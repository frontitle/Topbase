package warehouse

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

type Compiler func(queryir.Query) (queryir.Compiled, error)

type Service struct {
	Schedules core.ScheduleStore
	Runs      core.RunStore
	Tables    core.MaterializedTableStore
	Edges     core.LineageStore
	Questions core.QuestionStore
	Models    core.ModelStore
	Writer    core.WarehouseWriter
	Compile   Compiler
	Notify    func(title, body string)
	running   sync.Map
}

func (s *Service) Create(input core.Schedule, userID string) (core.Schedule, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Cron) == "" {
		return core.Schedule{}, fmt.Errorf("name and cron are required")
	}
	if strings.TrimSpace(input.QuestionID) == "" && strings.TrimSpace(input.ModelID) == "" {
		return core.Schedule{}, fmt.Errorf("question_id or model_id is required")
	}
	if _, err := ParseCron(input.Cron); err != nil {
		return core.Schedule{}, err
	}
	schema, table, err := core.ParseWarehouseTarget(input.MaterializeTo)
	if err != nil {
		return core.Schedule{}, err
	}
	if schema != "warehouse" && !input.ConfirmSourceWrite {
		return core.Schedule{}, fmt.Errorf("writing outside warehouse schema requires confirm_source_write")
	}
	if len(schema) > 63 || len(table) > 63 {
		return core.Schedule{}, fmt.Errorf("PostgreSQL 的目标 schema 和表名最长各为 63 字节")
	}
	input.MaterializeTo = schema + "." + table
	question, err := s.sourceQuestion(input)
	if err != nil {
		return core.Schedule{}, err
	}
	if input.DatabaseID == "" {
		input.DatabaseID = question.DatabaseID
		if input.DatabaseID == "" && question.QueryIR != nil {
			input.DatabaseID = question.QueryIR.Source.DatabaseID
		}
	}
	if input.DatabaseID == "" {
		return core.Schedule{}, fmt.Errorf("沉淀计划缺少目标数据库连接")
	}
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}
	if input.Strategy == "" {
		input.Strategy = "replace"
	}
	switch strings.ToLower(input.Strategy) {
	case "replace", "truncate_insert", "create_only":
	case "incremental":
		if strings.TrimSpace(input.WatermarkField) == "" {
			return core.Schedule{}, fmt.Errorf("watermark_field is required for incremental")
		}
		if err := queryir.CheckIdent("watermark", input.WatermarkField); err != nil {
			return core.Schedule{}, err
		}
	default:
		return core.Schedule{}, fmt.Errorf("unsupported strategy %q", input.Strategy)
	}
	input.ID = core.NewID("sch")
	input.Enabled = true
	input.CreatedBy = userID
	input.CreatedAt = time.Now().UTC()
	if err := s.Schedules.Create(input); err != nil {
		return core.Schedule{}, err
	}
	return input, nil
}

func (s *Service) List() ([]core.Schedule, error) {
	if s.Schedules == nil {
		return []core.Schedule{}, nil
	}
	return s.Schedules.List()
}

func (s *Service) ListTables() ([]core.MaterializedTable, error) {
	if s.Tables == nil {
		return []core.MaterializedTable{}, nil
	}
	return s.Tables.List()
}

func (s *Service) ListRuns(scheduleID string) ([]core.Run, error) {
	if s.Runs == nil {
		return []core.Run{}, nil
	}
	return s.Runs.List(scheduleID)
}

func (s *Service) ListLineage(entityType, id string) ([]core.LineageEdge, error) {
	if s.Edges == nil {
		return []core.LineageEdge{}, nil
	}
	return s.Edges.List(entityType, id)
}

func (s *Service) Run(ctx context.Context, scheduleID string) (core.Run, error) {
	if _, loaded := s.running.LoadOrStore(scheduleID, true); loaded {
		return core.Run{}, fmt.Errorf("schedule is already running")
	}
	defer s.running.Delete(scheduleID)

	schedule, err := s.Schedules.ByID(scheduleID)
	if err != nil {
		return core.Run{}, err
	}
	if s.Runs == nil || s.Tables == nil {
		return core.Run{}, fmt.Errorf("沉淀运行记录与目标表登记服务未配置")
	}
	previous, receipt, err := s.Runs.Unconfirmed(schedule.ID)
	if err == nil {
		if receipt != nil {
			// The writer confirmed the physical commit. Replay just the
			// application transaction, without querying or appending rows.
			recovered := previous
			recovered.Status, recovered.Error = "succeeded", ""
			recovered.FinishedAt, recovered.RowCount = receipt.LastRunAt, receipt.RowCount
			if err := s.Runs.Finish(recovered, receipt); err != nil {
				return previous, fmt.Errorf("恢复运行 %s 的登记失败: %w", previous.ID, err)
			}
			if s.Notify != nil {
				s.Notify("数仓物化已恢复", schedule.Name+" 的写入记录与水位已恢复")
			}
			return recovered, nil
		}
		return previous, fmt.Errorf("运行 %s 的写入尚未核对，请管理员核对 %s 的数据、水位与运行记录后恢复计划。%s", previous.ID, schedule.MaterializeTo, previous.Error)
	}
	if !errors.Is(err, core.ErrNotFound) {
		return core.Run{}, fmt.Errorf("检查上次写入状态失败: %w", err)
	}
	run := core.Run{ID: core.NewID("run"), ScheduleID: schedule.ID, Status: "running", StartedAt: time.Now().UTC()}
	if err := s.Runs.Create(run); err != nil {
		return core.Run{}, fmt.Errorf("record schedule run: %w", err)
	}
	question, err := s.sourceQuestion(schedule)
	if err != nil {
		return s.fail(run, schedule, err)
	}
	schema, table, err := core.ParseWarehouseTarget(schedule.MaterializeTo)
	if err != nil {
		return s.fail(run, schedule, err)
	}
	if schema != "warehouse" && !schedule.ConfirmSourceWrite {
		return s.fail(run, schedule, fmt.Errorf("writing outside warehouse schema requires confirm_source_write"))
	}
	watermark := ""
	requireNewTarget := schedule.ClaimedTarget
	if existing, err := s.Tables.ByTarget(schedule.DatabaseID, schema, table); err == nil {
		if schedule.ClaimedTarget && existing.ScheduleID != schedule.ID {
			return s.fail(run, schedule, core.ErrWarehouseTargetInUse)
		}
		requireNewTarget = false
		watermark = existing.Watermark
		if schedule.DatabaseID == "" {
			schedule.DatabaseID = existing.DatabaseID
		}
	} else if !errors.Is(err, core.ErrNotFound) {
		return s.fail(run, schedule, err)
	}
	sqlText, args, databaseID, err := s.compile(question, schedule, watermark)
	run.SQLCompiled = sqlText
	if err != nil {
		return s.fail(run, schedule, err)
	}
	if s.Writer == nil {
		return s.fail(run, schedule, fmt.Errorf("warehouse writer is not configured"))
	}
	run.Status = "writing"
	if err := s.Runs.Update(run); err != nil {
		return s.fail(run, schedule, fmt.Errorf("登记目标库写入意图失败，尚未写入目标表: %w", err))
	}
	result, err := s.Writer.Materialize(ctx, core.MaterializeSpec{
		RequireNewTarget: requireNewTarget,
		DatabaseID:       databaseID, Schema: schema, Table: table, Strategy: schedule.Strategy,
		SQL: sqlText, Args: args, WatermarkField: schedule.WatermarkField,
	})
	if errors.Is(err, core.ErrWarehouseCommitUnknown) {
		return s.needsReview(run, schedule, nil, err)
	}
	if err != nil {
		return s.fail(run, schedule, err)
	}
	now := time.Now().UTC()
	run.Status = "succeeded"
	run.RowCount = result.RowCount
	run.FinishedAt = &now
	mat := core.MaterializedTable{
		ID: core.NewID("mat"), DatabaseID: databaseID, Schema: schema, Name: table,
		ScheduleID: schedule.ID, QuestionID: question.ID, Warehouse: true,
		LastRunAt: &now, LastStatus: "succeeded", RowCount: result.RowCount, Watermark: result.Watermark,
	}
	if err := s.Runs.Finish(run, &mat); err != nil {
		return s.needsReview(run, schedule, &mat, fmt.Errorf("目标库已提交 %d 行，水位 %q，应用库登记未确认: %w", result.RowCount, result.Watermark, err))
	}
	if s.Notify != nil {
		s.Notify("数仓物化成功", fmt.Sprintf("%s 已写入 %s.%s，共 %d 行", schedule.Name, schema, table, result.RowCount))
	}
	return run, nil
}

func (s *Service) needsReview(run core.Run, schedule core.Schedule, receipt *core.MaterializedTable, cause error) (core.Run, error) {
	run.Status = "needs_review"
	run.FinishedAt = nil
	err := fmt.Errorf("运行 %s 需核对 %s，已暂停重写目标表: %w", run.ID, schedule.MaterializeTo, cause)
	if receipt != nil {
		err = fmt.Errorf("%w；重试将使用已确认的回执恢复登记", err)
	}
	run.Error = err.Error()
	if markErr := s.Runs.MarkUncertain(run, receipt); markErr != nil {
		return run, fmt.Errorf("写入回执未能登记，请管理员核对运行 %s 和目标 %s: %w", run.ID, schedule.MaterializeTo, errors.Join(cause, markErr))
	}
	return run, err
}

func (s *Service) sourceQuestion(schedule core.Schedule) (core.Question, error) {
	if schedule.ModelID != "" {
		if s.Models == nil {
			return core.Question{}, fmt.Errorf("model store is not configured")
		}
		model, err := s.Models.ByID(schedule.ModelID)
		if err != nil {
			return core.Question{}, err
		}
		return core.Question{ID: model.ID, Name: model.Name, QueryType: "queryir", DatabaseID: model.DatabaseID, QueryIR: model.QueryIR, NativeSQL: model.NativeSQL}, nil
	}
	if schedule.QuestionID == "" {
		return core.Question{}, fmt.Errorf("question_id or model_id is required")
	}
	return s.Questions.ByID(schedule.QuestionID)
}

func (s *Service) compile(question core.Question, schedule core.Schedule, watermark string) (string, []any, string, error) {
	databaseID := schedule.DatabaseID
	if databaseID == "" {
		databaseID = question.DatabaseID
	}
	if question.QueryType == "native" || (question.QueryIR == nil && question.NativeSQL != "") {
		sqlText, args, err := queryir.ApplyNative(question.NativeSQL, question.Parameters, map[string]any{})
		if err != nil {
			return "", nil, "", err
		}
		sqlText, args, err = applyWatermarkSQL(sqlText, args, schedule, watermark)
		return sqlText, args, databaseID, err
	}
	if question.QueryIR == nil {
		return "", nil, "", fmt.Errorf("question has no queryir")
	}
	if s.Compile == nil {
		return "", nil, "", fmt.Errorf("query compiler is not configured")
	}
	q := queryir.Clone(*question.QueryIR)
	if strings.EqualFold(schedule.Strategy, "incremental") && watermark != "" {
		q.Filters = append(q.Filters, queryir.Filter{Field: schedule.WatermarkField, Op: "gt", Value: watermark})
	}
	compiled, err := s.Compile(q)
	if err != nil {
		return "", nil, "", err
	}
	if databaseID == "" {
		databaseID = q.Source.DatabaseID
	}
	return compiled.SQL, compiled.Args, databaseID, nil
}

func applyWatermarkSQL(sqlText string, args []any, schedule core.Schedule, watermark string) (string, []any, error) {
	if !strings.EqualFold(schedule.Strategy, "incremental") || watermark == "" {
		return sqlText, args, nil
	}
	if err := queryir.CheckIdent("watermark", schedule.WatermarkField); err != nil {
		return "", nil, err
	}
	args = append(args, watermark)
	wrapped := "SELECT * FROM (" + sqlText + ") AS _wh WHERE " + queryir.Quote(schedule.WatermarkField) + " > $" + fmt.Sprintf("%d", len(args))
	return wrapped, args, nil
}

func (s *Service) fail(run core.Run, schedule core.Schedule, err error) (core.Run, error) {
	now := time.Now().UTC()
	run.Status = "failed"
	run.Error = err.Error()
	run.FinishedAt = &now
	if recordErr := s.Runs.Finish(run, nil); recordErr != nil {
		return run, errors.Join(err, fmt.Errorf("登记失败运行状态: %w", recordErr))
	}
	if s.Notify != nil {
		s.Notify("数仓物化失败", schedule.Name+"："+err.Error())
	}
	return run, err
}

package warehouse

import (
	"context"
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

func ParseTarget(raw string) (schema, table string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("materialize_to is required")
	}
	schema, table = "warehouse", raw
	if i := strings.LastIndex(raw, "."); i >= 0 {
		schema, table = raw[:i], raw[i+1:]
	}
	if !strings.HasPrefix(table, "wh_") {
		table = "wh_" + table
	}
	if err := queryir.CheckIdent("schema", schema); err != nil {
		return "", "", err
	}
	if err := queryir.CheckIdent("table", table); err != nil {
		return "", "", err
	}
	return schema, table, nil
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
	schema, _, err := ParseTarget(input.MaterializeTo)
	if err != nil {
		return core.Schedule{}, err
	}
	if schema != "warehouse" && !input.ConfirmSourceWrite {
		return core.Schedule{}, fmt.Errorf("writing outside warehouse schema requires confirm_source_write")
	}
	if input.QuestionID != "" {
		if _, err := s.Questions.ByID(input.QuestionID); err != nil {
			return core.Schedule{}, err
		}
	}
	if input.ModelID != "" {
		if s.Models == nil {
			return core.Schedule{}, fmt.Errorf("model store is not configured")
		}
		if _, err := s.Models.ByID(input.ModelID); err != nil {
			return core.Schedule{}, err
		}
	}
	if input.DatabaseID == "" && input.QuestionID != "" {
		question, _ := s.Questions.ByID(input.QuestionID)
		input.DatabaseID = question.DatabaseID
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
	question, err := s.sourceQuestion(schedule)
	if err != nil {
		return core.Run{}, err
	}
	schema, table, err := ParseTarget(schedule.MaterializeTo)
	if err != nil {
		return core.Run{}, err
	}
	if schema != "warehouse" && !schedule.ConfirmSourceWrite {
		return core.Run{}, fmt.Errorf("writing outside warehouse schema requires confirm_source_write")
	}
	watermark := ""
	if s.Tables != nil {
		if existing, err := s.Tables.ByTarget(schedule.DatabaseID, schema, table); err == nil {
			watermark = existing.Watermark
			if schedule.DatabaseID == "" {
				schedule.DatabaseID = existing.DatabaseID
			}
		}
	}
	sqlText, args, databaseID, err := s.compile(question, schedule, watermark)
	run := core.Run{
		ID: core.NewID("run"), ScheduleID: schedule.ID, Status: "running",
		SQLCompiled: sqlText, StartedAt: time.Now().UTC(),
	}
	if s.Runs != nil {
		_ = s.Runs.Create(run)
	}
	if err != nil {
		return s.fail(run, schedule, err)
	}
	if s.Writer == nil {
		return s.fail(run, schedule, fmt.Errorf("warehouse writer is not configured"))
	}
	result, err := s.Writer.Materialize(ctx, core.MaterializeSpec{
		DatabaseID: databaseID, Schema: schema, Table: table, Strategy: schedule.Strategy,
		SQL: sqlText, Args: args, WatermarkField: schedule.WatermarkField,
	})
	if err != nil {
		return s.fail(run, schedule, err)
	}
	now := time.Now().UTC()
	run.Status = "succeeded"
	run.RowCount = result.RowCount
	run.FinishedAt = &now
	if s.Runs != nil {
		_ = s.Runs.Update(run)
	}
	schedule.LastRunAt = &now
	_ = s.Schedules.Update(schedule)
	mat := core.MaterializedTable{
		ID: core.NewID("mat"), DatabaseID: databaseID, Schema: schema, Name: table,
		ScheduleID: schedule.ID, QuestionID: question.ID, Warehouse: true,
		LastRunAt: &now, LastStatus: "succeeded", RowCount: result.RowCount, Watermark: result.Watermark,
	}
	if s.Tables != nil {
		_ = s.Tables.Upsert(mat)
	}
	if s.Edges != nil {
		_ = s.Edges.Add(core.LineageEdge{FromType: "question", FromID: question.ID, ToType: "table", ToID: schema + "." + table})
		_ = s.Edges.Add(core.LineageEdge{FromType: "schedule", FromID: schedule.ID, ToType: "table", ToID: schema + "." + table})
	}
	if s.Notify != nil {
		s.Notify("数仓物化成功", fmt.Sprintf("%s 已写入 %s.%s，共 %d 行", schedule.Name, schema, table, result.RowCount))
	}
	return run, nil
}

func (s *Service) sourceQuestion(schedule core.Schedule) (core.Question, error) {
	if schedule.ModelID != "" && s.Models != nil {
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
	if s.Runs != nil {
		_ = s.Runs.Update(run)
	}
	schedule.LastRunAt = &now
	_ = s.Schedules.Update(schedule)
	if s.Notify != nil {
		s.Notify("数仓物化失败", schedule.Name+"："+err.Error())
	}
	return run, err
}

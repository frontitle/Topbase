package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/frontitle/Topbase/internal/core/queryir"
)

var ErrWarehouseTargetInUse = errors.New("已有沉淀计划或数据表使用此目标，请更换保存表名")
var ErrWarehouseCommitUnknown = errors.New("目标库提交结果未确认")

func ParseWarehouseTarget(raw string) (schema, table string, err error) {
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

type Schedule struct {
	ClaimedTarget      bool       `json:"-"`
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	QuestionID         string     `json:"question_id,omitempty"`
	ModelID            string     `json:"model_id,omitempty"`
	DatabaseID         string     `json:"database_id,omitempty"`
	Cron               string     `json:"cron"`
	Timezone           string     `json:"timezone,omitempty"`
	MaterializeTo      string     `json:"materialize_to"`
	Strategy           string     `json:"strategy,omitempty"`
	WatermarkField     string     `json:"watermark_field,omitempty"`
	ConfirmSourceWrite bool       `json:"confirm_source_write,omitempty"`
	Enabled            bool       `json:"enabled"`
	LastRunAt          *time.Time `json:"last_run_at,omitempty"`
	CreatedBy          string     `json:"created_by,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type ScheduleSummary struct {
	Schedule
	LastStatus string `json:"last_status,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

type Run struct {
	ID          string     `json:"id"`
	ScheduleID  string     `json:"schedule_id"`
	Status      string     `json:"status"`
	SQLCompiled string     `json:"sql_compiled,omitempty"`
	RowCount    int        `json:"row_count"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

type MaterializedTable struct {
	ID         string     `json:"id"`
	DatabaseID string     `json:"database_id"`
	Schema     string     `json:"schema"`
	Name       string     `json:"name"`
	ScheduleID string     `json:"schedule_id,omitempty"`
	QuestionID string     `json:"question_id,omitempty"`
	Warehouse  bool       `json:"warehouse"`
	LastRunAt  *time.Time `json:"last_run_at,omitempty"`
	LastStatus string     `json:"last_status,omitempty"`
	RowCount   int        `json:"row_count,omitempty"`
	Watermark  string     `json:"watermark,omitempty"`
}

type LineageEdge struct {
	FromType string `json:"from_type"`
	FromID   string `json:"from_id"`
	ToType   string `json:"to_type"`
	ToID     string `json:"to_id"`
}

type ScheduleProposal struct {
	Name            string `json:"name"`
	QuestionID      string `json:"question_id"`
	Cron            string `json:"cron"`
	Timezone        string `json:"timezone"`
	MaterializeTo   string `json:"materialize_to"`
	Strategy        string `json:"strategy"`
	WatermarkField  string `json:"watermark_field,omitempty"`
	RequiresConfirm bool   `json:"requires_confirm"`
	Rationale       string `json:"rationale,omitempty"`
}

type MaterializeSpec struct {
	RequireNewTarget bool
	DatabaseID       string
	Schema           string
	Table            string
	Strategy         string
	SQL              string
	Args             []any
	WatermarkField   string
}

type MaterializeResult struct {
	RowCount  int
	Watermark string
}

type WarehouseWriter interface {
	Materialize(ctx context.Context, spec MaterializeSpec) (MaterializeResult, error)
}

type ScheduleStore interface {
	Create(Schedule) error
	Update(Schedule) error
	ByID(id string) (Schedule, error)
	List() ([]Schedule, error)
}

type RunStore interface {
	Create(Run) error
	Update(Run) error
	List(scheduleID string) ([]Run, error)
	Unconfirmed(scheduleID string) (Run, *MaterializedTable, error)
	// Finish commits the final run, schedule timestamp and (when supplied)
	// materialization metadata and lineage together.
	Finish(Run, *MaterializedTable) error
	// MarkUncertain may only change a run whose external write is pending.
	MarkUncertain(Run, *MaterializedTable) error
}

type MaterializedTableStore interface {
	Upsert(MaterializedTable) error
	List() ([]MaterializedTable, error)
	ListByDatabase(databaseID string) ([]MaterializedTable, error)
	ByTarget(databaseID, schema, name string) (MaterializedTable, error)
}

type LineageStore interface {
	Add(LineageEdge) error
	List(entityType, id string) ([]LineageEdge, error)
}

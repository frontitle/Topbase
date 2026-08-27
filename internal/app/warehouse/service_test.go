package warehouse

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/topbase/topbase/internal/adapters"
	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

type memSchedules struct{ items map[string]core.Schedule }

func (m *memSchedules) Create(item core.Schedule) error {
	if m.items == nil {
		m.items = map[string]core.Schedule{}
	}
	m.items[item.ID] = item
	return nil
}
func (m *memSchedules) Update(item core.Schedule) error { m.items[item.ID] = item; return nil }
func (m *memSchedules) ByID(id string) (core.Schedule, error) {
	item, ok := m.items[id]
	if !ok {
		return core.Schedule{}, core.ErrNotFound
	}
	return item, nil
}
func (m *memSchedules) List() ([]core.Schedule, error) {
	out := []core.Schedule{}
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}

type memRuns struct{ items []core.Run }

func (m *memRuns) Create(item core.Run) error { m.items = append(m.items, item); return nil }
func (m *memRuns) Update(item core.Run) error {
	for i, existing := range m.items {
		if existing.ID == item.ID {
			m.items[i] = item
			return nil
		}
	}
	m.items = append(m.items, item)
	return nil
}
func (m *memRuns) List(string) ([]core.Run, error) { return m.items, nil }

type memTables struct{ items []core.MaterializedTable }

func (m *memTables) Upsert(item core.MaterializedTable) error {
	for i, existing := range m.items {
		if existing.DatabaseID == item.DatabaseID && existing.Schema == item.Schema && existing.Name == item.Name {
			m.items[i] = item
			return nil
		}
	}
	m.items = append(m.items, item)
	return nil
}
func (m *memTables) List() ([]core.MaterializedTable, error) { return m.items, nil }
func (m *memTables) ListByDatabase(string) ([]core.MaterializedTable, error) {
	return m.items, nil
}
func (m *memTables) ByTarget(databaseID, schema, name string) (core.MaterializedTable, error) {
	for _, item := range m.items {
		if item.DatabaseID == databaseID && item.Schema == schema && item.Name == name {
			return item, nil
		}
	}
	return core.MaterializedTable{}, core.ErrNotFound
}

type memLineage struct{ items []core.LineageEdge }

func (m *memLineage) Add(edge core.LineageEdge) error {
	m.items = append(m.items, edge)
	return nil
}
func (m *memLineage) List(string, string) ([]core.LineageEdge, error) { return m.items, nil }

type memQuestions struct{ item core.Question }

func (m memQuestions) Create(core.Question) error           { return nil }
func (m memQuestions) Update(core.Question) error           { return nil }
func (m memQuestions) ByID(string) (core.Question, error)   { return m.item, nil }
func (m memQuestions) List(bool) ([]core.Question, error)   { return []core.Question{m.item}, nil }
func (m memQuestions) SetArchived(string, *time.Time) error { return nil }

type memWriter struct {
	spec core.MaterializeSpec
}

func (m *memWriter) Materialize(_ context.Context, spec core.MaterializeSpec) (core.MaterializeResult, error) {
	m.spec = spec
	result := core.MaterializeResult{RowCount: 3}
	if spec.WatermarkField != "" {
		result.Watermark = "2026-08-17"
	}
	return result, nil
}

func TestRunMaterializesQuestion(t *testing.T) {
	question := core.Question{
		ID: "qst_1", Name: "日 GMV", QueryType: "queryir", DatabaseID: "pg_1",
		QueryIR: &queryir.Query{
			Version:      1,
			Source:       queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
			Aggregations: []queryir.Aggregation{{Fn: "sum", Field: "amount"}},
			GroupBy:      []queryir.Breakout{{Field: "created_at", Temporal: "day"}},
		},
	}
	writer := &memWriter{}
	tables := &memTables{}
	svc := &Service{
		Schedules: &memSchedules{}, Runs: &memRuns{}, Tables: tables, Edges: &memLineage{},
		Questions: memQuestions{item: question}, Writer: writer, Compile: adapters.CompilePostgresWarehouse,
	}
	schedule, err := svc.Create(core.Schedule{
		Name: "日 GMV", QuestionID: "qst_1", Cron: "0 9 * * *", MaterializeTo: "warehouse.wh_gmv_daily",
	}, "user_1")
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.Run(context.Background(), schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "succeeded" || run.RowCount != 3 {
		t.Fatalf("%+v", run)
	}
	if writer.spec.Schema != "warehouse" || writer.spec.Table != "wh_gmv_daily" {
		t.Fatalf("target %+v", writer.spec)
	}
	if strings.Contains(writer.spec.SQL, "LIMIT") {
		t.Fatalf("warehouse sql should not limit: %s", writer.spec.SQL)
	}
	if len(tables.items) != 1 || !tables.items[0].Warehouse {
		t.Fatalf("tables %+v", tables.items)
	}
}

func TestDueCronHour(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 8, 17, 9, 0, 10, 0, loc)
	if !Due("0 9 * * *", "Asia/Shanghai", now, nil) {
		t.Fatal("expected due at 09:00")
	}
	if Due("0 9 * * *", "Asia/Shanghai", now, &now) {
		t.Fatal("same minute should not re-run")
	}
}

func TestProposeScheduleHour(t *testing.T) {
	proposal := Propose(core.Question{ID: "qst_1", Name: "GMV daily"}, "每天 8 点写入数仓")
	if proposal.Cron != "0 8 * * *" || !proposal.RequiresConfirm {
		t.Fatalf("%+v", proposal)
	}
	if proposal.MaterializeTo != "warehouse.wh_gmvdaily" && !strings.Contains(proposal.MaterializeTo, "wh_") {
		t.Fatalf("target %s", proposal.MaterializeTo)
	}
	incremental := Propose(core.Question{ID: "qst_1", Name: "GMV daily"}, "每天 9 点增量写入数仓")
	if incremental.Strategy != "incremental" || incremental.WatermarkField != "created_at" {
		t.Fatalf("%+v", incremental)
	}
}

func TestCreateRejectsSourceWriteWithoutConfirm(t *testing.T) {
	question := core.Question{ID: "qst_1", Name: "订单", QueryType: "native", DatabaseID: "pg_1", NativeSQL: "SELECT 1"}
	svc := &Service{Schedules: &memSchedules{}, Questions: memQuestions{item: question}}
	_, err := svc.Create(core.Schedule{
		Name: "写源库", QuestionID: "qst_1", Cron: "0 9 * * *", MaterializeTo: "public.orders",
	}, "user_1")
	if err == nil || !strings.Contains(err.Error(), "confirm_source_write") {
		t.Fatalf("expected confirm_source_write, got %v", err)
	}
	if _, err := svc.Create(core.Schedule{
		Name: "写源库", QuestionID: "qst_1", Cron: "0 9 * * *", MaterializeTo: "public.orders", ConfirmSourceWrite: true,
	}, "user_1"); err != nil {
		t.Fatal(err)
	}
}

func TestIncrementalRunAppliesWatermark(t *testing.T) {
	question := core.Question{
		ID: "qst_1", Name: "日订单", QueryType: "queryir", DatabaseID: "pg_1",
		QueryIR: &queryir.Query{
			Version: 1,
			Source:  queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
			Fields:  []string{"id", "created_at"},
		},
	}
	writer := &memWriter{}
	tables := &memTables{}
	svc := &Service{
		Schedules: &memSchedules{}, Runs: &memRuns{}, Tables: tables,
		Questions: memQuestions{item: question}, Writer: writer, Compile: adapters.CompilePostgresWarehouse,
	}
	schedule, err := svc.Create(core.Schedule{
		Name: "日订单", QuestionID: "qst_1", Cron: "0 9 * * *", MaterializeTo: "warehouse.wh_orders",
		Strategy: "incremental", WatermarkField: "created_at",
	}, "user_1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Run(context.Background(), schedule.ID); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(writer.spec.SQL, ">") {
		t.Fatalf("first run should be full load: %s", writer.spec.SQL)
	}
	if tables.items[0].Watermark != "2026-08-17" {
		t.Fatalf("watermark %+v", tables.items[0])
	}
	if _, err := svc.Run(context.Background(), schedule.ID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(writer.spec.SQL, ">") {
		t.Fatalf("second run should filter watermark: %s", writer.spec.SQL)
	}
}

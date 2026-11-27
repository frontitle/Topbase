package appdb

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	appwarehouse "github.com/frontitle/Topbase/internal/app/warehouse"
	"github.com/frontitle/Topbase/internal/core"
)

type countingWarehouseWriter struct {
	calls     int
	uncertain bool
	last      core.MaterializeSpec
}

func (w *countingWarehouseWriter) Materialize(_ context.Context, spec core.MaterializeSpec) (core.MaterializeResult, error) {
	w.calls++
	w.last = spec
	if w.uncertain {
		return core.MaterializeResult{}, core.ErrWarehouseCommitUnknown
	}
	return core.MaterializeResult{RowCount: w.calls, Watermark: strconv.Itoa(w.calls)}, nil
}

func TestWarehouseFailureAndRecoveryDoNotRepeatPhysicalWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "warehouse.db")
	testWarehouseRecovery(t, func() (*Store, error) { return Open(path) })
}

func TestWarehouseProductionRecovery(t *testing.T) {
	engine, dsn := Engine(os.Getenv("TOPBASE_TEST_WAREHOUSE_ENGINE")), os.Getenv("TOPBASE_TEST_WAREHOUSE_DSN")
	if engine == "" || dsn == "" {
		t.Skip("set TOPBASE_TEST_WAREHOUSE_ENGINE and TOPBASE_TEST_WAREHOUSE_DSN to an empty fixture database")
	}
	testWarehouseRecovery(t, func() (*Store, error) {
		return OpenConfig(Config{Engine: engine, DSN: dsn, AppVersion: "warehouse-recovery-test"})
	})
}

func testWarehouseRecovery(t *testing.T, newStore func() (*Store, error)) {
	t.Helper()
	open := func() *Store {
		store, err := newStore()
		if err != nil {
			t.Fatal(err)
		}
		return store
	}
	store := open()
	defer func() { store.Close() }()
	writer := &countingWarehouseWriter{}
	notices := 0
	service := func() *appwarehouse.Service {
		return &appwarehouse.Service{
			Schedules: store.Schedules(), Runs: store.Runs(), Tables: store.Materialized(), Questions: store.Questions(), Writer: writer,
			Notify: func(string, string) { notices++ },
		}
	}
	if err := store.CreateQuestion(core.Question{ID: "q", Name: "orders", DatabaseID: "db", QueryType: "native", NativeSQL: "SELECT 1 AS n"}); err != nil {
		t.Fatal(err)
	}
	plan, err := service().Create(core.Schedule{Name: "orders", QuestionID: "q", Cron: "0 * * * *", MaterializeTo: "orders", Strategy: "incremental", WatermarkField: "n"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	run := func() (core.Run, error) { return service().Run(context.Background(), plan.ID) }
	initial, err := run()
	if err != nil || initial.Status != "succeeded" || writer.calls != 1 || !writer.last.RequireNewTarget || notices != 1 {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	exec := func(statement string) {
		if _, err := store.db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	var removeTrigger string
	switch store.db.engine {
	case EnginePostgres:
		exec(`CREATE FUNCTION reject_warehouse_success() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='succeeded' THEN RAISE EXCEPTION 'injected completion failure'; END IF; RETURN NEW; END; $$`)
		exec(`CREATE TRIGGER reject_success BEFORE UPDATE ON schedule_runs FOR EACH ROW EXECUTE FUNCTION reject_warehouse_success()`)
		removeTrigger = "DROP TRIGGER reject_success ON schedule_runs"
	case EngineMySQL:
		exec(`CREATE TRIGGER reject_success BEFORE UPDATE ON schedule_runs FOR EACH ROW BEGIN IF NEW.status='succeeded' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected completion failure'; END IF; END`)
		removeTrigger = "DROP TRIGGER reject_success"
	default:
		exec(`CREATE TRIGGER reject_success BEFORE UPDATE ON schedule_runs WHEN NEW.status='succeeded' BEGIN SELECT RAISE(ABORT,'injected completion failure'); END`)
		removeTrigger = "DROP TRIGGER reject_success"
	}
	pending, err := run()
	if err == nil || pending.Status != "needs_review" || writer.calls != 2 || notices != 1 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	mat, err := store.MaterializedTableByTarget("db", "warehouse", "wh_orders")
	if err != nil || mat.Watermark != "1" || mat.RowCount != 1 {
		t.Fatalf("partial metadata commit: %+v %v", mat, err)
	}
	savedPlan, err := store.ScheduleByID(plan.ID)
	if err != nil || savedPlan.LastRunAt == nil || !savedPlan.LastRunAt.Equal(*initial.FinishedAt) {
		t.Fatalf("partial plan timestamp: %+v %v", savedPlan, err)
	}
	if _, err = run(); err == nil || writer.calls != 2 || notices != 1 {
		t.Fatalf("recovery while fault persists: %v", err)
	}
	store.Close()
	store = open()
	exec(removeTrigger)
	recovered, err := run()
	if err != nil || recovered.Status != "succeeded" || recovered.ID != pending.ID || writer.calls != 2 || notices != 2 {
		t.Fatalf("confirmed commit replayed: %+v %v, writes=%d notices=%d", recovered, err, writer.calls, notices)
	}
	mat, err = store.MaterializedTableByTarget("db", "warehouse", "wh_orders")
	if err != nil || mat.Watermark != "2" || mat.RowCount != 2 {
		t.Fatalf("recovered table: %+v %v", mat, err)
	}
	edges, err := store.ListLineage("table", "warehouse.wh_orders")
	if err != nil || len(edges) != 2 {
		t.Fatalf("refresh duplicated lineage: %+v err=%v", edges, err)
	}
	writer.uncertain = true
	unknown, err := run()
	if !errors.Is(err, core.ErrWarehouseCommitUnknown) || unknown.Status != "needs_review" || writer.calls != 3 || notices != 2 {
		t.Fatalf("unknown write: %+v %v", unknown, err)
	}
	if len(writer.last.Args) != 1 || writer.last.Args[0] != "2" {
		t.Fatalf("recovered watermark was not used: %+v", writer.last.Args)
	}
	store.Close()
	store = open()
	again, err := run()
	if err == nil || again.ID != unknown.ID || writer.calls != 3 || notices != 2 {
		t.Fatalf("unconfirmed physical commit was replayed: %+v %v", again, err)
	}
	if store.db.engine == EngineSQLite {
		testWarehouseMissingReceipt(t, service(), store, writer)
	}
}

func testWarehouseMissingReceipt(t *testing.T, service *appwarehouse.Service, store *Store, writer *countingWarehouseWriter) {
	t.Helper()
	plan, err := service.Create(core.Schedule{Name: "receipt failure", QuestionID: "q", Cron: "0 * * * *", MaterializeTo: "receipt_failure"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	writer.uncertain = false
	before := writer.calls
	exec := func(stmt string) {
		if _, err := store.db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	exec(`CREATE TRIGGER reject_intent BEFORE INSERT ON settings WHEN NEW.key LIKE 'warehouse_active_run:%' BEGIN SELECT RAISE(ABORT,'intent unavailable'); END`)
	intent, err := service.Run(context.Background(), plan.ID)
	if err == nil || intent.Status != "failed" || writer.calls != before {
		t.Fatalf("intent failure: %+v %v", intent, err)
	}
	history, err := store.ListRuns(plan.ID)
	if err != nil || len(history) != 1 || history[0].Status != "failed" {
		t.Fatalf("intent failure history: %+v %v", history, err)
	}
	exec("DROP TRIGGER reject_intent")
	exec(`CREATE TRIGGER reject_finished BEFORE UPDATE ON schedule_runs WHEN NEW.status='succeeded' BEGIN SELECT RAISE(ABORT,'finish unavailable'); END`)
	exec(`CREATE TRIGGER reject_receipt BEFORE INSERT ON settings WHEN NEW.key LIKE 'warehouse_run_receipt:%' BEGIN SELECT RAISE(ABORT,'receipt unavailable'); END`)
	pending, err := service.Run(context.Background(), plan.ID)
	if err == nil || writer.calls != before+1 {
		t.Fatalf("missing receipt: %+v %v", pending, err)
	}
	exec("DROP TRIGGER reject_finished")
	exec("DROP TRIGGER reject_receipt")
	blocked, err := service.Run(context.Background(), plan.ID)
	if err == nil || blocked.ID != pending.ID || writer.calls != before+1 {
		t.Fatalf("replayed without receipt: %+v %v", blocked, err)
	}
}

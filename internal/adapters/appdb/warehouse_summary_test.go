package appdb

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/frontitle/Topbase/internal/core"
)

// Called only by the opt-in production migration integration test on its
// freshly populated fixture database. Exercise upgrade DDL as well as the
// new-table join against the old schedules table.
func testWarehouseUpgradeOnProductionEngine(t *testing.T, store *Store) {
	t.Helper()
	for _, statement := range []string{
		"DROP TABLE warehouse_target_claims",
		"DELETE FROM schema_migrations WHERE version=18",
	} {
		if _, err := store.db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := applyMigrations(context.Background(), store.db, "integration"); err != nil {
		t.Fatalf("upgrade warehouse target registry: %v", err)
	}
	plan := core.Schedule{ID: "warehouse-upgrade", Name: "Migration check", DatabaseID: "db1", QuestionID: "q", Cron: "0 * * * *", MaterializeTo: "warehouse.wh_orders", CreatedAt: time.Now()}
	if err := store.CreateSchedule(plan); err != nil {
		t.Fatalf("create plan after upgrade: %v", err)
	}
	plan.ID = "warehouse-duplicate"
	plan.MaterializeTo = "orders"
	if err := store.CreateSchedule(plan); !errors.Is(err, core.ErrWarehouseTargetInUse) {
		t.Fatalf("expected target conflict after upgrade, got %v", err)
	}
}

func TestWarehouseSummariesIncludeUnmaterializedPlansAndLatestAttempt(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "warehouse.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	for _, id := range []string{"empty", "retry"} {
		err := store.CreateSchedule(core.Schedule{ID: id, Name: id, QuestionID: "orders", Cron: "0 * * * *",
			Timezone: "UTC", MaterializeTo: "warehouse.wh_" + id, Strategy: "replace", CreatedAt: now})
		if err != nil {
			t.Fatal(err)
		}
	}
	failed := core.Run{ID: "z", ScheduleID: "retry", StartedAt: now, Status: "failed", Error: "timeout"}
	if err := store.CreateRun(failed); err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListScheduleSummaries()
	if err != nil || len(rows) != 2 || rows[0].LastStatus != "" || rows[1].LastError != "timeout" {
		t.Fatalf("before retry: rows=%+v err=%v", rows, err)
	}
	retry := core.Run{ID: "a", ScheduleID: "retry", StartedAt: now.Add(time.Millisecond), Status: "running"}
	if err := store.CreateRun(retry); err != nil {
		t.Fatal(err)
	}
	finished := now.Add(2 * time.Millisecond)
	retry.Status, retry.FinishedAt = "succeeded", &finished
	if err := store.UpdateRun(retry); err != nil {
		t.Fatal(err)
	}
	runs, err := store.ListRuns("retry")
	if err != nil || len(runs) != 2 || runs[0].FinishedAt == nil || runs[0].FinishedAt.Sub(runs[0].StartedAt) != time.Millisecond {
		t.Fatalf("retry duration: %+v %v", runs, err)
	}
	rows, err = store.ListScheduleSummaries()
	if err != nil || len(rows) != 2 || rows[1].LastStatus != "succeeded" || rows[1].LastError != "" {
		t.Fatalf("after retry: rows=%+v err=%v", rows, err)
	}
}

func TestWarehouseTargetsAreUniquePerConnectionAcrossConcurrentCreators(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claims.db")
	open := func() *Store {
		store, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { store.Close() })
		return store
	}
	one, two := open(), open()
	plan := core.Schedule{ID: "first", DatabaseID: "db1", MaterializeTo: "orders", CreatedAt: time.Now()}
	start, results := make(chan struct{}), make(chan error, 2)
	go func() { <-start; results <- one.CreateSchedule(plan) }()
	go func() {
		other := plan
		other.ID = "second"
		other.MaterializeTo = "warehouse.wh_orders"
		<-start
		results <- two.CreateSchedule(other)
	}()
	close(start)
	var successes, conflicts int
	for range 2 {
		switch err := <-results; {
		case err == nil:
			successes++
		case errors.Is(err, core.ErrWarehouseTargetInUse):
			conflicts++
		default:
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflict=%d", successes, conflicts)
	}
	plan.ID = "separate"
	plan.DatabaseID = "db2"
	if err := one.CreateSchedule(plan); err != nil {
		t.Fatal(err)
	}
	saved, err := two.ScheduleByID("separate")
	if err != nil || !saved.ClaimedTarget {
		t.Fatalf("claim lookup: %+v %v", saved, err)
	}
}

func TestLegacySharedTargetSurvivesAndBlocksAnotherPlan(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// Before claims, both of these target spellings wrote the same table.
	for _, target := range []string{"orders", "warehouse.wh_orders"} {
		_, err := store.db.Exec(`INSERT INTO schedules(id,name,question_id,database_id,cron,materialize_to,created_at) VALUES(?,?,?,?,?,?,?)`, target, target, "q", "db1", "0 * * * *", target, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			t.Fatal(err)
		}
	}
	plan := core.Schedule{ID: "another", DatabaseID: "db1", MaterializeTo: "wh_orders"}
	if err := store.CreateSchedule(plan); !errors.Is(err, core.ErrWarehouseTargetInUse) {
		t.Fatalf("expected conflict, got %v", err)
	}
	for _, id := range []string{"orders", "warehouse.wh_orders"} {
		saved, err := store.ScheduleByID(id)
		if err != nil || saved.ClaimedTarget || saved.MaterializeTo != id {
			t.Fatalf("legacy plan changed: %+v %v", saved, err)
		}
	}
	plan.MaterializeTo = "different"
	if err := store.CreateSchedule(plan); err != nil {
		t.Fatalf("failed reservation was not rolled back: %v", err)
	}
}

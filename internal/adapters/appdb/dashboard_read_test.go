package appdb

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/frontitle/Topbase/internal/core"
)

func dashboardReadFixture(t testing.TB, count int) *Store {
	t.Helper()
	store, err := OpenConfig(Config{Engine: EngineSQLite, Path: filepath.Join(t.TempDir(), "app.db"), MaxOpenConns: 8, MaxIdleConns: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	for i := range count {
		id := fmt.Sprintf("board-%04d", i)
		board := core.Dashboard{
			ID: id, PublicUUID: "public-" + id, Name: id, CreatedAt: time.Unix(int64(i), 0),
			Tabs:    []core.DashboardTab{{ID: id + "-later", Name: "Later", Position: 1}, {ID: id + "-first", Name: "First", Position: 0}},
			Cards:   []core.DashboardCard{{ID: id + "-card", TabID: id + "-first", Type: "text", Body: id, Layout: core.Layout{W: 4}}},
			Filters: []core.DashboardFilter{{ID: id + "-filter", Name: id, Type: "text", Mappings: []core.FilterMapping{{CardID: id + "-card", Field: "name"}}}},
		}
		if err := store.CreateDashboard(board); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

func TestDashboardReadScope(t *testing.T) {
	store := dashboardReadFixture(t, 260)
	now := time.Now()
	if err := store.SetDashboardArchived("board-0000", &now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DashboardByID("board-0000"); err != nil {
		t.Fatalf("archived dashboard must remain accessible by internal id: %v", err)
	}
	if _, err := store.DashboardByPublicUUID("public-board-0000"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("archived public dashboard: %v", err)
	}
	if _, err := store.DashboardByID("unknown"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("unknown dashboard: %v", err)
	}
	items, err := store.ListDashboards(false)
	if err != nil || len(items) != 259 {
		t.Fatalf("list: %d rows, %v", len(items), err)
	}
	for i, item := range items {
		id := fmt.Sprintf("board-%04d", 259-i)
		if item.ID != id || len(item.Tabs) != 2 || item.Tabs[0].ID != id+"-first" || item.Tabs[1].Position != 1 ||
			len(item.Cards) != 1 || item.Cards[0].Body != id || item.Cards[0].Layout.W != 4 ||
			len(item.Filters) != 1 || len(item.Filters[0].Mappings) != 1 || item.Filters[0].Mappings[0].CardID != id+"-card" {
			t.Fatalf("incorrect dashboard graph for %s: %+v", id, item)
		}
	}
	board, err := store.DashboardByPublicUUID("public-board-0001")
	if err != nil || board.ID != "board-0001" || len(board.Cards) != 1 {
		t.Fatalf("active public dashboard: %+v, %v", board, err)
	}
	// An unreadable row in another board must not prevent opening this one.
	// This also detects a regression to loading every board for a detail read.
	if _, err := store.db.Exec("UPDATE dashboard_tabs SET position='invalid' WHERE dashboard_id='board-0259'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DashboardByID("board-0001"); err != nil {
		t.Fatalf("detail read decoded an unrelated board: %v", err)
	}
}

func BenchmarkDashboardRead(b *testing.B) {
	store := dashboardReadFixture(b, 200)
	b.Run("DetailParallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := store.DashboardByID("board-0100"); err != nil {
					b.Error(err)
					return
				}
			}
		})
	})
	b.Run("List", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := store.ListDashboards(false); err != nil {
				b.Fatal(err)
			}
		}
	})
}

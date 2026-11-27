package appdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/frontitle/Topbase/internal/core"
)

func TestUploadedTablePreviewPreservesCompleteRowsWithinBudget(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	long := strings.Repeat("数", 65536)
	many := make([][]string, 100)
	for i := range many {
		many[i] = []string{fmt.Sprint(i)}
	}
	for _, tc := range []struct {
		name    string
		data    [][]string
		want    int
		limited bool
	}{
		{"empty", nil, 0, false},
		{"short", [][]string{{"伦敦", "a,b"}, {"杭州", "第二行"}}, 2, false},
		{"many", many, 50, false},
		{"large-first", [][]string{{long}}, 0, true},
		{"large-second", [][]string{{"首行"}, {long}}, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := store.SaveUploadedTable(core.UploadedTable{ID: tc.name, Name: tc.name, FileName: "qa.xlsx", SheetName: "Sheet", CreatedAt: time.Now().UTC(), RowCount: len(tc.data)}, tc.data)
			if err != nil {
				t.Fatal(err)
			}
			got, err := store.PreviewUploadedTable(context.Background(), tc.name)
			if err != nil || got.TotalRows != len(tc.data) || got.ByteLimited != tc.limited || len(got.Rows) != tc.want {
				t.Fatalf("rows=%d total=%d byteLimited=%t err=%v", len(got.Rows), got.TotalRows, got.ByteLimited, err)
			}
			for i, row := range got.Rows {
				if !slices.Equal(row, tc.data[i]) {
					t.Fatalf("preview differed at row %d", i)
				}
			}
		})
	}
	if _, err := store.PreviewUploadedTable(context.Background(), "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing upload: %v", err)
	}
}

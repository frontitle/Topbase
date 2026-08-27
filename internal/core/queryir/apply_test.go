package queryir

import (
	"testing"
	"time"
)

func TestApplyMappedFiltersDateRange(t *testing.T) {
	q := Query{
		Version: 1,
		Source:  Source{DatabaseID: "pg_1", Table: &TableRef{Schema: "public", Name: "orders"}},
	}
	out, err := ApplyMappedFilters(q, []MappedValue{{
		Field: "created_at", Type: "date", Value: map[string]any{"start": "2026-01-01", "end": "2026-01-31"},
	}}, time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Filters) != 2 || out.Filters[0].Op != "gte" || out.Filters[1].Op != "lte" {
		t.Fatalf("filters %+v", out.Filters)
	}
	if len(q.Filters) != 0 {
		t.Fatal("original query should stay unchanged")
	}
}

func TestApplyMappedFiltersRelativeDays(t *testing.T) {
	q := Query{Version: 1, Source: Source{DatabaseID: "pg_1", Table: &TableRef{Schema: "public", Name: "orders"}}}
	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	out, err := ApplyMappedFilters(q, []MappedValue{{Field: "created_at", Type: "date", Value: "past_7_days"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if out.Filters[0].Value != "2026-08-10" || out.Filters[1].Value != "2026-08-17" {
		t.Fatalf("relative date %+v", out.Filters)
	}
}

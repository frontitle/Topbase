package queryir

import (
	"strings"
	"testing"
)

func TestDrillRecordsAndFilter(t *testing.T) {
	q := Query{
		Version:      1,
		Source:       Source{DatabaseID: "pg_1", Table: &TableRef{Schema: "public", Name: "orders"}},
		Aggregations: []Aggregation{{Fn: "count"}},
		GroupBy:      []Breakout{{Field: "status"}},
	}
	out, err := Drill(q, DrillRequest{Kind: "records", Values: map[string]any{"status": "paid"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Aggregations) != 0 || len(out.Filters) != 1 || out.Filters[0].Value != "paid" {
		t.Fatalf("%+v", out)
	}
	filtered, err := Drill(q, DrillRequest{Kind: "filter", Field: "status", Value: "paid"})
	if err != nil || len(filtered.Aggregations) != 1 {
		t.Fatalf("filter drill should keep aggregation: %v %+v", err, filtered)
	}
}

func TestApplyNativeFieldFilter(t *testing.T) {
	sql, args, err := ApplyNative(
		`SELECT * FROM orders WHERE {{created_at}} AND status = {{status}}`,
		[]Parameter{{Name: "created_at", Type: "date", Field: "created_at"}, {Name: "status", Type: "text"}},
		map[string]any{"created_at": map[string]any{"start": "2026-01-01", "end": "2026-01-31"}, "status": "paid"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, `"created_at" >= $1`) || !strings.Contains(sql, `$3`) {
		t.Fatalf("sql %s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("args %v", args)
	}
}

func TestApplyNativeOptionalOmitted(t *testing.T) {
	sql, args, err := ApplyNative(`SELECT * FROM orders WHERE 1=1 [[ AND region = {{region}} ]]`, []Parameter{{Name: "region"}}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sql, "region") || len(args) != 0 {
		t.Fatalf("optional should drop, got %s %v", sql, args)
	}
}

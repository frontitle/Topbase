package viz

import (
	"testing"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func TestInferScalarAndBar(t *testing.T) {
	scalar := Infer(queryir.Query{Aggregations: []queryir.Aggregation{{Fn: "count"}}})
	if scalar.Type != "scalar" || len(scalar.Y) != 1 {
		t.Fatalf("scalar %+v", scalar)
	}
	bar := Infer(queryir.Query{
		Aggregations: []queryir.Aggregation{{Fn: "sum", Field: "amount", Alias: "amount"}},
		GroupBy:      []queryir.Breakout{{Field: "status"}},
	})
	if bar.Type != "bar" || bar.X != "status" || bar.Y[0] != "amount" {
		t.Fatalf("bar %+v", bar)
	}
	line := Infer(queryir.Query{
		Aggregations: []queryir.Aggregation{{Fn: "count"}},
		GroupBy:      []queryir.Breakout{{Field: "created_at", Temporal: "day"}},
	})
	if line.Type != "line" || line.X != "created_at_day" {
		t.Fatalf("line %+v", line)
	}
}

func TestMergePrefersSavedTypeAndFillsAxes(t *testing.T) {
	saved := &core.ChartSpec{Type: "pie", X: "status"}
	got := Merge(saved, core.ChartSpec{Type: "bar", X: "status", Y: []string{"count"}})
	if got.Type != "pie" || got.X != "status" || len(got.Y) != 1 || got.Y[0] != "count" {
		t.Fatalf("%+v", got)
	}
	color := "#509EE3"
	savedColor := &core.ChartSpec{
		Type:   "bar",
		Series: map[string]core.ChartSeriesStyle{"amount": {Color: color, Title: "金额"}},
	}
	merged := Merge(savedColor, core.ChartSpec{Type: "bar", X: "status", Y: []string{"amount"}})
	if merged.Series["amount"].Color != color || merged.Series["amount"].Title != "金额" {
		t.Fatalf("series style dropped: %+v", merged.Series)
	}
	if Merge(nil, core.ChartSpec{Type: "table"}).Type != "table" {
		t.Fatal("nil saved should keep inferred")
	}
	hidden := false
	fromQuestion := core.ChartSpec{
		Type:   "table",
		Search: "已完成",
		Columns: map[string]core.ChartColumnStyle{
			"status": {Visible: &hidden, Filter: "=done"},
		},
	}
	cardOnlyType := Merge(&core.ChartSpec{Type: "table"}, fromQuestion)
	if cardOnlyType.Search != "已完成" || cardOnlyType.Columns["status"].Filter != "=done" {
		t.Fatalf("table view should fill from question: %+v", cardOnlyType)
	}
}

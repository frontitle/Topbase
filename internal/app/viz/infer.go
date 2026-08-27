package viz

import (
	"strings"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func Merge(saved *core.ChartSpec, inferred core.ChartSpec) core.ChartSpec {
	if saved == nil || strings.TrimSpace(saved.Type) == "" {
		return inferred
	}
	out := *saved
	if out.X == "" {
		out.X = inferred.X
	}
	if len(out.Y) == 0 {
		out.Y = append([]string{}, inferred.Y...)
	}
	if out.Columns == nil {
		out.Columns = inferred.Columns
	}
	if out.Search == "" {
		out.Search = inferred.Search
	}
	if out.Sort == "" {
		out.Sort = inferred.Sort
	}
	if out.SortDir == "" {
		out.SortDir = inferred.SortDir
	}
	if out.Series == nil {
		out.Series = inferred.Series
	}
	return out
}

func Infer(q queryir.Query) core.ChartSpec {
	ys := make([]string, 0, len(q.Aggregations))
	for _, agg := range q.Aggregations {
		alias := agg.Alias
		if alias == "" {
			alias = strings.ToLower(agg.Fn)
		}
		ys = append(ys, alias)
	}
	if len(q.GroupBy) == 0 {
		if len(ys) == 1 {
			return core.ChartSpec{Type: "scalar", Y: ys}
		}
		return core.ChartSpec{Type: "table"}
	}
	x := q.GroupBy[0].Field
	if q.GroupBy[0].Temporal != "" {
		x = q.GroupBy[0].Field + "_" + strings.ToLower(q.GroupBy[0].Temporal)
		return core.ChartSpec{Type: "line", X: x, Y: ys}
	}
	if len(ys) > 0 {
		return core.ChartSpec{Type: "bar", X: x, Y: ys}
	}
	return core.ChartSpec{Type: "table", X: x}
}

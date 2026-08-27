package queryir

import (
	"fmt"
	"strings"
)

type DrillRequest struct {
	Kind     string         `json:"kind"`
	Field    string         `json:"field,omitempty"`
	Value    any            `json:"value,omitempty"`
	Values   map[string]any `json:"values,omitempty"`
	Temporal string         `json:"temporal,omitempty"`
	Join     *Join          `json:"join,omitempty"`
}

var finerTime = map[string]string{
	"year": "quarter", "quarter": "month", "month": "week", "week": "day", "day": "hour", "hour": "minute",
}

func Drill(q Query, req DrillRequest) (Query, error) {
	out := Clone(q)
	switch strings.ToLower(req.Kind) {
	case "filter":
		if req.Field == "" {
			return Query{}, fmt.Errorf("field is required")
		}
		out.Filters = append(out.Filters, Filter{Field: req.Field, Op: "eq", Value: req.Value})
	case "records":
		out.Aggregations = nil
		out.GroupBy = nil
		out.Having = nil
		for field, value := range req.Values {
			out.Filters = append(out.Filters, Filter{Field: field, Op: "eq", Value: value})
		}
		if req.Field != "" && req.Value != nil {
			out.Filters = append(out.Filters, Filter{Field: req.Field, Op: "eq", Value: req.Value})
		}
		if req.Join != nil {
			out.Joins = append(out.Joins, *req.Join)
		}
		if out.Limit == 0 || out.Limit > 2000 {
			out.Limit = 2000
		}
	case "zoom_time":
		if len(out.GroupBy) == 0 {
			return Query{}, fmt.Errorf("no temporal breakout to zoom")
		}
		current := strings.ToLower(out.GroupBy[0].Temporal)
		next := req.Temporal
		if next == "" {
			next = finerTime[current]
		}
		if next == "" {
			return Query{}, fmt.Errorf("cannot zoom further than %s", current)
		}
		out.GroupBy[0].Temporal = next
	case "breakout":
		if req.Field == "" {
			return Query{}, fmt.Errorf("field is required")
		}
		out.GroupBy = append(out.GroupBy, Breakout{Field: req.Field, Temporal: req.Temporal})
	default:
		return Query{}, fmt.Errorf("unsupported drill kind %q", req.Kind)
	}
	return out, nil
}

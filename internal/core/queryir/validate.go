package queryir

import (
	"fmt"
	"regexp"
	"strings"
)

var ident = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)
var identPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*(\.[A-Za-z_][A-Za-z0-9_$]*)?$`)

var aggregations = map[string]bool{
	"count": true, "sum": true, "avg": true, "min": true, "max": true, "distinct": true,
	"stddev": true, "variance": true,
}

var filterOps = map[string]bool{
	"eq": true, "neq": true, "gt": true, "gte": true, "lt": true, "lte": true,
	"contains": true, "not_contains": true, "starts_with": true, "ends_with": true,
	"is_null": true, "not_null": true, "between": true, "is_empty": true, "not_empty": true,
	"in": true, "not_in": true,
}

var temporals = map[string]bool{
	"": true, "minute": true, "hour": true, "day": true, "week": true, "month": true, "quarter": true, "year": true,
}

var joinTypes = map[string]bool{"inner": true, "left": true, "right": true, "full": true, "": true}

var exprOps = map[string]bool{"add": true, "sub": true, "mul": true, "div": true, "concat": true}

func (q Query) Validate() error {
	if q.Version != 0 && q.Version != 1 {
		return fmt.Errorf("unsupported queryir version %d", q.Version)
	}
	if strings.TrimSpace(q.Source.DatabaseID) == "" {
		return fmt.Errorf("source.database_id is required")
	}
	if q.Source.Table == nil && q.Source.ModelID == "" && q.Source.QuestionID == "" && q.Source.MetricID == "" {
		return fmt.Errorf("source.table is required")
	}
	if q.Source.Table != nil {
		if err := checkIdent("schema", q.Source.Table.Schema); err != nil {
			return err
		}
		if err := checkIdent("table", q.Source.Table.Name); err != nil {
			return err
		}
	}
	for _, join := range q.Joins {
		if !joinTypes[strings.ToLower(join.Type)] {
			return fmt.Errorf("unsupported join type %q", join.Type)
		}
		if join.Table == nil {
			return fmt.Errorf("join.table is required")
		}
		if err := checkIdent("join schema", join.Table.Schema); err != nil {
			return err
		}
		if err := checkIdent("join table", join.Table.Name); err != nil {
			return err
		}
		if join.Alias != "" {
			if err := checkIdent("join alias", join.Alias); err != nil {
				return err
			}
		}
		for _, cond := range join.Conditions {
			if err := checkPath("join left", cond.Left); err != nil {
				return err
			}
			if err := checkPath("join right", cond.Right); err != nil {
				return err
			}
		}
	}
	for _, expr := range q.Expressions {
		if err := checkIdent("expression alias", expr.Alias); err != nil {
			return err
		}
		if !exprOps[strings.ToLower(expr.Op)] {
			return fmt.Errorf("unsupported expression op %q", expr.Op)
		}
		if err := checkPath("expression left", expr.Left); err != nil {
			return err
		}
	}
	for _, field := range q.Fields {
		if err := checkPath("field", field); err != nil {
			return err
		}
	}
	if err := validateFilters("filter", q.Filters); err != nil {
		return err
	}
	if err := validateFilters("having", q.Having); err != nil {
		return err
	}
	for _, agg := range q.Aggregations {
		fn := strings.ToLower(agg.Fn)
		if !aggregations[fn] {
			return fmt.Errorf("unsupported aggregation %q", agg.Fn)
		}
		if fn != "count" {
			if err := checkPath("aggregation field", agg.Field); err != nil {
				return err
			}
		}
		if agg.Alias != "" {
			if err := checkIdent("aggregation alias", agg.Alias); err != nil {
				return err
			}
		}
	}
	for _, breakout := range q.GroupBy {
		if err := checkPath("group_by field", breakout.Field); err != nil {
			return err
		}
		if !temporals[strings.ToLower(breakout.Temporal)] {
			return fmt.Errorf("unsupported temporal unit %q", breakout.Temporal)
		}
		if breakout.BinWidth < 0 {
			return fmt.Errorf("bin_width must be non-negative")
		}
	}
	for _, order := range q.OrderBy {
		if err := checkPath("order_by field", order.Field); err != nil {
			return err
		}
		dir := strings.ToLower(order.Dir)
		if dir != "" && dir != "asc" && dir != "desc" {
			return fmt.Errorf("unsupported order direction %q", order.Dir)
		}
	}
	if q.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}
	return nil
}

func validateFilters(label string, filters []Filter) error {
	for _, filter := range filters {
		if len(filter.And) > 0 {
			if err := validateFilters(label, filter.And); err != nil {
				return err
			}
			continue
		}
		if len(filter.Or) > 0 {
			if err := validateFilters(label, filter.Or); err != nil {
				return err
			}
			continue
		}
		if filter.SegmentID != "" {
			if err := checkIdent("segment", filter.SegmentID); err != nil {
				return err
			}
			continue
		}
		if err := checkPath(label+" field", filter.Field); err != nil {
			return err
		}
		if !filterOps[strings.ToLower(filter.Op)] {
			return fmt.Errorf("unsupported filter op %q", filter.Op)
		}
		switch strings.ToLower(filter.Op) {
		case "is_null", "not_null", "is_empty", "not_empty":
		case "between":
			if len(FilterValues(filter.Value)) != 2 {
				return fmt.Errorf("between requires two values")
			}
		case "in", "not_in":
			if len(FilterValues(filter.Value)) == 0 {
				return fmt.Errorf("%s requires at least one value", filter.Op)
			}
		default:
			if filter.Value == nil || fmt.Sprint(filter.Value) == "" {
				return fmt.Errorf("filter %s requires a value", filter.Field)
			}
		}
	}
	return nil
}

func CheckIdent(label, value string) error {
	return checkIdent(label, value)
}

func checkIdent(label, value string) error {
	if !ident.MatchString(value) {
		return fmt.Errorf("invalid %s %q", label, value)
	}
	return nil
}

func checkPath(label, value string) error {
	if !identPath.MatchString(value) {
		return fmt.Errorf("invalid %s %q", label, value)
	}
	return nil
}

func Quote(identName string) string {
	return `"` + strings.ReplaceAll(identName, `"`, `""`) + `"`
}

func QuotePath(name string) string {
	parts := strings.Split(name, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, Quote(part))
	}
	return strings.Join(quoted, ".")
}

func FilterValues(value any) []any {
	switch items := value.(type) {
	case []any:
		return items
	case []string:
		out := make([]any, len(items))
		for i, item := range items {
			out[i] = item
		}
		return out
	default:
		if value == nil || fmt.Sprint(value) == "" {
			return nil
		}
		return []any{value}
	}
}

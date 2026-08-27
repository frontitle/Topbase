package queryir

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func Clone(q Query) Query {
	raw, _ := json.Marshal(q)
	var out Query
	_ = json.Unmarshal(raw, &out)
	return out
}

type MappedValue struct {
	Field string
	Type  string
	Value any
}

// ApplyMappedFilters adds dashboard filter values onto a card query.
func ApplyMappedFilters(q Query, mappings []MappedValue, now time.Time) (Query, error) {
	out := Clone(q)
	for _, mapping := range mappings {
		if mapping.Value == nil || strings.TrimSpace(mapping.Field) == "" {
			continue
		}
		filters, err := valueToFilters(mapping.Field, mapping.Type, mapping.Value, now)
		if err != nil {
			return Query{}, err
		}
		out.Filters = append(out.Filters, filters...)
	}
	return out, nil
}

func valueToFilters(field, filterType string, value any, now time.Time) ([]Filter, error) {
	switch strings.ToLower(filterType) {
	case "date":
		start, end, err := parseDateValue(value, now)
		if err != nil {
			return nil, err
		}
		filters := []Filter{}
		if start != "" {
			filters = append(filters, Filter{Field: field, Op: "gte", Value: start})
		}
		if end != "" {
			filters = append(filters, Filter{Field: field, Op: "lte", Value: end})
		}
		return filters, nil
	case "number":
		return []Filter{{Field: field, Op: "eq", Value: value}}, nil
	case "boolean":
		return []Filter{{Field: field, Op: "eq", Value: value}}, nil
	default:
		return []Filter{{Field: field, Op: "eq", Value: value}}, nil
	}
}

func parseDateValue(value any, now time.Time) (string, string, error) {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "past_") && strings.HasSuffix(v, "_days") {
			n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(v, "past_"), "_days"))
			if err != nil {
				return "", "", fmt.Errorf("invalid relative date")
			}
			start := now.AddDate(0, 0, -n).Format("2006-01-02")
			return start, now.Format("2006-01-02"), nil
		}
		return v, v, nil
	case map[string]any:
		start, _ := v["start"].(string)
		end, _ := v["end"].(string)
		if days, ok := asInt(v["relative_days"]); ok {
			start = now.AddDate(0, 0, -days).Format("2006-01-02")
			end = now.Format("2006-01-02")
		}
		return start, end, nil
	default:
		return "", "", fmt.Errorf("unsupported date filter value")
	}
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

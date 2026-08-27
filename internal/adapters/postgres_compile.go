package adapters

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/topbase/topbase/internal/core/queryir"
)

type compileLimits struct {
	defaultLimit int
	maxLimit     int
}

func CompilePostgres(q queryir.Query) (queryir.Compiled, error) {
	return compilePostgres(q, compileLimits{defaultLimit: 1000, maxLimit: 10000})
}

func CompilePostgresWarehouse(q queryir.Query) (queryir.Compiled, error) {
	return compilePostgres(q, compileLimits{})
}

func compilePostgres(q queryir.Query, limits compileLimits) (queryir.Compiled, error) {
	if err := q.Validate(); err != nil {
		return queryir.Compiled{}, err
	}
	if q.Source.Table == nil {
		return queryir.Compiled{}, fmt.Errorf("source.table is required to compile")
	}

	selects := make([]string, 0, len(q.Fields)+len(q.Aggregations)+len(q.GroupBy)+len(q.Expressions))
	for _, breakout := range q.GroupBy {
		selects = append(selects, postgresBreakout(breakout)+" AS "+queryir.Quote(breakoutAlias(breakout)))
	}
	for _, field := range q.Fields {
		already := false
		for _, breakout := range q.GroupBy {
			if breakout.Field == field && breakout.Temporal == "" && breakout.BinWidth == 0 {
				already = true
				break
			}
		}
		if already {
			continue
		}
		selects = append(selects, queryir.QuotePath(field))
	}
	for _, expr := range q.Expressions {
		selects = append(selects, postgresExpr(expr)+" AS "+queryir.Quote(expr.Alias))
	}
	for _, agg := range q.Aggregations {
		selects = append(selects, postgresAgg(agg))
	}
	if len(selects) == 0 {
		selects = []string{"*"}
	}

	from := queryir.Quote(q.Source.Table.Schema) + "." + queryir.Quote(q.Source.Table.Name)
	sourceAlias := q.Source.Alias
	if sourceAlias == "" && len(q.Joins) > 0 {
		sourceAlias = q.Source.Table.Name
	}
	if sourceAlias != "" {
		from += " AS " + queryir.Quote(sourceAlias)
	}
	args := []any{}
	for _, join := range q.Joins {
		joinSQL, nextArgs, err := postgresJoin(join, args)
		if err != nil {
			return queryir.Compiled{}, err
		}
		from += " " + joinSQL
		args = nextArgs
	}
	where, args := postgresFilters(q.Filters, args)

	sql := "SELECT " + strings.Join(selects, ", ") + " FROM " + from
	if where != "" {
		sql += " WHERE " + where
	}
	if len(q.Aggregations) > 0 && len(q.GroupBy) > 0 {
		groups := make([]string, 0, len(q.GroupBy))
		for _, breakout := range q.GroupBy {
			groups = append(groups, postgresBreakout(breakout))
		}
		sql += " GROUP BY " + strings.Join(groups, ", ")
	} else if len(q.Aggregations) > 0 && len(q.Fields) > 0 {
		groups := make([]string, 0, len(q.Fields))
		for _, field := range q.Fields {
			groups = append(groups, queryir.QuotePath(field))
		}
		sql += " GROUP BY " + strings.Join(groups, ", ")
	}
	having, args := postgresFilters(q.Having, args)
	if having != "" {
		sql += " HAVING " + having
	}
	if len(q.OrderBy) > 0 {
		orders := make([]string, 0, len(q.OrderBy))
		for _, order := range q.OrderBy {
			dir := "ASC"
			if strings.EqualFold(order.Dir, "desc") {
				dir = "DESC"
			}
			orders = append(orders, queryir.QuotePath(order.Field)+" "+dir)
		}
		sql += " ORDER BY " + strings.Join(orders, ", ")
	}
	limit := q.Limit
	if limit == 0 {
		limit = limits.defaultLimit
	}
	if limits.maxLimit > 0 && limit > limits.maxLimit {
		limit = limits.maxLimit
	}
	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", limit)
	}
	return queryir.Compiled{SQL: sql, Args: args}, nil
}

func postgresJoin(join queryir.Join, args []any) (string, []any, error) {
	kind := strings.ToUpper(join.Type)
	if kind == "" {
		kind = "LEFT"
	}
	if kind == "INNER" {
		kind = "INNER"
	}
	alias := join.Alias
	if alias == "" {
		alias = join.Table.Name
	}
	on := make([]string, 0, len(join.Conditions))
	if len(join.Conditions) == 0 {
		return "", args, fmt.Errorf("join conditions are required")
	}
	for _, cond := range join.Conditions {
		op := cond.Op
		if op == "" {
			op = "="
		}
		if op != "=" && op != "<>" {
			return "", args, fmt.Errorf("unsupported join op %q", op)
		}
		on = append(on, queryir.QuotePath(cond.Left)+" "+op+" "+queryir.QuotePath(cond.Right))
	}
	sql := kind + " JOIN " + queryir.Quote(join.Table.Schema) + "." + queryir.Quote(join.Table.Name) + " AS " + queryir.Quote(alias) + " ON " + strings.Join(on, " AND ")
	return sql, args, nil
}

func postgresBreakout(breakout queryir.Breakout) string {
	field := queryir.QuotePath(breakout.Field)
	if breakout.BinWidth > 0 {
		w := strconv.FormatFloat(breakout.BinWidth, 'f', -1, 64)
		return "floor(" + field + " / " + w + ") * " + w
	}
	switch strings.ToLower(breakout.Temporal) {
	case "minute", "hour", "day", "week", "month", "year":
		return "date_trunc('" + strings.ToLower(breakout.Temporal) + "', " + field + ")"
	case "quarter":
		return "date_trunc('quarter', " + field + ")"
	default:
		return field
	}
}

func breakoutAlias(breakout queryir.Breakout) string {
	name := breakout.Field
	if strings.Contains(name, ".") {
		name = strings.ReplaceAll(name, ".", "_")
	}
	if breakout.BinWidth > 0 {
		return name + "_bin"
	}
	if breakout.Temporal == "" {
		return name
	}
	return name + "_" + strings.ToLower(breakout.Temporal)
}

func postgresExpr(expr queryir.Expression) string {
	left := queryir.QuotePath(expr.Left)
	right := fmt.Sprint(expr.Right)
	if s, ok := expr.Right.(string); ok && queryirPath(s) {
		right = queryir.QuotePath(s)
	} else {
		switch expr.Right.(type) {
		case float64, int, int64:
			right = fmt.Sprint(expr.Right)
		default:
			right = quoteLiteral(fmt.Sprint(expr.Right))
		}
	}
	switch strings.ToLower(expr.Op) {
	case "add":
		return "(" + left + " + " + right + ")"
	case "sub":
		return "(" + left + " - " + right + ")"
	case "mul":
		return "(" + left + " * " + right + ")"
	case "div":
		return "(" + left + " / NULLIF(" + right + ", 0))"
	default:
		return "concat(" + left + ", " + right + ")"
	}
}

func queryirPath(value string) bool {
	return identPathMatch(value)
}

func identPathMatch(value string) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" {
			return false
		}
		for i, r := range part {
			if i == 0 && !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_') {
				return false
			}
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '$') {
				return false
			}
		}
	}
	return !strings.ContainsAny(value, " '\"")
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func postgresAgg(agg queryir.Aggregation) string {
	alias := agg.Alias
	fn := strings.ToLower(agg.Fn)
	if alias == "" {
		alias = fn
	}
	expr := "count(*)"
	switch fn {
	case "sum", "avg", "min", "max":
		expr = fn + "(" + queryir.QuotePath(agg.Field) + ")"
	case "distinct":
		expr = "count(DISTINCT " + queryir.QuotePath(agg.Field) + ")"
	case "stddev":
		expr = "stddev_samp(" + queryir.QuotePath(agg.Field) + ")"
	case "variance":
		expr = "var_samp(" + queryir.QuotePath(agg.Field) + ")"
	}
	return expr + " AS " + queryir.Quote(alias)
}

func postgresFilters(filters []queryir.Filter, args []any) (string, []any) {
	if len(filters) == 0 {
		return "", args
	}
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		part, next, ok := postgresFilter(filter, args)
		if !ok {
			continue
		}
		parts = append(parts, part)
		args = next
	}
	return strings.Join(parts, " AND "), args
}

func postgresFilter(filter queryir.Filter, args []any) (string, []any, bool) {
	if len(filter.And) > 0 {
		inner, args := postgresFilters(filter.And, args)
		if inner == "" {
			return "", args, false
		}
		return "(" + inner + ")", args, true
	}
	if len(filter.Or) > 0 {
		parts := make([]string, 0, len(filter.Or))
		for _, child := range filter.Or {
			part, next, ok := postgresFilter(child, args)
			if !ok {
				continue
			}
			parts = append(parts, part)
			args = next
		}
		if len(parts) == 0 {
			return "", args, false
		}
		return "(" + strings.Join(parts, " OR ") + ")", args, true
	}
	field := queryir.QuotePath(filter.Field)
	switch strings.ToLower(filter.Op) {
	case "eq":
		args = append(args, filter.Value)
		return fmt.Sprintf("%s = $%d", field, len(args)), args, true
	case "neq":
		args = append(args, filter.Value)
		return fmt.Sprintf("%s <> $%d", field, len(args)), args, true
	case "gt":
		args = append(args, filter.Value)
		return fmt.Sprintf("%s > $%d", field, len(args)), args, true
	case "gte":
		args = append(args, filter.Value)
		return fmt.Sprintf("%s >= $%d", field, len(args)), args, true
	case "lt":
		args = append(args, filter.Value)
		return fmt.Sprintf("%s < $%d", field, len(args)), args, true
	case "lte":
		args = append(args, filter.Value)
		return fmt.Sprintf("%s <= $%d", field, len(args)), args, true
	case "contains":
		args = append(args, fmt.Sprint(filter.Value))
		return fmt.Sprintf("%s ILIKE '%%' || $%d || '%%'", field, len(args)), args, true
	case "not_contains":
		args = append(args, fmt.Sprint(filter.Value))
		return fmt.Sprintf("%s NOT ILIKE '%%' || $%d || '%%'", field, len(args)), args, true
	case "starts_with":
		args = append(args, fmt.Sprint(filter.Value))
		return fmt.Sprintf("%s ILIKE $%d || '%%'", field, len(args)), args, true
	case "ends_with":
		args = append(args, fmt.Sprint(filter.Value))
		return fmt.Sprintf("%s ILIKE '%%' || $%d", field, len(args)), args, true
	case "is_null":
		return field + " IS NULL", args, true
	case "not_null":
		return field + " IS NOT NULL", args, true
	case "is_empty":
		return "(" + field + " IS NULL OR " + field + " = '')", args, true
	case "not_empty":
		return "(" + field + " IS NOT NULL AND " + field + " <> '')", args, true
	case "between":
		values := queryir.FilterValues(filter.Value)
		if len(values) != 2 {
			return "", args, false
		}
		args = append(args, values[0], values[1])
		return fmt.Sprintf("%s BETWEEN $%d AND $%d", field, len(args)-1, len(args)), args, true
	case "in", "not_in":
		values := queryir.FilterValues(filter.Value)
		if len(values) == 0 {
			return "", args, false
		}
		parts := make([]string, 0, len(values))
		for _, item := range values {
			args = append(args, item)
			parts = append(parts, fmt.Sprintf("$%d", len(args)))
		}
		op := "IN"
		if strings.ToLower(filter.Op) == "not_in" {
			op = "NOT IN"
		}
		return fmt.Sprintf("%s %s (%s)", field, op, strings.Join(parts, ", ")), args, true
	default:
		return "", args, false
	}
}

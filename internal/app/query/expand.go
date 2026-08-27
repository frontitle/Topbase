package query

import (
	"fmt"
	"strings"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

type Expander struct {
	Fields    core.FieldStore
	Models    core.ModelStore
	Metrics   core.MetricStore
	Segments  core.SegmentStore
	Questions core.QuestionStore
}

func (e *Expander) Expand(q queryir.Query) (queryir.Query, error) {
	if e == nil {
		return q, nil
	}
	out := queryir.Clone(q)
	if out.Source.ModelID != "" && e.Models != nil {
		model, err := e.Models.ByID(out.Source.ModelID)
		if err != nil {
			return queryir.Query{}, err
		}
		if model.QueryIR == nil {
			return queryir.Query{}, fmt.Errorf("model has no queryir")
		}
		base := queryir.Clone(*model.QueryIR)
		out.Source = base.Source
		out.Joins = append(base.Joins, out.Joins...)
		out.Filters = append(base.Filters, out.Filters...)
		if len(out.Fields) == 0 {
			out.Fields = base.Fields
		}
	}
	if out.Source.QuestionID != "" && e.Questions != nil {
		question, err := e.Questions.ByID(out.Source.QuestionID)
		if err != nil {
			return queryir.Query{}, err
		}
		if question.QueryIR == nil {
			return queryir.Query{}, fmt.Errorf("question has no queryir")
		}
		base := queryir.Clone(*question.QueryIR)
		out.Source = base.Source
		out.Joins = append(base.Joins, out.Joins...)
		out.Filters = append(base.Filters, out.Filters...)
	}
	if out.Source.MetricID != "" && e.Metrics != nil {
		metric, err := e.Metrics.ByID(out.Source.MetricID)
		if err != nil {
			return queryir.Query{}, err
		}
		out.Source.Table = &queryir.TableRef{Schema: metric.Schema, Name: metric.Table}
		out.Aggregations = append([]queryir.Aggregation{metric.Aggregation}, out.Aggregations...)
		out.Filters = append(metric.Filters, out.Filters...)
	}
	if err := e.expandSegments(&out); err != nil {
		return queryir.Query{}, err
	}
	if err := e.expandImplicitJoins(&out); err != nil {
		return queryir.Query{}, err
	}
	return out, nil
}

func (e *Expander) expandSegments(q *queryir.Query) error {
	if e.Segments == nil {
		return nil
	}
	filters, err := e.replaceSegments(q.Filters)
	if err != nil {
		return err
	}
	q.Filters = filters
	return nil
}

func (e *Expander) replaceSegments(filters []queryir.Filter) ([]queryir.Filter, error) {
	out := make([]queryir.Filter, 0, len(filters))
	for _, filter := range filters {
		if filter.SegmentID != "" {
			segment, err := e.Segments.ByID(filter.SegmentID)
			if err != nil {
				return nil, err
			}
			out = append(out, segment.Filters...)
			continue
		}
		if len(filter.And) > 0 {
			inner, err := e.replaceSegments(filter.And)
			if err != nil {
				return nil, err
			}
			filter.And = inner
		}
		if len(filter.Or) > 0 {
			inner, err := e.replaceSegments(filter.Or)
			if err != nil {
				return nil, err
			}
			filter.Or = inner
		}
		out = append(out, filter)
	}
	return out, nil
}

func (e *Expander) expandImplicitJoins(q *queryir.Query) error {
	if e.Fields == nil || q.Source.Table == nil {
		return nil
	}
	fields, err := e.Fields.ListDatabaseFields(q.Source.DatabaseID)
	if err != nil {
		return err
	}
	needed := map[string]bool{}
	collectJoinTables(q, needed)
	for table := range needed {
		if table == q.Source.Table.Name {
			continue
		}
		if hasJoin(q.Joins, table) {
			continue
		}
		join, ok := implicitJoin(q.Source.Table, table, fields)
		if !ok {
			return fmt.Errorf("fk is required for implicit join to %s", table)
		}
		q.Joins = append(q.Joins, join)
	}
	for i, join := range q.Joins {
		if !join.Implicit || len(join.Conditions) > 0 || join.Table == nil {
			continue
		}
		filled, ok := implicitJoin(q.Source.Table, join.Table.Name, fields)
		if !ok {
			return fmt.Errorf("fk is required for implicit join to %s", join.Table.Name)
		}
		filled.Type = join.Type
		if filled.Type == "" {
			filled.Type = "left"
		}
		q.Joins[i] = filled
	}
	return nil
}

func collectJoinTables(q *queryir.Query, needed map[string]bool) {
	source := ""
	if q.Source.Table != nil {
		source = q.Source.Table.Name
	}
	for _, field := range q.Fields {
		if table, _, ok := splitPath(field); ok && table != source {
			needed[table] = true
		}
	}
	for _, filter := range q.Filters {
		if table, _, ok := splitPath(filter.Field); ok && table != source {
			needed[table] = true
		}
	}
	for _, join := range q.Joins {
		if join.Implicit && join.Table != nil {
			needed[join.Table.Name] = true
		}
	}
}

func splitPath(name string) (string, string, bool) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func hasJoin(joins []queryir.Join, table string) bool {
	for _, join := range joins {
		if join.Table != nil && join.Table.Name == table {
			return true
		}
		if join.Alias == table {
			return true
		}
	}
	return false
}

func implicitJoin(from *queryir.TableRef, toTable string, fields []core.FieldMeta) (queryir.Join, bool) {
	for _, field := range fields {
		if field.Schema != from.Schema || field.Table != from.Name || field.FKTarget == nil {
			continue
		}
		if field.FKTarget.Table != toTable {
			continue
		}
		schema := field.FKTarget.Schema
		if schema == "" {
			schema = from.Schema
		}
		return queryir.Join{
			Type:     "left",
			Alias:    toTable,
			Table:    &queryir.TableRef{Schema: schema, Name: toTable},
			Implicit: true,
			Conditions: []queryir.JoinCondition{{
				Left:  from.Name + "." + field.Name,
				Right: toTable + "." + field.FKTarget.Name,
				Op:    "=",
			}},
		}, true
	}
	return queryir.Join{}, false
}

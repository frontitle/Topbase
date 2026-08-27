package query

import (
	"context"
	"fmt"

	"github.com/topbase/topbase/internal/app/viz"
	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

type DatasetResult struct {
	SQL       string         `json:"sql"`
	Columns   []string       `json:"columns"`
	Rows      [][]any        `json:"rows"`
	Meta      map[string]any `json:"meta"`
	ChartSpec core.ChartSpec `json:"chartspec"`
}

type Compiler func(queryir.Query) (queryir.Compiled, error)

type DatasetService struct {
	Queries core.QueryService
	Compile Compiler
	Expand  *Expander
}

func (s DatasetService) Run(ctx context.Context, q queryir.Query) (DatasetResult, error) {
	if q.Version == 0 {
		q.Version = 1
	}
	if s.Expand != nil {
		expanded, err := s.Expand.Expand(q)
		if err != nil {
			return DatasetResult{}, err
		}
		q = expanded
	}
	if s.Compile == nil {
		return DatasetResult{}, fmt.Errorf("query compiler is not configured")
	}
	compiled, err := s.Compile(q)
	if err != nil {
		return DatasetResult{}, err
	}
	result, err := s.Queries.RunArgs(ctx, q.Source.DatabaseID, compiled.SQL, compiled.Args)
	if err != nil {
		return DatasetResult{}, err
	}
	if result.Meta == nil {
		result.Meta = map[string]any{}
	}
	result.Meta["queryir_version"] = q.Version
	spec := viz.Infer(q)
	return DatasetResult{
		SQL:       compiled.SQL,
		Columns:   result.Columns,
		Rows:      result.Rows,
		Meta:      result.Meta,
		ChartSpec: spec,
	}, nil
}

func VisualRequestToQuery(databaseID, schema, table string, fields []string, aggregation, aggregationField, filterField, filterValue string) (queryir.Query, error) {
	if schema == "" || table == "" {
		return queryir.Query{}, fmt.Errorf("schema and table are required")
	}
	q := queryir.Query{
		Version: 1,
		Source:  queryir.Source{DatabaseID: databaseID, Table: &queryir.TableRef{Schema: schema, Name: table}},
		Fields:  fields,
		Limit:   1000,
	}
	if aggregation != "" {
		q.Aggregations = []queryir.Aggregation{{Fn: aggregation, Field: aggregationField}}
		for _, field := range fields {
			q.GroupBy = append(q.GroupBy, queryir.Breakout{Field: field})
		}
	}
	if filterField != "" && filterValue != "" {
		q.Filters = []queryir.Filter{{Field: filterField, Op: "eq", Value: filterValue}}
	}
	return q, nil
}

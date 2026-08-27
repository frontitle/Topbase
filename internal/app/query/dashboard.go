package query

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/topbase/topbase/internal/app/viz"
	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func (s DatasetService) RunDashboardCard(ctx context.Context, board core.Dashboard, cardID string, values map[string]any, questions core.QuestionStore) (DatasetResult, error) {
	var card core.DashboardCard
	found := false
	for _, item := range board.Cards {
		if item.ID == cardID {
			card = item
			found = true
			break
		}
	}
	if !found {
		return DatasetResult{}, fmt.Errorf("card not found")
	}
	if card.Type != "question" || card.QuestionID == "" {
		return DatasetResult{}, fmt.Errorf("card is not a question")
	}
	question, err := questions.ByID(card.QuestionID)
	if err != nil {
		return DatasetResult{}, err
	}
	mapped := mappedValues(board, card.ID, values)
	if question.QueryType == "native" || (question.QueryIR == nil && question.NativeSQL != "") {
		valueMap := nativeValues(mapped, values, question.Parameters)
		sql, args, err := queryir.ApplyNative(question.NativeSQL, question.Parameters, valueMap)
		if err != nil {
			return DatasetResult{}, err
		}
		result, err := s.Queries.RunArgs(ctx, question.DatabaseID, sql, args)
		if err != nil {
			return DatasetResult{}, err
		}
		out := DatasetResult{SQL: sql, Columns: result.Columns, Rows: result.Rows, Meta: result.Meta, ChartSpec: core.ChartSpec{Type: "table"}}
		applyChartSpec(&out, question, card)
		return out, nil
	}
	if question.QueryIR == nil {
		return DatasetResult{}, fmt.Errorf("question has no queryir")
	}
	q, err := queryir.ApplyMappedFilters(*question.QueryIR, mapped, time.Now().UTC())
	if err != nil {
		return DatasetResult{}, err
	}
	out, err := s.Run(ctx, q)
	if err != nil {
		return DatasetResult{}, err
	}
	applyChartSpec(&out, question, card)
	return out, nil
}

func applyChartSpec(result *DatasetResult, question core.Question, card core.DashboardCard) {
	inferred := result.ChartSpec
	if inferred.Type == "" {
		inferred = core.ChartSpec{Type: "table"}
	}
	spec := viz.Merge(question.ChartSpec, inferred)
	if card.Config != nil {
		if raw, ok := card.Config["chartspec"]; ok {
			body, err := json.Marshal(raw)
			if err == nil {
				var override core.ChartSpec
				if json.Unmarshal(body, &override) == nil {
					spec = viz.Merge(&override, spec)
				}
			}
		}
	}
	result.ChartSpec = spec
}

func nativeValues(mapped []queryir.MappedValue, raw map[string]any, params []queryir.Parameter) map[string]any {
	values := map[string]any{}
	for _, item := range mapped {
		values[item.Field] = item.Value
	}
	for name, value := range raw {
		if _, ok := values[name]; !ok {
			values[name] = value
		}
	}
	for _, param := range params {
		if value, ok := raw[param.Name]; ok {
			values[param.Name] = value
		}
	}
	return values
}

func mappedValues(board core.Dashboard, cardID string, values map[string]any) []queryir.MappedValue {
	items := []queryir.MappedValue{}
	for _, filter := range board.Filters {
		value, ok := values[filter.ID]
		if !ok {
			continue
		}
		for _, mapping := range filter.Mappings {
			if mapping.CardID != "" && mapping.CardID != cardID {
				continue
			}
			items = append(items, queryir.MappedValue{Field: mapping.Field, Type: filter.Type, Value: value})
		}
	}
	return items
}

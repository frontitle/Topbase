package core

import (
	"context"
	"testing"
)

type metadataExecutor struct{}

func (metadataExecutor) Execute(context.Context, string, string, ...any) (QueryResult, error) {
	return QueryResult{Columns: []string{"id"}, Rows: [][]any{{1}}}, nil
}

func TestQueryServiceMarksDirectExecutionFreshness(t *testing.T) {
	t.Parallel()

	service := QueryService{Executor: metadataExecutor{}}
	result, err := service.Run(context.Background(), "pg_test", "select 1 as id")
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta["execution"] != "direct" {
		t.Fatalf("execution = %v, want direct", result.Meta["execution"])
	}
	if result.Meta["cache_hit"] != false {
		t.Fatalf("cache_hit = %v, want false", result.Meta["cache_hit"])
	}
	if result.Meta["executed_at"] == "" {
		t.Fatal("executed_at should be populated")
	}
}

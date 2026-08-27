package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/topbase/topbase/internal/core"
)

// DemoExecutor keeps the UI runnable before a source adapter is configured.
type DemoExecutor struct{}

func (DemoExecutor) Execute(_ context.Context, databaseID, sql string, _ ...any) (core.QueryResult, error) {
	if databaseID != "demo" {
		return core.QueryResult{}, fmt.Errorf("database %q is not connected", databaseID)
	}
	return core.QueryResult{
		Columns: []string{"metric", "value", "query_preview"},
		Rows:    [][]any{{"active_users", 12842, strings.TrimSpace(sql)}},
		Meta:    map[string]any{"elapsed_ms": 14, "cached": false},
	}, nil
}

type DemoAI struct{}

func (DemoAI) Ask(_ context.Context, _ string, databaseID string) (core.ChatResponse, error) {
	if databaseID == "" {
		return core.ChatResponse{}, fmt.Errorf("database_id is required")
	}
	return core.ChatResponse{
		Answer: "我已把问题转为可预览的只读查询。确认后可运行，并可直接创建周期任务。",
		SQL:    "SELECT date_trunc('day', created_at) AS day, count(*) AS users FROM users GROUP BY 1 ORDER BY 1 DESC LIMIT 30",
		Safe:   true,
	}, nil
}

package aimcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/topbase/topbase/internal/topbaseapi"
)

func TestServerExposesSafeAnalyticsWorkflow(t *testing.T) {
	t.Parallel()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			http.Error(w, "missing API key", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/developer/ping":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "user_id": "usr_demo", "max_query_rows": 10})
		case "/api/databases":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "pg_demo", "name": "Demo", "engine": "postgres"}})
		case "/api/dataset":
			var input map[string]any
			_ = json.NewDecoder(r.Body).Decode(&input)
			if input["limit"] != float64(10) {
				t.Fatalf("bounded limit = %#v", input["limit"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"columns": []string{"count"}, "rows": [][]any{{2}}, "meta": map[string]any{}, "chartspec": map[string]any{"type": "scalar"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	client, err := topbaseapi.New(api.URL, "key", topbaseapi.WithMaxRows(10))
	if err != nil {
		t.Fatal(err)
	}

	server := New(client)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	clientSession, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	listed, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	toolNames := map[string]bool{}
	for _, tool := range listed.Tools {
		toolNames[tool.Name] = true
	}
	for _, name := range []string{"topbase_status", "topbase_describe_table", "topbase_query_data", "topbase_create_analysis"} {
		if !toolNames[name] {
			t.Fatalf("missing tool %q in %#v", name, toolNames)
		}
	}

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "topbase_query_data",
		Arguments: map[string]any{"queryir": map[string]any{
			"version": 1, "source": map[string]any{"database_id": "pg_demo", "table": map[string]any{"schema": "public", "name": "orders"}},
			"aggregations": []map[string]any{{"fn": "count"}}, "limit": 100,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool failed: %#v", result.Content)
	}
}

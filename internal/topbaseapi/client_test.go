package topbaseapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func TestPreviewUsesBearerAndBoundsRows(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dataset" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		var query queryir.Query
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			t.Fatal(err)
		}
		if query.Limit != 25 {
			t.Fatalf("limit = %d, want 25", query.Limit)
		}
		writeTestJSON(w, map[string]any{"columns": []string{"count"}, "rows": [][]any{{3}}, "meta": map[string]any{}, "chartspec": map[string]any{"type": "scalar"}})
	}))
	defer server.Close()

	client, err := New(server.URL, "secret", WithMaxRows(25))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Preview(context.Background(), queryir.Query{
		Source:       queryir.Source{DatabaseID: "pg_demo", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
		Aggregations: []queryir.Aggregation{{Fn: "count"}}, Limit: 999,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("rows = %#v", result.Rows)
	}
}

func TestDescribeTableCombinesLiveAndSemanticMetadata(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/databases/pg_demo/tables":
			writeTestJSON(w, []map[string]any{{"schema": "public", "name": "orders", "description": "订单事实表", "columns": []map[string]any{{"name": "amount", "data_type": "numeric", "description": "订单金额"}}}})
		case "/api/databases/pg_demo/tables/public/orders/fields":
			writeTestJSON(w, []map[string]any{{"database_id": "pg_demo", "schema": "public", "table": "orders", "name": "amount", "display_name": "订单金额", "semantic_type": "Currency"}})
		case "/api/databases/pg_demo/tables/public/orders/annotation":
			writeTestJSON(w, map[string]any{"display_name": "订单", "description": "运营订单", "field_types": map[string]string{"amount": "Currency"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "secret")
	description, err := client.DescribeTable(context.Background(), "pg_demo", "public", "orders")
	if err != nil {
		t.Fatal(err)
	}
	if description.Table.Description != "订单事实表" || description.Annotation.DisplayName != "订单" || len(description.Fields) != 1 {
		t.Fatalf("description = %#v", description)
	}
}

func TestCreateAnalysisRejectsNativeSQLBeforeHTTP(t *testing.T) {
	t.Parallel()
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	client, _ := New(server.URL, "secret")
	_, err := client.CreateAnalysis(context.Background(), core.Question{Name: "unsafe", QueryType: "native", NativeSQL: "select 1"})
	if err == nil {
		t.Fatal("CreateAnalysis unexpectedly succeeded")
	}
	if called {
		t.Fatal("HTTP API was called for rejected native SQL")
	}
}

func TestAPIErrorPreservesTopbaseMessage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeTestJSON(w, map[string]string{"error": "data view permission required"})
	}))
	defer server.Close()
	client, _ := New(server.URL, "secret")
	_, err := client.ListDatabases(context.Background())
	if err == nil || err.Error() != "Topbase API 403 Forbidden: data view permission required" {
		t.Fatalf("error = %v", err)
	}
}

func TestStatusReportsServerDeveloperLimits(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/developer/ping" || r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		writeTestJSON(w, map[string]any{"status": "ok", "user_id": "usr_1", "user_name": "Ada", "max_query_rows": 50, "allow_analysis_write": true})
	}))
	defer server.Close()
	client, _ := New(server.URL, "secret")
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "ok" || status.MaxQueryRows != 50 || !status.AllowAnalysisWrite {
		t.Fatalf("status = %#v", status)
	}
}

func writeTestJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

package adapters

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestQueryLimitsFromEnv(t *testing.T) {
	t.Setenv("TOPBASE_QUERY_RESULT_MAX_BYTES", "2048")
	t.Setenv("TOPBASE_QUERY_CELL_MAX_BYTES", "256")
	limits := queryLimitsFromEnv()
	if limits.rows != 1000 || limits.resultBytes != 2048 || limits.cellBytes != 256 {
		t.Fatalf("unexpected query limits: %#v", limits)
	}
}

func TestQueryLimitsClampDangerousEnvironmentValues(t *testing.T) {
	t.Setenv("TOPBASE_QUERY_RESULT_MAX_BYTES", "9999999999")
	t.Setenv("TOPBASE_QUERY_CELL_MAX_BYTES", "9999999999")
	t.Setenv("TOPBASE_MAX_CONCURRENT_QUERIES", "9999999999")
	limits := queryLimitsFromEnv()
	if limits.resultBytes != 256<<20 || limits.cellBytes != 16<<20 {
		t.Fatalf("query limits were not clamped: %#v", limits)
	}
	if got := cap(NewSQLConnector().querySlots); got != 128 {
		t.Fatalf("query concurrency = %d, want 128", got)
	}
}

func TestScanQueryRowsBoundsLargeCellsAndTotalResult(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE payloads (value TEXT); INSERT INTO payloads VALUES ('abcdefghijklmnop'), ('qrstuvwxyz')`); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(`SELECT value FROM payloads ORDER BY rowid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	result, err := scanQueryRows(rows, columns, queryLimits{rows: 100, resultBytes: 12, cellBytes: 8}, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(result.Rows))
	}
	value, ok := result.Rows[0][0].(string)
	if !ok || !strings.HasSuffix(value, "…") || len(value) > 8 {
		t.Fatalf("bounded value = %#v", result.Rows[0][0])
	}
	if result.Meta["truncated"] != true || result.Meta["truncation_reason"] != "result_byte_limit" {
		t.Fatalf("unexpected truncation metadata: %#v", result.Meta)
	}
	if result.Meta["result_bytes"] != 8 {
		t.Fatalf("result bytes = %#v", result.Meta["result_bytes"])
	}
}

func TestSQLConnectorQueryCapacityHonorsContextCancellation(t *testing.T) {
	connector := NewSQLConnector()
	connector.querySlots = make(chan struct{}, 1)
	connector.querySlots <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := connector.Execute(ctx, "missing", "SELECT 1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute error = %v, want context cancellation", err)
	}
}

func TestBoundedUTF8DoesNotSplitMultibyteCharacters(t *testing.T) {
	value, truncated := boundedUTF8("订单数据", 8)
	if !truncated || value != "订…" || len(value) > 8 {
		t.Fatalf("bounded UTF-8 value = %q, truncated = %v", value, truncated)
	}
}

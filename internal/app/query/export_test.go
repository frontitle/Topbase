package query

import (
	"strings"
	"testing"
)

func TestWriteCSV(t *testing.T) {
	raw, err := CSVBytes([]string{"day", "users"}, [][]any{{"2026-01-01", 12}, {"2026-01-02", 15}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "day,users") || !strings.Contains(text, "2026-01-01,12") {
		t.Fatalf("csv %q", text)
	}
}

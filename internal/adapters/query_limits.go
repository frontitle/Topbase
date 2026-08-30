package adapters

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/topbase/topbase/internal/core"
)

const (
	defaultQueryRowLimit       = 1000
	defaultQueryResultMaxBytes = 16 << 20
	defaultQueryCellMaxBytes   = 1 << 20
)

type queryLimits struct {
	rows        int
	resultBytes int
	cellBytes   int
}

func queryLimitsFromEnv() queryLimits {
	resultBytes := boundedPositiveEnvInt("TOPBASE_QUERY_RESULT_MAX_BYTES", defaultQueryResultMaxBytes, 256<<20)
	cellBytes := boundedPositiveEnvInt("TOPBASE_QUERY_CELL_MAX_BYTES", defaultQueryCellMaxBytes, 16<<20)
	if cellBytes > resultBytes {
		cellBytes = resultBytes
	}
	return queryLimits{
		rows:        defaultQueryRowLimit,
		resultBytes: resultBytes,
		cellBytes:   cellBytes,
	}
}

func positiveEnvInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func boundedPositiveEnvInt(name string, fallback, maximum int) int {
	value := positiveEnvInt(name, fallback)
	if maximum > 0 && value > maximum {
		return maximum
	}
	return value
}

func scanQueryRows(rows *sql.Rows, columns []string, limits queryLimits, engine string) (core.QueryResult, error) {
	if limits.rows <= 0 {
		limits.rows = defaultQueryRowLimit
	}
	if limits.resultBytes <= 0 {
		limits.resultBytes = defaultQueryResultMaxBytes
	}
	if limits.cellBytes <= 0 {
		limits.cellBytes = defaultQueryCellMaxBytes
	}
	meta := map[string]any{
		"row_limit":         limits.rows,
		"result_byte_limit": limits.resultBytes,
		"cell_byte_limit":   limits.cellBytes,
	}
	if engine != "" {
		meta["engine"] = engine
	}
	result := core.QueryResult{Columns: columns, Rows: make([][]any, 0), Meta: meta}
	retainedBytes := 0
	cellTruncated := false
	for rows.Next() {
		if len(result.Rows) >= limits.rows {
			meta["truncated"] = true
			meta["truncation_reason"] = "row_limit"
			break
		}
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return core.QueryResult{}, err
		}
		rowBytes := 0
		for i, value := range values {
			normalized, size, truncated := boundedQueryValue(value, limits.cellBytes)
			values[i] = normalized
			rowBytes += size
			cellTruncated = cellTruncated || truncated
		}
		if retainedBytes+rowBytes > limits.resultBytes {
			meta["truncated"] = true
			meta["truncation_reason"] = "result_byte_limit"
			break
		}
		retainedBytes += rowBytes
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return core.QueryResult{}, err
	}
	if cellTruncated {
		meta["truncated"] = true
		if _, exists := meta["truncation_reason"]; !exists {
			meta["truncation_reason"] = "cell_byte_limit"
		}
	}
	meta["result_bytes"] = retainedBytes
	return result, nil
}

func boundedQueryValue(value any, maxBytes int) (any, int, bool) {
	switch item := value.(type) {
	case nil:
		return nil, 0, false
	case []byte:
		text, truncated := boundedUTF8(string(item), maxBytes)
		return text, len(text), truncated
	case string:
		text, truncated := boundedUTF8(item, maxBytes)
		return text, len(text), truncated
	case time.Time:
		text := item.Format(time.RFC3339Nano)
		return text, len(text), false
	case bool:
		return item, 1, false
	case int64, float64:
		return item, 8, false
	default:
		return item, len(fmt.Sprint(item)), false
	}
}

func boundedUTF8(value string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value, false
	}
	if maxBytes <= len("…") {
		return "", true
	}
	end := maxBytes - len("…")
	for end > 0 && end < len(value) && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…", true
}

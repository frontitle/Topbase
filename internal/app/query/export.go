package query

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/topbase/topbase/internal/core/queryir"
)

func (s DatasetService) ExportCSV(ctx context.Context, q queryir.Query, w io.Writer) error {
	if q.Limit == 0 || q.Limit > 10000 {
		q.Limit = 10000
	}
	result, err := s.Run(ctx, q)
	if err != nil {
		return err
	}
	return WriteCSV(w, result.Columns, result.Rows)
}

func WriteCSV(w io.Writer, columns []string, rows [][]any) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(columns); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(row))
		for i, cell := range row {
			record[i] = csvCell(cell)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func CSVBytes(columns []string, rows [][]any) ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, columns, rows); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func csvCell(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprint(v)
	}
}

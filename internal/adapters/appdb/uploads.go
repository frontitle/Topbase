package appdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

// Old and new uploads store the sheet as a JSON array. Read a bounded
// prefix so opening a preview never transfers the full sheet to the server.
func (s *Store) PreviewUploadedTable(ctx context.Context, id string) (core.UploadedTablePreview, error) {
	const byteLimit = 128 << 10
	const rowLimit = 50
	preview := core.UploadedTablePreview{Rows: [][]string{}}
	var prefix string
	err := s.db.QueryRowContext(ctx, `SELECT row_count, SUBSTR(data_json, 1, ?) FROM uploaded_tables WHERE id=?`, byteLimit+1, id).
		Scan(&preview.TotalRows, &prefix)
	if err != nil || preview.TotalRows == 0 {
		return preview, err
	}
	// SUBSTR counts characters across supported engines. Apply the byte limit
	// before decoding, including for sheets with multibyte cell contents.
	clipped := len(prefix) > byteLimit
	if clipped {
		prefix = prefix[:byteLimit]
	}
	decoder := json.NewDecoder(strings.NewReader(prefix))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('[') {
		return preview, fmt.Errorf("invalid uploaded table data")
	}
	for len(preview.Rows) < min(preview.TotalRows, rowLimit) {
		if !decoder.More() {
			if clipped {
				preview.ByteLimited = true
				break
			}
			return preview, fmt.Errorf("uploaded table is missing rows")
		}
		var row []string
		if err := decoder.Decode(&row); err != nil {
			if clipped && (errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)) {
				preview.ByteLimited = true
				break
			}
			return preview, err
		}
		preview.Rows = append(preview.Rows, row)
	}
	return preview, nil
}

func (s *Store) SaveUploadedTable(item core.UploadedTable, rows [][]string) error {
	columns, err := json.Marshal(item.Columns)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO uploaded_tables(id,name,file_name,sheet_name,columns_json,row_count,data_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, item.ID, item.Name, item.FileName, item.SheetName, string(columns), item.RowCount, string(payload), item.CreatedAt)
	return err
}

func (s *Store) ListUploadedTables() ([]core.UploadedTable, error) {
	rows, err := s.db.Query(`SELECT id,name,file_name,sheet_name,columns_json,row_count,created_at FROM uploaded_tables ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.UploadedTable{}
	for rows.Next() {
		var item core.UploadedTable
		var columns string
		var created time.Time
		if err := rows.Scan(&item.ID, &item.Name, &item.FileName, &item.SheetName, &columns, &item.RowCount, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = created
		_ = json.Unmarshal([]byte(columns), &item.Columns)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteUploadedTable(id string) error {
	_, err := s.db.Exec(`DELETE FROM uploaded_tables WHERE id=?`, id)
	return err
}

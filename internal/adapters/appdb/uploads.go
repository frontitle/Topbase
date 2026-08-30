package appdb

import (
	"encoding/json"
	"time"

	"github.com/topbase/topbase/internal/core"
)

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

package core

import "time"

// UploadedTable is a user-provided workbook sheet retained by Topbase as part
// of its local warehouse. Rows are intentionally kept behind the store API;
// callers receive only safe catalog metadata.
type UploadedTable struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	FileName  string    `json:"file_name"`
	SheetName string    `json:"sheet_name"`
	Columns   []Column  `json:"columns"`
	RowCount  int       `json:"row_count"`
	CreatedAt time.Time `json:"created_at"`
}

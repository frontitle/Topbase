CREATE TABLE IF NOT EXISTS uploaded_tables (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  file_name TEXT NOT NULL,
  sheet_name TEXT NOT NULL,
  columns_json TEXT NOT NULL,
  row_count INTEGER NOT NULL DEFAULT 0,
  data_json TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL
);

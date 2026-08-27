-- W3 semantic fields, models, metrics, segments, glossary, native parameters.
ALTER TABLE questions ADD COLUMN parameters TEXT;

CREATE TABLE IF NOT EXISTS field_metadata (
  database_id TEXT NOT NULL,
  schema_name TEXT NOT NULL,
  table_name TEXT NOT NULL,
  field_name TEXT NOT NULL,
  display_name TEXT,
  description TEXT,
  semantic_type TEXT,
  visibility TEXT,
  format TEXT,
  fk_schema TEXT,
  fk_table TEXT,
  fk_field TEXT,
  PRIMARY KEY (database_id, schema_name, table_name, field_name)
);

CREATE TABLE IF NOT EXISTS models (
  id TEXT PRIMARY KEY,
  collection_id TEXT,
  name TEXT NOT NULL,
  database_id TEXT NOT NULL,
  queryir TEXT,
  native_sql TEXT,
  columns TEXT,
  created_by TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS metrics (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  database_id TEXT NOT NULL,
  schema_name TEXT NOT NULL,
  table_name TEXT NOT NULL,
  aggregation TEXT NOT NULL,
  filters TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS segments (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  database_id TEXT NOT NULL,
  schema_name TEXT NOT NULL,
  table_name TEXT NOT NULL,
  filters TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS glossary_terms (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  definition TEXT NOT NULL
);

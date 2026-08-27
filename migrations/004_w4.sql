-- W4 warehouse schedules, runs, materialized tables, lineage.
CREATE TABLE IF NOT EXISTS schedules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  question_id TEXT NOT NULL,
  database_id TEXT,
  cron TEXT NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
  materialize_to TEXT NOT NULL,
  strategy TEXT NOT NULL DEFAULT 'replace',
  enabled INTEGER NOT NULL DEFAULT 1,
  last_run_at TEXT,
  created_by TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS schedule_runs (
  id TEXT PRIMARY KEY,
  schedule_id TEXT NOT NULL,
  status TEXT NOT NULL,
  sql_compiled TEXT,
  row_count INTEGER NOT NULL DEFAULT 0,
  error TEXT,
  started_at TEXT NOT NULL,
  finished_at TEXT
);

CREATE TABLE IF NOT EXISTS materialized_tables (
  id TEXT PRIMARY KEY,
  database_id TEXT NOT NULL,
  schema_name TEXT NOT NULL,
  table_name TEXT NOT NULL,
  schedule_id TEXT,
  question_id TEXT,
  last_run_at TEXT,
  last_status TEXT,
  row_count INTEGER NOT NULL DEFAULT 0,
  UNIQUE(database_id, schema_name, table_name)
);

CREATE TABLE IF NOT EXISTS lineage_edges (
  from_type TEXT NOT NULL,
  from_id TEXT NOT NULL,
  to_type TEXT NOT NULL,
  to_id TEXT NOT NULL
);

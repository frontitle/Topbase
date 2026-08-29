-- Topbase application database (SQLite for local; SQL kept portable).
-- Reference snapshot for maintainers. Runtime schema changes are applied only
-- through the append-only numbered files in /migrations.
CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  avatar_url TEXT NOT NULL DEFAULT '',
  password_hash TEXT,
  feishu_open_id TEXT UNIQUE,
  locale TEXT NOT NULL DEFAULT 'zh-CN',
  theme TEXT NOT NULL DEFAULT 'dark',
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS groups (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  kind TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS group_members (
  group_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  PRIMARY KEY (group_id, user_id)
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS collections (
  id TEXT PRIMARY KEY,
  parent_id TEXT,
  name TEXT NOT NULL,
  personal_owner_user_id TEXT,
	owner_group_id TEXT,
	kind TEXT NOT NULL DEFAULT 'team_project',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS collection_shares (
  collection_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (collection_id, user_id)
);

CREATE TABLE IF NOT EXISTS questions (
  id TEXT PRIMARY KEY,
  collection_id TEXT,
  dashboard_id TEXT,
  name TEXT NOT NULL,
  description TEXT,
  queryir TEXT,
  native_sql TEXT,
  chartspec TEXT,
  query_type TEXT NOT NULL,
  database_id TEXT,
  created_by TEXT,
  created_at TEXT NOT NULL,
  archived_at TEXT,
  parameters TEXT
);

CREATE TABLE IF NOT EXISTS catalog_databases (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  engine TEXT NOT NULL,
  host TEXT,
  status TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS table_annotations (
  database_id TEXT NOT NULL,
  schema_name TEXT NOT NULL,
  table_name TEXT NOT NULL,
  display_name TEXT,
  description TEXT,
  user_note TEXT NOT NULL DEFAULT '',
  hidden INTEGER NOT NULL DEFAULT 0,
  field_types TEXT,
  PRIMARY KEY (database_id, schema_name, table_name)
);

CREATE TABLE IF NOT EXISTS dashboards (
  id TEXT PRIMARY KEY,
  collection_id TEXT,
  name TEXT NOT NULL,
  description TEXT,
  auto_refresh_seconds INTEGER,
  appearance TEXT,
  public_uuid TEXT,
  public_embed_enabled INTEGER NOT NULL DEFAULT 0,
  archived_at TEXT,
  created_by TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dashboard_tabs (
  id TEXT PRIMARY KEY,
  dashboard_id TEXT NOT NULL,
  name TEXT NOT NULL,
  position INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS dashboard_cards (
  id TEXT PRIMARY KEY,
  dashboard_id TEXT NOT NULL,
  tab_id TEXT,
  type TEXT NOT NULL,
  question_id TEXT,
  title TEXT,
  body TEXT,
  config TEXT,
  layout TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dashboard_filters (
  id TEXT PRIMARY KEY,
  dashboard_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  config TEXT,
  mappings TEXT
);

CREATE TABLE IF NOT EXISTS bookmarks (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(user_id, target_type, target_id)
);

CREATE TABLE IF NOT EXISTS revisions (
  id TEXT PRIMARY KEY,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  actor_id TEXT,
  snapshot TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
  id TEXT PRIMARY KEY,
  question_id TEXT NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  cron TEXT,
  channel TEXT NOT NULL DEFAULT 'inbox',
  goal REAL,
  once INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_by TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  alert_id TEXT,
  title TEXT NOT NULL,
  body TEXT,
  created_at TEXT NOT NULL,
  read_at TEXT
);

CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  prefix TEXT NOT NULL,
  hash TEXT NOT NULL,
  user_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT
);

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

CREATE TABLE IF NOT EXISTS schedules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  question_id TEXT,
  database_id TEXT,
  cron TEXT NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
  materialize_to TEXT NOT NULL,
  strategy TEXT NOT NULL DEFAULT 'replace',
  watermark_field TEXT,
  model_id TEXT,
  confirm_source_write INTEGER NOT NULL DEFAULT 0,
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
  watermark TEXT,
  UNIQUE(database_id, schema_name, table_name)
);

CREATE TABLE IF NOT EXISTS lineage_edges (
  from_type TEXT NOT NULL,
  from_id TEXT NOT NULL,
  to_type TEXT NOT NULL,
  to_id TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS subscriptions (
  id TEXT PRIMARY KEY,
  dashboard_id TEXT NOT NULL,
  cron TEXT NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
  channel TEXT NOT NULL DEFAULT 'inbox',
  enabled INTEGER NOT NULL DEFAULT 1,
  last_run_at TEXT,
  created_by TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS schema_snapshots (
  database_id TEXT PRIMARY KEY,
  tables_json TEXT NOT NULL,
  synced_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS topbase_installation (
  id TEXT PRIMARY KEY,
  installation_id TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS connection_secrets (
  database_id TEXT PRIMARY KEY,
  ciphertext TEXT NOT NULL,
  nonce TEXT NOT NULL,
  key_version INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS distributed_leases (
  name TEXT PRIMARY KEY,
  owner TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

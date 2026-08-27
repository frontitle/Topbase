-- Topbase application database (SQLite for local; SQL kept portable).
CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
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
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS questions (
  id TEXT PRIMARY KEY,
  collection_id TEXT,
  name TEXT NOT NULL,
  description TEXT,
  queryir TEXT,
  native_sql TEXT,
  chartspec TEXT,
  query_type TEXT NOT NULL,
  database_id TEXT,
  created_by TEXT,
  created_at TEXT NOT NULL,
  archived_at TEXT
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
  field_types TEXT,
  PRIMARY KEY (database_id, schema_name, table_name)
);

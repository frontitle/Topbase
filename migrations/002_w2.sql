-- W2 dashboards, sharing, alerts, API keys.
ALTER TABLE questions ADD COLUMN dashboard_id TEXT;

CREATE TABLE IF NOT EXISTS dashboards (
  id TEXT PRIMARY KEY,
  collection_id TEXT,
  name TEXT NOT NULL,
  description TEXT,
  auto_refresh_seconds INTEGER,
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
  created_at TEXT NOT NULL
);

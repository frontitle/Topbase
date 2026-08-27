-- W4b incremental watermark, Feishu department groups, dashboard subscriptions.
ALTER TABLE schedules ADD COLUMN watermark_field TEXT;
ALTER TABLE schedules ADD COLUMN model_id TEXT;
ALTER TABLE schedules ADD COLUMN confirm_source_write INTEGER NOT NULL DEFAULT 0;
ALTER TABLE materialized_tables ADD COLUMN watermark TEXT;
ALTER TABLE users ADD COLUMN feishu_open_id TEXT;

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

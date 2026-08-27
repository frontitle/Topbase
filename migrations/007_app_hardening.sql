-- Application hardening: explicit data-group ownership and dashboard sharing.
ALTER TABLE dashboards ADD COLUMN public_uuid TEXT;
ALTER TABLE dashboards ADD COLUMN appearance TEXT;
ALTER TABLE collections ADD COLUMN kind TEXT NOT NULL DEFAULT 'team_project';
ALTER TABLE collections ADD COLUMN owner_group_id TEXT;

UPDATE collections
SET kind = 'personal_project'
WHERE personal_owner_user_id IS NOT NULL AND personal_owner_user_id <> '';

UPDATE collections
SET name = '我的分析'
WHERE name = '我的问数'
  AND personal_owner_user_id IS NOT NULL
  AND personal_owner_user_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS dashboards_public_uuid_unique
ON dashboards(public_uuid)
WHERE public_uuid IS NOT NULL AND public_uuid <> '';

-- Migration 004 made question_id mandatory. Model-backed schedules do not have
-- a question, so rebuild the table with the current nullable contract.
CREATE TABLE schedules_v7 (
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

INSERT INTO schedules_v7 (
  id, name, question_id, database_id, cron, timezone, materialize_to,
  strategy, watermark_field, model_id, confirm_source_write, enabled,
  last_run_at, created_by, created_at
)
SELECT
  id, name, question_id, database_id, cron, timezone, materialize_to,
  strategy, watermark_field, model_id, confirm_source_write, enabled,
  last_run_at, created_by, created_at
FROM schedules;

DROP TABLE schedules;
ALTER TABLE schedules_v7 RENAME TO schedules;

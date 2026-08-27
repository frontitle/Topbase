-- Schema snapshots for database sync / field rescan.
CREATE TABLE IF NOT EXISTS schema_snapshots (
  database_id TEXT PRIMARY KEY,
  tables_json TEXT NOT NULL,
  synced_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS warehouse_target_claims (
  target_key VARCHAR(64) PRIMARY KEY,
  schedule_id VARCHAR(64) NOT NULL UNIQUE
);

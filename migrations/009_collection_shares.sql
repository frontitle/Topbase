-- Personal analysis groups can be shared directly with individual accounts.
CREATE TABLE IF NOT EXISTS collection_shares (
  collection_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (collection_id, user_id)
);

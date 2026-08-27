package appdb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func (s *Store) SaveSnapshot(item core.SchemaSnapshot) error {
	raw, err := json.Marshal(item.Tables)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO schema_snapshots(database_id, tables_json, synced_at) VALUES(?,?,?)
		ON CONFLICT(database_id) DO UPDATE SET tables_json=excluded.tables_json, synced_at=excluded.synced_at`,
		item.DatabaseID, string(raw), item.SyncedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetSnapshot(databaseID string) (core.SchemaSnapshot, error) {
	var raw, synced string
	err := s.db.QueryRow(`SELECT tables_json, synced_at FROM schema_snapshots WHERE database_id=?`, databaseID).Scan(&raw, &synced)
	if errors.Is(err, sql.ErrNoRows) {
		return core.SchemaSnapshot{}, core.ErrNotFound
	}
	if err != nil {
		return core.SchemaSnapshot{}, err
	}
	item := core.SchemaSnapshot{DatabaseID: databaseID, Tables: []core.Table{}}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &item.Tables)
	}
	item.SyncedAt, _ = time.Parse(time.RFC3339, synced)
	return item, nil
}

func (s *Store) DeleteSnapshot(databaseID string) error {
	_, err := s.db.Exec(`DELETE FROM schema_snapshots WHERE database_id=?`, databaseID)
	return err
}

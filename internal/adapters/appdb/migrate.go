package appdb

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	appmigrations "github.com/topbase/topbase/migrations"
)

const migrationTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  checksum TEXT NOT NULL,
  applied_at TEXT NOT NULL,
  app_version TEXT NOT NULL
)`

func applyMigrations(ctx context.Context, db *sql.DB, appVersion string) (int, error) {
	if _, err := db.ExecContext(ctx, migrationTableSQL); err != nil {
		return 0, fmt.Errorf("create migration ledger: %w", err)
	}
	files, err := appmigrations.Files()
	if err != nil {
		return 0, err
	}
	applied, err := readAppliedMigrations(ctx, db)
	if err != nil {
		return 0, err
	}
	latest := 0
	for _, file := range files {
		checksum := migrationChecksum(file.SQL)
		if existing, ok := applied[file.Version]; ok {
			if existing != checksum {
				return 0, fmt.Errorf("migration %03d checksum changed; restore the released SQL and add a new migration", file.Version)
			}
			latest = file.Version
			continue
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return 0, fmt.Errorf("begin migration %03d: %w", file.Version, err)
		}
		for _, statement := range splitMigrationStatements(file.SQL) {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				// Databases created before the migration ledger used the current
				// schema snapshot plus additive ALTER statements. A duplicate
				// column therefore means this exact additive change already landed.
				if !isDuplicateColumn(err) {
					_ = tx.Rollback()
					return 0, fmt.Errorf("apply migration %03d (%s): %w", file.Version, file.Name, err)
				}
			}
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, name, checksum, applied_at, app_version) VALUES(?,?,?,?,?)`,
			file.Version, file.Name, checksum, time.Now().UTC().Format(time.RFC3339Nano), appVersion,
		); err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("record migration %03d: %w", file.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return 0, fmt.Errorf("commit migration %03d: %w", file.Version, err)
		}
		latest = file.Version
	}
	return latest, nil
}

func readAppliedMigrations(ctx context.Context, db *sql.DB) (map[int]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read migration ledger: %w", err)
	}
	defer rows.Close()
	items := map[int]string{}
	for rows.Next() {
		var version int
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		items[version] = checksum
	}
	return items, rows.Err()
}

func splitMigrationStatements(raw string) []string {
	lines := strings.Split(raw, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		clean = append(clean, line)
	}
	parts := strings.Split(strings.Join(clean, "\n"), ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		if statement := strings.TrimSpace(part); statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}

func migrationChecksum(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func isDuplicateColumn(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
}

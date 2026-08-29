package appdb

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const productionMigrationSetting = "application_database_production_migrated_at"

type TransferReport struct {
	SourceEngine    Engine           `json:"source_engine"`
	TargetEngine    Engine           `json:"target_engine"`
	Tables          int              `json:"tables"`
	Rows            int64            `json:"rows"`
	RowsByTable     map[string]int64 `json:"rows_by_table"`
	CompletedAt     time.Time        `json:"completed_at"`
	RestartRequired bool             `json:"restart_required"`
}

type BackupManifest struct {
	Format        string           `json:"format"`
	Version       string           `json:"version"`
	Engine        Engine           `json:"engine"`
	SchemaVersion int              `json:"schema_version"`
	CreatedAt     time.Time        `json:"created_at"`
	RowsByTable   map[string]int64 `json:"rows_by_table"`
	Sensitive     bool             `json:"sensitive"`
}

func (s *Store) ProductionMigrationCompleted() (bool, string, error) {
	value, ok, err := s.Get(productionMigrationSetting)
	return ok && strings.TrimSpace(value) != "", value, err
}

// MigrateSQLiteToProduction copies a consistent snapshot into a freshly
// initialized PostgreSQL or MySQL application database. It never switches the
// running process; operators must apply the returned configuration and restart.
func (s *Store) MigrateSQLiteToProduction(ctx context.Context, targetConfig Config) (TransferReport, error) {
	if s.Engine() != EngineSQLite {
		return TransferReport{}, errors.New("one-time production migration is only available while Topbase is using SQLite")
	}
	if done, completedAt, err := s.ProductionMigrationCompleted(); err != nil {
		return TransferReport{}, err
	} else if done {
		return TransferReport{}, fmt.Errorf("production migration was already completed at %s", completedAt)
	}
	if targetConfig.Engine != EnginePostgres && targetConfig.Engine != EngineMySQL {
		return TransferReport{}, errors.New("target engine must be postgres or mysql")
	}
	if targetConfig.AppVersion == "" {
		targetConfig.AppVersion = "dev"
	}
	target, err := OpenConfig(targetConfig)
	if err != nil {
		return TransferReport{}, fmt.Errorf("prepare production application database: %w", err)
	}
	defer target.Close()

	tables, err := s.applicationTables(ctx)
	if err != nil {
		return TransferReport{}, err
	}
	for _, table := range tables {
		if table == "schema_migrations" || table == "topbase_installation" {
			continue
		}
		var count int64
		if err := target.db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+target.db.quoteIdentifier(table)).Scan(&count); err != nil {
			return TransferReport{}, fmt.Errorf("inspect target table %s: %w", table, err)
		}
		if count != 0 {
			return TransferReport{}, fmt.Errorf("target Topbase database is not empty: table %s contains %d rows", table, count)
		}
	}

	sourceTx, err := s.db.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelSerializable})
	if err != nil {
		return TransferReport{}, fmt.Errorf("start SQLite snapshot: %w", err)
	}
	defer sourceTx.Rollback()
	targetTx, err := target.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return TransferReport{}, fmt.Errorf("start target transaction: %w", err)
	}
	rollback := func(cause error) (TransferReport, error) {
		return TransferReport{}, errors.Join(cause, targetTx.Rollback())
	}
	// OpenConfig creates a target-local installation identity as part of schema
	// bootstrap. Replace it with the source identity so encrypted runtime state
	// and deployment identity continue as one installation after restart.
	if _, err := targetTx.ExecContext(ctx, "DELETE FROM "+target.db.quoteIdentifier("topbase_installation")); err != nil {
		return rollback(fmt.Errorf("prepare target installation identity: %w", err))
	}

	report := TransferReport{
		SourceEngine: EngineSQLite, TargetEngine: target.Engine(), RowsByTable: map[string]int64{},
		CompletedAt: time.Now().UTC(), RestartRequired: true,
	}
	for _, table := range tables {
		if table == "schema_migrations" || table == "distributed_leases" || table == "sessions" {
			continue
		}
		count, err := copyTable(ctx, sourceTx, targetTx, target.db, table)
		if err != nil {
			return rollback(err)
		}
		report.Tables++
		report.Rows += count
		report.RowsByTable[table] = count
	}
	marker := report.CompletedAt.Format(time.RFC3339)
	markerSQL := target.db.rewrite(`INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`)
	if _, err := targetTx.ExecContext(ctx, markerSQL, productionMigrationSetting, marker); err != nil {
		return rollback(fmt.Errorf("record target migration: %w", err))
	}
	if err := targetTx.Commit(); err != nil {
		return TransferReport{}, fmt.Errorf("commit production migration: %w", err)
	}
	if err := sourceTx.Commit(); err != nil {
		return TransferReport{}, fmt.Errorf("finish SQLite snapshot: %w", err)
	}
	if err := s.Set(productionMigrationSetting, marker); err != nil {
		return TransferReport{}, fmt.Errorf("production data was copied but the local completion marker failed: %w", err)
	}
	return report, nil
}

func copyTable(ctx context.Context, source *sql.Tx, target *sql.Tx, targetDB *database, table string) (int64, error) {
	rows, err := source.QueryContext(ctx, "SELECT * FROM "+quoteIdentifier(EngineSQLite, table))
	if err != nil {
		return 0, fmt.Errorf("read source table %s: %w", table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	quoted := make([]string, len(columns))
	placeholders := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = targetDB.quoteIdentifier(column)
		placeholders[i] = "?"
	}
	statement := "INSERT INTO " + targetDB.quoteIdentifier(table) + " (" + strings.Join(quoted, ",") + ") VALUES (" + strings.Join(placeholders, ",") + ")"
	if targetDB.engine == EnginePostgres {
		statement = rebindPostgres(statement)
	}
	var count int64
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return 0, fmt.Errorf("scan source table %s: %w", table, err)
		}
		if _, err := target.ExecContext(ctx, statement, values...); err != nil {
			return 0, fmt.Errorf("write target table %s: %w", table, err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var targetCount int64
	if err := target.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+targetDB.quoteIdentifier(table)).Scan(&targetCount); err != nil {
		return 0, fmt.Errorf("verify target table %s: %w", table, err)
	}
	if targetCount != count {
		return 0, fmt.Errorf("verify target table %s: copied %d rows but found %d", table, count, targetCount)
	}
	return count, nil
}

// ExportLogical writes a consistent, engine-neutral ZIP. Each table is a JSONL
// stream whose first row declares columns; remaining rows contain values.
func (s *Store) ExportLogical(ctx context.Context, writer io.Writer, version string) (BackupManifest, error) {
	tables, err := s.applicationTables(ctx)
	if err != nil {
		return BackupManifest{}, err
	}
	tx, err := s.db.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return BackupManifest{}, err
	}
	defer tx.Rollback()
	archive := zip.NewWriter(writer)
	manifest := BackupManifest{
		Format: "topbase-logical-backup-v1", Version: version, Engine: s.Engine(), SchemaVersion: s.SchemaVersion(),
		CreatedAt: time.Now().UTC(), RowsByTable: map[string]int64{}, Sensitive: true,
	}
	for _, table := range tables {
		entry, err := archive.Create("tables/" + table + ".jsonl")
		if err != nil {
			return BackupManifest{}, err
		}
		rows, err := tx.QueryContext(ctx, "SELECT * FROM "+s.db.quoteIdentifier(table))
		if err != nil {
			return BackupManifest{}, fmt.Errorf("export table %s: %w", table, err)
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return BackupManifest{}, err
		}
		encoder := json.NewEncoder(entry)
		if err := encoder.Encode(map[string]any{"type": "columns", "columns": columns}); err != nil {
			rows.Close()
			return BackupManifest{}, err
		}
		var count int64
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for i := range values {
				destinations[i] = &values[i]
			}
			if err := rows.Scan(destinations...); err != nil {
				rows.Close()
				return BackupManifest{}, err
			}
			if err := encoder.Encode(map[string]any{"type": "row", "values": values}); err != nil {
				rows.Close()
				return BackupManifest{}, err
			}
			count++
		}
		if err := rows.Close(); err != nil {
			return BackupManifest{}, err
		}
		manifest.RowsByTable[table] = count
	}
	manifestEntry, err := archive.Create("manifest.json")
	if err != nil {
		return BackupManifest{}, err
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	if _, err := manifestEntry.Write(append(manifestJSON, '\n')); err != nil {
		return BackupManifest{}, err
	}
	if err := archive.Close(); err != nil {
		return BackupManifest{}, err
	}
	if err := tx.Commit(); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

func (s *Store) applicationTables(ctx context.Context) ([]string, error) {
	var query string
	switch s.Engine() {
	case EngineSQLite:
		query = `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	case EnginePostgres:
		query = `SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() AND table_type='BASE TABLE' ORDER BY table_name`
	case EngineMySQL:
		query = `SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type='BASE TABLE' ORDER BY table_name`
	default:
		return nil, fmt.Errorf("unsupported application database engine %s", s.Engine())
	}
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	sort.Strings(tables)
	return tables, rows.Err()
}

func (d *database) quoteIdentifier(value string) string { return quoteIdentifier(d.engine, value) }

func quoteIdentifier(engine Engine, value string) string {
	if engine == EngineMySQL {
		return "`" + strings.ReplaceAll(value, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

package appdb

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"regexp"
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

// legacyMigration010Checksum is the checksum written by the development build
// that first introduced migration 010. Its SQL only differed from the released
// file by one trailing blank line, which has no effect on the schema.
const legacyMigration010Checksum = "4a04457ec92ce1f35d127af9ef18426bfd4beb2586fb73cf7c093622f26bf1b6"
const releasedMigration010Checksum = "ae81ddd40e199a3dfd7590281c7707966d5a187b683da3fe163ee8c6a7cbb01d"

//go:embed schema.sql
var currentSchemaSQL string

func applyMigrations(ctx context.Context, db *database, appVersion string) (int, error) {
	var version int
	err := withMigrationLock(ctx, db, func() error {
		state, err := inspectNamespace(ctx, db)
		if err != nil {
			return err
		}
		if state.occupied && !state.hasLedger && db.engine != EngineSQLite {
			return fmt.Errorf("application database namespace is not empty and does not contain Topbase migration metadata; use an empty database or dedicated schema")
		}
		if !state.occupied && db.engine != EngineSQLite {
			version, err = bootstrapCurrentSchema(ctx, db, appVersion)
			return err
		}
		version, err = migrateIncrementally(ctx, db, appVersion)
		return err
	})
	return version, err
}

func migrateIncrementally(ctx context.Context, db *database, appVersion string) (int, error) {
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
			if !migrationChecksumMatches(file.Version, existing, checksum) {
				return 0, fmt.Errorf("migration %03d checksum changed; restore the released SQL and add a new migration", file.Version)
			}
			latest = file.Version
			continue
		}
		tx, err := db.DB.BeginTx(ctx, nil)
		if err != nil {
			return 0, fmt.Errorf("begin migration %03d: %w", file.Version, err)
		}
		for _, statement := range splitMigrationStatements(file.SQL) {
			if _, err := tx.ExecContext(ctx, db.rewrite(statement)); err != nil {
				// Databases created before the migration ledger used the current
				// schema snapshot plus additive ALTER statements. A duplicate
				// column therefore means this exact additive change already landed.
				if !isDuplicateColumn(err) {
					_ = tx.Rollback()
					return 0, fmt.Errorf("apply migration %03d (%s): %w", file.Version, file.Name, err)
				}
			}
		}
		if _, err := tx.ExecContext(ctx, db.rewrite(
			`INSERT INTO schema_migrations(version, name, checksum, applied_at, app_version) VALUES(?,?,?,?,?)`),
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
	if err := ensureInstallationRecord(ctx, db); err != nil {
		return 0, err
	}
	return latest, nil
}

func migrationChecksumMatches(version int, existing, current string) bool {
	if existing == current {
		return true
	}
	// The released 010 migration removed one trailing blank line from the
	// development build. Accept only that exact, schema-equivalent checksum.
	return version == 10 && existing == legacyMigration010Checksum && current == releasedMigration010Checksum
}

func readAppliedMigrations(ctx context.Context, db *database) (map[int]string, error) {
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
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column name") ||
		(strings.Contains(message, "column") && strings.Contains(message, "already exists"))
}

type namespaceState struct {
	occupied  bool
	hasLedger bool
}

func inspectNamespace(ctx context.Context, db *database) (namespaceState, error) {
	var query string
	switch db.engine {
	case EnginePostgres:
		query = `SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = current_schema()`
	case EngineMySQL:
		query = `SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()`
	default:
		query = `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return namespaceState{}, fmt.Errorf("inspect application database namespace: %w", err)
	}
	defer rows.Close()
	state := namespaceState{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return namespaceState{}, err
		}
		state.occupied = true
		if strings.EqualFold(name, "schema_migrations") {
			state.hasLedger = true
		}
	}
	return state, rows.Err()
}

func bootstrapCurrentSchema(ctx context.Context, db *database, appVersion string) (int, error) {
	files, err := appmigrations.Files()
	if err != nil {
		return 0, err
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin application database initialization: %w", err)
	}
	rollback := func(cause error) (int, error) {
		_ = tx.Rollback()
		return 0, cause
	}
	statements := splitMigrationStatements(currentSchemaSQL)
	for _, statement := range statements {
		if db.engine == EngineMySQL {
			statement = mysqlBootstrapStatement(statement)
		}
		if _, err := tx.ExecContext(ctx, db.rewrite(statement)); err != nil {
			return rollback(fmt.Errorf("initialize application database schema: %w", err))
		}
	}
	if db.engine == EnginePostgres {
		if _, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS dashboards_public_uuid_unique ON dashboards(public_uuid) WHERE public_uuid IS NOT NULL AND public_uuid <> ''`); err != nil {
			return rollback(fmt.Errorf("create application database indexes: %w", err))
		}
	} else if db.engine == EngineMySQL {
		if _, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX dashboards_public_uuid_unique ON dashboards(public_uuid)`); err != nil {
			return rollback(fmt.Errorf("create application database indexes: %w", err))
		}
	}
	if _, err := tx.ExecContext(ctx, db.rewrite(migrationTableSQL)); err != nil {
		return rollback(fmt.Errorf("create migration ledger: %w", err))
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, file := range files {
		if _, err := tx.ExecContext(ctx, db.rewrite(
			`INSERT INTO schema_migrations(version, name, checksum, applied_at, app_version) VALUES(?,?,?,?,?)`),
			file.Version, file.Name, migrationChecksum(file.SQL), now, appVersion,
		); err != nil {
			return rollback(fmt.Errorf("record bootstrap migration %03d: %w", file.Version, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit application database initialization: %w", err)
	}
	if err := ensureInstallationRecord(ctx, db); err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, nil
	}
	return files[len(files)-1].Version, nil
}

func ensureInstallationRecord(ctx context.Context, db *database) error {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Errorf("generate installation id: %w", err)
	}
	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO topbase_installation(id, installation_id, created_at) VALUES(?,?,?)`,
		"topbase", hex.EncodeToString(random[:]), time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("record Topbase installation identity: %w", err)
	}
	return nil
}

func withMigrationLock(ctx context.Context, db *database, fn func() error) error {
	if db.engine == EngineSQLite {
		return fn()
	}
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve migration lock connection: %w", err)
	}
	defer conn.Close()
	if db.engine == EnginePostgres {
		if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(607235432102941003)`); err != nil {
			return fmt.Errorf("acquire PostgreSQL migration lock: %w", err)
		}
		defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(607235432102941003)`)
	} else {
		var acquired sql.NullInt64
		if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK('topbase_schema_migrations', 120)`).Scan(&acquired); err != nil {
			return fmt.Errorf("acquire MySQL migration lock: %w", err)
		}
		if !acquired.Valid || acquired.Int64 != 1 {
			return fmt.Errorf("acquire MySQL migration lock: timed out")
		}
		defer conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK('topbase_schema_migrations')`)
	}
	return fn()
}

var mysqlColumnLine = regexp.MustCompile(`(?i)^(\s*)([a-z_][a-z0-9_]*)(\s+)TEXT(.*)$`)

func mysqlBootstrapStatement(statement string) string {
	upper := strings.ToUpper(strings.TrimSpace(statement))
	if !strings.HasPrefix(upper, "CREATE TABLE") {
		return statement
	}
	keyed := mysqlIndexedColumns(statement)
	lines := strings.Split(statement, "\n")
	for index, line := range lines {
		match := mysqlColumnLine.FindStringSubmatch(line)
		if len(match) != 5 {
			continue
		}
		name := strings.ToLower(match[2])
		typeName := "LONGTEXT"
		if keyed[name] || mysqlShortTextColumn(name) || strings.Contains(strings.ToUpper(match[4]), " DEFAULT ") {
			typeName = "VARCHAR(255)"
		}
		if keyed[name] {
			typeName = "VARCHAR(191)"
		}
		lines[index] = match[1] + match[2] + match[3] + typeName + match[4]
	}
	return strings.Join(lines, "\n") + " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"
}

func mysqlIndexedColumns(statement string) map[string]bool {
	items := map[string]bool{}
	constraint := regexp.MustCompile(`(?i)(?:PRIMARY\s+KEY|UNIQUE)\s*\(([^)]*)\)`)
	for _, match := range constraint.FindAllStringSubmatch(statement, -1) {
		for _, value := range strings.Split(match[1], ",") {
			items[strings.ToLower(strings.Trim(strings.TrimSpace(value), "`\""))] = true
		}
	}
	inline := regexp.MustCompile(`(?im)^\s*([a-z_][a-z0-9_]*)\s+TEXT[^\n]*(?:PRIMARY\s+KEY|UNIQUE)`)
	for _, match := range inline.FindAllStringSubmatch(statement, -1) {
		items[strings.ToLower(match[1])] = true
	}
	return items
}

func mysqlShortTextColumn(name string) bool {
	short := map[string]bool{
		"aggregation": true, "channel": true, "cron": true, "email": true, "engine": true,
		"expires_at": true, "field_name": true, "format": true, "host": true, "kind": true,
		"locale": true, "materialize_to": true, "name": true, "prefix": true, "public_uuid": true,
		"schema_name": true, "semantic_type": true, "status": true, "strategy": true, "table_name": true,
		"theme": true, "timezone": true, "type": true, "visibility": true, "watermark_field": true,
	}
	if name == "id" || strings.HasSuffix(name, "_id") || strings.HasSuffix(name, "_at") {
		return true
	}
	return short[name]
}

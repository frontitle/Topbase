package appdb

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func TestProductionMigrationIntegration(t *testing.T) {
	engine := Engine(os.Getenv("TOPBASE_TEST_MIGRATION_ENGINE"))
	dsn := os.Getenv("TOPBASE_TEST_MIGRATION_DSN")
	if engine == "" || dsn == "" {
		t.Skip("set TOPBASE_TEST_MIGRATION_ENGINE and TOPBASE_TEST_MIGRATION_DSN")
	}
	source, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := source.Set("site_name", "Migrated Lab"); err != nil {
		t.Fatal(err)
	}
	report, err := source.MigrateSQLiteToProduction(context.Background(), Config{Engine: engine, DSN: dsn, AppVersion: "integration", ConnectTimeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if report.TargetEngine != engine || report.RowsByTable["settings"] < 1 || !report.RestartRequired {
		t.Fatalf("report = %+v", report)
	}
	target, err := OpenConfig(Config{Engine: engine, DSN: dsn, AppVersion: "integration", ConnectTimeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if value, ok, err := target.Get("site_name"); err != nil || !ok || value != "Migrated Lab" {
		t.Fatalf("migrated site_name = %q, %v, %v", value, ok, err)
	}
	if _, err := source.MigrateSQLiteToProduction(context.Background(), Config{Engine: engine, DSN: dsn, AppVersion: "integration"}); err == nil {
		t.Fatal("second production migration unexpectedly succeeded")
	}
}

func TestLogicalBackupContainsManifestAndApplicationRows(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Set("site_name", "Backup Lab"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	manifest, err := store.ExportLogical(context.Background(), &output, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Format != "topbase-logical-backup-v1" || manifest.RowsByTable["settings"] != 1 || !manifest.Sensitive {
		t.Fatalf("manifest = %+v", manifest)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]bool{}
	for _, file := range archive.File {
		files[file.Name] = true
	}
	if !files["manifest.json"] || !files["tables/settings.jsonl"] || !files["tables/users.jsonl"] {
		t.Fatalf("backup files = %#v", files)
	}
}

func TestPortableQueryRewrite(t *testing.T) {
	postgres := (&database{engine: EnginePostgres}).rewrite(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`)
	if !strings.Contains(postgres, `VALUES($1,$2)`) || !strings.Contains(postgres, `excluded.value`) {
		t.Fatalf("unexpected PostgreSQL query: %s", postgres)
	}
	mysql := (&database{engine: EngineMySQL}).rewrite(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`)
	if !strings.Contains(mysql, `ON DUPLICATE KEY UPDATE value=VALUES(value)`) {
		t.Fatalf("unexpected MySQL query: %s", mysql)
	}
	ignored := (&database{engine: EnginePostgres}).rewrite(`INSERT OR IGNORE INTO group_members(group_id,user_id) VALUES(?,?)`)
	if !strings.Contains(ignored, `ON CONFLICT DO NOTHING`) || !strings.Contains(ignored, `$1,$2`) {
		t.Fatalf("unexpected PostgreSQL insert-ignore query: %s", ignored)
	}
}

func TestMySQLBootstrapUsesIndexablePublicUUID(t *testing.T) {
	statement := mysqlBootstrapStatement(`CREATE TABLE dashboards (
  id TEXT PRIMARY KEY,
  public_uuid TEXT,
  body TEXT
)`)
	if !strings.Contains(statement, "public_uuid VARCHAR(255)") || !strings.Contains(statement, "body LONGTEXT") {
		t.Fatalf("unexpected MySQL bootstrap statement: %s", statement)
	}
}

func TestConfigFromEnvBuildsPostgresRDSDSN(t *testing.T) {
	t.Setenv("TOPBASE_APP_DB_ENGINE", "postgresql")
	t.Setenv("TOPBASE_APP_DB_HOST", "topbase.pg.rds.example")
	t.Setenv("TOPBASE_APP_DB_NAME", "topbase")
	t.Setenv("TOPBASE_APP_DB_USER", "topbase_app")
	t.Setenv("TOPBASE_APP_DB_PASSWORD", "secret value")
	t.Setenv("TOPBASE_APP_DB_SCHEMA", "topbase")
	t.Setenv("TOPBASE_APP_DB_TLS_MODE", "verify-full")
	cfg, err := ConfigFromEnv(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	driver, dsn, err := cfg.driverAndDSN()
	if err != nil {
		t.Fatal(err)
	}
	if driver != "pgx" || !strings.Contains(dsn, "sslmode=verify-full") || !strings.Contains(dsn, "search_path=topbase") {
		t.Fatalf("unexpected PostgreSQL config: %s / %s", driver, dsn)
	}
}

func TestEncryptedConnectionSecretsRoundTrip(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index + 1)
	}
	secrets, err := store.ConnectionSecrets(key)
	if err != nil {
		t.Fatal(err)
	}
	want := core.ConnectionRequest{ID: "pg_orders", Name: "orders", Engine: "postgres", Host: "db.internal", Password: "never-plaintext"}
	if err := secrets.SaveConnectionSecret(want.ID, want); err != nil {
		t.Fatal(err)
	}
	var ciphertext string
	if err := store.db.QueryRow(`SELECT ciphertext FROM connection_secrets WHERE database_id=?`, want.ID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil || strings.Contains(string(decoded), want.Password) {
		t.Fatalf("connection password was not encrypted")
	}
	got, err := secrets.GetConnectionSecret(want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != want.Password || got.Host != want.Host {
		t.Fatalf("connection secret mismatch: %+v", got)
	}
}

func TestDistributedLeaseExcludesOtherInstances(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if acquired, err := store.AcquireLease(ctx, "schedule:test", "node-a", time.Minute); err != nil || !acquired {
		t.Fatalf("first lease = %v, %v", acquired, err)
	}
	if acquired, err := store.AcquireLease(ctx, "schedule:test", "node-b", time.Minute); err != nil || acquired {
		t.Fatalf("competing lease = %v, %v", acquired, err)
	}
	if err := store.ReleaseLease(ctx, "schedule:test", "node-a"); err != nil {
		t.Fatal(err)
	}
	if acquired, err := store.AcquireLease(ctx, "schedule:test", "node-b", time.Minute); err != nil || !acquired {
		t.Fatalf("lease after release = %v, %v", acquired, err)
	}
}

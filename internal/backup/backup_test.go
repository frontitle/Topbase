package backup

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCreateProducesReadableConsistentBackup(t *testing.T) {
	dataDir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE marker(value TEXT); INSERT INTO marker VALUES ('kept')`); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := os.WriteFile(filepath.Join(dataDir, "connection-secrets.json"), []byte(`{"secret":true}`), 0600); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(t.TempDir(), "backup-1")
	manifest, err := Create(dataDir, destination, "v0.1.0-test", time.Date(2026, 8, 27, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "v0.1.0-test" || len(manifest.Files) != 3 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	backupDB, err := sql.Open("sqlite", filepath.Join(destination, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer backupDB.Close()
	var value string
	if err := backupDB.QueryRow(`SELECT value FROM marker`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "kept" {
		t.Fatalf("value = %q", value)
	}
	if _, err := os.Stat(filepath.Join(destination, "connection-secrets.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(dataDir, destination, "v0.1.0-test", time.Now()); err == nil {
		t.Fatal("expected existing destination to be rejected")
	}
}

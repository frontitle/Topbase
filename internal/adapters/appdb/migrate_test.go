package appdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/topbase/topbase/internal/core"

	_ "modernc.org/sqlite"
)

func TestOpenAppliesAndRecordsMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	store, err := OpenWithVersion(path, "v0.1.0-test")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.SchemaVersion() != 9 {
		t.Fatalf("schema version = %d, want 9", store.SchemaVersion())
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 9 {
		t.Fatalf("migration count = %d, want 9", count)
	}
	var appVersion string
	if err := store.db.QueryRow(`SELECT app_version FROM schema_migrations WHERE version=9`).Scan(&appVersion); err != nil {
		t.Fatal(err)
	}
	if appVersion != "v0.1.0-test" {
		t.Fatalf("app version = %q", appVersion)
	}
}

func TestOpenAdoptsLegacyDatabaseWithoutLosingRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE collections (
			id TEXT PRIMARY KEY, parent_id TEXT, name TEXT NOT NULL,
			personal_owner_user_id TEXT, created_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatal(err)
	}
	legacyName := "\u6211\u7684\u95ee\u6570"
	if _, err := db.Exec(`INSERT INTO collections(id,name,personal_owner_user_id,created_at) VALUES(?,?,?,?)`,
		"personal", legacyName, "user_1", "2026-08-27T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	store, err := OpenWithVersion(path, "v0.1.0-test")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var name, kind string
	if err := store.db.QueryRow(`SELECT name, kind FROM collections WHERE id='personal'`).Scan(&name, &kind); err != nil {
		t.Fatal(err)
	}
	if name != "我的分析" || kind != "personal_project" {
		t.Fatalf("legacy collection = %q/%q", name, kind)
	}
}

func TestOpenRejectsChangedAppliedMigrationChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE schema_migrations SET checksum='changed' WHERE version=1`); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	_, err = Open(path)
	if err == nil || !strings.Contains(err.Error(), "checksum changed") {
		t.Fatalf("expected checksum protection, got %v", err)
	}
}

func TestStorePing(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCollectionOwnerGroupPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	item := core.Collection{
		ID: "col_team", Name: "经营团队", Kind: "team_project",
		OwnerGroupID: "grp_ops", CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateCollection(item); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	saved, err := store.CollectionByID(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.OwnerGroupID != item.OwnerGroupID || saved.Kind != item.Kind {
		t.Fatalf("saved collection ownership = %+v", saved)
	}
}

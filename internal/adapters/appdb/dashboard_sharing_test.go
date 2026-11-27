package appdb

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/frontitle/Topbase/internal/app/content"
	"github.com/frontitle/Topbase/internal/core"
)

func TestDashboardPublicPolicyUpgradeKeepsLegacyLink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateDashboard(core.Dashboard{ID: "old", Name: "Old", PublicUUID: "legacy-link", PublicEmbedEnabled: true, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		"ALTER TABLE dashboards DROP COLUMN public_expires_at",
		"ALTER TABLE dashboards DROP COLUMN public_password_hash",
		"DELETE FROM schema_migrations WHERE version=19",
	} {
		if _, err := store.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	access, err := store.PublicDashboardAccess(context.Background(), "legacy-link")
	if err != nil || access.ExpiresAt != nil || access.PasswordHash != "" || !access.EmbedEnabled {
		t.Fatalf("legacy link no longer permanently available: %v", err)
	}
}

func TestDashboardSharingAndCanvasSavesAreIndependent(t *testing.T) {
	store := dashboardReadFixture(t, 1)
	service := content.Service{Dashboards: store.Dashboards(), Revisions: store.Revisions()}
	stale, err := service.GetDashboard("board-0000")
	if err != nil {
		t.Fatal(err)
	}
	originalUUID := stale.PublicUUID
	expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)
	password := "一二三四五六七八"
	protected, err := service.PublishDashboard(stale.ID, "actor", core.DashboardPublicLinkUpdate{SetExpiry: true, ExpiresAt: &expiry, Password: &password})
	if err != nil || protected.PublicUUID != originalUUID || !core.VerifyPassword(protected.PublicPasswordHash, password) {
		t.Fatalf("adding a password must preserve the existing link: %v", err)
	}
	stale.Name = "New title"
	stale.PublicUUID = "forged"
	stale.PublicEmbedEnabled = true
	updated, err := service.UpdateDashboard(stale, "actor")
	if err != nil || updated.PublicUUID != originalUUID || updated.PublicPasswordHash != protected.PublicPasswordHash ||
		updated.PublicExpiresAt == nil || !updated.PublicExpiresAt.Equal(expiry) || updated.PublicEmbedEnabled {
		t.Fatalf("stale canvas save overwrote sharing configuration: %v", err)
	}
	forever, err := service.UpdateDashboardPublicLink(stale.ID, "actor", core.DashboardPublicLinkUpdate{SetExpiry: true})
	if err != nil || forever.PublicExpiresAt != nil || forever.PublicPasswordHash != protected.PublicPasswordHash || forever.Name != "New title" {
		t.Fatalf("returning to permanent changed the password or canvas: %v", err)
	}
	copy, err := service.DuplicateDashboard(stale.ID, "actor")
	if err != nil || copy.PublicPasswordHash != "" || copy.PublicUUID != "" || copy.PublicExpiresAt != nil || copy.PublicEmbedEnabled {
		t.Fatalf("duplicate should be private: %v", err)
	}
	revisions, err := service.ListRevisions("dashboard", stale.ID)
	if err != nil || len(revisions) == 0 {
		t.Fatalf("failed to read revision history: %v", err)
	}
	for _, r := range revisions {
		if strings.Contains(r.Snapshot, protected.PublicPasswordHash) || strings.Contains(r.Snapshot, password) {
			t.Fatal("password or verifier persisted in revision history")
		}
	}
	if _, err := service.DisableDashboardPublicLink(stale.ID, "actor"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateDashboardPublicLink(stale.ID, "actor", core.DashboardPublicLinkUpdate{Password: &password}); err == nil {
		t.Fatal("a settings edit resurrected a revoked link")
	}
	fresh, err := service.EnableDashboardPublicLink(stale.ID, "actor")
	if err != nil || fresh.PublicUUID == originalUUID || fresh.PublicUUID == "" || fresh.PublicExpiresAt != nil || fresh.PublicPasswordHash != "" {
		t.Fatalf("re-published link is not fresh and permanent: %v", err)
	}
}

func TestExpiredDashboardPolicyBlocksPublicReadsButCanBeRenewed(t *testing.T) {
	store := dashboardReadFixture(t, 1)
	service := content.Service{Dashboards: store.Dashboards()}
	past := time.Now().Add(-time.Second)
	if err := service.Dashboards.SetPublicLink("board-0000", "", true, &past, nil); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	uuid := "public-board-0000"
	if _, err := service.GetDashboardByPublicUUID(uuid); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expired public data: %v", err)
	}
	if _, err := store.PublicDashboardAccess(ctx, uuid); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expired password gate: %v", err)
	}
	if available, err := store.PublicDashboardPageAvailable(ctx, uuid, false); err != nil || available {
		t.Fatalf("expired public page: %t %v", available, err)
	}
	short := "仅三字"
	if _, err := service.UpdateDashboardPublicLink("board-0000", "", core.DashboardPublicLinkUpdate{Password: &short, SetExpiry: true}); err == nil {
		t.Fatal("short multibyte password accepted")
	}
	if _, err := service.GetDashboardByPublicUUID(uuid); !errors.Is(err, core.ErrNotFound) {
		t.Fatal("invalid password update partially applied the new expiry")
	}
	if _, err := service.UpdateDashboardPublicLink("board-0000", "", core.DashboardPublicLinkUpdate{SetExpiry: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetDashboardByPublicUUID(uuid); err != nil {
		t.Fatalf("owner could not renew the same link: %v", err)
	}
}

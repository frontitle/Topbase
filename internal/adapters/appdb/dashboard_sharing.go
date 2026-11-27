package appdb

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/frontitle/Topbase/internal/core"
)

const activeDashboardLink = ` WHERE public_uuid=? AND public_uuid<>'' AND archived_at IS NULL
	AND (public_expires_at IS NULL OR public_expires_at>?)`

func shareExpiryValue(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UnixMilli()
}

func shareExpiryTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := time.UnixMilli(value.Int64).UTC()
	return &t
}

func (s *Store) DashboardByPublicUUID(uuid string) (core.Dashboard, error) {
	return s.readOneDashboard(activeDashboardLink, uuid, time.Now().UnixMilli())
}

// PublicDashboardAccess checks access without reading the dashboard's
// contents. The card and metadata handlers check the same condition when
// loading the dashboard snapshot used for the request.
func (s *Store) PublicDashboardAccess(ctx context.Context, uuid string) (core.DashboardPublicAccess, error) {
	var access core.DashboardPublicAccess
	var hash sql.NullString
	var expires sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT public_embed_enabled, public_expires_at, public_password_hash FROM dashboards`+activeDashboardLink,
		uuid, time.Now().UnixMilli()).Scan(&access.EmbedEnabled, &expires, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return access, core.ErrNotFound
	}
	access.ExpiresAt = shareExpiryTime(expires)
	access.PasswordHash = hash.String
	return access, err
}

func (s *Store) PublicDashboardPageAvailable(ctx context.Context, uuid string, embed bool) (bool, error) {
	access, err := s.PublicDashboardAccess(ctx, uuid)
	if errors.Is(err, core.ErrNotFound) {
		return false, nil
	}
	return err == nil && (!embed || access.EmbedEnabled), err
}

func (a dashboardAdapter) SetPublicLink(id, publishUUID string, setExpiry bool, expiresAt *time.Time, passwordHash *string) error {
	var changes []string
	var args []any
	if publishUUID != "" {
		changes = append(changes, "public_uuid=COALESCE(NULLIF(public_uuid,''),?)")
		args = append(args, publishUUID)
	}
	if setExpiry {
		changes = append(changes, "public_expires_at=?")
		args = append(args, shareExpiryValue(expiresAt))
	}
	if passwordHash != nil {
		changes = append(changes, "public_password_hash=?")
		args = append(args, nullString(*passwordHash))
	}
	return a.updateSharing(id, publishUUID == "", strings.Join(changes, ", "), args...)
}

func (a dashboardAdapter) ClearPublicLink(id string) error {
	return a.updateSharing(id, false, "public_uuid=NULL, public_embed_enabled=0, public_expires_at=NULL, public_password_hash=NULL")
}

func (a dashboardAdapter) SetEmbedding(id string, enabled bool) error {
	return a.updateSharing(id, enabled, "public_embed_enabled=?", enabled)
}

// A single conditional UPDATE changes only the requested policy fields.
// A simultaneous canvas save or change to another setting cannot undo it.
func (a dashboardAdapter) updateSharing(id string, publishedOnly bool, assignments string, args ...any) error {
	where := " WHERE id=? AND archived_at IS NULL"
	if publishedOnly {
		where += " AND public_uuid IS NOT NULL AND public_uuid<>''"
	}
	if assignments != "" {
		result, err := a.db.Exec("UPDATE dashboards SET "+assignments+where, append(args, id)...)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows > 0 {
			return err
		}
	}
	// MySQL can report zero changed rows for an idempotent policy update.
	var exists int
	err := a.db.QueryRow("SELECT 1 FROM dashboards"+where, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return core.ErrNotFound
	}
	return err
}

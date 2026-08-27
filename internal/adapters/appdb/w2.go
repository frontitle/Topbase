package appdb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func (s *Store) CreateDashboard(d core.Dashboard) error {
	return s.saveDashboard(d, true)
}

func (s *Store) UpdateDashboard(d core.Dashboard) error {
	return s.saveDashboard(d, false)
}

func (s *Store) saveDashboard(d core.Dashboard, insert bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	appearance, _ := json.Marshal(d.Appearance)
	if insert {
		if _, err := tx.Exec(`INSERT INTO dashboards(id, collection_id, name, description, auto_refresh_seconds, appearance, public_uuid, archived_at, created_by, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			d.ID, nullString(d.CollectionID), d.Name, d.Description, d.AutoRefreshSeconds, string(appearance), nullString(d.PublicUUID), nil, nullString(d.CreatedBy), d.CreatedAt.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(`UPDATE dashboards SET collection_id=?, name=?, description=?, auto_refresh_seconds=?, appearance=?, public_uuid=? WHERE id=?`,
			nullString(d.CollectionID), d.Name, d.Description, d.AutoRefreshSeconds, string(appearance), nullString(d.PublicUUID), d.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM dashboard_tabs WHERE dashboard_id=?`, d.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM dashboard_cards WHERE dashboard_id=?`, d.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM dashboard_filters WHERE dashboard_id=?`, d.ID); err != nil {
			return err
		}
	}
	for _, tab := range d.Tabs {
		if _, err := tx.Exec(`INSERT INTO dashboard_tabs(id, dashboard_id, name, position) VALUES(?,?,?,?)`, tab.ID, d.ID, tab.Name, tab.Position); err != nil {
			return err
		}
	}
	for _, card := range d.Cards {
		config, err := encodeCard(card)
		if err != nil {
			return err
		}
		layout, _ := json.Marshal(card.Layout)
		if _, err := tx.Exec(`INSERT INTO dashboard_cards(id, dashboard_id, tab_id, type, question_id, title, body, config, layout, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			card.ID, d.ID, nullString(card.TabID), card.Type, nullString(card.QuestionID), card.Title, card.Body, config, string(layout), time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	for _, filter := range d.Filters {
		config, _ := json.Marshal(filter.Config)
		mappings, _ := json.Marshal(filter.Mappings)
		if _, err := tx.Exec(`INSERT INTO dashboard_filters(id, dashboard_id, name, type, config, mappings) VALUES(?,?,?,?,?,?)`,
			filter.ID, d.ID, filter.Name, filter.Type, string(config), string(mappings)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DashboardByPublicUUID(uuid string) (core.Dashboard, error) {
	if strings.TrimSpace(uuid) == "" {
		return core.Dashboard{}, core.ErrNotFound
	}
	var id string
	err := s.db.QueryRow(`SELECT id FROM dashboards WHERE public_uuid=? AND archived_at IS NULL`, uuid).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Dashboard{}, core.ErrNotFound
		}
		return core.Dashboard{}, err
	}
	return s.DashboardByID(id)
}

func (s *Store) DashboardByID(id string) (core.Dashboard, error) {
	items, err := s.ListDashboards(true)
	if err != nil {
		return core.Dashboard{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return core.Dashboard{}, core.ErrNotFound
}

func (s *Store) ListDashboards(includeArchived bool) ([]core.Dashboard, error) {
	sqlText := `SELECT id, collection_id, name, description, auto_refresh_seconds, appearance, public_uuid, archived_at, created_by, created_at FROM dashboards`
	if !includeArchived {
		sqlText += ` WHERE archived_at IS NULL`
	}
	sqlText += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Dashboard{}
	for rows.Next() {
		var item core.Dashboard
		var collection, desc, appearance, createdBy, archived, created, publicUUID sql.NullString
		var refresh sql.NullInt64
		if err := rows.Scan(&item.ID, &collection, &item.Name, &desc, &refresh, &appearance, &publicUUID, &archived, &createdBy, &created); err != nil {
			return nil, err
		}
		item.CollectionID, item.Description, item.CreatedBy, item.PublicUUID = collection.String, desc.String, createdBy.String, publicUUID.String
		item.AutoRefreshSeconds = int(refresh.Int64)
		if appearance.String != "" {
			_ = json.Unmarshal([]byte(appearance.String), &item.Appearance)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		if archived.Valid {
			t, _ := time.Parse(time.RFC3339, archived.String)
			item.ArchivedAt = &t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		if err := s.loadDashboardChildren(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Store) loadDashboardChildren(d *core.Dashboard) error {
	tabRows, err := s.db.Query(`SELECT id, name, position FROM dashboard_tabs WHERE dashboard_id=? ORDER BY position`, d.ID)
	if err != nil {
		return err
	}
	d.Tabs = []core.DashboardTab{}
	for tabRows.Next() {
		var tab core.DashboardTab
		if err := tabRows.Scan(&tab.ID, &tab.Name, &tab.Position); err != nil {
			tabRows.Close()
			return err
		}
		tab.DashboardID = d.ID
		d.Tabs = append(d.Tabs, tab)
	}
	tabRows.Close()
	cardRows, err := s.db.Query(`SELECT id, tab_id, type, question_id, title, body, config, layout FROM dashboard_cards WHERE dashboard_id=?`, d.ID)
	if err != nil {
		return err
	}
	d.Cards = []core.DashboardCard{}
	for cardRows.Next() {
		var card core.DashboardCard
		var tabID, questionID, title, body, config, layout sql.NullString
		if err := cardRows.Scan(&card.ID, &tabID, &card.Type, &questionID, &title, &body, &config, &layout); err != nil {
			cardRows.Close()
			return err
		}
		card.DashboardID, card.TabID, card.QuestionID, card.Title, card.Body = d.ID, tabID.String, questionID.String, title.String, body.String
		decodeCard(config.String, layout.String, &card)
		d.Cards = append(d.Cards, card)
	}
	cardRows.Close()
	filterRows, err := s.db.Query(`SELECT id, name, type, config, mappings FROM dashboard_filters WHERE dashboard_id=?`, d.ID)
	if err != nil {
		return err
	}
	defer filterRows.Close()
	d.Filters = []core.DashboardFilter{}
	for filterRows.Next() {
		var filter core.DashboardFilter
		var config, mappings sql.NullString
		if err := filterRows.Scan(&filter.ID, &filter.Name, &filter.Type, &config, &mappings); err != nil {
			return err
		}
		filter.DashboardID = d.ID
		if config.String != "" {
			_ = json.Unmarshal([]byte(config.String), &filter.Config)
		}
		if mappings.String != "" {
			_ = json.Unmarshal([]byte(mappings.String), &filter.Mappings)
		}
		d.Filters = append(d.Filters, filter)
	}
	return filterRows.Err()
}

func (s *Store) SetDashboardArchived(id string, archivedAt *time.Time) error {
	var value any
	if archivedAt != nil {
		value = archivedAt.UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`UPDATE dashboards SET archived_at=? WHERE id=?`, value, id)
	return err
}

func encodeCard(card core.DashboardCard) (string, error) {
	payload := map[string]any{"config": card.Config, "click": card.Click}
	raw, err := json.Marshal(payload)
	return string(raw), err
}

func decodeCard(configJSON, layoutJSON string, card *core.DashboardCard) {
	if layoutJSON != "" {
		_ = json.Unmarshal([]byte(layoutJSON), &card.Layout)
	}
	if configJSON == "" {
		return
	}
	var payload struct {
		Config map[string]any      `json:"config"`
		Click  *core.ClickBehavior `json:"click"`
	}
	if err := json.Unmarshal([]byte(configJSON), &payload); err == nil {
		card.Config = payload.Config
		card.Click = payload.Click
		return
	}
	_ = json.Unmarshal([]byte(configJSON), &card.Config)
}

func (s *Store) CreateBookmark(b core.Bookmark) error {
	_, err := s.db.Exec(`INSERT INTO bookmarks(id, user_id, target_type, target_id, created_at) VALUES(?,?,?,?,?)`,
		b.ID, b.UserID, b.TargetType, b.TargetID, b.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListBookmarks(userID string) ([]core.Bookmark, error) {
	rows, err := s.db.Query(`SELECT id, user_id, target_type, target_id, created_at FROM bookmarks WHERE user_id=? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Bookmark{}
	for rows.Next() {
		var item core.Bookmark
		var created string
		if err := rows.Scan(&item.ID, &item.UserID, &item.TargetType, &item.TargetID, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteBookmark(id, userID string) error {
	_, err := s.db.Exec(`DELETE FROM bookmarks WHERE id=? AND user_id=?`, id, userID)
	return err
}

func (s *Store) CreateRevision(r core.Revision) error {
	_, err := s.db.Exec(`INSERT INTO revisions(id, target_type, target_id, actor_id, snapshot, created_at) VALUES(?,?,?,?,?,?)`,
		r.ID, r.TargetType, r.TargetID, nullString(r.ActorID), r.Snapshot, r.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListRevisions(targetType, targetID string) ([]core.Revision, error) {
	rows, err := s.db.Query(`SELECT id, target_type, target_id, actor_id, snapshot, created_at FROM revisions WHERE target_type=? AND target_id=? ORDER BY created_at DESC`, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Revision{}
	for rows.Next() {
		var item core.Revision
		var actor, created sql.NullString
		if err := rows.Scan(&item.ID, &item.TargetType, &item.TargetID, &actor, &item.Snapshot, &created); err != nil {
			return nil, err
		}
		item.ActorID = actor.String
		item.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateAlert(a core.Alert) error {
	_, err := s.db.Exec(`INSERT INTO alerts(id, question_id, name, kind, cron, channel, goal, once, enabled, created_by, created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.QuestionID, a.Name, a.Kind, a.Cron, a.Channel, a.Goal, boolInt(a.Once), boolInt(a.Enabled), nullString(a.CreatedBy), a.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) UpdateAlert(a core.Alert) error {
	_, err := s.db.Exec(`UPDATE alerts SET name=?, kind=?, cron=?, channel=?, goal=?, once=?, enabled=? WHERE id=?`,
		a.Name, a.Kind, a.Cron, a.Channel, a.Goal, boolInt(a.Once), boolInt(a.Enabled), a.ID)
	return err
}

func (s *Store) AlertByID(id string) (core.Alert, error) {
	row := s.db.QueryRow(`SELECT id, question_id, name, kind, cron, channel, goal, once, enabled, created_by, created_at FROM alerts WHERE id=?`, id)
	return scanAlert(row)
}

func (s *Store) ListAlerts() ([]core.Alert, error) {
	rows, err := s.db.Query(`SELECT id, question_id, name, kind, cron, channel, goal, once, enabled, created_by, created_at FROM alerts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Alert{}
	for rows.Next() {
		item, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteAlert(id string) error {
	_, err := s.db.Exec(`DELETE FROM alerts WHERE id=?`, id)
	return err
}

func scanAlert(row scanner) (core.Alert, error) {
	var item core.Alert
	var cron, createdBy, created sql.NullString
	var once, enabled int
	err := row.Scan(&item.ID, &item.QuestionID, &item.Name, &item.Kind, &cron, &item.Channel, &item.Goal, &once, &enabled, &createdBy, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Alert{}, core.ErrNotFound
	}
	if err != nil {
		return core.Alert{}, err
	}
	item.Cron, item.CreatedBy = cron.String, createdBy.String
	item.Once, item.Enabled = once == 1, enabled == 1
	item.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	return item, nil
}

func (s *Store) CreateNotification(n core.Notification) error {
	_, err := s.db.Exec(`INSERT INTO notifications(id, user_id, alert_id, title, body, created_at, read_at) VALUES(?,?,?,?,?,?,?)`,
		n.ID, nullString(n.UserID), nullString(n.AlertID), n.Title, n.Body, n.CreatedAt.UTC().Format(time.RFC3339), nil)
	return err
}

func (s *Store) ListNotifications(userID string) ([]core.Notification, error) {
	rows, err := s.db.Query(`SELECT id, user_id, alert_id, title, body, created_at, read_at FROM notifications WHERE user_id=? OR user_id IS NULL ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Notification{}
	for rows.Next() {
		var item core.Notification
		var user, alertID, body, created, read sql.NullString
		if err := rows.Scan(&item.ID, &user, &alertID, &item.Title, &body, &created, &read); err != nil {
			return nil, err
		}
		item.UserID, item.AlertID, item.Body = user.String, alertID.String, body.String
		item.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		if read.Valid {
			t, _ := time.Parse(time.RFC3339, read.String)
			item.ReadAt = &t
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Search(query string) ([]core.SearchHit, error) {
	like := "%" + strings.TrimSpace(query) + "%"
	rows, err := s.db.Query(`
		SELECT 'question' AS type, id, name FROM questions WHERE archived_at IS NULL AND name LIKE ?
		UNION ALL
		SELECT 'dashboard', id, name FROM dashboards WHERE archived_at IS NULL AND name LIKE ?
		UNION ALL
		SELECT 'collection', id, name FROM collections WHERE name LIKE ?
		UNION ALL
		SELECT 'database', id, name FROM catalog_databases WHERE name LIKE ?
		UNION ALL
		SELECT 'model', id, name FROM models WHERE name LIKE ?
		UNION ALL
		SELECT 'metric', id, name FROM metrics WHERE name LIKE ?
		UNION ALL
		SELECT 'warehouse', id, schema_name || '.' || table_name FROM materialized_tables WHERE table_name LIKE ? OR schema_name || '.' || table_name LIKE ?
		LIMIT 50`, like, like, like, like, like, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.SearchHit{}
	for rows.Next() {
		var item core.SearchHit
		if err := rows.Scan(&item.Type, &item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateAPIKey(key core.APIKey) error {
	_, err := s.db.Exec(`INSERT INTO api_keys(id, name, prefix, hash, user_id, created_at) VALUES(?,?,?,?,?,?)`,
		key.ID, key.Name, key.Prefix, key.Hash, key.UserID, key.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListAPIKeys(userID string) ([]core.APIKey, error) {
	rows, err := s.db.Query(`SELECT id, name, prefix, hash, user_id, created_at FROM api_keys WHERE user_id=? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.APIKey{}
	for rows.Next() {
		var item core.APIKey
		var created string
		if err := rows.Scan(&item.ID, &item.Name, &item.Prefix, &item.Hash, &item.UserID, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) APIKeyByHash(hash string) (core.APIKey, error) {
	var item core.APIKey
	var created string
	err := s.db.QueryRow(`SELECT id, name, prefix, hash, user_id, created_at FROM api_keys WHERE hash=?`, hash).
		Scan(&item.ID, &item.Name, &item.Prefix, &item.Hash, &item.UserID, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return core.APIKey{}, core.ErrNotFound
	}
	if err != nil {
		return core.APIKey{}, err
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return item, nil
}

func (s *Store) DeleteAPIKey(id string) error {
	_, err := s.db.Exec(`DELETE FROM api_keys WHERE id=?`, id)
	return err
}

type dashboardAdapter struct{ *Store }

func (s *Store) Dashboards() core.DashboardStore { return dashboardAdapter{s} }

func (a dashboardAdapter) Create(d core.Dashboard) error { return a.CreateDashboard(d) }
func (a dashboardAdapter) Update(d core.Dashboard) error { return a.UpdateDashboard(d) }
func (a dashboardAdapter) ByID(id string) (core.Dashboard, error) {
	return a.DashboardByID(id)
}
func (a dashboardAdapter) ByPublicUUID(uuid string) (core.Dashboard, error) {
	return a.DashboardByPublicUUID(uuid)
}
func (a dashboardAdapter) List(includeArchived bool) ([]core.Dashboard, error) {
	return a.ListDashboards(includeArchived)
}
func (a dashboardAdapter) SetArchived(id string, archivedAt *time.Time) error {
	return a.SetDashboardArchived(id, archivedAt)
}

type bookmarkAdapter struct{ *Store }

func (s *Store) Bookmarks() core.BookmarkStore { return bookmarkAdapter{s} }

func (a bookmarkAdapter) Create(b core.Bookmark) error { return a.CreateBookmark(b) }
func (a bookmarkAdapter) ListByUser(userID string) ([]core.Bookmark, error) {
	return a.ListBookmarks(userID)
}
func (a bookmarkAdapter) Delete(id, userID string) error { return a.DeleteBookmark(id, userID) }

type revisionAdapter struct{ *Store }

func (s *Store) Revisions() core.RevisionStore { return revisionAdapter{s} }

func (a revisionAdapter) Create(r core.Revision) error { return a.CreateRevision(r) }
func (a revisionAdapter) List(targetType, targetID string) ([]core.Revision, error) {
	return a.ListRevisions(targetType, targetID)
}

type alertAdapter struct{ *Store }

func (s *Store) Alerts() core.AlertStore { return alertAdapter{s} }

func (a alertAdapter) Create(alert core.Alert) error { return a.CreateAlert(alert) }
func (a alertAdapter) Update(alert core.Alert) error { return a.UpdateAlert(alert) }
func (a alertAdapter) ByID(id string) (core.Alert, error) {
	return a.AlertByID(id)
}
func (a alertAdapter) List() ([]core.Alert, error) { return a.ListAlerts() }
func (a alertAdapter) Delete(id string) error      { return a.DeleteAlert(id) }

type notificationAdapter struct{ *Store }

func (s *Store) Notifications() core.NotificationStore { return notificationAdapter{s} }

func (a notificationAdapter) Create(n core.Notification) error { return a.CreateNotification(n) }
func (a notificationAdapter) List(userID string) ([]core.Notification, error) {
	return a.ListNotifications(userID)
}

type apiKeyAdapter struct{ *Store }

func (s *Store) APIKeys() core.APIKeyStore { return apiKeyAdapter{s} }

func (a apiKeyAdapter) Create(key core.APIKey) error { return a.CreateAPIKey(key) }
func (a apiKeyAdapter) ListByUser(userID string) ([]core.APIKey, error) {
	return a.ListAPIKeys(userID)
}
func (a apiKeyAdapter) ByHash(hash string) (core.APIKey, error) { return a.APIKeyByHash(hash) }
func (a apiKeyAdapter) Delete(id string) error                  { return a.DeleteAPIKey(id) }

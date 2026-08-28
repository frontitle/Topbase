package appdb

import (
	"database/sql"
	"errors"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func (s *Store) ListGroups() ([]core.Group, error) {
	rows, err := s.db.Query(`SELECT id, name, kind FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Group{}
	for rows.Next() {
		var item core.Group
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertGroup(group core.Group) error {
	_, err := s.db.Exec(`INSERT INTO groups(id, name, kind) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, kind=excluded.kind`, group.ID, group.Name, group.Kind)
	return err
}

func (s *Store) ReplaceMembers(groupID string, userIDs []string) error {
	if _, err := s.db.Exec(`DELETE FROM group_members WHERE group_id=?`, groupID); err != nil {
		return err
	}
	for _, userID := range userIDs {
		if err := s.AddMember(groupID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UserByFeishuOpenID(openID string) (core.User, error) {
	return s.scanUser(s.db.QueryRow(`SELECT id, email, name, avatar_url, password_hash, locale, theme, is_active, created_at FROM users WHERE feishu_open_id = ?`, openID))
}

func (s *Store) CreateSubscription(item core.Subscription) error {
	_, err := s.db.Exec(`INSERT INTO subscriptions(id, dashboard_id, cron, timezone, channel, enabled, last_run_at, created_by, created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		item.ID, item.DashboardID, item.Cron, item.Timezone, item.Channel, boolInt(item.Enabled), timePtr(item.LastRunAt), nullString(item.CreatedBy), item.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) UpdateSubscription(item core.Subscription) error {
	_, err := s.db.Exec(`UPDATE subscriptions SET cron=?, timezone=?, channel=?, enabled=?, last_run_at=? WHERE id=?`,
		item.Cron, item.Timezone, item.Channel, boolInt(item.Enabled), timePtr(item.LastRunAt), item.ID)
	return err
}

func (s *Store) DeleteSubscription(id string) error {
	_, err := s.db.Exec(`DELETE FROM subscriptions WHERE id=?`, id)
	return err
}

func (s *Store) SubscriptionByID(id string) (core.Subscription, error) {
	row := s.db.QueryRow(`SELECT id, dashboard_id, cron, timezone, channel, enabled, last_run_at, created_by, created_at FROM subscriptions WHERE id=?`, id)
	item, err := scanSubscription(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Subscription{}, core.ErrNotFound
	}
	return item, err
}

func (s *Store) ListSubscriptions() ([]core.Subscription, error) {
	return s.querySubscriptions(`SELECT id, dashboard_id, cron, timezone, channel, enabled, last_run_at, created_by, created_at FROM subscriptions ORDER BY created_at DESC`)
}

func (s *Store) ListSubscriptionsByDashboard(dashboardID string) ([]core.Subscription, error) {
	return s.querySubscriptions(`SELECT id, dashboard_id, cron, timezone, channel, enabled, last_run_at, created_by, created_at FROM subscriptions WHERE dashboard_id=? ORDER BY created_at DESC`, dashboardID)
}

func (s *Store) querySubscriptions(sqlText string, args ...any) ([]core.Subscription, error) {
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Subscription{}
	for rows.Next() {
		item, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanSubscription(row scanner) (core.Subscription, error) {
	var item core.Subscription
	var last, createdBy, created sql.NullString
	var enabled int
	if err := row.Scan(&item.ID, &item.DashboardID, &item.Cron, &item.Timezone, &item.Channel, &enabled, &last, &createdBy, &created); err != nil {
		return core.Subscription{}, err
	}
	item.Enabled, item.CreatedBy = enabled == 1, createdBy.String
	item.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	if last.String != "" {
		t, _ := time.Parse(time.RFC3339, last.String)
		item.LastRunAt = &t
	}
	return item, nil
}

func (g groupAdapter) List() ([]core.Group, error) { return g.ListGroups() }
func (g groupAdapter) Upsert(group core.Group) error {
	return g.UpsertGroup(group)
}
func (g groupAdapter) ReplaceMembers(groupID string, userIDs []string) error {
	return g.Store.ReplaceMembers(groupID, userIDs)
}
func (g groupAdapter) HasMember(groupID, userID string) (bool, error) {
	return g.Store.HasMember(groupID, userID)
}
func (g groupAdapter) GroupsForUser(userID string) ([]core.Group, error) {
	return g.Store.GroupsForUser(userID)
}

func (s *Store) HasMember(groupID, userID string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM group_members WHERE group_id=? AND user_id=?`, groupID, userID).Scan(&n)
	return n > 0, err
}

func (s *Store) GroupsForUser(userID string) ([]core.Group, error) {
	rows, err := s.db.Query(`SELECT g.id, g.name, g.kind FROM groups g JOIN group_members m ON m.group_id=g.id WHERE m.user_id=? ORDER BY g.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Group{}
	for rows.Next() {
		var item core.Group
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a userAdapter) ByFeishuOpenID(openID string) (core.User, error) {
	return a.UserByFeishuOpenID(openID)
}

type subscriptionAdapter struct{ *Store }

func (s *Store) Subscriptions() core.SubscriptionStore { return subscriptionAdapter{s} }

func (a subscriptionAdapter) Create(item core.Subscription) error { return a.CreateSubscription(item) }
func (a subscriptionAdapter) Update(item core.Subscription) error { return a.UpdateSubscription(item) }
func (a subscriptionAdapter) Delete(id string) error              { return a.DeleteSubscription(id) }
func (a subscriptionAdapter) ByID(id string) (core.Subscription, error) {
	return a.SubscriptionByID(id)
}
func (a subscriptionAdapter) ListByDashboard(dashboardID string) ([]core.Subscription, error) {
	return a.ListSubscriptionsByDashboard(dashboardID)
}
func (a subscriptionAdapter) List() ([]core.Subscription, error) { return a.ListSubscriptions() }

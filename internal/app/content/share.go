package content

import (
	"fmt"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func (s Service) Search(query string) ([]core.SearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return []core.SearchHit{}, nil
	}
	if s.SearchStore == nil {
		return []core.SearchHit{}, nil
	}
	return s.SearchStore.Search(query)
}

func (s Service) AddBookmark(userID, targetType, targetID string) (core.Bookmark, error) {
	if userID == "" {
		return core.Bookmark{}, fmt.Errorf("sign in to bookmark")
	}
	if targetType == "" || targetID == "" {
		return core.Bookmark{}, fmt.Errorf("target_type and target_id are required")
	}
	item := core.Bookmark{ID: core.NewID("bmk"), UserID: userID, TargetType: targetType, TargetID: targetID, CreatedAt: time.Now().UTC()}
	if err := s.Bookmarks.Create(item); err != nil {
		return core.Bookmark{}, err
	}
	return item, nil
}

func (s Service) ListBookmarks(userID string) ([]core.Bookmark, error) {
	if s.Bookmarks == nil || userID == "" {
		return []core.Bookmark{}, nil
	}
	items, err := s.Bookmarks.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	for i, item := range items {
		items[i].Name = s.bookmarkName(item)
	}
	return items, nil
}

func (s Service) DeleteBookmark(id, userID string) error {
	return s.Bookmarks.Delete(id, userID)
}

func (s Service) bookmarkName(item core.Bookmark) string {
	switch item.TargetType {
	case "question":
		if q, err := s.Questions.ByID(item.TargetID); err == nil {
			return q.Name
		}
	case "dashboard":
		if d, err := s.Dashboards.ByID(item.TargetID); err == nil {
			return d.Name
		}
	}
	return item.TargetID
}

func (s Service) Trash() ([]core.TrashItem, error) {
	items := []core.TrashItem{}
	questions, err := s.Questions.List(true)
	if err != nil {
		return nil, err
	}
	for _, q := range questions {
		if q.ArchivedAt != nil {
			items = append(items, core.TrashItem{Type: "question", ID: q.ID, Name: q.Name, ArchivedAt: *q.ArchivedAt})
		}
	}
	if s.Dashboards != nil {
		dashboards, err := s.Dashboards.List(true)
		if err != nil {
			return nil, err
		}
		for _, d := range dashboards {
			if d.ArchivedAt != nil {
				items = append(items, core.TrashItem{Type: "dashboard", ID: d.ID, Name: d.Name, ArchivedAt: *d.ArchivedAt})
			}
		}
	}
	return items, nil
}

func (s Service) Restore(targetType, id string) error {
	switch targetType {
	case "question":
		return s.RestoreQuestion(id)
	case "dashboard":
		return s.RestoreDashboard(id)
	default:
		return fmt.Errorf("unsupported trash type %q", targetType)
	}
}

func (s Service) CreateAlert(a core.Alert, userID string) (core.Alert, error) {
	if strings.TrimSpace(a.Name) == "" || strings.TrimSpace(a.QuestionID) == "" {
		return core.Alert{}, fmt.Errorf("name and question_id are required")
	}
	if a.Kind == "" {
		a.Kind = "results"
	}
	if a.Channel == "" {
		a.Channel = "inbox"
	}
	a.ID = core.NewID("alt")
	a.Enabled = true
	a.CreatedBy = userID
	a.CreatedAt = time.Now().UTC()
	if err := s.Alerts.Create(a); err != nil {
		return core.Alert{}, err
	}
	return a, nil
}

func (s Service) ListAlerts() ([]core.Alert, error) {
	if s.Alerts == nil {
		return []core.Alert{}, nil
	}
	return s.Alerts.List()
}

func (s Service) GetAlert(id string) (core.Alert, error) {
	return s.Alerts.ByID(id)
}

func (s Service) DeleteAlert(id string) error {
	return s.Alerts.Delete(id)
}

func (s Service) ListNotifications(userID string) ([]core.Notification, error) {
	if s.Notifications == nil {
		return []core.Notification{}, nil
	}
	return s.Notifications.List(userID)
}

func (s Service) RecordNotification(n core.Notification) error {
	if s.Notifications == nil {
		return nil
	}
	if n.ID == "" {
		n.ID = core.NewID("ntf")
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	return s.Notifications.Create(n)
}

package core

import (
	"context"
	"time"
)

type SettingStore interface {
	Get(key string) (string, bool, error)
	Set(key, value string) error
}

type UserStore interface {
	Create(User) error
	ByEmail(email string) (User, error)
	ByID(id string) (User, error)
	ByFeishuOpenID(openID string) (User, error)
	List() ([]User, error)
	SetActive(id string, active bool) error
	SetPassword(id, passwordHash string) error
	UpdateProfile(id, name, email, locale, theme, avatarURL string) error
}

type GroupStore interface {
	Create(Group) error
	AddMember(groupID, userID string) error
	List() ([]Group, error)
	Upsert(Group) error
	ReplaceMembers(groupID string, userIDs []string) error
	HasMember(groupID, userID string) (bool, error)
	GroupsForUser(userID string) ([]Group, error)
}

type OrgUnit struct {
	ExternalID string   `json:"external_id"`
	Name       string   `json:"name"`
	MemberIDs  []string `json:"member_ids,omitempty"`
}

type OrgDirectory interface {
	ListUnits(context.Context) ([]OrgUnit, error)
}

type SessionStore interface {
	Create(Session) error
	ByID(id string) (Session, error)
	Delete(id string) error
	DeleteExpired(now time.Time) error
}

type APIKeyStore interface {
	Create(APIKey) error
	ListByUser(userID string) ([]APIKey, error)
	List() ([]APIKey, error)
	ByHash(hash string) (APIKey, error)
	Delete(id string) error
}

type CollectionStore interface {
	Create(Collection) error
	Update(Collection) error
	List() ([]Collection, error)
	ByID(id string) (Collection, error)
	Delete(id string) error
}

type QuestionStore interface {
	Create(Question) error
	Update(Question) error
	ByID(id string) (Question, error)
	List(includeArchived bool) ([]Question, error)
	SetArchived(id string, archivedAt *time.Time) error
}

type DashboardStore interface {
	Create(Dashboard) error
	Update(Dashboard) error
	ByID(id string) (Dashboard, error)
	ByPublicUUID(uuid string) (Dashboard, error)
	List(includeArchived bool) ([]Dashboard, error)
	SetArchived(id string, archivedAt *time.Time) error
}

type BookmarkStore interface {
	Create(Bookmark) error
	ListByUser(userID string) ([]Bookmark, error)
	Delete(id, userID string) error
}

type RevisionStore interface {
	Create(Revision) error
	List(targetType, targetID string) ([]Revision, error)
}

type AlertStore interface {
	Create(Alert) error
	Update(Alert) error
	ByID(id string) (Alert, error)
	List() ([]Alert, error)
	Delete(id string) error
}

type NotificationStore interface {
	Create(Notification) error
	List(userID string) ([]Notification, error)
}

type SearchStore interface {
	Search(query string) ([]SearchHit, error)
}

type QueryExecutor interface {
	Execute(ctx context.Context, databaseID, sql string, args ...any) (QueryResult, error)
}

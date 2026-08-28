package core

import "time"

type Bookmark struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id,omitempty"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Name       string    `json:"name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Revision struct {
	ID         string    `json:"id"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	ActorID    string    `json:"actor_id,omitempty"`
	Snapshot   string    `json:"snapshot"`
	CreatedAt  time.Time `json:"created_at"`
}

type Alert struct {
	ID         string    `json:"id"`
	QuestionID string    `json:"question_id"`
	Name       string    `json:"name"`
	Kind       string    `json:"kind"` // results | goal | progress
	Cron       string    `json:"cron,omitempty"`
	Channel    string    `json:"channel"`
	Goal       float64   `json:"goal,omitempty"`
	Once       bool      `json:"once"`
	Enabled    bool      `json:"enabled"`
	CreatedBy  string    `json:"created_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id,omitempty"`
	AlertID   string     `json:"alert_id,omitempty"`
	Title     string     `json:"title"`
	Body      string     `json:"body,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

type SearchHit struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIKey struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Hash      string    `json:"-"`
	UserID    string    `json:"user_id,omitempty"`
	Key       string    `json:"key,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PermissionGraph struct {
	Revision        int            `json:"revision"`
	DataGraph       map[string]any `json:"data_graph"`
	CollectionGraph map[string]any `json:"collection_graph"`
}

type TrashItem struct {
	Type       string    `json:"type"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ArchivedAt time.Time `json:"archived_at"`
}

type Subscription struct {
	ID          string     `json:"id"`
	DashboardID string     `json:"dashboard_id"`
	Cron        string     `json:"cron"`
	Timezone    string     `json:"timezone,omitempty"`
	Channel     string     `json:"channel"`
	Enabled     bool       `json:"enabled"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	CreatedBy   string     `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SubscriptionStore interface {
	Create(Subscription) error
	Update(Subscription) error
	Delete(id string) error
	ByID(id string) (Subscription, error)
	ListByDashboard(dashboardID string) ([]Subscription, error)
	List() ([]Subscription, error)
}

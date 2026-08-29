package core

import "time"

type Layout struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type ClickBehavior struct {
	Type        string `json:"type,omitempty"` // update_filter | link
	FilterID    string `json:"filter_id,omitempty"`
	URL         string `json:"url,omitempty"`
	QuestionID  string `json:"question_id,omitempty"`
	DashboardID string `json:"dashboard_id,omitempty"`
}

type DashboardCard struct {
	ID          string         `json:"id"`
	DashboardID string         `json:"dashboard_id,omitempty"`
	TabID       string         `json:"tab_id,omitempty"`
	Type        string         `json:"type"` // question | heading | text | link | iframe
	QuestionID  string         `json:"question_id,omitempty"`
	Title       string         `json:"title,omitempty"`
	Body        string         `json:"body,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	Click       *ClickBehavior `json:"click,omitempty"`
	Layout      Layout         `json:"layout"`
}

type DashboardTab struct {
	ID          string `json:"id"`
	DashboardID string `json:"dashboard_id,omitempty"`
	Name        string `json:"name"`
	Position    int    `json:"position"`
}

type FilterMapping struct {
	CardID string `json:"card_id"`
	Field  string `json:"field"`
}

type DashboardFilter struct {
	ID          string          `json:"id"`
	DashboardID string          `json:"dashboard_id,omitempty"`
	Name        string          `json:"name"`
	Type        string          `json:"type"` // date | category | id | number | text | boolean
	Config      map[string]any  `json:"config,omitempty"`
	Mappings    []FilterMapping `json:"mappings,omitempty"`
}

type Dashboard struct {
	ID                 string            `json:"id"`
	CollectionID       string            `json:"collection_id,omitempty"`
	Name               string            `json:"name"`
	Description        string            `json:"description,omitempty"`
	AutoRefreshSeconds int               `json:"auto_refresh_seconds,omitempty"`
	Appearance         map[string]any    `json:"appearance,omitempty"`
	Tabs               []DashboardTab    `json:"tabs,omitempty"`
	Cards              []DashboardCard   `json:"cards,omitempty"`
	Filters            []DashboardFilter `json:"filters,omitempty"`
	ArchivedAt         *time.Time        `json:"archived_at,omitempty"`
	CreatedBy          string            `json:"created_by,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	PublicUUID         string            `json:"public_uuid,omitempty"`
	PublicEmbedEnabled bool              `json:"public_embed_enabled"`
}

type FilterValue map[string]any

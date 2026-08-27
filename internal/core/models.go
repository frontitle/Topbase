package core

import "time"

type Database struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Engine       string     `json:"engine"`
	Host         string     `json:"host"`
	Status       string     `json:"status"`
	Connected    bool       `json:"connected"`
	CreatedAt    time.Time  `json:"created_at"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	TableCount   int        `json:"table_count,omitempty"`
}

type QueryResult struct {
	Columns []string
	Rows    [][]any
	Meta    map[string]any
}

type ChatRequest struct {
	Message    string
	DatabaseID string
}

type ChatResponse struct {
	Answer string
	SQL    string
	Safe   bool
}

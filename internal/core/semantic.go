package core

import (
	"time"

	"github.com/topbase/topbase/internal/core/queryir"
)

var SemanticTypes = []string{
	"EntityKey", "ForeignKey",
	"Quantity", "Score", "Percentage", "Currency", "Discount", "Income", "Latitude", "Longitude",
	"CreationDate", "CreationTime", "CreationTimestamp",
	"JoinedDate", "JoinedTime", "JoinedTimestamp", "Birthday",
	"EntityName", "Email", "URL", "ImageURL", "AvatarURL", "Category", "Name", "Title",
	"Description", "Product", "Source", "City", "State", "Country", "ZipCode",
	"FieldContainingJSON",
}

func ValidSemanticType(value string) bool {
	if value == "" {
		return true
	}
	for _, item := range SemanticTypes {
		if item == value {
			return true
		}
	}
	return false
}

type FieldRef struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Name   string `json:"name"`
}

type FieldMeta struct {
	DatabaseID   string         `json:"database_id"`
	Schema       string         `json:"schema"`
	Table        string         `json:"table"`
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name,omitempty"`
	Description  string         `json:"description,omitempty"`
	SemanticType string         `json:"semantic_type,omitempty"`
	Visibility   string         `json:"visibility,omitempty"`
	Format       map[string]any `json:"format,omitempty"`
	FKTarget     *FieldRef      `json:"fk_target,omitempty"`
}

type ModelColumn struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name,omitempty"`
	SemanticType string `json:"semantic_type,omitempty"`
	Description  string `json:"description,omitempty"`
}

type Model struct {
	ID           string         `json:"id"`
	CollectionID string         `json:"collection_id,omitempty"`
	Name         string         `json:"name"`
	DatabaseID   string         `json:"database_id"`
	QueryIR      *queryir.Query `json:"queryir,omitempty"`
	NativeSQL    string         `json:"native_sql,omitempty"`
	Columns      []ModelColumn  `json:"columns,omitempty"`
	CreatedBy    string         `json:"created_by,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Metric struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	DatabaseID  string              `json:"database_id"`
	Schema      string              `json:"schema"`
	Table       string              `json:"table"`
	Aggregation queryir.Aggregation `json:"aggregation"`
	Filters     []queryir.Filter    `json:"filters,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
}

type Segment struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	DatabaseID string           `json:"database_id"`
	Schema     string           `json:"schema"`
	Table      string           `json:"table"`
	Filters    []queryir.Filter `json:"filters"`
	CreatedAt  time.Time        `json:"created_at"`
}

type GlossaryTerm struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

type FieldStore interface {
	SaveField(FieldMeta) error
	ListFields(databaseID, schema, table string) ([]FieldMeta, error)
	ListDatabaseFields(databaseID string) ([]FieldMeta, error)
}

type ModelStore interface {
	Create(Model) error
	Update(Model) error
	ByID(id string) (Model, error)
	List() ([]Model, error)
}

type MetricStore interface {
	Create(Metric) error
	ByID(id string) (Metric, error)
	List() ([]Metric, error)
}

type SegmentStore interface {
	Create(Segment) error
	ByID(id string) (Segment, error)
	List() ([]Segment, error)
}

type GlossaryStore interface {
	Create(GlossaryTerm) error
	List() ([]GlossaryTerm, error)
}

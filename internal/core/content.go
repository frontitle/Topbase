package core

import (
	"encoding/json"
	"time"

	"github.com/topbase/topbase/internal/core/queryir"
)

type Collection struct {
	ID                  string    `json:"id"`
	ParentID            string    `json:"parent_id,omitempty"`
	Name                string    `json:"name"`
	PersonalOwnerUserID string    `json:"personal_owner_user_id,omitempty"`
	OwnerGroupID        string    `json:"owner_group_id,omitempty"`
	Kind                string    `json:"kind"` // personal_project | team_project
	CreatedAt           time.Time `json:"created_at"`
	// These are presentation fields, populated only for a recipient of a
	// personal-group share. Shared groups are always view-only.
	ReadOnly     bool   `json:"read_only,omitempty"`
	SharedByName string `json:"shared_by_name,omitempty"`
}

type CollectionShare struct {
	CollectionID string    `json:"collection_id"`
	UserID       string    `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProjectAccessRule grants a team group access to a team project. Roles are
// cumulative: view < edit < manage. Personal projects do not use these rules.
type ProjectAccessRule struct {
	ProjectID string `json:"project_id"`
	GroupID   string `json:"group_id"`
	Role      string `json:"role"`
}

type ChartSeriesStyle struct {
	Color       string `json:"color,omitempty"`
	Title       string `json:"title,omitempty"`
	Display     string `json:"display,omitempty"` // line | bar | area
	Axis        string `json:"axis,omitempty"`    // left | right | auto
	LineStyle   string `json:"line_style,omitempty"`
	LineSize    string `json:"line_size,omitempty"`
	Visible     *bool  `json:"visible,omitempty"`
	Markers     string `json:"markers,omitempty"` // auto | on | off
	Missing     string `json:"missing,omitempty"`
	Interpolate string `json:"interpolate,omitempty"`
	ShowValues  *bool  `json:"show_values,omitempty"`
	ShowTrend   *bool  `json:"show_trend,omitempty"`
}

type ChartAxisSpec struct {
	Title     string   `json:"title,omitempty"`
	Scale     string   `json:"scale,omitempty"` // linear | log | pow | timeseries | ordinal | histogram
	Enabled   string   `json:"enabled,omitempty"`
	Labels    *bool    `json:"labels,omitempty"`
	AutoRange *bool    `json:"auto_range,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	UnpinZero bool     `json:"unpin_zero,omitempty"`
	Ticks     *int     `json:"ticks,omitempty"`
}

type ChartColorSegment struct {
	Min   *float64 `json:"min,omitempty"`
	Max   *float64 `json:"max,omitempty"`
	Color string   `json:"color,omitempty"`
}

type ChartColumnStyle struct {
	Title   string `json:"title,omitempty"`
	Visible *bool  `json:"visible,omitempty"`
	Filter  string `json:"filter,omitempty"`
}

type ChartSpec struct {
	Type           string                      `json:"type"`
	X              string                      `json:"x,omitempty"`
	Y              []string                    `json:"y,omitempty"`
	Breakout       string                      `json:"breakout,omitempty"`
	Stacked        bool                        `json:"stacked,omitempty"`
	Stacking       string                      `json:"stacking,omitempty"`
	ShowLegend     *bool                       `json:"show_legend,omitempty"`
	Legend         string                      `json:"legend,omitempty"`
	ShowLabels     bool                        `json:"show_labels,omitempty"`
	Smooth         *bool                       `json:"smooth,omitempty"`
	Interpolation  string                      `json:"interpolation,omitempty"`
	Missing        string                      `json:"missing,omitempty"`
	XTitle         string                      `json:"x_title,omitempty"`
	YTitle         string                      `json:"y_title,omitempty"`
	XAxis          *ChartAxisSpec              `json:"x_axis,omitempty"`
	YAxis          *ChartAxisSpec              `json:"y_axis,omitempty"`
	Series         map[string]ChartSeriesStyle `json:"series,omitempty"`
	Columns        map[string]ChartColumnStyle `json:"columns,omitempty"`
	Search         string                      `json:"search,omitempty"`
	Sort           string                      `json:"sort,omitempty"`
	SortDir        string                      `json:"sort_dir,omitempty"`
	MaxCategories  int                         `json:"max_categories,omitempty"`
	OtherColor     string                      `json:"other_color,omitempty"`
	SliceThreshold *float64                    `json:"slice_threshold,omitempty"`
	ShowGoal       *bool                       `json:"show_goal,omitempty"`
	Goal           *float64                    `json:"goal,omitempty"`
	GoalLabel      string                      `json:"goal_label,omitempty"`
	Trendline      bool                        `json:"trendline,omitempty"`
	AutoSplit      *bool                       `json:"auto_split,omitempty"`
	LabelFrequency string                      `json:"label_frequency,omitempty"`
	ValueFormat    string                      `json:"value_format,omitempty"`
	StackValues    string                      `json:"stack_values,omitempty"`
	Markers        string                      `json:"markers,omitempty"`
	ShowTotal      *bool                       `json:"show_total,omitempty"`
	Percent        string                      `json:"percent,omitempty"`
	Donut          *bool                       `json:"donut,omitempty"`
	Prefix         string                      `json:"prefix,omitempty"`
	Suffix         string                      `json:"suffix,omitempty"`
	Decimals       *int                        `json:"decimals,omitempty"`
	NumberStyle    string                      `json:"number_style,omitempty"`
	Multiply       *float64                    `json:"multiply,omitempty"`
	Color          string                      `json:"color,omitempty"`
	Segments       []ChartColorSegment         `json:"segments,omitempty"`
}

type Question struct {
	ID           string              `json:"id"`
	CollectionID string              `json:"collection_id,omitempty"`
	DashboardID  string              `json:"dashboard_id,omitempty"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	QueryIR      *queryir.Query      `json:"queryir,omitempty"`
	NativeSQL    string              `json:"native_sql,omitempty"`
	ChartSpec    *ChartSpec          `json:"chartspec,omitempty"`
	QueryType    string              `json:"query_type"`
	DatabaseID   string              `json:"database_id,omitempty"`
	CreatedBy    string              `json:"created_by,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	ArchivedAt   *time.Time          `json:"archived_at,omitempty"`
	Parameters   []queryir.Parameter `json:"parameters,omitempty"`
	RawQueryIR   json.RawMessage     `json:"-"`
}

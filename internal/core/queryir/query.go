package queryir

type Query struct {
	Version      int           `json:"version"`
	Source       Source        `json:"source"`
	Joins        []Join        `json:"joins,omitempty"`
	Expressions  []Expression  `json:"expressions,omitempty"`
	Fields       []string      `json:"fields,omitempty"`
	Filters      []Filter      `json:"filters,omitempty"`
	Aggregations []Aggregation `json:"aggregations,omitempty"`
	GroupBy      []Breakout    `json:"group_by,omitempty"`
	Having       []Filter      `json:"having,omitempty"`
	OrderBy      []Order       `json:"order_by,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	Parameters   []Parameter   `json:"parameters,omitempty"`
}

type Source struct {
	DatabaseID string    `json:"database_id"`
	Table      *TableRef `json:"table,omitempty"`
	ModelID    string    `json:"model_id,omitempty"`
	QuestionID string    `json:"question_id,omitempty"`
	MetricID   string    `json:"metric_id,omitempty"`
	Alias      string    `json:"alias,omitempty"`
}

type TableRef struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type Join struct {
	Type       string          `json:"type"`
	Alias      string          `json:"alias,omitempty"`
	Table      *TableRef       `json:"table,omitempty"`
	Conditions []JoinCondition `json:"conditions,omitempty"`
	Implicit   bool            `json:"implicit,omitempty"`
}

type JoinCondition struct {
	Left  string `json:"left"`
	Right string `json:"right"`
	Op    string `json:"op,omitempty"`
}

type Expression struct {
	Alias string `json:"alias"`
	Op    string `json:"op"`
	Left  string `json:"left"`
	Right any    `json:"right,omitempty"`
}

type Filter struct {
	Field     string   `json:"field,omitempty"`
	Op        string   `json:"op,omitempty"`
	Value     any      `json:"value,omitempty"`
	SegmentID string   `json:"segment_id,omitempty"`
	And       []Filter `json:"and,omitempty"`
	Or        []Filter `json:"or,omitempty"`
}

type Aggregation struct {
	Fn    string `json:"fn"`
	Field string `json:"field,omitempty"`
	Alias string `json:"alias,omitempty"`
}

type Breakout struct {
	Field    string  `json:"field"`
	Temporal string  `json:"temporal,omitempty"`
	BinWidth float64 `json:"bin_width,omitempty"`
}

type Order struct {
	Field string `json:"field"`
	Dir   string `json:"dir,omitempty"`
}

type Parameter struct {
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"`
	Field string `json:"field,omitempty"`
}

type Compiled struct {
	SQL  string
	Args []any
}

package topbaseapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	appquery "github.com/topbase/topbase/internal/app/query"
	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

const defaultMaxRows = 200

// Client calls the public Topbase HTTP API. It deliberately does not have
// access to source-database credentials: every operation remains subject to
// the API key owner's Topbase permissions.
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	maxRows    int
}

type Option func(*Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func WithMaxRows(limit int) Option {
	return func(c *Client) {
		if limit > 0 {
			c.maxRows = limit
		}
	}
}

func New(rawURL, apiKey string, options ...Option) (*Client, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("Topbase URL is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("Topbase API key is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Topbase URL %q", rawURL)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	client := &Client{
		baseURL: parsed,
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		maxRows: defaultMaxRows,
	}
	for _, option := range options {
		option(client)
	}
	return client, nil
}

type TableSummary struct {
	Schema      string        `json:"schema"`
	Name        string        `json:"name"`
	DisplayName string        `json:"display_name,omitempty"`
	Description string        `json:"description,omitempty"`
	Hidden      bool          `json:"hidden,omitempty"`
	Warehouse   bool          `json:"warehouse,omitempty"`
	Columns     []core.Column `json:"columns,omitempty"`
}

type TableDescription struct {
	Table      TableSummary         `json:"table"`
	Fields     []core.FieldMeta     `json:"fields"`
	Annotation core.TableAnnotation `json:"annotation"`
}

type DeveloperStatus struct {
	Status             string `json:"status"`
	UserID             string `json:"user_id"`
	UserName           string `json:"user_name"`
	MaxQueryRows       int    `json:"max_query_rows"`
	AllowAnalysisWrite bool   `json:"allow_analysis_write"`
}

func (c *Client) Status(ctx context.Context) (DeveloperStatus, error) {
	var result DeveloperStatus
	err := c.do(ctx, http.MethodGet, "/api/developer/ping", nil, &result)
	return result, err
}

func (c *Client) ListDatabases(ctx context.Context) ([]core.Database, error) {
	var result []core.Database
	err := c.do(ctx, http.MethodGet, "/api/databases", nil, &result)
	return result, err
}

func (c *Client) ListTables(ctx context.Context, databaseID string) ([]TableSummary, error) {
	if strings.TrimSpace(databaseID) == "" {
		return nil, errors.New("database_id is required")
	}
	var result []TableSummary
	err := c.do(ctx, http.MethodGet, joinPath("/api/databases", databaseID, "tables"), nil, &result)
	return result, err
}

func (c *Client) DescribeTable(ctx context.Context, databaseID, schema, table string) (TableDescription, error) {
	if strings.TrimSpace(databaseID) == "" || strings.TrimSpace(schema) == "" || strings.TrimSpace(table) == "" {
		return TableDescription{}, errors.New("database_id, schema and table are required")
	}
	tables, err := c.ListTables(ctx, databaseID)
	if err != nil {
		return TableDescription{}, err
	}
	var selected *TableSummary
	for i := range tables {
		if tables[i].Schema == schema && tables[i].Name == table {
			selected = &tables[i]
			break
		}
	}
	if selected == nil {
		return TableDescription{}, fmt.Errorf("table %s.%s was not found", schema, table)
	}
	base := joinPath("/api/databases", databaseID, "tables", schema, table)
	var fields []core.FieldMeta
	if err := c.do(ctx, http.MethodGet, base+"/fields", nil, &fields); err != nil {
		return TableDescription{}, err
	}
	var annotation core.TableAnnotation
	if err := c.do(ctx, http.MethodGet, base+"/annotation", nil, &annotation); err != nil {
		return TableDescription{}, err
	}
	return TableDescription{Table: *selected, Fields: fields, Annotation: annotation}, nil
}

func (c *Client) ListCollections(ctx context.Context) ([]core.Collection, error) {
	var result []core.Collection
	err := c.do(ctx, http.MethodGet, "/api/collections", nil, &result)
	return result, err
}

func (c *Client) ListAnalyses(ctx context.Context) ([]core.Question, error) {
	var result []core.Question
	err := c.do(ctx, http.MethodGet, "/api/questions", nil, &result)
	return result, err
}

func (c *Client) GetAnalysis(ctx context.Context, id string) (core.Question, error) {
	if strings.TrimSpace(id) == "" {
		return core.Question{}, errors.New("analysis_id is required")
	}
	var result core.Question
	err := c.do(ctx, http.MethodGet, joinPath("/api/questions", id), nil, &result)
	return result, err
}

func (c *Client) Preview(ctx context.Context, q queryir.Query) (appquery.DatasetResult, error) {
	q = c.boundedQuery(q)
	if err := q.Validate(); err != nil {
		return appquery.DatasetResult{}, fmt.Errorf("invalid QueryIR: %w", err)
	}
	var result appquery.DatasetResult
	err := c.do(ctx, http.MethodPost, "/api/dataset", q, &result)
	return result, err
}

func (c *Client) CreateAnalysis(ctx context.Context, input core.Question) (core.Question, error) {
	if strings.TrimSpace(input.Name) == "" {
		return core.Question{}, errors.New("name is required")
	}
	if input.QueryIR == nil {
		return core.Question{}, errors.New("queryir is required; AI integrations cannot create native SQL analyses")
	}
	if strings.TrimSpace(input.NativeSQL) != "" || input.QueryType == "native" {
		return core.Question{}, errors.New("native SQL is not accepted by the AI integration")
	}
	bounded := c.boundedQuery(*input.QueryIR)
	if err := bounded.Validate(); err != nil {
		return core.Question{}, fmt.Errorf("invalid QueryIR: %w", err)
	}
	input.ID = ""
	input.CreatedBy = ""
	input.CreatedAt = time.Time{}
	input.NativeSQL = ""
	input.QueryType = "queryir"
	input.DatabaseID = bounded.Source.DatabaseID
	input.QueryIR = &bounded
	var result core.Question
	err := c.do(ctx, http.MethodPost, "/api/questions", input, &result)
	return result, err
}

func (c *Client) RunAnalysis(ctx context.Context, id string) (appquery.DatasetResult, error) {
	analysis, err := c.GetAnalysis(ctx, id)
	if err != nil {
		return appquery.DatasetResult{}, err
	}
	if analysis.QueryIR == nil {
		return appquery.DatasetResult{}, errors.New("native SQL analyses cannot be run by the AI integration")
	}
	return c.Preview(ctx, *analysis.QueryIR)
}

func (c *Client) boundedQuery(q queryir.Query) queryir.Query {
	if q.Version == 0 {
		q.Version = 1
	}
	if q.Limit <= 0 || q.Limit > c.maxRows {
		q.Limit = c.maxRows
	}
	return q
}

func (c *Client) do(ctx context.Context, method, endpoint string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	target := *c.baseURL
	target.Path = strings.TrimRight(c.baseURL.Path, "/") + endpoint
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "topbase-ai-client/1")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call Topbase: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("read Topbase response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiError struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &apiError)
		if apiError.Error == "" {
			apiError.Error = strings.TrimSpace(string(data))
		}
		if apiError.Error == "" {
			apiError.Error = response.Status
		}
		return fmt.Errorf("Topbase API %s: %s", response.Status, apiError.Error)
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode Topbase response: %w", err)
	}
	return nil
}

func joinPath(base string, values ...string) string {
	parts := []string{base}
	for _, value := range values {
		parts = append(parts, url.PathEscape(value))
	}
	return path.Join(parts...)
}

package aimcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	appquery "github.com/topbase/topbase/internal/app/query"
	"github.com/topbase/topbase/internal/buildinfo"
	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
	"github.com/topbase/topbase/internal/topbaseapi"
)

const instructions = `Topbase exposes permission-scoped analytics tools. For a natural-language data question:
1. list databases and tables;
2. describe the relevant tables so comments, field meanings and relationships are understood;
3. translate the request into QueryIR and call topbase_query_data;
4. explain the returned rows in plain language and mention assumptions;
5. only call topbase_create_analysis when the user explicitly asks to save the result.
Never invent fields. Prefer QueryIR over SQL. Results are capped to protect source systems and model context.`

type emptyInput struct{}

type databaseInput struct {
	DatabaseID string `json:"database_id" jsonschema:"Topbase database ID returned by topbase_list_databases"`
}

type tableInput struct {
	DatabaseID string `json:"database_id" jsonschema:"Topbase database ID"`
	Schema     string `json:"schema" jsonschema:"Database schema name"`
	Table      string `json:"table" jsonschema:"Table name"`
}

type analysisInput struct {
	AnalysisID string `json:"analysis_id" jsonschema:"Saved Topbase analysis ID"`
}

type queryInput struct {
	Query map[string]any `json:"queryir" jsonschema:"Topbase QueryIR object. Use fields and aliases exactly as returned by table metadata."`
}

type createAnalysisInput struct {
	Name         string          `json:"name" jsonschema:"User-facing analysis name"`
	Description  string          `json:"description,omitempty" jsonschema:"Plain-language purpose and assumptions"`
	CollectionID string          `json:"collection_id,omitempty" jsonschema:"Optional writable data group ID. Omit to save in the caller's personal analysis group."`
	Query        map[string]any  `json:"queryir" jsonschema:"Validated read-only Topbase QueryIR object"`
	ChartSpec    *core.ChartSpec `json:"chartspec,omitempty" jsonschema:"Optional visualization settings; omit to use Topbase inference"`
	Confirmed    bool            `json:"confirmed" jsonschema:"Must be true only after the user explicitly asked to save an analysis"`
}

func New(client *topbaseapi.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "topbase",
		Title:   "Topbase AI Analytics",
		Version: buildinfo.Version,
	}, &mcp.ServerOptions{
		Instructions: instructions,
		Capabilities: &mcp.ServerCapabilities{},
	})
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	additive := false
	closedWorld := false

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_status",
		Title:       "检查 Topbase 连接",
		Description: "Verify the Topbase URL, API key, developer-mode limits and whether this key may create analyses. Use this first when setup or authentication is uncertain.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, topbaseapi.DeveloperStatus, error) {
		item, err := client.Status(ctx)
		return nil, item, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_list_databases",
		Title:       "列出数据源",
		Description: "List the source databases visible to the API key owner. Start here when the target database is unknown.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, []core.Database, error) {
		items, err := client.ListDatabases(ctx)
		return nil, items, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_list_tables",
		Title:       "列出数据表",
		Description: "List live tables and database comments in one visible Topbase database. Hidden tables remain marked hidden.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input databaseInput) (*mcp.CallToolResult, []topbaseapi.TableSummary, error) {
		items, err := client.ListTables(ctx, input.DatabaseID)
		return nil, items, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_describe_table",
		Title:       "理解数据表",
		Description: "Read a table's live columns, database comments, Topbase display names, semantic types, visibility settings and relationships. Always call this before constructing a query.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input tableInput) (*mcp.CallToolResult, topbaseapi.TableDescription, error) {
		item, err := client.DescribeTable(ctx, input.DatabaseID, input.Schema, input.Table)
		return nil, item, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_list_data_groups",
		Title:       "列出分析分组",
		Description: "List analysis data groups visible to the API key owner. Use this to choose where an explicitly requested analysis should be saved.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, []core.Collection, error) {
		items, err := client.ListCollections(ctx)
		return nil, items, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_list_analyses",
		Title:       "列出分析",
		Description: "List saved analyses visible to the API key owner.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
		items, err := client.ListAnalyses(ctx)
		return nil, items, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:  "topbase_query_data",
		Title: "查询并问答数据",
		Description: "Run a read-only visual query using Topbase QueryIR and return bounded live rows plus inferred visualization settings. " +
			"Supported filters include eq, neq, gt, gte, lt, lte, contains, between, in and null checks; aggregations include count, sum, avg, min, max and distinct. " +
			"The object starts with {version:1, source:{database_id, table:{schema,name}}}; optional keys include fields, filters, joins, aggregations, group_by, order_by and limit. " +
			"Use the result to answer the user's question in plain language. This tool never accepts native SQL.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input queryInput) (*mcp.CallToolResult, appquery.DatasetResult, error) {
		query, err := decodeQuery(input.Query)
		if err != nil {
			return nil, appquery.DatasetResult{}, err
		}
		result, err := client.Preview(ctx, query)
		return nil, result, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_run_analysis",
		Title:       "运行已有分析",
		Description: "Run one visible saved visual analysis and return bounded live data. Native SQL analyses are intentionally excluded from the AI integration.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input analysisInput) (*mcp.CallToolResult, appquery.DatasetResult, error) {
		result, err := client.RunAnalysis(ctx, input.AnalysisID)
		return nil, result, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "topbase_create_analysis",
		Title:       "创建分析",
		Description: "Save a validated visual QueryIR analysis. This is an additive write and must only be called after the user explicitly asks to save or create an analysis. Set confirmed=true to acknowledge that request.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &additive, OpenWorldHint: &closedWorld},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input createAnalysisInput) (*mcp.CallToolResult, any, error) {
		if !input.Confirmed {
			return nil, nil, errors.New("confirmed must be true after the user explicitly asks to save the analysis")
		}
		query, err := decodeQuery(input.Query)
		if err != nil {
			return nil, nil, err
		}
		result, err := client.CreateAnalysis(ctx, core.Question{
			Name: input.Name, Description: input.Description, CollectionID: input.CollectionID,
			QueryIR: &query, ChartSpec: input.ChartSpec,
		})
		return nil, result, err
	})

	return server
}

func decodeQuery(value map[string]any) (queryir.Query, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return queryir.Query{}, fmt.Errorf("encode QueryIR: %w", err)
	}
	var query queryir.Query
	if err := json.Unmarshal(data, &query); err != nil {
		return queryir.Query{}, fmt.Errorf("decode QueryIR: %w", err)
	}
	return query, nil
}

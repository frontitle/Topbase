package core

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

type AIProvider interface {
	Ask(context.Context, string, string) (ChatResponse, error)
}

type QueryService struct{ Executor QueryExecutor }

func (s QueryService) Run(ctx context.Context, databaseID, sql string) (QueryResult, error) {
	return s.RunArgs(ctx, databaseID, sql, nil)
}

func (s QueryService) RunArgs(ctx context.Context, databaseID, sql string, args []any) (QueryResult, error) {
	if strings.TrimSpace(databaseID) == "" || strings.TrimSpace(sql) == "" {
		return QueryResult{}, errors.New("database_id and sql are required")
	}
	statement := strings.ToLower(strings.TrimSpace(sql))
	if strings.Contains(statement, ";") {
		return QueryResult{}, errors.New("multiple statements are not allowed")
	}
	if !strings.HasPrefix(statement, "select") && !strings.HasPrefix(statement, "with") && !strings.HasPrefix(statement, "explain") {
		return QueryResult{}, errors.New("only read-only queries may run in the explorer")
	}
	if regexp.MustCompile("(?i)\\b(insert|update|delete|merge|drop|alter|create|grant|revoke|copy|call|vacuum)\\b").MatchString(statement) {
		return QueryResult{}, errors.New("query contains a write or administrative operation")
	}
	result, err := s.Executor.Execute(ctx, databaseID, sql, args...)
	if err != nil {
		return QueryResult{}, err
	}
	if result.Meta == nil {
		result.Meta = map[string]any{}
	}
	result.Meta["executed_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	result.Meta["cache_hit"] = false
	result.Meta["execution"] = "direct"
	return result, nil
}

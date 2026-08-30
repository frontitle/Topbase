package httpapi

import (
	"net/http"
	"strings"

	appquery "github.com/topbase/topbase/internal/app/query"
	"github.com/topbase/topbase/internal/core/queryir"
)

func (s *server) visualQuery(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	var request struct {
		Schema           string   `json:"schema"`
		Table            string   `json:"table"`
		Fields           []string `json:"fields"`
		Aggregation      string   `json:"aggregation"`
		AggregationField string   `json:"aggregation_field"`
		FilterField      string   `json:"filter_field"`
		FilterValue      string   `json:"filter_value"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	q, err := appquery.VisualRequestToQuery(r.PathValue("id"), request.Schema, request.Table, request.Fields, request.Aggregation, request.AggregationField, request.FilterField, request.FilterValue)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.catalog.EnsureConnected(r.Context(), r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := s.dataset.Run(r.Context(), q)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.attachDatasetSemantics(&result, q)
	writeJSON(w, http.StatusOK, map[string]any{
		"sql": result.SQL, "columns": result.Columns, "rows": result.Rows, "meta": result.Meta, "chartspec": result.ChartSpec, "queryir": q,
	})
}

func (s *server) runDataset(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	var q queryir.Query
	if !decodeJSON(w, r, &q) {
		return
	}
	if isAPIKeyRequest(r) {
		settings, err := s.identity.DeveloperSettings()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if q.Limit <= 0 || q.Limit > settings.MaxQueryRows {
			q.Limit = settings.MaxQueryRows
		}
	}
	if q.Source.DatabaseID != "" {
		if err := s.catalog.EnsureConnected(r.Context(), q.Source.DatabaseID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	result, err := s.dataset.Run(r.Context(), q)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.attachDatasetSemantics(&result, q)
	writeJSON(w, http.StatusOK, result)
}

func (s *server) attachDatasetSemantics(result *appquery.DatasetResult, q queryir.Query) {
	if result == nil || q.Source.Table == nil || q.Source.DatabaseID == "" {
		return
	}
	fields, err := s.content.ListFields(q.Source.DatabaseID, q.Source.Table.Schema, q.Source.Table.Name)
	if err != nil {
		return
	}
	byName := make(map[string]string, len(fields))
	for _, field := range fields {
		if field.SemanticType != "" {
			byName[field.Name] = field.SemanticType
		}
	}
	semanticTypes := map[string]string{}
	for _, column := range result.Columns {
		name := column
		if dot := strings.LastIndex(name, "."); dot >= 0 {
			name = name[dot+1:]
		}
		if semantic := byName[name]; semantic != "" {
			semanticTypes[column] = semantic
		}
	}
	if len(semanticTypes) == 0 {
		return
	}
	if result.Meta == nil {
		result.Meta = map[string]any{}
	}
	result.Meta["semantic_types"] = semanticTypes
}

package httpapi

import (
	"net/http"

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
	writeJSON(w, http.StatusOK, result)
}

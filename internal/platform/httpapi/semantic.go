package httpapi

import (
	"net/http"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func (s *server) listFields(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	items, err := s.content.ListFields(r.PathValue("id"), r.PathValue("schema"), r.PathValue("table"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) saveField(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var field core.FieldMeta
	if !decodeJSON(w, r, &field) {
		return
	}
	field.DatabaseID = r.PathValue("id")
	field.Schema = r.PathValue("schema")
	field.Table = r.PathValue("table")
	saved, err := s.content.SaveField(field)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) listSemanticTypes(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	writeJSON(w, http.StatusOK, core.SemanticTypes)
}

func (s *server) listModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	items, err := s.content.ListModels()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createModel(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCapability(w, r, "data", "curate")
	if !ok {
		return
	}
	var m core.Model
	if !decodeJSON(w, r, &m) {
		return
	}
	saved, err := s.content.CreateModel(m, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) getModel(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	item, err := s.content.GetModel(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) listMetrics(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	items, err := s.content.ListMetrics()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createMetric(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "curate"); !ok {
		return
	}
	var m core.Metric
	if !decodeJSON(w, r, &m) {
		return
	}
	saved, err := s.content.CreateMetric(m)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) listSegments(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	items, err := s.content.ListSegments()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createSegment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "curate"); !ok {
		return
	}
	var seg core.Segment
	if !decodeJSON(w, r, &seg) {
		return
	}
	saved, err := s.content.CreateSegment(seg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) listGlossary(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	items, err := s.content.ListGlossary()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createGlossary(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "curate"); !ok {
		return
	}
	var term core.GlossaryTerm
	if !decodeJSON(w, r, &term) {
		return
	}
	saved, err := s.content.CreateGlossary(term)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) drillDataset(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	var input struct {
		Query     queryir.Query        `json:"queryir"`
		Drill     queryir.DrillRequest `json:"drill"`
		JoinTable string               `json:"join_table"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.JoinTable != "" && input.Query.Source.Table != nil {
		fields, _ := s.content.ListFields(input.Query.Source.DatabaseID, input.Query.Source.Table.Schema, input.Query.Source.Table.Name)
		for _, field := range fields {
			if field.FKTarget != nil && field.FKTarget.Table == input.JoinTable {
				schema := field.FKTarget.Schema
				if schema == "" {
					schema = input.Query.Source.Table.Schema
				}
				input.Drill.Join = &queryir.Join{
					Type: "left", Implicit: true,
					Table: &queryir.TableRef{Schema: schema, Name: input.JoinTable},
					Conditions: []queryir.JoinCondition{{
						Left:  input.Query.Source.Table.Name + "." + field.Name,
						Right: input.JoinTable + "." + field.FKTarget.Name,
					}},
				}
				break
			}
		}
		if input.Drill.Join == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fk is required for implicit join to " + input.JoinTable})
			return
		}
	}
	next, err := queryir.Drill(input.Query, input.Drill)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if next.Source.DatabaseID != "" {
		if err := s.catalog.EnsureConnected(r.Context(), next.Source.DatabaseID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "queryir": next})
			return
		}
	}
	result, err := s.dataset.Run(r.Context(), next)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "queryir": next})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"queryir": next, "sql": result.SQL, "columns": result.Columns, "rows": result.Rows, "chartspec": result.ChartSpec})
}

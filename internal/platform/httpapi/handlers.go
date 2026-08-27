package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/topbase/topbase/internal/adapters"
	"github.com/topbase/topbase/internal/core"
)

func (s *server) requireAdmin(w http.ResponseWriter, r *http.Request) (core.User, bool) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return core.User{}, false
	}
	if !s.identity.IsAdmin(user.ID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
		return core.User{}, false
	}
	return user, true
}

func (s *server) databases(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	databases, err := s.catalog.Store.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "read catalog: " + err.Error()})
		return
	}
	for i, item := range databases {
		databases[i] = s.catalog.LiveStatus(item)
		if snap, err := s.catalog.Snapshot(item.ID); err == nil {
			databases[i].LastSyncedAt = &snap.SyncedAt
			databases[i].TableCount = len(snap.Tables)
		}
	}
	writeJSON(w, http.StatusOK, databases)
}

func (s *server) databaseEngines(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, adapters.EngineDefinitions())
}

func (s *server) getDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	id := r.PathValue("id")
	databases, err := s.catalog.Store.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var found *core.Database
	for i := range databases {
		if databases[i].ID == id {
			found = &databases[i]
			break
		}
	}
	if found == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "database not found"})
		return
	}
	payload := map[string]any{"database": found, "tables": []core.Table{}}
	*found = s.catalog.LiveStatus(*found)
	payload["database"] = found
	if snap, err := s.catalog.Snapshot(id); err == nil {
		found.LastSyncedAt = &snap.SyncedAt
		found.TableCount = len(snap.Tables)
		payload["database"] = found
		payload["tables"] = snap.Tables
		payload["synced_at"] = snap.SyncedAt
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *server) connectDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input core.ConnectionRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	database, err := s.catalog.Add(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, database)
}

func (s *server) testDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input core.ConnectionRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.catalog.Test(r.Context(), input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

func (s *server) testSavedDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input core.ConnectionRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &input) {
		return
	}
	input.ID = r.PathValue("id")
	if err := s.catalog.Test(r.Context(), input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

func (s *server) getDatabaseConnection(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	item, err := s.catalog.ConnectionSettings(r.PathValue("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, core.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) updateDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input core.ConnectionRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	database, err := s.catalog.Update(r.Context(), r.PathValue("id"), input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, core.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, database)
}

func (s *server) tables(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	id := r.PathValue("id")
	var tables []core.Table
	var err error
	if r.URL.Query().Get("cached") == "1" {
		snap, snapErr := s.catalog.Snapshot(id)
		if snapErr != nil {
			if errors.Is(snapErr, core.ErrNotFound) {
				writeJSON(w, http.StatusOK, []core.Table{})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": snapErr.Error()})
			return
		}
		tables = snap.Tables
	} else {
		if err := s.catalog.EnsureConnected(r.Context(), id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		tables, err = s.catalog.Connector.Tables(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	marked := map[string]bool{}
	if s.warehouse != nil {
		items, _ := s.warehouse.Tables.ListByDatabase(id)
		for _, item := range items {
			marked[item.Schema+"."+item.Name] = true
		}
	}
	payload := make([]map[string]any, 0, len(tables))
	for _, table := range tables {
		payload = append(payload, map[string]any{
			"schema": table.Schema, "name": table.Name, "description": table.Description, "columns": table.Columns,
			"warehouse": marked[table.Schema+"."+table.Name] || table.Schema == "warehouse",
		})
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *server) deleteDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if err := s.catalog.Remove(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) syncDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	snap, err := s.catalog.Sync(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (s *server) rescanTable(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	snap, err := s.catalog.RescanTable(r.Context(), r.PathValue("id"), r.PathValue("schema"), r.PathValue("table"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (s *server) getAnnotation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	annotation, err := s.metadata.GetTableAnnotation(r.PathValue("id"), r.PathValue("schema"), r.PathValue("table"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, annotation)
}

func (s *server) saveAnnotation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var annotation core.TableAnnotation
	if !decodeJSON(w, r, &annotation) {
		return
	}
	if annotation.FieldTypes == nil {
		annotation.FieldTypes = map[string]string{}
	}
	if err := s.metadata.SaveTableAnnotation(r.PathValue("id"), r.PathValue("schema"), r.PathValue("table"), annotation); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, annotation)
}

func (s *server) runQuery(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "sql", "native"); !ok {
		return
	}
	var request map[string]string
	if !decodeJSON(w, r, &request) {
		return
	}
	if request["database_id"] != "" {
		if err := s.catalog.EnsureConnected(r.Context(), request["database_id"]); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	result, err := s.queries.Run(r.Context(), request["database_id"], request["sql"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"columns": result.Columns, "rows": result.Rows, "meta": result.Meta})
}

func (s *server) chat(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	var request map[string]string
	if !decodeJSON(w, r, &request) {
		return
	}
	answer, err := s.ai.Ask(r.Context(), request["message"], request["database_id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"answer": answer.Answer, "sql": answer.SQL, "safe": answer.Safe})
}

func (s *server) feishuLogin(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "Feishu OAuth requires application credentials"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		path := r.URL.Path
		if strings.HasPrefix(path, "/public/") || strings.HasPrefix(path, "/embed/") || strings.HasPrefix(path, "/api/public/") {
			w.Header().Set("Content-Security-Policy", "frame-ancestors *")
		} else {
			w.Header().Set("X-Frame-Options", "DENY")
		}
		next.ServeHTTP(w, r)
	})
}

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func (s *server) currentUserOrKey(r *http.Request) (core.User, bool) {
	if user, ok := s.currentSessionUser(r); ok {
		return user, true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		user, err := s.identity.UserForAPIKey(strings.TrimPrefix(auth, "Bearer "))
		if err == nil {
			return user, true
		}
	}
	return core.User{}, false
}

func (s *server) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.identity.ListUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) inviteUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := s.identity.InviteUser(input.Name, input.Email, input.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *server) setUserActive(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		IsActive bool `json:"is_active"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.SetUserActive(r.PathValue("id"), input.IsActive); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "is_active": input.IsActive})
}

func (s *server) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.ResetUserPassword(r.PathValue("id"), input.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "password_reset"})
}

func (s *server) replaceUserGroups(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		GroupIDs []string `json:"group_ids"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.ReplaceUserManualGroups(r.PathValue("id"), input.GroupIDs); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "groups_updated"})
}

func (s *server) bindUserExternalIdentity(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		ProviderID string `json:"provider_id"`
		Subject    string `json:"subject"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.BindExternalIdentity(core.ExternalIdentityLink{ProviderID: input.ProviderID, Subject: input.Subject, UserID: r.PathValue("id")}); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "bound"})
}

func (s *server) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	items, err := s.identity.ListAPIKeys(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	key, err := s.identity.CreateAPIKey(user.ID, input.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, key)
}

func (s *server) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserOrKey(r); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	if err := s.identity.DeleteAPIKey(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) getPermissionGraph(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	graph, err := s.identity.PermissionGraph()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *server) putPermissionGraph(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var graph core.PermissionGraph
	if !decodeJSON(w, r, &graph) {
		return
	}
	saved, err := s.identity.SavePermissionGraph(graph)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	items, err := s.content.Search(r.URL.Query().Get("q"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) listBookmarks(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	items, err := s.content.ListBookmarks(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createBookmark(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	var input struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := s.content.AddBookmark(user.ID, input.TargetType, input.TargetID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *server) deleteBookmark(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	if err := s.content.DeleteBookmark(r.PathValue("id"), user.ID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listRevisions(w http.ResponseWriter, r *http.Request) {
	items, err := s.content.ListRevisions(r.URL.Query().Get("target_type"), r.URL.Query().Get("target_id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) listTrash(w http.ResponseWriter, _ *http.Request) {
	items, err := s.content.Trash()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) restoreTrash(w http.ResponseWriter, r *http.Request) {
	if err := s.content.Restore(r.PathValue("type"), r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func (s *server) archiveQuestion(w http.ResponseWriter, r *http.Request) {
	if err := s.content.ArchiveQuestion(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) exportDataset(w http.ResponseWriter, r *http.Request) {
	var q struct {
		QueryIR json.RawMessage `json:"queryir"`
	}
	if !decodeJSON(w, r, &q) {
		return
	}
	var query queryir.Query
	if err := json.Unmarshal(q.QueryIR, &query); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid queryir"})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="topbase.csv"`)
	if err := s.dataset.ExportCSV(r.Context(), query, w); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (s *server) exportQuestion(w http.ResponseWriter, r *http.Request) {
	question, err := s.content.GetQuestion(r.PathValue("id"))
	if err != nil || question.QueryIR == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question not exportable"})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+question.ID+`.csv"`)
	if err := s.dataset.ExportCSV(r.Context(), *question.QueryIR, w); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

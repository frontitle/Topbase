package httpapi

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/topbase/topbase/internal/core"
)

func (s *server) listDashboards(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	items, err := s.content.ListDashboards()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	visible := make([]core.Dashboard, 0, len(items))
	for _, item := range items {
		if s.canAccessDashboard(user, item, "view") {
			visible = append(visible, item)
		}
	}
	writeJSON(w, http.StatusOK, visible)
}

func (s *server) createDashboard(w http.ResponseWriter, r *http.Request) {
	var d core.Dashboard
	if !decodeJSON(w, r, &d) {
		return
	}
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	if d.CollectionID == "" {
		if project, err := s.content.EnsurePersonalCollection(user); err == nil {
			d.CollectionID = project.ID
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if !s.canAccessCollection(user, d.CollectionID, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "data-group edit permission required"})
		return
	}
	saved, err := s.content.CreateDashboard(d, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) getDashboard(w http.ResponseWriter, r *http.Request) {
	_, item, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "view")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) updateDashboard(w http.ResponseWriter, r *http.Request) {
	user, existing, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "edit")
	if !ok {
		return
	}
	var d core.Dashboard
	if !decodeJSON(w, r, &d) {
		return
	}
	d.ID = r.PathValue("id")
	// External publishing settings are controlled by dedicated, permissioned
	// endpoints; a normal editor must not publish or turn embedding on.
	d.PublicUUID = existing.PublicUUID
	d.PublicEmbedEnabled = existing.PublicEmbedEnabled
	if d.CollectionID == "" {
		d.CollectionID = existing.CollectionID
	}
	if d.CollectionID != existing.CollectionID && !s.canAccessCollection(user, d.CollectionID, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "destination data-group edit permission required"})
		return
	}
	saved, err := s.content.UpdateDashboard(d, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) archiveDashboard(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "edit"); !ok {
		return
	}
	if err := s.content.ArchiveDashboard(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) runDashboardCard(w http.ResponseWriter, r *http.Request) {
	_, board, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "view")
	if !ok {
		return
	}
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	var input struct {
		Filters map[string]any `json:"filters"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	if input.Filters == nil {
		input.Filters = map[string]any{}
	}
	result, err := s.dataset.RunDashboardCard(r.Context(), board, r.PathValue("cardId"), input.Filters, s.content.Questions)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) listAlerts(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	items, err := s.content.ListAlerts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	visible := make([]core.Alert, 0, len(items))
	for _, item := range items {
		if q, err := s.content.GetQuestion(item.QuestionID); err == nil && s.canAccessQuestion(user, q, "view") {
			visible = append(visible, item)
		}
	}
	writeJSON(w, http.StatusOK, visible)
}

func (s *server) createAlert(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	var a core.Alert
	if !decodeJSON(w, r, &a) {
		return
	}
	question, err := s.content.GetQuestion(a.QuestionID)
	if err != nil || !s.canAccessQuestion(user, question, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "analysis edit permission required"})
		return
	}
	saved, err := s.content.CreateAlert(a, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) deleteAlert(w http.ResponseWriter, r *http.Request) {
	alert, err := s.content.GetAlert(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found"})
		return
	}
	if _, _, ok := s.requireQuestionAccess(w, r, alert.QuestionID, "edit"); !ok {
		return
	}
	if err := s.content.DeleteAlert(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) runAlert(w http.ResponseWriter, r *http.Request) {
	alert, err := s.content.GetAlert(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found"})
		return
	}
	if _, _, ok := s.requireQuestionAccess(w, r, alert.QuestionID, "edit"); !ok {
		return
	}
	note, err := s.notify.RunAlert(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (s *server) listNotifications(w http.ResponseWriter, r *http.Request) {
	userID := ""
	if user, ok := s.currentUserOrKey(r); ok {
		userID = user.ID
	}
	items, err := s.content.ListNotifications(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) enableDashboardPublicLink(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "manage")
	if !ok {
		return
	}
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.PublicSharingEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "public sharing is disabled by the administrator"})
		return
	}
	saved, err := s.content.EnableDashboardPublicLink(r.PathValue("id"), user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s.publicLinkPayload(r, saved))
}

func (s *server) disableDashboardPublicLink(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "manage")
	if !ok {
		return
	}
	saved, err := s.content.DisableDashboardPublicLink(r.PathValue("id"), user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) setDashboardEmbedding(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.EmbeddingEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "embedding is disabled by the administrator"})
		return
	}
	saved, err := s.content.SetDashboardEmbedding(r.PathValue("id"), input.Enabled, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) copyDashboard(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "edit")
	if !ok {
		return
	}
	saved, err := s.content.DuplicateDashboard(r.PathValue("id"), user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) publicLinkPayload(r *http.Request, d core.Dashboard) map[string]any {
	origin := s.publicBaseURL(r)
	publicURL := origin + "/public/dashboard/" + d.PublicUUID + "/"
	embedURL := origin + "/embed/dashboard/" + d.PublicUUID + "/"
	return map[string]any{
		"dashboard":   d,
		"public_uuid": d.PublicUUID,
		"public_url":  publicURL,
		"embed_url":   embedURL,
		"iframe":      `<iframe src="` + embedURL + `" frameborder="0" width="100%" height="800" allowtransparency></iframe>`,
	}
}

func (s *server) getPublicDashboard(w http.ResponseWriter, r *http.Request) {
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.PublicSharingEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "public sharing is disabled by the administrator"})
		return
	}
	board, err := s.content.GetDashboardByPublicUUID(r.PathValue("uuid"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "公开链接不存在或已关闭"})
		return
	}
	questions := []core.Question{}
	seen := map[string]bool{}
	for _, card := range board.Cards {
		if card.QuestionID == "" || seen[card.QuestionID] {
			continue
		}
		seen[card.QuestionID] = true
		q, err := s.content.GetQuestion(card.QuestionID)
		if err != nil {
			continue
		}
		q.NativeSQL = ""
		questions = append(questions, q)
	}
	writeJSON(w, http.StatusOK, map[string]any{"dashboard": board, "questions": questions})
}

func (s *server) runPublicDashboardCard(w http.ResponseWriter, r *http.Request) {
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.PublicSharingEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "public sharing is disabled by the administrator"})
		return
	}
	board, err := s.content.GetDashboardByPublicUUID(r.PathValue("uuid"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "公开链接不存在或已关闭"})
		return
	}
	var input struct {
		Filters map[string]any `json:"filters"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	if input.Filters == nil {
		input.Filters = map[string]any{}
	}
	result, err := s.dataset.RunDashboardCard(r.Context(), board, r.PathValue("cardId"), input.Filters, s.content.Questions)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) servePublicDashboard(w http.ResponseWriter, r *http.Request) {
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.PublicSharingEnabled || (strings.HasPrefix(r.URL.Path, "/embed/") && !settings.EmbeddingEnabled) {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/embed/") {
		board, err := s.content.GetDashboardByPublicUUID(r.PathValue("uuid"))
		if err != nil || !board.PublicEmbedEnabled {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Security-Policy", "frame-ancestors *")
	} else {
		// Published pages remain visitable links, but cannot be repurposed as an
		// iframe without an administrator explicitly enabling the embed endpoint.
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'self'")
	}
	s.serveHTML(w, r, "embed/dashboard.html")
}

// serveDashboardView serves the authenticated dashboard workspace. It is kept
// separate from the public/embed view because it includes editing controls.
func (s *server) serveDashboardView(w http.ResponseWriter, r *http.Request) {
	s.serveHTML(w, r, "dashboard/view/index.html")
}

func (s *server) serveQuestionView(w http.ResponseWriter, r *http.Request) {
	s.serveHTML(w, r, "questions/view/index.html")
}

func (s *server) serveNewAnalysis(w http.ResponseWriter, r *http.Request) {
	s.serveHTML(w, r, "questions/new/index.html")
}

func (s *server) serveCollectionView(w http.ResponseWriter, r *http.Request) {
	s.serveHTML(w, r, "collections/view/index.html")
}

func (s *server) serveHTML(w http.ResponseWriter, r *http.Request, name string) {
	data, err := fs.ReadFile(s.static, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

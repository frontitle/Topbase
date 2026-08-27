package httpapi

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/topbase/topbase/internal/core"
)

func (s *server) listDashboards(w http.ResponseWriter, _ *http.Request) {
	items, err := s.content.ListDashboards()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createDashboard(w http.ResponseWriter, r *http.Request) {
	var d core.Dashboard
	if !decodeJSON(w, r, &d) {
		return
	}
	userID := ""
	if user, ok := s.currentUserOrKey(r); ok {
		userID = user.ID
		if d.CollectionID == "" {
			if project, err := s.content.EnsurePersonalCollection(user); err == nil {
				d.CollectionID = project.ID
			}
		}
	}
	saved, err := s.content.CreateDashboard(d, userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) getDashboard(w http.ResponseWriter, r *http.Request) {
	item, err := s.content.GetDashboard(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) updateDashboard(w http.ResponseWriter, r *http.Request) {
	var d core.Dashboard
	if !decodeJSON(w, r, &d) {
		return
	}
	d.ID = r.PathValue("id")
	userID := ""
	if user, ok := s.currentUserOrKey(r); ok {
		userID = user.ID
		if d.CollectionID == "" {
			if project, err := s.content.EnsurePersonalCollection(user); err == nil {
				d.CollectionID = project.ID
			}
		}
	}
	saved, err := s.content.UpdateDashboard(d, userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) archiveDashboard(w http.ResponseWriter, r *http.Request) {
	if err := s.content.ArchiveDashboard(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) runDashboardCard(w http.ResponseWriter, r *http.Request) {
	board, err := s.content.GetDashboard(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
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

func (s *server) listAlerts(w http.ResponseWriter, _ *http.Request) {
	items, err := s.content.ListAlerts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createAlert(w http.ResponseWriter, r *http.Request) {
	var a core.Alert
	if !decodeJSON(w, r, &a) {
		return
	}
	userID := ""
	if user, ok := s.currentUserOrKey(r); ok {
		userID = user.ID
	}
	saved, err := s.content.CreateAlert(a, userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) deleteAlert(w http.ResponseWriter, r *http.Request) {
	if err := s.content.DeleteAlert(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) runAlert(w http.ResponseWriter, r *http.Request) {
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
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.PublicSharingEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "public sharing is disabled by the administrator"})
		return
	}
	userID := ""
	if user, ok := s.currentUserOrKey(r); ok {
		userID = user.ID
	}
	saved, err := s.content.EnableDashboardPublicLink(r.PathValue("id"), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, publicLinkPayload(r, saved))
}

func (s *server) disableDashboardPublicLink(w http.ResponseWriter, r *http.Request) {
	userID := ""
	if user, ok := s.currentUserOrKey(r); ok {
		userID = user.ID
	}
	saved, err := s.content.DisableDashboardPublicLink(r.PathValue("id"), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) copyDashboard(w http.ResponseWriter, r *http.Request) {
	userID := ""
	if user, ok := s.currentUserOrKey(r); ok {
		userID = user.ID
	}
	saved, err := s.content.DuplicateDashboard(r.PathValue("id"), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func publicLinkPayload(r *http.Request, d core.Dashboard) map[string]any {
	origin := r.Header.Get("Origin")
	if origin == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		origin = scheme + "://" + r.Host
	}
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

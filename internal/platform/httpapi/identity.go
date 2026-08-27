package httpapi

import (
	"net/http"

	"github.com/topbase/topbase/internal/core"
)

func (s *server) setupStatus(w http.ResponseWriter, _ *http.Request) {
	done, err := s.identity.SetupCompleted()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"completed": done})
}

func (s *server) completeSetup(w http.ResponseWriter, r *http.Request) {
	var input core.SetupRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := s.identity.CompleteSetup(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if _, err := s.content.EnsurePersonalCollection(user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	session, _, err := s.identity.Login(input.AdminEmail, input.AdminPassword)
	if err != nil {
		writeJSON(w, http.StatusCreated, s.identity.WithRole(user))
		return
	}
	setSessionCookie(w, session.ID, session.ExpiresAt)
	writeJSON(w, http.StatusCreated, s.identity.WithRole(user))
}

func (s *server) createSession(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	session, user, err := s.identity.Login(input.Email, input.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	setSessionCookie(w, session.ID, session.ExpiresAt)
	writeJSON(w, http.StatusOK, s.identity.WithRole(user))
}

func (s *server) authOptions(w http.ResponseWriter, _ *http.Request) {
	settings, err := s.identity.AuthSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	providers, err := s.identity.IdentityProviders()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	visible := []map[string]string{}
	for _, provider := range providers {
		if (provider.Type == "google" || provider.Type == "wechat") && provider.Enabled && provider.ClientID != "" && provider.ClientSecret != "" {
			visible = append(visible, map[string]string{"id": provider.ID, "type": provider.Type, "name": provider.Name, "login_url": "/auth/oauth/" + provider.ID + "/login"})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"password_login_enabled": settings.PasswordLoginEnabled, "providers": visible})
}

func (s *server) getAuthSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	settings, err := s.identity.AuthSettings()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, settings)
}

func (s *server) saveAuthSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var settings core.AuthSettings
	if !decodeJSON(w, r, &settings) {
		return
	}
	if err := s.identity.SaveAuthSettings(settings); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, settings)
}

func (s *server) deleteSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		_ = s.identity.Logout(cookie.Value)
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) currentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentSessionUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	writeJSON(w, http.StatusOK, s.identity.WithRole(user))
}

func (s *server) listQuestions(w http.ResponseWriter, _ *http.Request) {
	items, err := s.content.ListQuestions()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createQuestion(w http.ResponseWriter, r *http.Request) {
	var q core.Question
	if !decodeJSON(w, r, &q) {
		return
	}
	userID := ""
	if user, ok := s.currentSessionUser(r); ok {
		userID = user.ID
		if q.CollectionID == "" {
			collection, err := s.content.EnsurePersonalCollection(user)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			q.CollectionID = collection.ID
		}
	}
	saved, err := s.content.CreateQuestion(q, userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) getQuestion(w http.ResponseWriter, r *http.Request) {
	item, err := s.content.GetQuestion(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) listCollections(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	items, err := s.content.ListCollections()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	visible := []core.Collection{}
	for _, item := range items {
		if s.identity.CanAccessProject(user, item, "view") {
			visible = append(visible, item)
		}
	}
	writeJSON(w, http.StatusOK, visible)
}

func (s *server) createCollection(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name         string `json:"name"`
		ParentID     string `json:"parent_id"`
		Kind         string `json:"kind"`
		OwnerGroupID string `json:"owner_group_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	ownerID := ""
	if input.Kind == "personal_project" {
		user, ok := s.currentUserOrKey(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
			return
		}
		ownerID = user.ID
	}
	item, err := s.content.CreateCollection(input.Name, input.ParentID, ownerID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if item.Kind == "team_project" {
		rules, _ := s.identity.ProjectAccessRules()
		if input.ParentID != "" {
			for _, rule := range rules {
				if rule.ProjectID == input.ParentID {
					rules = append(rules, core.ProjectAccessRule{ProjectID: item.ID, GroupID: rule.GroupID, Role: rule.Role})
				}
			}
		} else if input.OwnerGroupID != "" {
			rules = append(rules, core.ProjectAccessRule{ProjectID: item.ID, GroupID: input.OwnerGroupID, Role: "manage"})
			item.OwnerGroupID = input.OwnerGroupID
		}
		_ = s.identity.SaveProjectAccessRules(rules)
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *server) getProjectAccess(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "not signed in"})
		return
	}
	project, err := s.content.Collections.ByID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "project not found"})
		return
	}
	if !s.identity.CanAccessProject(user, project, "manage") {
		writeJSON(w, 403, map[string]string{"error": "project management permission required"})
		return
	}
	rules, _ := s.identity.ProjectAccessRules()
	out := []core.ProjectAccessRule{}
	for _, rule := range rules {
		if rule.ProjectID == project.ID {
			out = append(out, rule)
		}
	}
	writeJSON(w, 200, out)
}
func (s *server) putProjectAccess(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "not signed in"})
		return
	}
	project, err := s.content.Collections.ByID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "project not found"})
		return
	}
	if !s.identity.CanAccessProject(user, project, "manage") {
		writeJSON(w, 403, map[string]string{"error": "project management permission required"})
		return
	}
	var incoming []core.ProjectAccessRule
	if !decodeJSON(w, r, &incoming) {
		return
	}
	for i := range incoming {
		incoming[i].ProjectID = project.ID
	}
	all, _ := s.identity.ProjectAccessRules()
	kept := []core.ProjectAccessRule{}
	for _, rule := range all {
		if rule.ProjectID != project.ID {
			kept = append(kept, rule)
		}
	}
	kept = append(kept, incoming...)
	if err := s.identity.SaveProjectAccessRules(kept); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, incoming)
}

func (s *server) getCollection(w http.ResponseWriter, r *http.Request) {
	item, err := s.content.Collections.ByID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	user, ok := s.currentUserOrKey(r)
	if !ok || !s.identity.CanAccessProject(user, item, "view") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "project access denied"})
		return
	}
	questions, err := s.content.ListQuestions()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	children, _ := s.content.ListCollections()
	inCol := []core.Question{}
	kids := []core.Collection{}
	boards, _ := s.content.ListDashboards()
	inBoards := []core.Dashboard{}
	for _, q := range questions {
		if q.CollectionID == item.ID {
			inCol = append(inCol, q)
		}
	}
	for _, c := range children {
		if c.ParentID == item.ID {
			kids = append(kids, c)
		}
	}
	for _, board := range boards {
		if board.CollectionID == item.ID {
			inBoards = append(inBoards, board)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"collection": item, "questions": inCol, "dashboards": inBoards, "children": kids})
}

func (s *server) updateCollection(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := s.content.UpdateCollection(r.PathValue("id"), input.Name, input.ParentID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) deleteCollection(w http.ResponseWriter, r *http.Request) {
	if err := s.content.DeleteCollection(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) updateQuestion(w http.ResponseWriter, r *http.Request) {
	var q core.Question
	if !decodeJSON(w, r, &q) {
		return
	}
	q.ID = r.PathValue("id")
	if q.CollectionID == "" {
		if user, ok := s.currentUserOrKey(r); ok {
			if project, err := s.content.EnsurePersonalCollection(user); err == nil {
				q.CollectionID = project.ID
			}
		}
	}
	saved, err := s.content.UpdateQuestion(q)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *server) getSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"site_name": s.identity.SiteName()})
}

func (s *server) putSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		SiteName string `json:"site_name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.SetSiteName(input.SiteName); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"site_name": s.identity.SiteName()})
}

func (s *server) getAdminSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	settings, err := s.identity.AdminSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *server) putAdminSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input core.AdminSettings
	if !decodeJSON(w, r, &input) {
		return
	}
	settings, err := s.identity.SaveAdminSettings(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

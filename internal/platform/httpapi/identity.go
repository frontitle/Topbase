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

func (s *server) listQuestions(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	items, err := s.content.ListQuestions()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	visible := make([]core.Question, 0, len(items))
	for _, item := range items {
		if s.canAccessQuestion(user, item, "view") {
			visible = append(visible, item)
		}
	}
	writeJSON(w, http.StatusOK, visible)
}

func (s *server) createQuestion(w http.ResponseWriter, r *http.Request) {
	var q core.Question
	if !decodeJSON(w, r, &q) {
		return
	}
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	if q.CollectionID == "" {
		collection, err := s.content.EnsurePersonalCollection(user)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		q.CollectionID = collection.ID
	}
	if !s.canAccessCollection(user, q.CollectionID, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "data-group edit permission required"})
		return
	}
	if !s.identity.HasCapability(user.ID, "data", "view") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "data view permission required"})
		return
	}
	if (q.QueryType == "native" || q.NativeSQL != "") && !s.identity.HasCapability(user.ID, "sql", "native") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "native SQL permission required"})
		return
	}
	saved, err := s.content.CreateQuestion(q, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) getQuestion(w http.ResponseWriter, r *http.Request) {
	_, item, ok := s.requireQuestionAccess(w, r, r.PathValue("id"), "view")
	if !ok {
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
		if s.canAccessCollection(user, item.ID, "view") {
			if item.Kind == "personal_project" && item.PersonalOwnerUserID != user.ID {
				item.ReadOnly = true
				if owner, err := s.identity.Users.ByID(item.PersonalOwnerUserID); err == nil {
					item.SharedByName = owner.Name
				}
			}
			visible = append(visible, item)
		}
	}
	writeJSON(w, http.StatusOK, visible)
}

func (s *server) createCollection(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	var input struct {
		Name         string `json:"name"`
		ParentID     string `json:"parent_id"`
		Kind         string `json:"kind"`
		OwnerGroupID string `json:"owner_group_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Kind == "" {
		input.Kind = "team_project"
	}
	if input.Kind != "personal_project" && input.Kind != "team_project" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid data-group type"})
		return
	}
	ownerID := ""
	if input.Kind == "personal_project" {
		ownerID = user.ID
		input.OwnerGroupID = ""
	}
	if input.ParentID != "" && !s.canAccessCollection(user, input.ParentID, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "parent data-group edit permission required"})
		return
	}
	if input.ParentID == "" && input.Kind == "team_project" {
		if !s.identity.IsAdmin(user.ID) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only administrators can create a root team data group"})
			return
		}
	}
	item, err := s.content.CreateCollection(input.Name, input.ParentID, ownerID, input.OwnerGroupID)
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
		} else if item.OwnerGroupID != "" {
			rules = append(rules, core.ProjectAccessRule{ProjectID: item.ID, GroupID: item.OwnerGroupID, Role: "manage"})
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
	if !ok || !s.canAccessCollection(user, item.ID, "view") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "project access denied"})
		return
	}
	if item.Kind == "personal_project" && item.PersonalOwnerUserID != user.ID {
		item.ReadOnly = true
		if owner, err := s.identity.Users.ByID(item.PersonalOwnerUserID); err == nil {
			item.SharedByName = owner.Name
		}
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
		if c.ParentID == item.ID && s.canAccessCollection(user, c.ID, "view") {
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
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	existing, err := s.content.Collections.ByID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "data group not found"})
		return
	}
	if !s.canAccessCollection(user, existing.ID, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "data-group edit permission required"})
		return
	}
	var input struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ParentID != "" && !s.canAccessCollection(user, input.ParentID, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "destination data-group edit permission required"})
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
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	item, err := s.content.Collections.ByID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "data group not found"})
		return
	}
	if !s.canAccessCollection(user, item.ID, "manage") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "data-group management permission required"})
		return
	}
	if err := s.content.DeleteCollection(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) getCollectionShares(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	collection, err := s.content.Collections.ByID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "data group not found"})
		return
	}
	if collection.Kind != "personal_project" || collection.PersonalOwnerUserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only the owner can manage personal-group sharing"})
		return
	}
	shares, err := s.store.ListCollectionShares(collection.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	users := []core.User{}
	for _, share := range shares {
		if recipient, err := s.identity.Users.ByID(share.UserID); err == nil {
			users = append(users, recipient)
		}
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *server) putCollectionShares(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	collection, err := s.content.Collections.ByID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "data group not found"})
		return
	}
	if collection.Kind != "personal_project" || collection.PersonalOwnerUserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only the owner can share a personal group"})
		return
	}
	var input struct {
		UserIDs []string `json:"user_ids"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	unique := map[string]bool{}
	recipients := []string{}
	for _, id := range input.UserIDs {
		if id == user.ID || id == "" || unique[id] {
			continue
		}
		if _, err := s.identity.Users.ByID(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "shared recipient not found"})
			return
		}
		unique[id] = true
		recipients = append(recipients, id)
	}
	if err := s.store.ReplaceCollectionShares(collection.ID, recipients); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) updateQuestion(w http.ResponseWriter, r *http.Request) {
	user, existing, ok := s.requireQuestionAccess(w, r, r.PathValue("id"), "edit")
	if !ok {
		return
	}
	var q core.Question
	if !decodeJSON(w, r, &q) {
		return
	}
	q.ID = r.PathValue("id")
	if q.CollectionID == "" {
		project, err := s.content.EnsurePersonalCollection(user)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		q.CollectionID = project.ID
	}
	if q.CollectionID != existing.CollectionID && !s.canAccessCollection(user, q.CollectionID, "edit") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "destination data-group edit permission required"})
		return
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

package httpapi

import (
	"net/http"

	"github.com/topbase/topbase/internal/core"
)

func (s *server) requireCapability(w http.ResponseWriter, r *http.Request, capability, required string) (core.User, bool) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return core.User{}, false
	}
	if !s.identity.HasCapability(user.ID, capability, required) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": capability + " permission required: " + required})
		return core.User{}, false
	}
	return user, true
}

func (s *server) canAccessCollection(user core.User, collectionID, required string) bool {
	if user.ID == "" || collectionID == "" {
		return false
	}
	collection, err := s.content.Collections.ByID(collectionID)
	if err != nil {
		return false
	}
	if s.identity.CanAccessProject(user, collection, required) {
		return true
	}
	// A personal analysis group can be shared directly. It deliberately grants
	// only viewing: recipients can inspect its analyses but cannot edit, move,
	// create, or delete anything inside it.
	if required == "view" && collection.Kind == "personal_project" {
		shared, err := s.store.IsCollectionSharedWith(collectionID, user.ID)
		return err == nil && shared
	}
	return false
}

func (s *server) canAccessQuestion(user core.User, question core.Question, required string) bool {
	if s.identity.IsAdmin(user.ID) {
		return true
	}
	if question.CollectionID != "" {
		return s.canAccessCollection(user, question.CollectionID, required)
	}
	// Legacy ungrouped content remains private to its creator.
	return question.CreatedBy != "" && question.CreatedBy == user.ID
}

func (s *server) canAccessDashboard(user core.User, dashboard core.Dashboard, required string) bool {
	if s.identity.IsAdmin(user.ID) {
		return true
	}
	if dashboard.CollectionID != "" {
		return s.canAccessCollection(user, dashboard.CollectionID, required)
	}
	return dashboard.CreatedBy != "" && dashboard.CreatedBy == user.ID
}

func (s *server) requireQuestionAccess(w http.ResponseWriter, r *http.Request, id, required string) (core.User, core.Question, bool) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return core.User{}, core.Question{}, false
	}
	question, err := s.content.GetQuestion(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "analysis not found"})
		return core.User{}, core.Question{}, false
	}
	if !s.canAccessQuestion(user, question, required) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "analysis access denied"})
		return core.User{}, core.Question{}, false
	}
	return user, question, true
}

func (s *server) requireDashboardAccess(w http.ResponseWriter, r *http.Request, id, required string) (core.User, core.Dashboard, bool) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return core.User{}, core.Dashboard{}, false
	}
	dashboard, err := s.content.GetDashboard(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "dashboard not found"})
		return core.User{}, core.Dashboard{}, false
	}
	if !s.canAccessDashboard(user, dashboard, required) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "dashboard access denied"})
		return core.User{}, core.Dashboard{}, false
	}
	return user, dashboard, true
}

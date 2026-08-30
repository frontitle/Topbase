package httpapi

import (
	"net/http"
	"time"

	"github.com/topbase/topbase/internal/core"
)

type developerStatus struct {
	core.DeveloperSettings
	CanCreateKey bool `json:"can_create_key"`
	ActiveKeys   int  `json:"active_keys"`
}

type adminAPIKey struct {
	core.APIKey
	OwnerName  string `json:"owner_name"`
	OwnerEmail string `json:"owner_email"`
	Expired    bool   `json:"expired"`
}

func (s *server) developerPing(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUserOrKey(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
		return
	}
	settings, err := s.identity.DeveloperSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok", "user_id": user.ID, "user_name": user.Name,
		"max_query_rows": settings.MaxQueryRows, "allow_analysis_write": settings.AllowAnalysisWrite,
	})
}

func (s *server) getDeveloperStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentSessionUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	settings, err := s.identity.DeveloperSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if settings.PublicBaseURL == "" {
		settings.PublicBaseURL = s.publicBaseURL(r)
	}
	keys, err := s.identity.ListAPIKeys(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	active := 0
	now := time.Now().UTC()
	for _, key := range keys {
		if key.ExpiresAt == nil || now.Before(*key.ExpiresAt) {
			active++
		}
	}
	writeJSON(w, http.StatusOK, developerStatus{
		DeveloperSettings: settings,
		CanCreateKey:      settings.Enabled && (settings.AllowPersonalKeys || s.identity.IsAdmin(user.ID)),
		ActiveKeys:        active,
	})
}

func (s *server) getDeveloperSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	settings, err := s.identity.DeveloperSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *server) putDeveloperSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input core.DeveloperSettings
	if !decodeJSON(w, r, &input) {
		return
	}
	settings, err := s.identity.SaveDeveloperSettings(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *server) listAllAPIKeys(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	keys, err := s.identity.ListAllAPIKeys()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	now := time.Now().UTC()
	items := make([]adminAPIKey, 0, len(keys))
	for _, key := range keys {
		item := adminAPIKey{APIKey: key, Expired: key.ExpiresAt != nil && !now.Before(*key.ExpiresAt)}
		if owner, ownerErr := s.identity.Users.ByID(key.UserID); ownerErr == nil {
			item.OwnerName, item.OwnerEmail = owner.Name, owner.Email
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) deleteAnyAPIKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if err := s.identity.DeleteAPIKey(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

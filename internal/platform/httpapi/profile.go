package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/topbase/topbase/internal/core"
)

type profileProvider struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	SelfBindable bool   `json:"self_bindable"`
	Linked       bool   `json:"linked"`
	CanUnbind    bool   `json:"can_unbind"`
	LoginURL     string `json:"login_url,omitempty"`
}

func (s *server) getUserProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentSessionUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	providers, err := s.identity.IdentityProviders()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	links, err := s.identity.ExternalIdentityLinksForUser(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	linked := map[string]bool{}
	for _, link := range links {
		linked[link.ProviderID] = true
	}
	authSettings, _ := s.identity.AuthSettings()
	usableLinks := usableProfileLinks(providers, linked)
	visibleProviders := make([]profileProvider, 0, len(providers))
	for _, provider := range providers {
		configured := provider.Enabled && strings.TrimSpace(provider.ClientID) != "" && strings.TrimSpace(provider.ClientSecret) != ""
		selfBindable := configured && (provider.Type == "google" || provider.Type == "wechat")
		item := profileProvider{
			ID: provider.ID, Type: provider.Type, Name: provider.Name, Enabled: configured,
			SelfBindable: selfBindable, Linked: linked[provider.ID],
		}
		item.CanUnbind = authSettings.PasswordLoginEnabled || !usableLinks[provider.ID] || len(usableLinks) > 1
		if selfBindable && !item.Linked {
			item.LoginURL = "/auth/oauth/" + provider.ID + "/login?intent=bind"
		}
		visibleProviders = append(visibleProviders, item)
	}
	groups, _ := s.identity.Groups.GroupsForUser(user.ID)
	groupNames := make([]string, 0, len(groups))
	for _, group := range groups {
		groupNames = append(groupNames, group.Name)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":                   s.identity.WithRole(user),
		"groups":                 groupNames,
		"providers":              visibleProviders,
		"password_login_enabled": authSettings.PasswordLoginEnabled,
		"password_configured":    user.PasswordHash != "",
	})
}

func usableProfileLinks(providers []core.IdentityProvider, linked map[string]bool) map[string]bool {
	usable := map[string]bool{}
	for _, provider := range providers {
		if linked[provider.ID] && provider.Enabled && provider.ClientID != "" && provider.ClientSecret != "" && (provider.Type == "google" || provider.Type == "wechat") {
			usable[provider.ID] = true
		}
	}
	return usable
}

func (s *server) updateUserProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentSessionUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	var input struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		Locale          string `json:"locale"`
		Theme           string `json:"theme"`
		AvatarURL       string `json:"avatar_url"`
		CurrentPassword string `json:"current_password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	updated, err := s.identity.UpdateProfile(user.ID, input.Name, input.Email, input.Locale, input.Theme, input.AvatarURL, input.CurrentPassword)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *server) changeUserPassword(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentSessionUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.ChangePassword(user.ID, input.CurrentPassword, input.NewPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "password_changed"})
}

func (s *server) unbindUserExternalIdentity(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentSessionUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
		return
	}
	authSettings, _ := s.identity.AuthSettings()
	if !authSettings.PasswordLoginEnabled {
		links, err := s.identity.ExternalIdentityLinksForUser(user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		linked := map[string]bool{}
		for _, link := range links {
			linked[link.ProviderID] = true
		}
		providers, err := s.identity.IdentityProviders()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		usable := usableProfileLinks(providers, linked)
		if usable[r.PathValue("provider")] && len(usable) <= 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot unlink the only sign-in method while password login is disabled"})
			return
		}
	}
	if err := s.identity.UnbindExternalIdentity(user.ID, r.PathValue("provider")); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "binding not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

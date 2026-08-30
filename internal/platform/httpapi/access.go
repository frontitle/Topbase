package httpapi

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

// accessControl is the server-side boundary for the workbench. UI redirects
// remain useful for navigation, but they are not an authorization mechanism.
func (s *server) accessControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		if _, sessionAuth := s.currentSessionUser(r); sessionAuth {
			if _, err := r.Cookie(csrfCookie); err != nil {
				setCSRFCookie(w, newCSRFToken(), time.Now().Add(24*time.Hour))
			}
			next.ServeHTTP(w, r)
			return
		}
		if isAPIKeyRequest(r) {
			raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if _, err := s.identity.UserForAPIKey(raw); err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
				return
			}
			settings, err := s.identity.DeveloperSettings()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "read developer settings: " + err.Error()})
				return
			}
			if !developerAPIRequestAllowed(r, settings) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "this endpoint is not available to MCP or CLI API keys"})
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
			return
		}
		http.Redirect(w, r, "/auth/login/", http.StatusFound)
	})
}

func isAPIKeyRequest(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")
}

func developerAPIRequestAllowed(r *http.Request, settings core.DeveloperSettings) bool {
	p := strings.TrimSuffix(r.URL.Path, "/")
	if r.Method == http.MethodGet {
		if p == "/api/developer/ping" || p == "/api/databases" || p == "/api/collections" || p == "/api/questions" {
			return true
		}
		if strings.HasPrefix(p, "/api/databases/") {
			parts := strings.Split(strings.TrimPrefix(p, "/api/databases/"), "/")
			if len(parts) == 2 && parts[0] != "" && parts[1] == "tables" {
				return true
			}
			return len(parts) == 5 && parts[0] != "" && parts[1] == "tables" && parts[2] != "" && parts[3] != "" && (parts[4] == "fields" || parts[4] == "annotation")
		}
		questionID := strings.TrimPrefix(p, "/api/questions/")
		if p != questionID && questionID != "" && !strings.Contains(questionID, "/") {
			return true
		}
	}
	if r.Method == http.MethodPost && p == "/api/dataset" {
		return true
	}
	return r.Method == http.MethodPost && p == "/api/questions" && settings.AllowAnalysisWrite
}

func isPublicRequest(r *http.Request) bool {
	p := r.URL.Path
	if p == "/api/health" || p == "/api/ready" || p == "/api/version" || p == "/api/setup/status" || p == "/api/setup" || p == "/api/session" || p == "/api/auth/options" || strings.HasPrefix(p, "/api/public/") {
		return true
	}
	if strings.HasPrefix(p, "/auth/") || strings.HasPrefix(p, "/setup/") || strings.HasPrefix(p, "/public/") || strings.HasPrefix(p, "/embed/") {
		return true
	}
	// Shared JS/CSS/images are needed by both the authenticated product and
	// public dashboard embeds. They do not contain application data.
	ext := strings.ToLower(path.Ext(p))
	if strings.HasPrefix(p, "/assets/maps/") && ext == ".json" {
		return true
	}
	switch ext {
	case ".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2":
		return true
	}
	return false
}

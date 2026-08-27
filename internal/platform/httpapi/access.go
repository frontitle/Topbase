package httpapi

import (
	"net/http"
	"path"
	"strings"
	"time"
)

// accessControl is the server-side boundary for the workbench. UI redirects
// remain useful for navigation, but they are not an authorization mechanism.
func (s *server) accessControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := s.currentUserOrKey(r); ok {
			if _, sessionAuth := s.currentSessionUser(r); sessionAuth {
				if _, err := r.Cookie(csrfCookie); err != nil {
					setCSRFCookie(w, newCSRFToken(), time.Now().Add(24*time.Hour))
				}
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
	switch ext {
	case ".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2":
		return true
	}
	return false
}

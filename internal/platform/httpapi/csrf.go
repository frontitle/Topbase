package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

const csrfCookie = "topbase_csrf"

func newCSRFToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

// csrfProtection uses a host-only double-submit cookie for authenticated
// browser mutations. Bearer API clients and public read-only card execution do
// not use browser sessions and are not subject to this check.
func (s *server) csrfProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || isPublicRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := s.currentSessionUser(r); !ok {
			next.ServeHTTP(w, r)
			return
		}
		// Non-browser clients that deliberately authenticate with a session
		// cookie remain compatible. Browsers send Origin/Sec-Fetch-Site and must
		// prove same-origin script access with the CSRF header.
		browserRequest := r.Header.Get("Origin") != "" || r.Header.Get("Sec-Fetch-Site") != ""
		if !browserRequest {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(csrfCookie)
		header := r.Header.Get("X-Topbase-CSRF")
		if err != nil || cookie.Value == "" || header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "CSRF validation failed; refresh the page and try again"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

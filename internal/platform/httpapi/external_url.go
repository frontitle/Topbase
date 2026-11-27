package httpapi

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/frontitle/Topbase/internal/core"
)

func requestScheme(r *http.Request) string {
	if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
		for _, part := range strings.Split(strings.Split(forwarded, ",")[0], ";") {
			key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
			if ok && strings.EqualFold(key, "proto") {
				value = strings.ToLower(strings.Trim(value, `"`))
				if value == "http" || value == "https" {
					return value
				}
			}
		}
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		forwarded = strings.ToLower(forwarded)
		if forwarded == "http" || forwarded == "https" {
			return forwarded
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func requestHostname(r *http.Request) string {
	host := r.Host
	if value, _, err := net.SplitHostPort(host); err == nil {
		return value
	}
	return strings.Trim(host, "[]")
}

func hostWithPort(host string, port int) string {
	if port <= 0 {
		return host
	}
	return net.JoinHostPort(strings.Trim(host, "[]"), strconv.Itoa(port))
}

func (s *server) publicBaseURL(r *http.Request) string {
	scheme := requestScheme(r)
	host := r.Host
	settings, err := s.identity.AdminSettings()
	if err != nil {
		return scheme + "://" + host
	}
	configured := settings.PublicScheme != "" && settings.PublicScheme != "auto"
	if configured {
		scheme = settings.PublicScheme
	}
	if settings.CustomDomain != "" {
		host = settings.CustomDomain
	} else if configured || settings.PublicPort > 0 {
		host = requestHostname(r)
	}
	if settings.PublicPort > 0 {
		host = hostWithPort(host, settings.PublicPort)
	}
	return strings.TrimRight(scheme+"://"+host, "/")
}

func (s *server) externalURL(r *http.Request, path string) string {
	return s.publicBaseURL(r) + "/" + strings.TrimLeft(path, "/")
}

// Scheduled deliveries have no browser request from which to infer an origin.
// Use the administrator's workspace domain and the authenticated dashboard route.
func (s *server) subscriptionDashboardURL(hook core.WebhookEndpoint, dashboardID string) string {
	switch hook.Provider {
	case "slack", "teams", "google_chat":
	default:
		return ""
	}
	if dashboardID == "" {
		return ""
	}
	settings, err := s.identity.AdminSettings()
	if err != nil || settings.CustomDomain == "" {
		return ""
	}
	scheme := settings.PublicScheme
	if scheme == "auto" || scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + hostWithPort(settings.CustomDomain, settings.PublicPort) + "/dashboard/" + url.PathEscape(dashboardID) + "/"
}

func secureCookieRequest(r *http.Request) bool {
	return strings.EqualFold(os.Getenv("TOPBASE_SECURE_COOKIES"), "true") || requestScheme(r) == "https"
}

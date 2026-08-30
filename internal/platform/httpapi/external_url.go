package httpapi

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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

func secureCookieRequest(r *http.Request) bool {
	return strings.EqualFold(os.Getenv("TOPBASE_SECURE_COOKIES"), "true") || requestScheme(r) == "https"
}

package httpapi

import (
	"net/http/httptest"
	"testing"

	identityapp "github.com/frontitle/Topbase/internal/app/identity"
	"github.com/frontitle/Topbase/internal/core"
)

type externalURLSettings map[string]string

func (s externalURLSettings) Get(key string) (string, bool, error) {
	value, ok := s[key]
	return value, ok, nil
}
func (s externalURLSettings) Set(key, value string) error { s[key] = value; return nil }

func TestRequestSchemeSupportsForwardedHTTPS(t *testing.T) {
	request := httptest.NewRequest("GET", "http://internal:8080/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	if got := requestScheme(request); got != "https" {
		t.Fatalf("expected https, got %q", got)
	}
}

func TestHostWithPortSupportsIPv6(t *testing.T) {
	if got := hostWithPort("[2001:db8::1]", 8443); got != "[2001:db8::1]:8443" {
		t.Fatalf("unexpected IPv6 host: %q", got)
	}
}

func TestPublicBaseURLUsesConfiguredDomainAndPort(t *testing.T) {
	settings := externalURLSettings{
		"admin_settings": `{"site_name":"Topbase","timezone":"Asia/Shanghai","public_scheme":"https","custom_domain":"bi.example.com","public_port":8443}`,
	}
	server := server{identity: identityapp.Service{Settings: settings}}
	request := httptest.NewRequest("GET", "http://internal:8080/", nil)
	if got := server.publicBaseURL(request); got != "https://bi.example.com:8443" {
		t.Fatalf("unexpected public base URL: %q", got)
	}
}

func TestSubscriptionDashboardURL(t *testing.T) {
	settings := externalURLSettings{}
	s := server{identity: identityapp.Service{Settings: settings}}
	hook := core.WebhookEndpoint{Provider: "slack"}
	if got := s.subscriptionDashboardURL(hook, "board"); got != "" {
		t.Fatalf("unexpected unconfigured dashboard URL %q", got)
	}
	settings["admin_settings"] = `{"custom_domain":"bi.example.com","public_port":8443,"public_scheme":"auto","public_sharing_enabled":false}`
	for _, provider := range []string{"slack", "teams", "google_chat"} {
		hook.Provider = provider
		if got := s.subscriptionDashboardURL(hook, "sales"); got != "https://bi.example.com:8443/dashboard/sales/" {
			t.Errorf("%s dashboard URL: %q", provider, got)
		}
	}
}

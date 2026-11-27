package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/topbase/topbase/internal/core"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendToOfficeWebhooks(t *testing.T) {
	prior := client.Transport
	t.Cleanup(func() { client.Transport = prior })
	for _, tc := range []struct {
		provider, url, envelope string
		status                  int
	}{
		{"slack", "https://hooks.slack.com/services/T/B/test-token", "blocks", 200},
		{"teams", "https://tenant.environment.api.powerplatform.com/powerautomate/automations/direct/workflows/a/triggers/manual/paths/invoke?sig=test-token", "attachments", 202},
		{"google_chat", "https://chat.googleapis.com/v1/spaces/AAA/messages?key=test-key&token=test-token", "cardsV2", 200},
	} {
		hook := core.WebhookEndpoint{Provider: tc.provider, URL: tc.url, TemplateType: "card"}
		client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || r.URL.String() != tc.url {
				t.Fatalf("wrong request for %s", tc.provider)
			}
			var payload object
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload[tc.envelope] == nil {
				t.Fatalf("wrong envelope for %s: %v", tc.provider, err)
			}
			return &http.Response{StatusCode: tc.status, Body: http.NoBody}, nil
		})
		if err := Send(context.Background(), hook, "测试", "消息"); err != nil {
			t.Fatalf("%s: %v", tc.provider, err)
		}
	}
}

func TestDeliveryFailureKeepsCredentialsPrivate(t *testing.T) {
	prior := client.Transport
	t.Cleanup(func() { client.Transport = prior })
	hook := core.WebhookEndpoint{Provider: "slack", URL: "https://hooks.slack.com/services/T/B/test-token"}
	for _, status := range []int{0, http.StatusTooManyRequests, http.StatusTemporaryRedirect} {
		requests := 0
		client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			if status == 0 {
				return nil, errors.New("could not reach " + r.URL.String())
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://example.com/"}}, Body: io.NopCloser(strings.NewReader(r.URL.String()))}, nil
		})
		err := Send(context.Background(), hook, "日报", "汇总")
		if err == nil || strings.Contains(err.Error(), "test-token") || requests != 1 {
			t.Fatalf("unexpected delivery result for status %d: %v, %d requests", status, err, requests)
		}
	}
}

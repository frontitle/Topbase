package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/frontitle/Topbase/internal/core"
)

func TestWebhookQuickBindingManagement(t *testing.T) {
	handler := testServer(t)
	admin := adminSession(t, handler)
	var received atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Topbase-Signature") != "private-signature" {
			t.Error("stored webhook signature lost on edit")
		}
		received.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	request := func(method, path string, hooks []core.WebhookEndpoint, status int) string {
		t.Helper()
		body, err := json.Marshal(hooks)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.AddCookie(admin)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s %s = %d: %s", method, path, rec.Code, rec.Body)
		}
		result := rec.Body.String()
		for _, secret := range []string{"private-endpoint", "private-signature"} {
			if strings.Contains(result, secret) {
				t.Fatal("returned webhook credentials")
			}
		}
		return result
	}
	hooks := []core.WebhookEndpoint{
		{ID: "generic", Name: "Local", Provider: "generic", URL: upstream.URL + "/private-endpoint", Secret: "private-signature", Enabled: true},
		{ID: "chat", Name: "Chat", Provider: "google_chat", URL: "https://chat.googleapis.com/v1/spaces/room/messages?key=x&token=private-endpoint", Enabled: true},
	}
	request("PUT", "/api/webhooks", hooks, 200)
	config := request("GET", "/api/webhooks", nil, 200)
	var masked []core.WebhookEndpoint
	if err := json.Unmarshal([]byte(config), &masked); err != nil || len(masked) != 2 {
		t.Fatalf("read hooks = %s", config)
	}
	masked[0].Name = "Updated"
	request("PUT", "/api/webhooks", masked, 200)
	if received.Load() != 0 {
		t.Fatal("saving a channel sent a message")
	}
	request("POST", "/api/webhooks/generic/test", nil, 200)
	masked[0].Enabled = false
	request("PUT", "/api/webhooks", masked, 200)
	request("POST", "/api/webhooks/generic/test", nil, 400)
	request("POST", "/api/webhooks/missing/test", nil, 404)
	if received.Load() != 1 {
		t.Fatalf("received %d test notifications, want one", received.Load())
	}
	masked[1].Provider = "slack"
	request("PUT", "/api/webhooks", masked, 400)
	channels := request("GET", "/api/subscription-channels", nil, 200)
	if !strings.Contains(channels, `"id":"chat"`) || strings.Contains(channels, `"id":"generic"`) {
		t.Fatalf("enabled channels = %s", channels)
	}
}

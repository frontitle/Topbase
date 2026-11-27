package core

import "testing"

func TestVendorWebhookAddresses(t *testing.T) {
	tests := []struct {
		provider string
		url      string
		valid    bool
	}{
		{"slack", "https://hooks.slack.com/services/T1/B2/test_token", true},
		{"slack", "https://hooks.slack-gov.com/services/T1/B2/test_token", true},
		{"slack", "https://hooks.slack.com.example.org/services/T1/B2/test", false},
		{"slack", "https://hooks.slack.com@localhost/services/T1/B2/test", false},
		{"slack", "https://hooks.slack.com:8080/services/T1/B2/test", false},
		{"slack", "https://hooks.slack.com/services/T1/B2", false},
		{"teams", "https://tenant.aa.environment.api.powerplatform.com:443/powerautomate/automations/direct/workflows/id/triggers/manual/paths/invoke?sig=test", true},
		{"teams", "https://tenant.aa.environment.api.powerplatform.com/powerautomate/automations/direct/cu/20/workflows/id/triggers/manual/paths/invoke?sig=test", true},
		{"teams", "https://prod-1.westeurope.logic.azure.com/workflows/id/triggers/manual/paths/invoke?sig=test", true},
		{"teams", "https://tenant.aa.environment.api.powerplatform.com/powerautomate/automations/direct/workflows/id/triggers/manual/paths/invoke", false},
		{"teams", "https://prod-1.westeurope.logic.azure.com/other?sig=test", false},
		{"google_chat", "https://chat.googleapis.com/v1/spaces/abc/messages?key=x&token=y", true},
		{"google_chat", "https://chat.googleapis.com/v1/spaces/abc/messages?key=x", false},
		{"google_chat", "http://chat.googleapis.com/v1/spaces/abc/messages?key=x&token=y", false},
		{"generic", "http://127.0.0.1:8000/notifications", true},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if err := ValidateWebhookURL(tt.provider, tt.url); (err == nil) != tt.valid {
				t.Fatalf("valid=%v, got error %v", tt.valid, err)
			}
		})
	}
}

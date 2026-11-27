package webhook

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/topbase/topbase/internal/core"
)

func TestOfficeMessageFormats(t *testing.T) {
	const url = "https://bi.example.com/dashboard/sales/"
	for _, tc := range []struct {
		provider, style, rootKey, expected string
	}{
		{"slack", "text", "text", "|打开仪表盘>"},
		{"slack", "post", "blocks", "mrkdwn"},
		{"slack", "card", "blocks", "open_dashboard"},
		{"teams", "text", "attachments", "[打开仪表盘]("},
		{"teams", "post", "attachments", "Bolder"},
		{"teams", "card", "attachments", "Action.OpenUrl"},
		{"google_chat", "text", "text", "|打开仪表盘>"},
		{"google_chat", "post", "text", "*运营日报*"},
		{"google_chat", "card", "cardsV2", "openLink"},
	} {
		t.Run(tc.provider+"/"+tc.style, func(t *testing.T) {
			hook := core.WebhookEndpoint{Provider: tc.provider, TemplateType: tc.style, TemplateTitle: "运营{{title}}", TemplateBody: "汇总 {{body}}"}
			raw, err := render(hook, "日报", "42", url)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatal(err)
			}
			// Compare decoded strings to account for HTML escaping in JSON.
			if payload[tc.rootKey] == nil || !containsText(payload[tc.rootKey], tc.expected) || !containsText(payload[tc.rootKey], "运营日报") || !containsText(payload[tc.rootKey], "汇总 42") || !containsText(payload[tc.rootKey], url) {
				t.Fatalf("unexpected native payload: %s", raw)
			}
			withoutLink, err := Render(hook, "日报", "42")
			if err != nil || strings.Contains(string(withoutLink), "打开仪表盘") {
				t.Fatalf("unexpected test notification: %s, %v", withoutLink, err)
			}
		})
	}
}

func containsText(value any, text string) bool {
	switch v := value.(type) {
	case string:
		return strings.Contains(v, text)
	case []any:
		for _, item := range v {
			if containsText(item, text) {
				return true
			}
		}
	case map[string]any:
		for key, item := range v {
			if key == text || containsText(item, text) {
				return true
			}
		}
	}
	return false
}

func TestWebhookCustomAndLegacyTemplates(t *testing.T) {
	const title = "报表 \"A\" {{body}}"
	const body = "共两行\nA\\B"
	const url = "https://bi.example.com/dashboard/b/"
	template := `{"text":"{{title}}\n{{body}}","link":"{{dashboard_url}}","id":9007199254740995}`
	for _, provider := range []string{"slack", "teams", "google_chat", "wecom", "dingtalk", "generic", "feishu"} {
		hook := core.WebhookEndpoint{Provider: provider, TemplateType: "custom", Template: template}
		raw, err := render(hook, title, body, url)
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			Text, Link string
			ID         int64
		}
		if err := json.Unmarshal(raw, &got); err != nil || got.Text != title+"\n"+body || got.Link != url || got.ID != 9007199254740995 {
			t.Fatalf("%s: %s", provider, raw)
		}
	}
	legacy := core.WebhookEndpoint{Provider: "feishu", TemplateType: "text", Template: `{"msg_type":"text","content":{"text":"固定前缀 {{body}}"}}`}
	raw, err := Render(legacy, title, body)
	if err != nil {
		t.Fatal(err)
	}
	var got object
	if err := json.Unmarshal(raw, &got); err != nil || !reflect.DeepEqual(got, object{"msg_type": "text", "content": object{"text": "固定前缀 " + body}}) {
		t.Fatalf("legacy Feishu preset changed: %s", raw)
	}
}

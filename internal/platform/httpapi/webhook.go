package httpapi

import (
	"net/http"

	"github.com/topbase/topbase/internal/adapters/webhook"
	"github.com/topbase/topbase/internal/core"
)

// A Slack path, a Teams signature, and Chat query parameters are bearer
// credentials. Editors leave the address blank to retain the stored value.
func redactWebhook(hook core.WebhookEndpoint) core.WebhookEndpoint {
	hook.URL = ""
	hook.Secret = ""
	return hook
}

func (s *server) testWebhook(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	hooks, err := s.identity.Webhooks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取 Webhook 配置失败"})
		return
	}
	for _, hook := range hooks {
		if hook.ID != r.PathValue("id") {
			continue
		}
		if !hook.Enabled {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先启用此通道再发送测试通知"})
			return
		}
		if err := webhook.Send(r.Context(), hook, "Topbase 测试通知", "这是一条由管理员发送的通道测试消息。"); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Webhook 通道不存在"})
}

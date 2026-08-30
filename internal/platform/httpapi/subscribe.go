package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/adapters/feishu"
	"github.com/topbase/topbase/internal/core"
)

func (s *server) listGroups(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.identity.ListGroups()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) createGroup(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	group, err := s.identity.CreateGroup(input.Name)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, group)
}
func (s *server) replaceGroupMembers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		MemberIDs []string `json:"member_ids"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	group, err := s.identity.ReplaceGroupMembers(r.PathValue("id"), input.MemberIDs)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, group)
}

func (s *server) syncFeishuDepartments(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	groups, err := s.identity.SyncDirectory(r.Context(), feishu.EnvDirectory())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *server) syncIdentityProvider(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	providers, err := s.identity.IdentityProviders()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var provider *core.IdentityProvider
	for i := range providers {
		if providers[i].ID == r.PathValue("id") {
			provider = &providers[i]
			break
		}
	}
	if provider == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "enterprise account provider not found"})
		return
	}
	if !provider.Enabled || strings.TrimSpace(provider.ClientID) == "" || strings.TrimSpace(provider.ClientSecret) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先填写应用凭据、启用接入并保存配置"})
		return
	}
	var directory core.OrgDirectory
	switch provider.Type {
	case "feishu":
		directory = feishu.APIDirectory{AppID: provider.ClientID, AppSecret: provider.ClientSecret, HTTP: &http.Client{Timeout: 15 * time.Second}}
	case "dingtalk", "wecom":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "当前部署尚未安装该平台的组织同步适配器"})
		return
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "该账号平台不提供组织同步"})
		return
	}
	groups, err := s.identity.SyncDirectory(r.Context(), directory)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *server) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "view"); !ok {
		return
	}
	items, err := s.notify.ListSubscriptions(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) listAllSubscriptions(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.notify.ListSubscriptions("")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) adminMonitor(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	alerts, err := s.content.ListAlerts()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	schedules, err := s.warehouse.List()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	runs, err := s.warehouse.ListRuns("")
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	subs, err := s.notify.ListSubscriptions("")
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	writeJSON(w, http.StatusOK, map[string]any{
		"alerts": alerts, "schedules": schedules, "runs": runs, "subscriptions": subs,
		"runtime": map[string]any{
			"heap_alloc_bytes":   memory.HeapAlloc,
			"heap_inuse_bytes":   memory.HeapInuse,
			"heap_objects":       memory.HeapObjects,
			"sys_bytes":          memory.Sys,
			"gc_cycles":          memory.NumGC,
			"goroutines":         runtime.NumGoroutine(),
			"memory_limit_bytes": debug.SetMemoryLimit(-1),
		},
	})
}

func (s *server) listAdminNotifications(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.content.ListNotifications("")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) listIdentityProviders(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.identity.IdentityProviders()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	for i := range items {
		items[i].Configured = strings.TrimSpace(items[i].ClientID) != "" && strings.TrimSpace(items[i].ClientSecret) != ""
		items[i].ClientSecret = ""
	}
	writeJSON(w, 200, items)
}
func (s *server) saveIdentityProviders(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var items []core.IdentityProvider
	if !decodeJSON(w, r, &items) {
		return
	}
	existing, _ := s.identity.IdentityProviders()
	for i := range items {
		for _, old := range existing {
			if items[i].ID == old.ID && strings.TrimSpace(items[i].ClientSecret) == "" {
				items[i].ClientSecret = old.ClientSecret
			}
		}
		items[i].Configured = false
	}
	if err := s.identity.SaveIdentityProviders(items); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	for i := range items {
		items[i].Configured = strings.TrimSpace(items[i].ClientID) != "" && strings.TrimSpace(items[i].ClientSecret) != ""
		items[i].ClientSecret = ""
	}
	writeJSON(w, 200, items)
}
func (s *server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.identity.Webhooks()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	for i := range items {
		items[i].Secret = ""
	}
	writeJSON(w, 200, items)
}

func (s *server) listSubscriptionChannels(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	items, err := s.identity.Webhooks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	channels := make([]map[string]string, 0, len(items))
	for _, item := range items {
		if item.Enabled {
			channels = append(channels, map[string]string{"id": item.ID, "name": item.Name, "provider": item.Provider})
		}
	}
	writeJSON(w, http.StatusOK, channels)
}
func (s *server) saveWebhooks(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var items []core.WebhookEndpoint
	if !decodeJSON(w, r, &items) {
		return
	}
	existing, _ := s.identity.Webhooks()
	for i := range items {
		for _, old := range existing {
			if items[i].ID == old.ID && strings.TrimSpace(items[i].Secret) == "" {
				items[i].Secret = old.Secret
			}
		}
	}
	if err := s.identity.SaveWebhooks(items); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, items)
}

// deliverNotification resolves a stored endpoint only at delivery time, so a
// subscription never needs to contain a webhook URL or a secret.
func (s *server) deliverNotification(title, body, channel string) {
	if channel == "feishu" {
		_ = feishu.NotifyCard(title, body)
		return
	}
	if !strings.HasPrefix(channel, "webhook:") {
		return
	}
	id := strings.TrimPrefix(channel, "webhook:")
	hooks, err := s.identity.Webhooks()
	if err != nil {
		return
	}
	for _, hook := range hooks {
		if hook.ID != id || !hook.Enabled {
			continue
		}
		text := title + "\n" + body
		var payload any = map[string]string{"title": title, "body": body}
		if hook.Template != "" {
			rawTemplate := strings.NewReplacer("{{title}}", title, "{{body}}", body).Replace(hook.Template)
			if json.Unmarshal([]byte(rawTemplate), &payload) != nil {
				return
			}
		}
		switch hook.Provider {
		case "feishu":
			if hook.Template == "" {
				payload = map[string]any{"msg_type": "text", "content": map[string]string{"text": text}}
			}
		case "dingtalk":
			payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": text}}
		case "wecom":
			payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": text}}
		}
		raw, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, hook.URL, bytes.NewReader(raw))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if hook.Secret != "" {
			req.Header.Set("X-Topbase-Signature", hook.Secret)
		}
		resp, err := (&http.Client{Timeout: 8_000_000_000}).Do(req)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
		return
	}
}

func (s *server) createSubscription(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "edit")
	if !ok {
		return
	}
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	var input core.Subscription
	if !decodeJSON(w, r, &input) {
		return
	}
	hooks, err := s.identity.Webhooks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	validChannel := false
	for _, hook := range hooks {
		if hook.Enabled && input.Channel == "webhook:"+hook.ID {
			validChannel = true
			break
		}
	}
	if !validChannel {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先在通知与订阅中创建并启用 Webhook 通道"})
		return
	}
	input.DashboardID = r.PathValue("id")
	saved, err := s.notify.CreateSubscription(input, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *server) runSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := s.notify.ListSubscriptions("")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	dashboardID := ""
	for _, item := range subscriptions {
		if item.ID == r.PathValue("id") {
			dashboardID = item.DashboardID
			break
		}
	}
	if dashboardID == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "subscription not found"})
		return
	}
	if _, _, ok := s.requireDashboardAccess(w, r, dashboardID, "edit"); !ok {
		return
	}
	if _, ok := s.requireCapability(w, r, "data", "view"); !ok {
		return
	}
	note, err := s.notify.RunSubscription(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (s *server) updateSubscription(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	sub, err := s.notify.SetSubscriptionEnabled(r.PathValue("id"), input.Enabled)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (s *server) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if err := s.notify.DeleteSubscription(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

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
	writeJSON(w, http.StatusOK, map[string]any{"alerts": alerts, "schedules": schedules, "runs": runs, "subscriptions": subs})
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
	}
	if err := s.identity.SaveIdentityProviders(items); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
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
		switch hook.Provider {
		case "feishu":
			payload = map[string]any{"msg_type": "text", "content": map[string]string{"text": text}}
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

package core

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	slackWebhookPath = regexp.MustCompile(`^/services/[A-Za-z0-9_-]+/[A-Za-z0-9_-]+/[A-Za-z0-9_-]+$`)
	chatWebhookPath  = regexp.MustCompile(`^/v1/spaces/[A-Za-z0-9_-]+/messages$`)
	teamsTriggerPath = regexp.MustCompile(`^/powerautomate/automations/direct/(cu/[0-9]+/)?workflows/[A-Za-z0-9_-]+/triggers/[A-Za-z0-9_-]+/paths/invoke/?$`)
	azureTriggerPath = regexp.MustCompile(`^/workflows/[A-Za-z0-9_-]+/triggers/[A-Za-z0-9_-]+/paths/invoke/?$`)
)

// ValidateWebhookURL checks the address without echoing credentials or contacting
// the provider. Generic webhooks may point to an organization's own HTTP server.
func ValidateWebhookURL(provider, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("请填写完整的 HTTP 或 HTTPS Webhook 地址")
	}
	switch provider {
	case "slack", "teams", "google_chat":
		if u.Scheme != "https" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") {
			return errors.New("请粘贴平台生成的完整 HTTPS Webhook 地址")
		}
	default:
		return nil
	}
	host, path := strings.ToLower(u.Hostname()), u.EscapedPath()
	switch provider {
	case "slack":
		if (host != "hooks.slack.com" && host != "hooks.slack-gov.com") || !slackWebhookPath.MatchString(path) {
			return errors.New("请粘贴 Slack Incoming Webhooks 生成的频道地址")
		}
	case "teams":
		powerPlatform := strings.HasSuffix(host, ".environment.api.powerplatform.com") && teamsTriggerPath.MatchString(path)
		logicApps := strings.HasSuffix(host, ".logic.azure.com") && azureTriggerPath.MatchString(path)
		if (!powerPlatform && !logicApps) || u.Query().Get("sig") == "" {
			return errors.New("请从 Teams 工作流复制含 sig 参数的完整触发器地址，并将触发权限设为“任何人”")
		}
	case "google_chat":
		if host != "chat.googleapis.com" || !chatWebhookPath.MatchString(path) || u.Query().Get("key") == "" || u.Query().Get("token") == "" {
			return errors.New("请粘贴 Google Chat 聊天室生成的完整 Webhook 地址，包含 key 和 token")
		}
	}
	return nil
}

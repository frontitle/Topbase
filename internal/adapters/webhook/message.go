// Package webhook builds and delivers notifications to configured channels.
package webhook

import (
	"encoding/json"
	"errors"
	"html"
	"strings"

	"github.com/topbase/topbase/internal/core"
)

type object = map[string]any

func Render(hook core.WebhookEndpoint, title, body string) ([]byte, error) {
	return render(hook, title, body, "")
}

func render(hook core.WebhookEndpoint, title, body, dashboardURL string) ([]byte, error) {
	style := hook.TemplateType
	if style == "custom" || usesStoredTemplate(hook) {
		return renderJSON(hook.Template, title, body, dashboardURL)
	}
	if style != "" && style != "text" && style != "post" && style != "card" {
		return nil, errors.New("不支持的 Webhook 消息样式")
	}
	variables := strings.NewReplacer("{{title}}", title, "{{body}}", body, "{{dashboard_url}}", dashboardURL)
	if hook.TemplateTitle != "" {
		title = variables.Replace(hook.TemplateTitle)
	}
	if hook.TemplateBody != "" {
		body = variables.Replace(hook.TemplateBody)
	}
	var payload object
	switch hook.Provider {
	case "slack":
		payload = slackMessage(style, title, body, dashboardURL)
	case "teams":
		payload = teamsMessage(style, title, body, dashboardURL)
	case "google_chat":
		payload = googleChatMessage(style, title, body, dashboardURL)
	case "feishu":
		payload = feishuMessage(style, title, body)
	case "dingtalk", "wecom":
		if (style == "card" || style == "post") && hook.Template == "" {
			markdown := object{"content": "**" + title + "**\n" + body}
			if hook.Provider == "dingtalk" {
				markdown = object{"title": title, "text": "## " + title + "\n" + body}
			}
			payload = object{"msgtype": "markdown", "markdown": markdown}
		} else {
			payload = object{"msgtype": "text", "text": object{"content": title + "\n" + body}}
		}
	default:
		payload = object{"title": title, "body": body}
	}
	return json.Marshal(payload)
}

// Older clients stored an already-generated preset as JSON. DingTalk and
// WeCom used a text envelope even when those clients supplied Feishu JSON.
func usesStoredTemplate(hook core.WebhookEndpoint) bool {
	return hook.Template != "" && hook.Provider != "dingtalk" && hook.Provider != "wecom"
}

func renderJSON(template, title, body, url string) ([]byte, error) {
	template = strings.TrimSpace(template)
	if !strings.HasPrefix(template, "{") || !json.Valid([]byte(template)) {
		return nil, errors.New("Webhook 模板需要是有效的 JSON 对象")
	}
	// Insert JSON string contents so quotes, line breaks and literal
	// placeholders in dashboard data survive a round trip.
	escaped := func(value string) string {
		encoded, _ := json.Marshal(value)
		return string(encoded[1 : len(encoded)-1])
	}
	replacer := strings.NewReplacer("{{title}}", escaped(title), "{{body}}", escaped(body), "{{dashboard_url}}", escaped(url))
	return []byte(replacer.Replace(template)), nil
}

var slackEntities = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func slackMessage(style, title, body, url string) object {
	summary := slackEntities.Replace(title + "\n" + body)
	link := ""
	if url != "" {
		link = "\n<" + slackEntities.Replace(url) + "|打开仪表盘>"
	}
	payload := object{"text": summary + link, "unfurl_links": false, "unfurl_media": false}
	if style == "" || style == "text" {
		return payload
	}
	if style == "post" {
		text := "*" + slackEntities.Replace(title) + "*\n" + slackEntities.Replace(body) + link
		payload["blocks"] = []object{{"type": "section", "text": object{"type": "mrkdwn", "text": text, "verbatim": true}}}
		return payload
	}
	// Slack caps a Block Kit header at 150 Unicode characters.
	header := []rune(title)
	if len(header) > 150 {
		header = append(header[:149], '…')
	}
	blocks := []object{{"type": "header", "text": object{"type": "plain_text", "text": string(header)}}}
	if body != "" {
		blocks = append(blocks, object{"type": "section", "text": object{"type": "plain_text", "text": body}})
	}
	if url != "" {
		button := object{"type": "button", "action_id": "open_dashboard", "text": object{"type": "plain_text", "text": "打开仪表盘"}, "url": url}
		blocks = append(blocks, object{"type": "actions", "elements": []object{button}})
	}
	payload["blocks"] = blocks
	return payload
}

func teamsMessage(style, title, body, url string) object {
	// Teams Workflows expects an attachment even for a text notification.
	text := func(value string) object { return object{"type": "TextBlock", "text": value, "wrap": true} }
	blocks := []object{text(title + "\n" + body)}
	if style == "post" || style == "card" {
		heading := text(title)
		heading["weight"] = "Bolder"
		if style == "card" {
			heading["size"] = "Medium"
		}
		blocks = []object{heading, text(body)}
	}
	card := object{"type": "AdaptiveCard", "$schema": "http://adaptivecards.io/schemas/adaptive-card.json", "version": "1.2"}
	if url != "" {
		if style == "card" {
			card["actions"] = []object{{"type": "Action.OpenUrl", "title": "打开仪表盘", "url": url}}
		} else {
			blocks = append(blocks, text("[打开仪表盘]("+url+")"))
		}
	}
	card["body"] = blocks
	attachment := object{"contentType": "application/vnd.microsoft.card.adaptive", "contentUrl": nil, "content": card}
	return object{"type": "message", "attachments": []object{attachment}}
}

func googleChatMessage(style, title, body, url string) object {
	if style == "card" {
		paragraph := strings.ReplaceAll(html.EscapeString(body), "\n", "<br>")
		widgets := []object{{"textParagraph": object{"text": paragraph}}}
		if url != "" {
			button := object{"text": "打开仪表盘", "onClick": object{"openLink": object{"url": url}}}
			widgets = append(widgets, object{"buttonList": object{"buttons": []object{button}}})
		}
		card := object{"header": object{"title": title}, "sections": []object{{"widgets": widgets}}}
		return object{"cardsV2": []object{{"cardId": "topbase_notification", "card": card}}}
	}
	if style == "post" {
		title = "*" + title + "*"
	}
	content := title + "\n" + body
	if url != "" {
		content += "\n<" + url + "|打开仪表盘>"
	}
	return object{"text": content}
}

func feishuMessage(style, title, body string) object {
	switch style {
	case "post":
		post := object{"title": title, "content": [][]object{{{"tag": "text", "text": body}}}}
		return object{"msg_type": "post", "content": object{"post": object{"zh-CN": post}}}
	case "card":
		header := object{"template": "blue", "title": object{"tag": "plain_text", "content": title}}
		elements := []object{{"tag": "div", "text": object{"tag": "lark_md", "content": body}}}
		return object{"msg_type": "interactive", "card": object{"header": header, "elements": elements}}
	default:
		return object{"msg_type": "text", "content": object{"text": title + "\n" + body}}
	}
}

package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func NotifyCard(title, body string) error {
	webhook := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhook == "" {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"msg_type": "interactive",
		"card": map[string]any{
			"header": map[string]any{"title": map[string]string{"tag": "plain_text", "content": title}, "template": "turquoise"},
			"elements": []map[string]any{
				{"tag": "div", "text": map[string]string{"tag": "lark_md", "content": body}},
			},
		},
	})
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(webhook, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook status %d", resp.StatusCode)
	}
	return nil
}

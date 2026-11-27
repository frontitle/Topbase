package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/frontitle/Topbase/internal/core"
)

var client = &http.Client{
	Timeout: 8 * time.Second,
	// A redirect could forward the payload and its credentials to another host.
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func Send(ctx context.Context, hook core.WebhookEndpoint, title, body string) error {
	return SendWithDashboard(ctx, hook, title, body, "")
}

func SendWithDashboard(ctx context.Context, hook core.WebhookEndpoint, title, body, dashboardURL string) error {
	if err := core.ValidateWebhookURL(hook.Provider, hook.URL); err != nil {
		return err
	}
	payload, err := render(hook, title, body, dashboardURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(payload))
	if err != nil {
		return errors.New("Webhook 地址格式错误")
	}
	req.Header.Set("Content-Type", "application/json")
	if hook.Secret != "" {
		req.Header.Set("X-Topbase-Signature", hook.Secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		// net/http errors include the full URL, which is itself a credential.
		return errors.New("Webhook 请求失败，请检查网络和接收端地址")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Webhook 接收端返回 HTTP %d", resp.StatusCode)
	}
	if hook.Provider == "feishu" || hook.Provider == "dingtalk" || hook.Provider == "wecom" {
		var result struct {
			Code    int `json:"code"`
			ErrCode int `json:"errcode"`
			Status  int `json:"StatusCode"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&result); err == nil && (result.Code != 0 || result.ErrCode != 0 || result.Status != 0) {
			return errors.New("Webhook 平台拒绝了消息，请检查群机器人配置")
		}
	}
	return nil
}

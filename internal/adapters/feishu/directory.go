package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/topbase/topbase/internal/core"
)

type APIDirectory struct {
	AppID     string
	AppSecret string
	BaseURL   string
	HTTP      *http.Client
}

func EnvDirectory() core.OrgDirectory {
	appID := os.Getenv("FEISHU_APP_ID")
	secret := os.Getenv("FEISHU_APP_SECRET")
	if appID == "" || secret == "" {
		return nil
	}
	return APIDirectory{AppID: appID, AppSecret: secret, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (d APIDirectory) ListUnits(ctx context.Context) ([]core.OrgUnit, error) {
	token, err := d.tenantToken(ctx)
	if err != nil {
		return nil, err
	}
	base := d.BaseURL
	if base == "" {
		base = "https://open.feishu.cn"
	}
	u, _ := url.Parse(base + "/open-apis/contact/v3/departments/0/children")
	q := u.Query()
	q.Set("fetch_child", "true")
	q.Set("page_size", "50")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := d.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				DepartmentID string `json:"department_id"`
				Name         string `json:"name"`
				OpenID       string `json:"open_department_id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu departments: %s", payload.Msg)
	}
	units := []core.OrgUnit{}
	for _, item := range payload.Data.Items {
		id := item.OpenID
		if id == "" {
			id = item.DepartmentID
		}
		units = append(units, core.OrgUnit{ExternalID: id, Name: item.Name})
	}
	return units, nil
}

func (d APIDirectory) tenantToken(ctx context.Context) (string, error) {
	base := d.BaseURL
	if base == "" {
		base = "https://open.feishu.cn"
	}
	raw, _ := json.Marshal(map[string]string{"app_id": d.AppID, "app_secret": d.AppSecret})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var payload struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Code != 0 || payload.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu token: %s", payload.Msg)
	}
	return payload.TenantAccessToken, nil
}

func (d APIDirectory) client() *http.Client {
	if d.HTTP != nil {
		return d.HTTP
	}
	return http.DefaultClient
}

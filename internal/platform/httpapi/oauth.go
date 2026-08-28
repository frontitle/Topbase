package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

const (
	oauthStateCookie = "topbase_oauth_state"
	oauthBindCookie  = "topbase_oauth_bind"
)

func (s *server) oauthLogin(w http.ResponseWriter, r *http.Request) {
	provider, ok := s.oauthProvider(r.PathValue("provider"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	state, err := newOAuthState()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: state, Path: "/auth/oauth/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, MaxAge: 600})
	if r.URL.Query().Get("intent") == "bind" {
		user, signedIn := s.currentSessionUser(r)
		if !signedIn {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "sign in before linking an account"})
			return
		}
		http.SetCookie(w, &http.Cookie{Name: oauthBindCookie, Value: state + ":" + user.ID, Path: "/auth/oauth/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, MaxAge: 600})
	} else {
		clearOAuthBindCookie(w, r)
	}
	redirectURI := externalURL(r, "/auth/oauth/"+provider.ID+"/callback")
	values := url.Values{"state": {state}, "redirect_uri": {redirectURI}}
	switch provider.Type {
	case "google":
		values.Set("client_id", provider.ClientID)
		values.Set("response_type", "code")
		values.Set("scope", "openid email profile")
		redirectTo(w, r, provider.AuthorizationURL, "https://accounts.google.com/o/oauth2/v2/auth", values)
	case "wechat":
		values.Set("appid", provider.ClientID)
		values.Set("response_type", "code")
		values.Set("scope", "snsapi_login")
		redirectTo(w, r, provider.AuthorizationURL, "https://open.weixin.qq.com/connect/qrconnect", values)
	default:
		writeJSON(w, 400, map[string]string{"error": "this provider does not support browser login yet"})
	}
}

func (s *server) oauthCallback(w http.ResponseWriter, r *http.Request) {
	provider, ok := s.oauthProvider(r.PathValue("provider"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || r.URL.Query().Get("state") == "" || cookie.Value != r.URL.Query().Get("state") {
		writeJSON(w, 400, map[string]string{"error": "invalid OAuth state"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", Path: "/auth/oauth/", MaxAge: -1})
	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, 400, map[string]string{"error": "OAuth authorization was cancelled or failed"})
		return
	}
	subject, email, err := s.oauthProfile(r, provider, code)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": err.Error()})
		return
	}
	if bindCookie, cookieErr := r.Cookie(oauthBindCookie); cookieErr == nil {
		bindState, bindUserID, found := strings.Cut(bindCookie.Value, ":")
		current, signedIn := s.currentSessionUser(r)
		clearOAuthBindCookie(w, r)
		if found && bindState == r.URL.Query().Get("state") && signedIn && current.ID == bindUserID {
			if err := s.identity.BindExternalIdentity(core.ExternalIdentityLink{ProviderID: provider.ID, Subject: subject, UserID: current.ID}); err != nil {
				http.Redirect(w, r, "/account/?binding=failed", http.StatusFound)
				return
			}
			http.Redirect(w, r, "/account/?binding=success", http.StatusFound)
			return
		}
	}
	session, _, err := s.identity.LoginExternalIdentity(provider.ID, subject, email)
	if err != nil {
		writeJSON(w, 403, map[string]string{"error": err.Error()})
		return
	}
	setSessionCookie(w, session.ID, session.ExpiresAt)
	http.Redirect(w, r, "/", http.StatusFound)
}

func clearOAuthBindCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: oauthBindCookie, Value: "", Path: "/auth/oauth/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, MaxAge: -1})
}

func (s *server) oauthProvider(id string) (core.IdentityProvider, bool) {
	providers, err := s.identity.IdentityProviders()
	if err != nil {
		return core.IdentityProvider{}, false
	}
	for _, provider := range providers {
		if provider.ID == id && provider.Enabled && provider.ClientID != "" && provider.ClientSecret != "" {
			return provider, true
		}
	}
	return core.IdentityProvider{}, false
}

func (s *server) oauthProfile(r *http.Request, provider core.IdentityProvider, code string) (string, string, error) {
	redirectURI := externalURL(r, "/auth/oauth/"+provider.ID+"/callback")
	switch provider.Type {
	case "google":
		form := url.Values{"code": {code}, "client_id": {provider.ClientID}, "client_secret": {provider.ClientSecret}, "redirect_uri": {redirectURI}, "grant_type": {"authorization_code"}}
		var token struct {
			AccessToken string `json:"access_token"`
		}
		if err := oauthJSON(http.MethodPost, fallback(provider.TokenURL, "https://oauth2.googleapis.com/token"), form, &token); err != nil {
			return "", "", err
		}
		var profile struct {
			ID            string `json:"id"`
			Email         string `json:"email"`
			VerifiedEmail bool   `json:"verified_email"`
		}
		if err := oauthGetJSON("https://openidconnect.googleapis.com/v1/userinfo", token.AccessToken, &profile); err != nil {
			return "", "", err
		}
		if profile.ID == "" || profile.Email == "" || !profile.VerifiedEmail {
			return "", "", fmt.Errorf("Google did not return a verified email")
		}
		return profile.ID, profile.Email, nil
	case "wechat":
		endpoint := fallback(provider.TokenURL, "https://api.weixin.qq.com/sns/oauth2/access_token")
		values := url.Values{"appid": {provider.ClientID}, "secret": {provider.ClientSecret}, "code": {code}, "grant_type": {"authorization_code"}}
		var token struct {
			AccessToken string `json:"access_token"`
			OpenID      string `json:"openid"`
			ErrMsg      string `json:"errmsg"`
		}
		if err := oauthJSON(http.MethodGet, endpoint, values, &token); err != nil {
			return "", "", err
		}
		if token.OpenID == "" {
			return "", "", fmt.Errorf("WeChat authorization failed: %s", token.ErrMsg)
		}
		return token.OpenID, "", nil
	default:
		return "", "", fmt.Errorf("unsupported OAuth provider")
	}
}

func oauthJSON(method, endpoint string, values url.Values, target any) error {
	var req *http.Request
	var err error
	if method == http.MethodPost {
		req, err = http.NewRequest(method, endpoint, strings.NewReader(values.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		req, err = http.NewRequest(method, endpoint+"?"+values.Encode(), nil)
	}
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("OAuth token request failed: %s", strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target)
}
func oauthGetJSON(endpoint, accessToken string, target any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("OAuth profile request failed")
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target)
}
func redirectTo(w http.ResponseWriter, r *http.Request, custom, standard string, values url.Values) {
	target := fallback(custom, standard)
	http.Redirect(w, r, target+"?"+values.Encode(), http.StatusFound)
}
func fallback(value, standard string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return standard
}
func externalURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}
func newOAuthState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frontitle/Topbase/internal/core"
)

const dashboardShareTokenHeader = "X-Topbase-Share-Token"

func (s *server) saveDashboardPublicLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	user, _, ok := s.requireDashboardAccess(w, r, r.PathValue("id"), "manage")
	if !ok {
		return
	}
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.PublicSharingEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "工作区尚未允许公开分享"})
		return
	}
	var input struct {
		ExpiresAt json.RawMessage `json:"expires_at"`
		Password  json.RawMessage `json:"password"`
	}
	publishing := r.Method == http.MethodPost
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&input)
	if err != nil && !(publishing && errors.Is(err, io.EOF)) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "分享设置无效"})
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "分享设置无效"})
		return
	}
	update := core.DashboardPublicLinkUpdate{SetExpiry: input.ExpiresAt != nil}
	if update.SetExpiry && json.Unmarshal(input.ExpiresAt, &update.ExpiresAt) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请输入有效的到期时间"})
		return
	}
	if input.Password != nil {
		var password string // Both null and "" remove the password.
		if json.Unmarshal(input.Password, &password) != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "访问密码格式无效"})
			return
		}
		update.Password = &password
	}
	var board core.Dashboard
	if publishing {
		board, err = s.content.PublishDashboard(r.PathValue("id"), user.ID, update)
	} else {
		board, err = s.content.UpdateDashboardPublicLink(r.PathValue("id"), user.ID, update)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s.publicLinkPayload(r, board))
}

func publicLinkUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"code": "share_unavailable", "error": "分享链接已失效或已关闭"})
}

func (s *server) publicShareEnabled(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	settings, err := s.identity.AdminSettings()
	if err != nil || !settings.PublicSharingEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"code": "share_unavailable", "error": "工作区已暂停公开分享"})
		return false
	}
	if r.URL.Query().Get("embed") == "1" && !settings.EmbeddingEnabled {
		publicLinkUnavailable(w)
		return false
	}
	return true
}

func (s *server) publicDashboardAccess(w http.ResponseWriter, r *http.Request) (core.DashboardPublicAccess, bool) {
	if !s.publicShareEnabled(w, r) {
		return core.DashboardPublicAccess{}, false
	}
	policy, err := s.store.PublicDashboardAccess(r.Context(), r.PathValue("uuid"))
	if err != nil || (r.URL.Query().Get("embed") == "1" && !policy.EmbedEnabled) {
		publicLinkUnavailable(w)
		return core.DashboardPublicAccess{}, false
	}
	return policy, true
}

func (s *server) acceptShareGrant(w http.ResponseWriter, r *http.Request, hash string, expiry *time.Time) bool {
	if hash == "" || s.shareAccess.validGrant(r.Header.Get(dashboardShareTokenHeader), r.PathValue("uuid"), hash, expiry, time.Now()) {
		return true
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "password_required", "error": "请输入访问密码"})
	return false
}

// Check the lightweight link record before loading cards or starting a query.
func (s *server) authorizePublicPolicy(w http.ResponseWriter, r *http.Request) bool {
	policy, ok := s.publicDashboardAccess(w, r)
	return ok && s.acceptShareGrant(w, r, policy.PasswordHash, policy.ExpiresAt)
}

func (s *server) authorizePublicDashboard(w http.ResponseWriter, r *http.Request) (core.Dashboard, bool) {
	if !s.authorizePublicPolicy(w, r) {
		return core.Dashboard{}, false
	}
	board, err := s.content.GetDashboardByPublicUUID(r.PathValue("uuid"))
	if err != nil || (r.URL.Query().Get("embed") == "1" && !board.PublicEmbedEnabled) {
		publicLinkUnavailable(w)
		return core.Dashboard{}, false
	}
	// Check the policy in the same snapshot as the returned contents as well.
	if !s.acceptShareGrant(w, r, board.PublicPasswordHash, board.PublicExpiresAt) {
		return core.Dashboard{}, false
	}
	return board, true
}

func (s *server) verifyDashboardPassword(w http.ResponseWriter, r *http.Request) {
	policy, ok := s.publicDashboardAccess(w, r)
	if !ok {
		return
	}
	uuid := r.PathValue("uuid")
	if policy.PasswordHash == "" {
		writeJSON(w, http.StatusOK, map[string]string{"token": ""})
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !decodeJSON(w, r, &input) {
		return
	}
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}
	key := uuid + "/" + peer
	now := time.Now()
	retry := s.shareAccess.reserve(key, now)
	if retry != 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"code": "rate_limited", "error": "验证过于频繁，请稍后再试"})
		return
	}
	defer func() { <-s.shareAccess.workers }()
	if !core.VerifyPassword(policy.PasswordHash, input.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "invalid_password", "error": "访问密码不正确"})
		return
	}
	s.shareAccess.releaseAttempt(key)
	now = time.Now()
	if policy.ExpiresAt != nil && !now.Before(*policy.ExpiresAt) {
		publicLinkUnavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": s.shareAccess.grant(uuid, policy.PasswordHash, policy.ExpiresAt, now)})
}

type shareAttemptWindow struct {
	until time.Time
	count int
}
type dashboardShareAccess struct {
	key         []byte
	workers     chan struct{}
	mu          sync.Mutex
	attempts    map[string]shareAttemptWindow
	nextCleanup time.Time
}

func newDashboardShareAccess(masterKey []byte) *dashboardShareAccess {
	derive := hmac.New(sha256.New, masterKey)
	derive.Write([]byte("topbase/dashboard-share/grant/v1"))
	return &dashboardShareAccess{key: derive.Sum(nil), workers: make(chan struct{}, 2), attempts: make(map[string]shareAttemptWindow)}
}

// Reserve an attempt before the memory-intensive password check, including for
// simultaneous requests. Limit live Argon2 work across all links on this node.
func (a *dashboardShareAccess) reserve(key string, now time.Time) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !now.Before(a.nextCleanup) {
		for k, entry := range a.attempts {
			if !now.Before(entry.until) {
				delete(a.attempts, k)
			}
		}
		a.nextCleanup = now.Add(time.Minute)
	}
	window, exists := a.attempts[key]
	if !exists && len(a.attempts) >= 4096 {
		return 60
	}
	if !now.Before(window.until) {
		window = shareAttemptWindow{until: now.Add(time.Minute)}
	}
	if window.count >= 6 {
		return 1 + int(window.until.Sub(now)/time.Second)
	}
	select {
	case a.workers <- struct{}{}:
		window.count++
		a.attempts[key] = window
		return 0
	default:
		return 1
	}
}
func (a *dashboardShareAccess) releaseAttempt(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	window := a.attempts[key]
	if window.count <= 1 {
		delete(a.attempts, key)
	} else {
		window.count--
		a.attempts[key] = window
	}
}

// Bind the grant to the currently stored policy. Rotating a password, changing
// the deadline, or republishing under a new UUID invalidates earlier grants.
func (a *dashboardShareAccess) signature(uuid, hash string, expiresAt *time.Time, deadline string) []byte {
	mac := hmac.New(sha256.New, a.key)
	mac.Write([]byte(uuid + "\x00" + hash + "\x00" + deadline + "\x00"))
	if expiresAt != nil {
		mac.Write([]byte(expiresAt.UTC().Format(time.RFC3339Nano)))
	}
	return mac.Sum(nil)
}
func (a *dashboardShareAccess) grant(uuid, hash string, expiresAt *time.Time, now time.Time) string {
	until := now.Add(12 * time.Hour)
	if expiresAt != nil && expiresAt.Before(until) {
		until = *expiresAt
	}
	deadline := strconv.FormatInt(until.UnixMilli(), 10)
	return deadline + "." + base64.RawURLEncoding.EncodeToString(a.signature(uuid, hash, expiresAt, deadline))
}
func (a *dashboardShareAccess) validGrant(token, uuid, hash string, expiresAt *time.Time, now time.Time) bool {
	if len(token) > 96 {
		return false
	}
	deadline, signature, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	unix, err := strconv.ParseInt(deadline, 10, 64)
	if err != nil || unix <= now.UnixMilli() {
		return false
	}
	mac, err := base64.RawURLEncoding.Strict().DecodeString(signature)
	return err == nil && hmac.Equal(a.signature(uuid, hash, expiresAt, deadline), mac)
}

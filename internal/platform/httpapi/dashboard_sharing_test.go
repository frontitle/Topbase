package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDashboardSharingPolicyAndRevocation(t *testing.T) {
	handler := testServer(t)
	admin := adminSession(t, handler)
	call := func(cookie *http.Cookie, method, url, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, url, strings.NewReader(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s %s: %d, want %d: %s", method, url, rec.Code, status, rec.Body.String())
		}
		return rec
	}
	fields := func(rec *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var data map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	board := fields(call(admin, "POST", "/api/dashboards", `{"name":"Sharing test"}`, 201))
	url := "/api/dashboards/" + board["id"].(string)
	policy := func(cookie *http.Cookie) map[string]any {
		return fields(call(cookie, "GET", url+"?view=sharing", "", 200))
	}
	initial := policy(admin)
	if initial["allow_public"] != false || initial["can_manage"] != true || initial["is_admin"] != true || initial["allow_embed"] != false {
		t.Fatal(initial)
	}
	call(admin, "POST", url+"/public-link", "", 403)
	unavailable := func(path string) string {
		t.Helper()
		rec := call(nil, "GET", path, "", http.StatusNotFound)
		body := rec.Body.String()
		if !strings.Contains(body, "分享链接不可用") || strings.Contains(body, "board-canvas") ||
			rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("expected a non-cacheable unavailable page")
		}
		return body
	}
	disabledPage := unavailable("/public/dashboard/missing/")
	user := fields(call(admin, "POST", "/api/users", `{"name":"Viewer","email":"viewer@example.com","password":"secret123"}`, 201))
	login := call(nil, "POST", "/api/session", `{"email":"viewer@example.com","password":"secret123"}`, 200)
	viewer := cookieNamed(t, login.Result().Cookies(), sessionCookie)
	call(viewer, "GET", url+"?view=sharing", "", 403)
	call(admin, "PUT", "/api/collections/"+board["collection_id"].(string)+"/shares", `{"user_ids":["`+user["id"].(string)+`"]}`, 204)
	call(admin, "PUT", "/api/admin/settings", `{"site_name":"QA","public_sharing_enabled":true,"embedding_enabled":true}`, 200)
	readOnly := policy(viewer)
	if readOnly["allow_public"] != true || readOnly["can_manage"] != false || readOnly["is_admin"] != false {
		t.Fatal(readOnly)
	}
	call(viewer, "POST", url+"/public-link", "", 403)
	published := fields(call(admin, "POST", url+"/public-link", "", 200))
	uuid := published["public_uuid"].(string)
	publicPath := "/public/dashboard/" + uuid + "/"
	embedPath := "/embed/dashboard/" + uuid + "/"
	page := call(nil, "GET", publicPath, "", 200)
	if !strings.Contains(page.Body.String(), "board-canvas") || page.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected the active public dashboard")
	}
	if unavailable(embedPath) != disabledPage || unavailable("/public/dashboard/unknown/") != disabledPage {
		t.Fatal("unavailable pages disclosed different link states")
	}
	call(viewer, "PUT", url+"/embedding", `{"enabled":true}`, 403)
	call(admin, "PUT", url+"/embedding", `{"enabled":true}`, 200)
	call(nil, "GET", embedPath, "", 200)
	call(admin, "PUT", "/api/admin/settings", `{"site_name":"QA","public_sharing_enabled":false,"embedding_enabled":false}`, 200)
	paused := policy(admin)
	if unavailable(publicPath) != disabledPage || unavailable(embedPath) != disabledPage {
		t.Fatal("paused sharing should show the unavailable page")
	}
	if paused["public_uuid"] == "" || paused["allow_public"] != false || paused["public_embed_enabled"] != true {
		t.Fatal(paused)
	}
	call(admin, "PUT", url+"/embedding", `{"enabled":false}`, 200)
	call(admin, "PUT", url+"/embedding", `{"enabled":true}`, 403)
	call(viewer, "DELETE", url+"/public-link", "", 403)
	call(admin, "DELETE", url+"/public-link", "", 200)
	revoked := policy(admin)
	if revoked["public_uuid"] != "" || revoked["public_embed_enabled"] != false {
		t.Fatal(revoked)
	}
	call(admin, "PUT", "/api/admin/settings", `{"site_name":"QA","public_sharing_enabled":true,"embedding_enabled":true}`, 200)
	for _, path := range []string{publicPath, publicPath, embedPath} {
		if unavailable(path) != disabledPage {
			t.Fatal("revoked links should stay unavailable on refresh")
		}
	}
}

func TestPublicDashboardPasswordAndExpiry(t *testing.T) {
	handler := testServer(t)
	admin := adminSession(t, handler)
	call := func(cookie *http.Cookie, method, path, body, token string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if token != "" {
			r.Header.Set(dashboardShareTokenHeader, token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, want, w.Body.String())
		}
		if strings.HasPrefix(path, "/api/public/") && w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("public response is cacheable")
		}
		if strings.Contains(w.Body.String(), "argon2id$") || strings.Contains(w.Body.String(), "私有访问口令123") {
			t.Fatal("share credentials in response")
		}
		return w
	}
	asMap := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var m map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	call(admin, "PUT", "/api/admin/settings", `{"site_name":"QA","public_sharing_enabled":true,"embedding_enabled":true}`, "", 200)
	board := asMap(call(admin, "POST", "/api/dashboards", `{"name":"protected-board","cards":[{"type":"text","body":"only visible after unlock"}]}`, "", 201))
	manage := "/api/dashboards/" + board["id"].(string) + "/public-link"
	draft := asMap(call(admin, "GET", strings.TrimSuffix(manage, "/public-link")+"?view=sharing", "", "", 200))
	if draft["expires_at"] != nil || draft["password_required"] != false {
		t.Fatal(draft)
	}
	call(admin, "PUT", manage, `{"password":"私有访问口令123"}`, "", 400)
	call(admin, "POST", manage, `{"expires_at":"2020-12-01T00:00:00Z"}`, "", 400)
	call(admin, "POST", manage, `{"password":"tiny"}`, "", 400)
	config := asMap(call(admin, "POST", manage, `{"password":"私有访问口令123","expires_at":"2099-05-01T00:00:00Z"}`, "", 200))
	if config["expires_at"] != "2099-05-01T00:00:00Z" || config["password_required"] != true {
		t.Fatal(config)
	}
	uuid := config["public_uuid"].(string)
	path := "/api/public/dashboard/" + uuid
	card := path + "/cards/" + board["cards"].([]any)[0].(map[string]any)["id"].(string) + "/dataset"
	locked := call(nil, "GET", path, "", "", 401)
	if asMap(locked)["code"] != "password_required" || strings.Contains(locked.Body.String(), "only visible after unlock") {
		t.Fatal("expected a password prompt without board data")
	}
	call(nil, "POST", card, "{}", "", 401)
	call(nil, "POST", path+"/verify", `{"password":"wrong"}`, "", 401)
	call(nil, "POST", path+"/verify?embed=1", `{"password":"私有访问口令123"}`, "", 404)
	verify := func(password string) string {
		t.Helper()
		return asMap(call(nil, "POST", path+"/verify", `{"password":"`+password+`"}`, "", 200))["token"].(string)
	}
	token := verify("私有访问口令123")
	if token == "" {
		t.Fatal("missing grant")
	}
	got := asMap(call(nil, "GET", path, "", token, 200))["dashboard"].(map[string]any)
	if got["name"] != "protected-board" || got["public_expires_at"] != config["expires_at"] {
		t.Fatal("missing authorized board or expiry")
	}
	call(nil, "POST", card, "{}", token, 400) // Auth succeeds; text cards have no dataset.
	call(nil, "GET", path+"?embed=1", "", token, 404)
	call(admin, "PUT", strings.TrimSuffix(manage, "/public-link")+"/embedding", `{"enabled":true}`, "", 200)
	call(nil, "GET", path+"?embed=1", "", token, 200)
	call(nil, "POST", card+"?embed=1", "{}", "", 401)
	call(admin, "PUT", manage, `{"password":"replacement-123"}`, "", 200)
	call(nil, "GET", path, "", token, 401)
	call(nil, "POST", path+"/verify", `{"password":"私有访问口令123"}`, "", 401)
	token = verify("replacement-123")
	config = asMap(call(admin, "PUT", manage, `{"expires_at":null}`, "", 200))
	if config["expires_at"] != nil || config["password_required"] != true || config["public_uuid"] != uuid {
		t.Fatal(config)
	}
	call(nil, "GET", path, "", token, 401)
	verify("replacement-123")
	config = asMap(call(admin, "PUT", manage, `{"password":null}`, "", 200))
	if config["password_required"] != false || config["public_uuid"] != uuid {
		t.Fatal(config)
	}
	call(nil, "GET", path, "", "", 200)
	// Exercise the clock edge without mutating the database behind the service.
	deadline := time.Now().Add(500 * time.Millisecond)
	call(admin, "PUT", manage, `{"expires_at":"`+deadline.UTC().Format(time.RFC3339Nano)+`"}`, "", 200)
	time.Sleep(time.Until(deadline))
	for _, page := range []string{"/public/dashboard/" + uuid + "/", "/embed/dashboard/" + uuid + "/"} {
		rec := call(nil, "GET", page, "", "", 404)
		if !strings.Contains(rec.Body.String(), "分享链接不可用") {
			t.Fatal("expired page should render unavailability")
		}
	}
	call(nil, "GET", path, "", token, 404)
	call(nil, "POST", card+"?embed=1", "{}", token, 404)
	call(nil, "POST", path+"/verify", `{"password":"replacement-123"}`, "", 404)
	config = asMap(call(admin, "PUT", manage, `{"expires_at":null}`, "", 200))
	if config["public_uuid"] != uuid {
		t.Fatal("renewal changed the public link")
	}
	call(nil, "GET", path, "", "", 200)
	call(admin, "DELETE", manage, "", "", 200)
	call(nil, "GET", path, "", token, 404)
	call(nil, "POST", card, "{}", token, 404)
	call(nil, "POST", path+"/verify", `{"password":"replacement-123"}`, "", 404)
}

func TestShareGrantsAndPasswordThrottling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	access := newDashboardShareAccess([]byte("test-instance-key"))
	second := newDashboardShareAccess([]byte("test-instance-key"))
	expires := now.Add(time.Minute)
	grant := access.grant("uuid", "current", &expires, now)
	if !second.validGrant(grant, "uuid", "current", &expires, now) {
		t.Fatal("shared application key must validate grant")
	}
	if second.validGrant(grant, "other", "current", &expires, now) || second.validGrant(grant, "uuid", "old", &expires, now) || second.validGrant(grant, "uuid", "current", &expires, expires) || second.validGrant(grant+"x", "uuid", "current", &expires, now) {
		t.Fatal("grant had wrong scope or lifetime")
	}
	permanent := access.grant("uuid", "current", nil, now)
	if second.validGrant(permanent, "uuid", "current", nil, now.Add(12*time.Hour)) {
		t.Fatal("permanent link must use bounded grants")
	}
	for i := 0; i < 6; i++ {
		if access.reserve("uuid/ip", now) != 0 {
			t.Fatal("valid attempt was throttled")
		}
		<-access.workers
	}
	if access.reserve("uuid/ip", now) == 0 || access.reserve("uuid/other-ip", now) != 0 {
		t.Fatal("unexpected retry window")
	}
	<-access.workers
	if access.reserve("uuid/ip", now.Add(time.Minute)) != 0 {
		t.Fatal("expected retry after expiry")
	}
	<-access.workers
}

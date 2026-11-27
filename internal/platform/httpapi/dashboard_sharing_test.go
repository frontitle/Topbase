package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

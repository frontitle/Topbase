package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/topbase/topbase/internal/core"
)

func testServer(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("TOPBASE_DATA_DIR", t.TempDir())
	t.Setenv("TOPBASE_CRON", "off")
	handler := NewServer()
	if closer, ok := handler.(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closer.Close() })
	}
	return handler
}

func adminSession(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Result().Cookies()[0]
}

func setupCookies(t *testing.T, handler http.Handler) []*http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Result().Cookies()
}

func cookieNamed(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}

func TestQueryRejectsMutation(t *testing.T) {
	handler := testServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/queries/run", bytes.NewBufferString(`{"database_id":"demo","sql":"delete from users"}`))
	r.AddCookie(adminSession(t, handler))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, expected 400", w.Code)
	}
}

func TestQueryRejectsWriteHiddenInCTE(t *testing.T) {
	handler := testServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/queries/run", bytes.NewBufferString(`{"database_id":"demo","sql":"with changed as (delete from users returning id) select * from changed"}`))
	r.AddCookie(adminSession(t, handler))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, expected 400", w.Code)
	}
}

func TestQueryRequiresConnectedDatabase(t *testing.T) {
	handler := testServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/queries/run", bytes.NewBufferString(`{"database_id":"not-connected","sql":"select 1"}`))
	r.AddCookie(adminSession(t, handler))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, expected 400", w.Code)
	}
}

func TestDatabaseTestDoesNotAcceptIncompleteConnection(t *testing.T) {
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup %d %s", setupRec.Code, setupRec.Body.String())
	}
	r := httptest.NewRequest(http.MethodPost, "/api/databases/test", bytes.NewBufferString(`{"name":"warehouse","engine":"postgres"}`))
	r.AddCookie(setupRec.Result().Cookies()[0])
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, expected 400: %s", w.Code, w.Body.String())
	}
}

func TestAdminPagesRequireAdministratorSession(t *testing.T) {
	handler := testServer(t)
	workbenchReq := httptest.NewRequest(http.MethodGet, "/questions/", nil)
	workbenchRec := httptest.NewRecorder()
	handler.ServeHTTP(workbenchRec, workbenchReq)
	if workbenchRec.Code != http.StatusFound || workbenchRec.Header().Get("Location") != "/auth/login/" {
		t.Fatalf("guest workbench page %d %q", workbenchRec.Code, workbenchRec.Header().Get("Location"))
	}
	apiReq := httptest.NewRequest(http.MethodGet, "/api/questions", nil)
	apiRec := httptest.NewRecorder()
	handler.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusUnauthorized {
		t.Fatalf("guest workbench API %d", apiRec.Code)
	}
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	adminCookie := setupRec.Result().Cookies()[0]

	guestReq := httptest.NewRequest(http.MethodGet, "/admin/people/", nil)
	guestRec := httptest.NewRecorder()
	handler.ServeHTTP(guestRec, guestReq)
	if guestRec.Code != http.StatusFound || guestRec.Header().Get("Location") != "/auth/login/" {
		t.Fatalf("guest admin page %d %q", guestRec.Code, guestRec.Header().Get("Location"))
	}

	memberReq := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(`{"name":"Ben","email":"ben@example.com","password":"secret123"}`))
	memberReq.AddCookie(adminCookie)
	memberRec := httptest.NewRecorder()
	handler.ServeHTTP(memberRec, memberReq)
	if memberRec.Code != http.StatusCreated {
		t.Fatalf("invite member %d", memberRec.Code)
	}
	loginReq := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewBufferString(`{"email":"ben@example.com","password":"secret123"}`))
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	nonAdminReq := httptest.NewRequest(http.MethodGet, "/admin/people/", nil)
	nonAdminReq.AddCookie(loginRec.Result().Cookies()[0])
	nonAdminRec := httptest.NewRecorder()
	handler.ServeHTTP(nonAdminRec, nonAdminReq)
	if nonAdminRec.Code != http.StatusFound || nonAdminRec.Header().Get("Location") != "/" {
		t.Fatalf("member admin page %d %q", nonAdminRec.Code, nonAdminRec.Header().Get("Location"))
	}
}

func TestDashboardQuickCreateAssignsName(t *testing.T) {
	handler := testServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewBufferString(`{}`))
	req.AddCookie(adminSession(t, handler))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("quick create dashboard %d: %s", rec.Code, rec.Body.String())
	}
	var dashboard map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &dashboard); err != nil {
		t.Fatal(err)
	}
	if dashboard["id"] == "" || !strings.HasPrefix(dashboard["name"].(string), "新建仪表盘 ") {
		t.Fatalf("dashboard did not receive generated identity: %+v", dashboard)
	}
}

func TestSetupLoginAndSaveQuestion(t *testing.T) {
	handler := testServer(t)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	statusRec := httptest.NewRecorder()
	handler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("setup status %d", statusRec.Code)
	}

	setupBody := `{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123","site_name":"Ada Lab"}`
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(setupBody))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup %d: %s", setupRec.Code, setupRec.Body.String())
	}
	if setupRec.Result().Cookies()[0].Name != sessionCookie {
		t.Fatalf("expected session cookie after setup")
	}

	againReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(setupBody))
	againRec := httptest.NewRecorder()
	handler.ServeHTTP(againRec, againReq)
	if againRec.Code != http.StatusBadRequest {
		t.Fatalf("second setup %d, expected 400", againRec.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewBufferString(`{"email":"ada@example.com","password":"secret123"}`))
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login %d: %s", loginRec.Code, loginRec.Body.String())
	}
	cookie := loginRec.Result().Cookies()[0]

	meReq := httptest.NewRequest(http.MethodGet, "/api/user/current", nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("current user %d: %s", meRec.Code, meRec.Body.String())
	}
	if !bytes.Contains(meRec.Body.Bytes(), []byte(`"is_admin":true`)) {
		t.Fatalf("setup admin should be is_admin, got %s", meRec.Body.String())
	}

	question := map[string]any{
		"name":       "订单计数",
		"query_type": "queryir",
		"queryir": map[string]any{
			"version": 1,
			"source": map[string]any{
				"database_id": "pg_demo",
				"table":       map[string]string{"schema": "public", "name": "orders"},
			},
			"aggregations": []map[string]string{{"fn": "count"}},
			"limit":        100,
		},
	}
	raw, _ := json.Marshal(question)
	saveReq := httptest.NewRequest(http.MethodPost, "/api/questions", bytes.NewReader(raw))
	saveReq.AddCookie(cookie)
	saveRec := httptest.NewRecorder()
	handler.ServeHTTP(saveRec, saveReq)
	if saveRec.Code != http.StatusCreated {
		t.Fatalf("save question %d: %s", saveRec.Code, saveRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/questions", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list questions %d", listRec.Code)
	}

	var saved map[string]any
	if err := json.Unmarshal(saveRec.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	questionID, _ := saved["id"].(string)
	boardBody, _ := json.Marshal(map[string]any{
		"name": "经营看板",
		"cards": []map[string]any{
			{"type": "heading", "title": "概览", "layout": map[string]int{"x": 0, "y": 0, "w": 12, "h": 1}},
			{"type": "text", "body": "日期筛选", "layout": map[string]int{"x": 0, "y": 1, "w": 12, "h": 1}},
			{"type": "question", "question_id": questionID, "layout": map[string]int{"x": 0, "y": 2, "w": 4, "h": 4}},
		},
		"filters": []map[string]any{
			{"name": "日期", "type": "date", "mappings": []map[string]string{{"field": "created_at"}}},
		},
	})
	boardReq := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewReader(boardBody))
	boardReq.AddCookie(cookie)
	boardRec := httptest.NewRecorder()
	handler.ServeHTTP(boardRec, boardReq)
	if boardRec.Code != http.StatusCreated {
		t.Fatalf("create dashboard %d: %s", boardRec.Code, boardRec.Body.String())
	}
	var board map[string]any
	_ = json.Unmarshal(boardRec.Body.Bytes(), &board)
	boardID, _ := board["id"].(string)

	searchReq := httptest.NewRequest(http.MethodGet, "/api/search?q=经营", nil)
	searchReq.AddCookie(cookie)
	searchRec := httptest.NewRecorder()
	handler.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusOK || !bytes.Contains(searchRec.Body.Bytes(), []byte("经营看板")) {
		t.Fatalf("search %d %s", searchRec.Code, searchRec.Body.String())
	}

	bmReq := httptest.NewRequest(http.MethodPost, "/api/bookmarks", bytes.NewBufferString(`{"target_type":"dashboard","target_id":"`+boardID+`"}`))
	bmReq.AddCookie(cookie)
	bmRec := httptest.NewRecorder()
	handler.ServeHTTP(bmRec, bmReq)
	if bmRec.Code != http.StatusCreated {
		t.Fatalf("bookmark %d: %s", bmRec.Code, bmRec.Body.String())
	}

	alertReq := httptest.NewRequest(http.MethodPost, "/api/alerts", bytes.NewBufferString(`{"name":"有结果","question_id":"`+questionID+`","kind":"results"}`))
	alertReq.AddCookie(cookie)
	alertRec := httptest.NewRecorder()
	handler.ServeHTTP(alertRec, alertReq)
	if alertRec.Code != http.StatusCreated {
		t.Fatalf("alert %d: %s", alertRec.Code, alertRec.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/dashboards/"+boardID, nil)
	delReq.AddCookie(cookie)
	delRec := httptest.NewRecorder()
	handler.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("archive dashboard %d", delRec.Code)
	}
	trashReq := httptest.NewRequest(http.MethodGet, "/api/trash", nil)
	trashReq.AddCookie(cookie)
	trashRec := httptest.NewRecorder()
	handler.ServeHTTP(trashRec, trashReq)
	if trashRec.Code != http.StatusOK || !bytes.Contains(trashRec.Body.Bytes(), []byte(boardID)) {
		t.Fatalf("trash %d %s", trashRec.Code, trashRec.Body.String())
	}
}

func TestPersonalProfilePasswordAndBindingLifecycle(t *testing.T) {
	handler := testServer(t)
	session := adminSession(t, handler)

	profileReq := httptest.NewRequest(http.MethodGet, "/api/user/profile", nil)
	profileReq.AddCookie(session)
	profileRec := httptest.NewRecorder()
	handler.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusOK || !bytes.Contains(profileRec.Body.Bytes(), []byte(`"name":"Ada"`)) {
		t.Fatalf("profile %d: %s", profileRec.Code, profileRec.Body.String())
	}
	var initial struct {
		User core.User `json:"user"`
	}
	if err := json.Unmarshal(profileRec.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}

	deniedEmail := httptest.NewRequest(http.MethodPut, "/api/user/profile", bytes.NewBufferString(`{"name":"Ada Lovelace","email":"new@example.com","locale":"zh-CN","theme":"dark"}`))
	deniedEmail.AddCookie(session)
	deniedEmailRec := httptest.NewRecorder()
	handler.ServeHTTP(deniedEmailRec, deniedEmail)
	if deniedEmailRec.Code != http.StatusBadRequest {
		t.Fatalf("email change without password %d: %s", deniedEmailRec.Code, deniedEmailRec.Body.String())
	}

	avatar := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	updateBody, _ := json.Marshal(map[string]string{
		"name": "Ada Lovelace", "email": "new@example.com", "locale": "zh-CN", "theme": "dark",
		"avatar_url": avatar, "current_password": "secret123",
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/api/user/profile", bytes.NewReader(updateBody))
	updateReq.AddCookie(session)
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK || !bytes.Contains(updateRec.Body.Bytes(), []byte(`"email":"new@example.com"`)) || !bytes.Contains(updateRec.Body.Bytes(), []byte(`"avatar_url"`)) {
		t.Fatalf("profile update %d: %s", updateRec.Code, updateRec.Body.String())
	}

	wrongPassword := httptest.NewRequest(http.MethodPut, "/api/user/password", bytes.NewBufferString(`{"current_password":"wrong","new_password":"newsecret456"}`))
	wrongPassword.AddCookie(session)
	wrongPasswordRec := httptest.NewRecorder()
	handler.ServeHTTP(wrongPasswordRec, wrongPassword)
	if wrongPasswordRec.Code != http.StatusBadRequest {
		t.Fatalf("wrong current password %d: %s", wrongPasswordRec.Code, wrongPasswordRec.Body.String())
	}
	changePassword := httptest.NewRequest(http.MethodPut, "/api/user/password", bytes.NewBufferString(`{"current_password":"secret123","new_password":"newsecret456"}`))
	changePassword.AddCookie(session)
	changePasswordRec := httptest.NewRecorder()
	handler.ServeHTTP(changePasswordRec, changePassword)
	if changePasswordRec.Code != http.StatusOK {
		t.Fatalf("change password %d: %s", changePasswordRec.Code, changePasswordRec.Body.String())
	}
	oldLogin := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewBufferString(`{"email":"new@example.com","password":"secret123"}`))
	oldLoginRec := httptest.NewRecorder()
	handler.ServeHTTP(oldLoginRec, oldLogin)
	if oldLoginRec.Code != http.StatusUnauthorized {
		t.Fatalf("old password login %d", oldLoginRec.Code)
	}
	newLogin := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewBufferString(`{"email":"new@example.com","password":"newsecret456"}`))
	newLoginRec := httptest.NewRecorder()
	handler.ServeHTTP(newLoginRec, newLogin)
	if newLoginRec.Code != http.StatusOK {
		t.Fatalf("new password login %d: %s", newLoginRec.Code, newLoginRec.Body.String())
	}

	providers := `[{"id":"google-main","type":"google","name":"Google","enabled":true,"client_id":"client","client_secret":"secret"}]`
	providersReq := httptest.NewRequest(http.MethodPut, "/api/identity/providers", bytes.NewBufferString(providers))
	providersReq.AddCookie(session)
	providersRec := httptest.NewRecorder()
	handler.ServeHTTP(providersRec, providersReq)
	if providersRec.Code != http.StatusOK {
		t.Fatalf("save providers %d: %s", providersRec.Code, providersRec.Body.String())
	}
	bindReq := httptest.NewRequest(http.MethodPost, "/api/users/"+initial.User.ID+"/external-identities", bytes.NewBufferString(`{"provider_id":"google-main","subject":"google-user-1"}`))
	bindReq.AddCookie(session)
	bindRec := httptest.NewRecorder()
	handler.ServeHTTP(bindRec, bindReq)
	if bindRec.Code != http.StatusOK {
		t.Fatalf("bind identity %d: %s", bindRec.Code, bindRec.Body.String())
	}
	linkedReq := httptest.NewRequest(http.MethodGet, "/api/user/profile", nil)
	linkedReq.AddCookie(session)
	linkedRec := httptest.NewRecorder()
	handler.ServeHTTP(linkedRec, linkedReq)
	if linkedRec.Code != http.StatusOK || !bytes.Contains(linkedRec.Body.Bytes(), []byte(`"linked":true`)) {
		t.Fatalf("linked profile %d: %s", linkedRec.Code, linkedRec.Body.String())
	}
	unbindReq := httptest.NewRequest(http.MethodDelete, "/api/user/external-identities/google-main", nil)
	unbindReq.AddCookie(session)
	unbindRec := httptest.NewRecorder()
	handler.ServeHTTP(unbindRec, unbindReq)
	if unbindRec.Code != http.StatusNoContent {
		t.Fatalf("unbind identity %d: %s", unbindRec.Code, unbindRec.Body.String())
	}
}

func TestPublicVersionAndReadinessExposeMigrationState(t *testing.T) {
	handler := testServer(t)
	for _, path := range []string{"/api/health", "/api/ready", "/api/version"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s = %d: %s", path, rec.Code, rec.Body.String())
		}
	}
	versionReq := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	versionRec := httptest.NewRecorder()
	handler.ServeHTTP(versionRec, versionReq)
	if !bytes.Contains(versionRec.Body.Bytes(), []byte(`"schema_version":8`)) || !bytes.Contains(versionRec.Body.Bytes(), []byte(`"version":"0.1.0-alpha.0-dev"`)) {
		t.Fatalf("unexpected version payload: %s", versionRec.Body.String())
	}
}

func TestBrowserMutationRequiresCSRFToken(t *testing.T) {
	handler := testServer(t)
	cookies := setupCookies(t, handler)
	session := cookieNamed(t, cookies, sessionCookie)
	csrf := cookieNamed(t, cookies, csrfCookie)

	blocked := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewBufferString(`{}`))
	blocked.AddCookie(session)
	blocked.Header.Set("Origin", "http://example.com")
	blocked.Header.Set("Sec-Fetch-Site", "same-origin")
	blockedRec := httptest.NewRecorder()
	handler.ServeHTTP(blockedRec, blocked)
	if blockedRec.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF token = %d: %s", blockedRec.Code, blockedRec.Body.String())
	}

	allowed := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewBufferString(`{}`))
	allowed.AddCookie(session)
	allowed.AddCookie(csrf)
	allowed.Header.Set("Origin", "http://example.com")
	allowed.Header.Set("Sec-Fetch-Site", "same-origin")
	allowed.Header.Set("X-Topbase-CSRF", csrf.Value)
	allowedRec := httptest.NewRecorder()
	handler.ServeHTTP(allowedRec, allowed)
	if allowedRec.Code != http.StatusCreated {
		t.Fatalf("valid CSRF token = %d: %s", allowedRec.Code, allowedRec.Body.String())
	}
}

func TestPersonalAnalysisAndDashboardAreIsolatedByDataGroup(t *testing.T) {
	handler := testServer(t)
	adminCookie := adminSession(t, handler)

	questionBody := `{"name":"管理员分析","query_type":"queryir","queryir":{"version":1,"source":{"database_id":"pg_demo","table":{"schema":"public","name":"orders"}},"limit":10}}`
	createQuestion := httptest.NewRequest(http.MethodPost, "/api/questions", bytes.NewBufferString(questionBody))
	createQuestion.AddCookie(adminCookie)
	createQuestionRec := httptest.NewRecorder()
	handler.ServeHTTP(createQuestionRec, createQuestion)
	if createQuestionRec.Code != http.StatusCreated {
		t.Fatalf("create admin analysis %d: %s", createQuestionRec.Code, createQuestionRec.Body.String())
	}
	var adminQuestion core.Question
	if err := json.Unmarshal(createQuestionRec.Body.Bytes(), &adminQuestion); err != nil {
		t.Fatal(err)
	}

	createDashboard := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewBufferString(`{}`))
	createDashboard.AddCookie(adminCookie)
	createDashboardRec := httptest.NewRecorder()
	handler.ServeHTTP(createDashboardRec, createDashboard)
	if createDashboardRec.Code != http.StatusCreated {
		t.Fatalf("create admin dashboard %d: %s", createDashboardRec.Code, createDashboardRec.Body.String())
	}
	var adminDashboard core.Dashboard
	if err := json.Unmarshal(createDashboardRec.Body.Bytes(), &adminDashboard); err != nil {
		t.Fatal(err)
	}

	invite := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(`{"name":"Ben","email":"ben@example.com","password":"secret123"}`))
	invite.AddCookie(adminCookie)
	inviteRec := httptest.NewRecorder()
	handler.ServeHTTP(inviteRec, invite)
	if inviteRec.Code != http.StatusCreated {
		t.Fatalf("invite member %d: %s", inviteRec.Code, inviteRec.Body.String())
	}
	login := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewBufferString(`{"email":"ben@example.com","password":"secret123"}`))
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, login)
	memberCookie := cookieNamed(t, loginRec.Result().Cookies(), sessionCookie)

	deniedData := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
	deniedData.AddCookie(memberCookie)
	deniedDataRec := httptest.NewRecorder()
	handler.ServeHTTP(deniedDataRec, deniedData)
	if deniedDataRec.Code != http.StatusForbidden {
		t.Fatalf("member data access before grant = %d: %s", deniedDataRec.Code, deniedDataRec.Body.String())
	}
	grantBody := `{"revision":1,"data_graph":{"grp_all_users":{"data":"view","sql":"query","collections":"none","admin":"none"}},"collection_graph":{}}`
	grant := httptest.NewRequest(http.MethodPut, "/api/permissions/graph", bytes.NewBufferString(grantBody))
	grant.AddCookie(adminCookie)
	grantRec := httptest.NewRecorder()
	handler.ServeHTTP(grantRec, grant)
	if grantRec.Code != http.StatusOK {
		t.Fatalf("grant data capability %d: %s", grantRec.Code, grantRec.Body.String())
	}
	allowedData := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
	allowedData.AddCookie(memberCookie)
	allowedDataRec := httptest.NewRecorder()
	handler.ServeHTTP(allowedDataRec, allowedData)
	if allowedDataRec.Code != http.StatusOK {
		t.Fatalf("member data access after grant = %d: %s", allowedDataRec.Code, allowedDataRec.Body.String())
	}
	deniedNative := httptest.NewRequest(http.MethodPost, "/api/queries/run", bytes.NewBufferString(`{"database_id":"demo","sql":"select 1"}`))
	deniedNative.AddCookie(memberCookie)
	deniedNativeRec := httptest.NewRecorder()
	handler.ServeHTTP(deniedNativeRec, deniedNative)
	if deniedNativeRec.Code != http.StatusForbidden {
		t.Fatalf("member native SQL access = %d: %s", deniedNativeRec.Code, deniedNativeRec.Body.String())
	}

	for _, path := range []string{"/api/questions/" + adminQuestion.ID, "/api/dashboards/" + adminDashboard.ID} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(memberCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("member access %s = %d: %s", path, rec.Code, rec.Body.String())
		}
	}
	listQuestions := httptest.NewRequest(http.MethodGet, "/api/questions", nil)
	listQuestions.AddCookie(memberCookie)
	listQuestionsRec := httptest.NewRecorder()
	handler.ServeHTTP(listQuestionsRec, listQuestions)
	if bytes.Contains(listQuestionsRec.Body.Bytes(), []byte(adminQuestion.ID)) {
		t.Fatalf("private analysis leaked in list: %s", listQuestionsRec.Body.String())
	}
	listDashboards := httptest.NewRequest(http.MethodGet, "/api/dashboards", nil)
	listDashboards.AddCookie(memberCookie)
	listDashboardsRec := httptest.NewRecorder()
	handler.ServeHTTP(listDashboardsRec, listDashboards)
	if bytes.Contains(listDashboardsRec.Body.Bytes(), []byte(adminDashboard.ID)) {
		t.Fatalf("private dashboard leaked in list: %s", listDashboardsRec.Body.String())
	}
}

func TestQuestionAndCollectionManagement(t *testing.T) {
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup %d %s", setupRec.Code, setupRec.Body.String())
	}
	cookie := setupRec.Result().Cookies()[0]

	questionsPage := httptest.NewRecorder()
	questionsPageReq := httptest.NewRequest(http.MethodGet, "/questions/", nil)
	questionsPageReq.AddCookie(cookie)
	handler.ServeHTTP(questionsPage, questionsPageReq)
	if questionsPage.Code != http.StatusOK || !bytes.Contains(questionsPage.Body.Bytes(), []byte("分析")) {
		t.Fatalf("questions page %d", questionsPage.Code)
	}
	newAnalysisPage := httptest.NewRecorder()
	newAnalysisPageReq := httptest.NewRequest(http.MethodGet, "/questions/new/", nil)
	newAnalysisPageReq.AddCookie(cookie)
	handler.ServeHTTP(newAnalysisPage, newAnalysisPageReq)
	if newAnalysisPage.Code != http.StatusOK || !bytes.Contains(newAnalysisPage.Body.Bytes(), []byte("选择起始数据")) {
		t.Fatalf("new analysis page %d", newAnalysisPage.Code)
	}
	collectionsPage := httptest.NewRecorder()
	collectionsPageReq := httptest.NewRequest(http.MethodGet, "/collections/", nil)
	collectionsPageReq.AddCookie(cookie)
	handler.ServeHTTP(collectionsPage, collectionsPageReq)
	if collectionsPage.Code != http.StatusOK || !bytes.Contains(collectionsPage.Body.Bytes(), []byte("数据组")) {
		t.Fatalf("collections page %d", collectionsPage.Code)
	}

	qBody, _ := json.Marshal(map[string]any{
		"name": "订单计数", "query_type": "queryir",
		"queryir": map[string]any{
			"version":      1,
			"source":       map[string]any{"database_id": "pg_demo", "table": map[string]string{"schema": "public", "name": "orders"}},
			"aggregations": []map[string]string{{"fn": "count"}},
			"limit":        100,
		},
	})
	saveReq := httptest.NewRequest(http.MethodPost, "/api/questions", bytes.NewReader(qBody))
	saveReq.AddCookie(cookie)
	saveRec := httptest.NewRecorder()
	handler.ServeHTTP(saveRec, saveReq)
	if saveRec.Code != http.StatusCreated {
		t.Fatalf("save %d %s", saveRec.Code, saveRec.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(saveRec.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	questionID, _ := saved["id"].(string)
	personalID, _ := saved["collection_id"].(string)
	if questionID == "" || personalID == "" {
		t.Fatalf("saved question should land in personal collection, got %s", saveRec.Body.String())
	}

	viewRec := httptest.NewRecorder()
	viewReq := httptest.NewRequest(http.MethodGet, "/questions/"+questionID+"/", nil)
	viewReq.AddCookie(cookie)
	handler.ServeHTTP(viewRec, viewReq)
	if viewRec.Code != http.StatusOK || !bytes.Contains(viewRec.Body.Bytes(), []byte("grid.js")) {
		t.Fatalf("question view %d", viewRec.Code)
	}

	renameReq := httptest.NewRequest(http.MethodPut, "/api/questions/"+questionID, bytes.NewBufferString(`{"name":"订单计数 v2","description":"","collection_id":"`+personalID+`"}`))
	renameReq.AddCookie(cookie)
	renameRec := httptest.NewRecorder()
	handler.ServeHTTP(renameRec, renameReq)
	if renameRec.Code != http.StatusOK || !bytes.Contains(renameRec.Body.Bytes(), []byte("订单计数 v2")) {
		t.Fatalf("rename %d %s", renameRec.Code, renameRec.Body.String())
	}

	createCol := httptest.NewRequest(http.MethodPost, "/api/collections", bytes.NewBufferString(`{"name":"经营分析"}`))
	createCol.AddCookie(cookie)
	createColRec := httptest.NewRecorder()
	handler.ServeHTTP(createColRec, createCol)
	if createColRec.Code != http.StatusCreated {
		t.Fatalf("create collection %d %s", createColRec.Code, createColRec.Body.String())
	}
	var col map[string]any
	_ = json.Unmarshal(createColRec.Body.Bytes(), &col)
	colID, _ := col["id"].(string)
	if colID == "" {
		t.Fatal("missing collection id")
	}
	if owner, _ := col["personal_owner_user_id"].(string); owner != "" {
		t.Fatalf("team collection should not be personal: %s", createColRec.Body.String())
	}

	moveReq := httptest.NewRequest(http.MethodPut, "/api/questions/"+questionID, bytes.NewBufferString(`{"name":"订单计数 v2","collection_id":"`+colID+`"}`))
	moveReq.AddCookie(cookie)
	moveRec := httptest.NewRecorder()
	handler.ServeHTTP(moveRec, moveReq)
	if moveRec.Code != http.StatusOK {
		t.Fatalf("move %d %s", moveRec.Code, moveRec.Body.String())
	}

	getColRec := httptest.NewRecorder()
	getColReq := httptest.NewRequest(http.MethodGet, "/api/collections/"+colID, nil)
	getColReq.AddCookie(cookie)
	handler.ServeHTTP(getColRec, getColReq)
	if getColRec.Code != http.StatusOK || !bytes.Contains(getColRec.Body.Bytes(), []byte(questionID)) {
		t.Fatalf("get collection %d %s", getColRec.Code, getColRec.Body.String())
	}

	colViewRec := httptest.NewRecorder()
	colViewReq := httptest.NewRequest(http.MethodGet, "/collections/"+colID+"/", nil)
	colViewReq.AddCookie(cookie)
	handler.ServeHTTP(colViewRec, colViewReq)
	if colViewRec.Code != http.StatusOK || !bytes.Contains(colViewRec.Body.Bytes(), []byte("子数据组")) {
		t.Fatalf("collection view %d", colViewRec.Code)
	}

	delBusy := httptest.NewRequest(http.MethodDelete, "/api/collections/"+colID, nil)
	delBusy.AddCookie(cookie)
	delBusyRec := httptest.NewRecorder()
	handler.ServeHTTP(delBusyRec, delBusy)
	if delBusyRec.Code != http.StatusBadRequest {
		t.Fatalf("delete non-empty %d %s", delBusyRec.Code, delBusyRec.Body.String())
	}

	clearReq := httptest.NewRequest(http.MethodPut, "/api/questions/"+questionID, bytes.NewBufferString(`{"name":"订单计数 v2","collection_id":""}`))
	clearReq.AddCookie(cookie)
	clearRec := httptest.NewRecorder()
	handler.ServeHTTP(clearRec, clearReq)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear collection %d %s", clearRec.Code, clearRec.Body.String())
	}

	delEmpty := httptest.NewRequest(http.MethodDelete, "/api/collections/"+colID, nil)
	delEmpty.AddCookie(cookie)
	delEmptyRec := httptest.NewRecorder()
	handler.ServeHTTP(delEmptyRec, delEmpty)
	if delEmptyRec.Code != http.StatusNoContent {
		t.Fatalf("delete empty %d %s", delEmptyRec.Code, delEmptyRec.Body.String())
	}

	delPersonal := httptest.NewRequest(http.MethodDelete, "/api/collections/"+personalID, nil)
	delPersonal.AddCookie(cookie)
	delPersonalRec := httptest.NewRecorder()
	handler.ServeHTTP(delPersonalRec, delPersonal)
	if delPersonalRec.Code != http.StatusBadRequest {
		t.Fatalf("delete personal %d %s", delPersonalRec.Code, delPersonalRec.Body.String())
	}
}

func TestSemanticModelAndNativeQuestion(t *testing.T) {
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup %d %s", setupRec.Code, setupRec.Body.String())
	}
	cookie := setupRec.Result().Cookies()[0]

	fieldBody := `{"name":"user_id","semantic_type":"ForeignKey","fk_target":{"schema":"public","table":"users","name":"id"}}`
	fieldReq := httptest.NewRequest(http.MethodPut, "/api/databases/pg_demo/tables/public/orders/fields", bytes.NewBufferString(fieldBody))
	fieldReq.AddCookie(cookie)
	fieldRec := httptest.NewRecorder()
	handler.ServeHTTP(fieldRec, fieldReq)
	if fieldRec.Code != http.StatusOK {
		t.Fatalf("save field %d %s", fieldRec.Code, fieldRec.Body.String())
	}

	qBody, _ := json.Marshal(map[string]any{
		"name": "订单", "query_type": "queryir",
		"queryir": map[string]any{
			"version": 1,
			"source":  map[string]any{"database_id": "pg_demo", "table": map[string]string{"schema": "public", "name": "orders"}},
			"fields":  []string{"id", "user_id"},
		},
	})
	qReq := httptest.NewRequest(http.MethodPost, "/api/questions", bytes.NewReader(qBody))
	qReq.AddCookie(cookie)
	qRec := httptest.NewRecorder()
	handler.ServeHTTP(qRec, qReq)
	if qRec.Code != http.StatusCreated {
		t.Fatalf("question %d %s", qRec.Code, qRec.Body.String())
	}
	var saved map[string]any
	_ = json.Unmarshal(qRec.Body.Bytes(), &saved)
	modelBody, _ := json.Marshal(map[string]any{"name": "订单模型", "queryir": saved["queryir"]})
	mReq := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewReader(modelBody))
	mReq.AddCookie(cookie)
	mRec := httptest.NewRecorder()
	handler.ServeHTTP(mRec, mReq)
	if mRec.Code != http.StatusCreated {
		t.Fatalf("model %d %s", mRec.Code, mRec.Body.String())
	}

	nativeBody, _ := json.Marshal(map[string]any{
		"name": "SQL 日订单", "query_type": "native", "database_id": "pg_demo",
		"native_sql": "SELECT * FROM orders WHERE {{created_at}}",
		"parameters": []map[string]string{{"name": "created_at", "type": "date", "field": "created_at"}},
	})
	nReq := httptest.NewRequest(http.MethodPost, "/api/questions", bytes.NewReader(nativeBody))
	nReq.AddCookie(cookie)
	nRec := httptest.NewRecorder()
	handler.ServeHTTP(nRec, nReq)
	if nRec.Code != http.StatusCreated {
		t.Fatalf("native question %d %s", nRec.Code, nRec.Body.String())
	}

	drillBody, _ := json.Marshal(map[string]any{
		"queryir":    saved["queryir"],
		"drill":      map[string]any{"kind": "records", "values": map[string]any{"id": 1}},
		"join_table": "users",
	})
	dReq := httptest.NewRequest(http.MethodPost, "/api/dataset/drill", bytes.NewReader(drillBody))
	dReq.AddCookie(cookie)
	dRec := httptest.NewRecorder()
	handler.ServeHTTP(dRec, dReq)
	if dRec.Code != http.StatusBadRequest {
		t.Fatalf("drill expected execute error, got %d %s", dRec.Code, dRec.Body.String())
	}
	if !bytes.Contains(dRec.Body.Bytes(), []byte(`"implicit":true`)) && !bytes.Contains(dRec.Body.Bytes(), []byte(`"users"`)) {
		t.Fatalf("drill queryir should include implicit join, got %s", dRec.Body.String())
	}

	var nativeSaved map[string]any
	_ = json.Unmarshal(nRec.Body.Bytes(), &nativeSaved)
	dashBody, _ := json.Marshal(map[string]any{
		"name":    "日订单板",
		"cards":   []map[string]any{{"type": "question", "question_id": nativeSaved["id"], "layout": map[string]int{"x": 0, "y": 0, "w": 6, "h": 4}}},
		"filters": []map[string]any{{"name": "下单日", "type": "date", "mappings": []map[string]string{{"field": "created_at"}}}},
	})
	dashReq := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewReader(dashBody))
	dashReq.AddCookie(cookie)
	dashRec := httptest.NewRecorder()
	handler.ServeHTTP(dashRec, dashReq)
	if dashRec.Code != http.StatusCreated {
		t.Fatalf("dashboard %d %s", dashRec.Code, dashRec.Body.String())
	}
	var board map[string]any
	_ = json.Unmarshal(dashRec.Body.Bytes(), &board)
	cards, _ := board["cards"].([]any)
	if len(cards) == 0 {
		t.Fatal("expected dashboard card")
	}
	card := cards[0].(map[string]any)
	cardReq := httptest.NewRequest(http.MethodPost, "/api/dashboards/"+board["id"].(string)+"/cards/"+card["id"].(string)+"/dataset", bytes.NewBufferString(`{"filters":{}}`))
	cardReq.AddCookie(cookie)
	cardRec := httptest.NewRecorder()
	handler.ServeHTTP(cardRec, cardReq)
	if cardRec.Code == http.StatusNotFound {
		t.Fatalf("native dashboard card %d %s", cardRec.Code, cardRec.Body.String())
	}

	proposeReq := httptest.NewRequest(http.MethodPost, "/api/ai/propose-schedule", bytes.NewBufferString(`{"question_id":"`+saved["id"].(string)+`","message":"每天 9 点写入数仓"}`))
	proposeReq.AddCookie(cookie)
	proposeRec := httptest.NewRecorder()
	handler.ServeHTTP(proposeRec, proposeReq)
	if proposeRec.Code != http.StatusOK || !bytes.Contains(proposeRec.Body.Bytes(), []byte("requires_confirm")) {
		t.Fatalf("propose %d %s", proposeRec.Code, proposeRec.Body.String())
	}
	schBody, _ := json.Marshal(map[string]any{
		"name": "订单数仓", "question_id": saved["id"], "cron": "0 9 * * *", "materialize_to": "warehouse.wh_orders",
	})
	schReq := httptest.NewRequest(http.MethodPost, "/api/schedules", bytes.NewReader(schBody))
	schReq.AddCookie(cookie)
	schRec := httptest.NewRecorder()
	handler.ServeHTTP(schRec, schReq)
	if schRec.Code != http.StatusCreated {
		t.Fatalf("schedule %d %s", schRec.Code, schRec.Body.String())
	}
	var schedule map[string]any
	_ = json.Unmarshal(schRec.Body.Bytes(), &schedule)
	runReq := httptest.NewRequest(http.MethodPost, "/api/schedules/"+schedule["id"].(string)+"/run", nil)
	runReq.AddCookie(cookie)
	runRec := httptest.NewRecorder()
	handler.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusBadRequest {
		t.Fatalf("run without postgres expected 400, got %d %s", runRec.Code, runRec.Body.String())
	}
	if !bytes.Contains(runRec.Body.Bytes(), []byte("sql_compiled")) && !bytes.Contains(runRec.Body.Bytes(), []byte("SELECT")) {
		t.Fatalf("run error should include compiled sql, got %s", runRec.Body.String())
	}
}

func TestFeishuDepartmentSyncRequiresCredentials(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "")
	t.Setenv("FEISHU_APP_SECRET", "")
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	req := httptest.NewRequest(http.MethodPost, "/api/feishu/departments/sync", nil)
	req.AddCookie(setupRec.Result().Cookies()[0])
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestDashboardSubscriptionCreateAndRun(t *testing.T) {
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup %d %s", setupRec.Code, setupRec.Body.String())
	}
	cookie := setupRec.Result().Cookies()[0]

	groupsReq := httptest.NewRequest(http.MethodGet, "/api/groups", nil)
	groupsReq.AddCookie(cookie)
	groupsRec := httptest.NewRecorder()
	handler.ServeHTTP(groupsRec, groupsReq)
	if groupsRec.Code != http.StatusOK || !bytes.Contains(groupsRec.Body.Bytes(), []byte("all_users")) {
		t.Fatalf("groups %d %s", groupsRec.Code, groupsRec.Body.String())
	}

	boardBody, _ := json.Marshal(map[string]any{
		"name":  "经营看板",
		"cards": []map[string]any{{"type": "heading", "title": "概览", "layout": map[string]int{"x": 0, "y": 0, "w": 12, "h": 1}}},
	})
	boardReq := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewReader(boardBody))
	boardReq.AddCookie(cookie)
	boardRec := httptest.NewRecorder()
	handler.ServeHTTP(boardRec, boardReq)
	if boardRec.Code != http.StatusCreated {
		t.Fatalf("dashboard %d %s", boardRec.Code, boardRec.Body.String())
	}
	var board map[string]any
	_ = json.Unmarshal(boardRec.Body.Bytes(), &board)
	boardID, _ := board["id"].(string)

	subBody, _ := json.Marshal(map[string]any{"cron": "0 9 * * *", "channel": "inbox"})
	subReq := httptest.NewRequest(http.MethodPost, "/api/dashboards/"+boardID+"/subscriptions", bytes.NewReader(subBody))
	subReq.AddCookie(cookie)
	subRec := httptest.NewRecorder()
	handler.ServeHTTP(subRec, subReq)
	if subRec.Code != http.StatusCreated {
		t.Fatalf("subscription %d %s", subRec.Code, subRec.Body.String())
	}
	var sub map[string]any
	_ = json.Unmarshal(subRec.Body.Bytes(), &sub)

	listReq := httptest.NewRequest(http.MethodGet, "/api/dashboards/"+boardID+"/subscriptions", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK || !bytes.Contains(listRec.Body.Bytes(), []byte(sub["id"].(string))) {
		t.Fatalf("list subscriptions %d %s", listRec.Code, listRec.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/subscriptions/"+sub["id"].(string)+"/run", nil)
	runReq.AddCookie(cookie)
	runRec := httptest.NewRecorder()
	handler.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("run subscription %d %s", runRec.Code, runRec.Body.String())
	}
}

func TestNonAdminCannotManageDatabases(t *testing.T) {
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	adminCookie := setupRec.Result().Cookies()[0]

	inviteReq := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(`{"name":"Ben","email":"ben@example.com","password":"secret123"}`))
	inviteReq.AddCookie(adminCookie)
	inviteRec := httptest.NewRecorder()
	handler.ServeHTTP(inviteRec, inviteReq)
	if inviteRec.Code != http.StatusCreated {
		t.Fatalf("invite %d %s", inviteRec.Code, inviteRec.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewBufferString(`{"email":"ben@example.com","password":"secret123"}`))
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login %d %s", loginRec.Code, loginRec.Body.String())
	}
	userCookie := loginRec.Result().Cookies()[0]

	meReq := httptest.NewRequest(http.MethodGet, "/api/user/current", nil)
	meReq.AddCookie(userCookie)
	meRec := httptest.NewRecorder()
	handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK || bytes.Contains(meRec.Body.Bytes(), []byte(`"is_admin":true`)) {
		t.Fatalf("invited user should not be admin: %s", meRec.Body.String())
	}

	syncReq := httptest.NewRequest(http.MethodPost, "/api/databases/pg_demo/sync", nil)
	syncReq.AddCookie(userCookie)
	syncRec := httptest.NewRecorder()
	handler.ServeHTTP(syncRec, syncReq)
	if syncRec.Code != http.StatusForbidden {
		t.Fatalf("non-admin sync %d %s", syncRec.Code, syncRec.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/databases/pg_demo", bytes.NewBufferString(`{"name":"x","engine":"postgres"}`))
	putReq.AddCookie(userCookie)
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusForbidden {
		t.Fatalf("non-admin put %d %s", putRec.Code, putRec.Body.String())
	}

	connReq := httptest.NewRequest(http.MethodGet, "/api/databases/pg_demo/connection", nil)
	connReq.AddCookie(userCookie)
	connRec := httptest.NewRecorder()
	handler.ServeHTTP(connRec, connReq)
	if connRec.Code != http.StatusForbidden {
		t.Fatalf("non-admin connection %d %s", connRec.Code, connRec.Body.String())
	}

	noteReq := httptest.NewRequest(http.MethodPut, "/api/databases/pg_demo/tables/public/orders/annotation", bytes.NewBufferString(`{"display_name":"订单"}`))
	noteReq.AddCookie(userCookie)
	noteRec := httptest.NewRecorder()
	handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusForbidden {
		t.Fatalf("non-admin annotation %d %s", noteRec.Code, noteRec.Body.String())
	}

	fieldReq := httptest.NewRequest(http.MethodPut, "/api/databases/pg_demo/tables/public/orders/fields", bytes.NewBufferString(`{"name":"id","display_name":"编号"}`))
	fieldReq.AddCookie(userCookie)
	fieldRec := httptest.NewRecorder()
	handler.ServeHTTP(fieldRec, fieldReq)
	if fieldRec.Code != http.StatusForbidden {
		t.Fatalf("non-admin field %d %s", fieldRec.Code, fieldRec.Body.String())
	}
}

func TestAdminDatabaseSyncWithoutConnection(t *testing.T) {
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	req := httptest.NewRequest(http.MethodPost, "/api/databases/missing/sync", nil)
	req.AddCookie(setupRec.Result().Cookies()[0])
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	cachedReq := httptest.NewRequest(http.MethodGet, "/api/databases/missing/tables?cached=1", nil)
	cachedReq.AddCookie(setupRec.Result().Cookies()[0])
	cachedRec := httptest.NewRecorder()
	handler.ServeHTTP(cachedRec, cachedReq)
	if cachedRec.Code != http.StatusOK || cachedRec.Body.String() != "[]\n" && !bytes.Contains(cachedRec.Body.Bytes(), []byte("[]")) {
		t.Fatalf("cached tables %d %s", cachedRec.Code, cachedRec.Body.String())
	}

	connReq := httptest.NewRequest(http.MethodGet, "/api/databases/missing/connection", nil)
	connReq.AddCookie(setupRec.Result().Cookies()[0])
	connRec := httptest.NewRecorder()
	handler.ServeHTTP(connRec, connReq)
	if connRec.Code != http.StatusNotFound {
		t.Fatalf("missing connection %d %s", connRec.Code, connRec.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/databases/missing", bytes.NewBufferString(`{"name":"x","engine":"postgres","host":"127.0.0.1","database":"app","username":"u"}`))
	putReq.AddCookie(setupRec.Result().Cookies()[0])
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusNotFound {
		t.Fatalf("missing update %d %s", putRec.Code, putRec.Body.String())
	}
}

func TestQuestionChartSpecAndDashboardLayout(t *testing.T) {
	handler := testServer(t)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"language":"zh-CN","admin_name":"Ada","admin_email":"ada@example.com","admin_password":"secret123"}`))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup %d %s", setupRec.Code, setupRec.Body.String())
	}
	cookie := setupRec.Result().Cookies()[0]

	questionBody, _ := json.Marshal(map[string]any{
		"name":       "订单计数",
		"query_type": "queryir",
		"queryir": map[string]any{
			"version": 1,
			"source": map[string]any{
				"database_id": "pg_demo",
				"table":       map[string]string{"schema": "public", "name": "orders"},
			},
			"aggregations": []map[string]string{{"fn": "count"}},
			"group_by":     []map[string]string{{"field": "status"}},
		},
	})
	saveReq := httptest.NewRequest(http.MethodPost, "/api/questions", bytes.NewReader(questionBody))
	saveReq.AddCookie(cookie)
	saveRec := httptest.NewRecorder()
	handler.ServeHTTP(saveRec, saveReq)
	if saveRec.Code != http.StatusCreated {
		t.Fatalf("save question %d: %s", saveRec.Code, saveRec.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(saveRec.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	questionID, _ := saved["id"].(string)

	patchBody, _ := json.Marshal(map[string]any{
		"name":          "订单计数",
		"description":   "",
		"collection_id": saved["collection_id"],
		"chartspec": map[string]any{
			"type": "pie",
			"x":    "status",
			"y":    []string{"count"},
			"series": map[string]any{
				"completed": map[string]any{"color": "#509EE3", "title": "已完成"},
			},
			"show_total": true,
			"percent":    "legend",
			"donut":      true,
		},
	})
	patchReq := httptest.NewRequest(http.MethodPut, "/api/questions/"+questionID, bytes.NewReader(patchBody))
	patchReq.AddCookie(cookie)
	patchRec := httptest.NewRecorder()
	handler.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch question %d: %s", patchRec.Code, patchRec.Body.String())
	}
	var patched map[string]any
	_ = json.Unmarshal(patchRec.Body.Bytes(), &patched)
	spec, _ := patched["chartspec"].(map[string]any)
	if spec == nil || spec["type"] != "pie" || spec["x"] != "status" {
		t.Fatalf("chartspec not saved: %s", patchRec.Body.String())
	}
	series, _ := spec["series"].(map[string]any)
	completed, _ := series["completed"].(map[string]any)
	if completed["color"] != "#509EE3" || completed["title"] != "已完成" {
		t.Fatalf("series style not saved: %s", patchRec.Body.String())
	}

	tableBody, _ := json.Marshal(map[string]any{
		"name":          "订单计数",
		"description":   "",
		"collection_id": patched["collection_id"],
		"chartspec": map[string]any{
			"type":   "table",
			"search": "已完成",
			"columns": map[string]any{
				"status": map[string]any{"visible": false, "filter": "=done", "title": "状态"},
			},
		},
	})
	tableReq := httptest.NewRequest(http.MethodPut, "/api/questions/"+questionID, bytes.NewReader(tableBody))
	tableReq.AddCookie(cookie)
	tableRec := httptest.NewRecorder()
	handler.ServeHTTP(tableRec, tableReq)
	if tableRec.Code != http.StatusOK {
		t.Fatalf("patch table spec %d: %s", tableRec.Code, tableRec.Body.String())
	}
	var tableQ map[string]any
	_ = json.Unmarshal(tableRec.Body.Bytes(), &tableQ)
	tableSpec, _ := tableQ["chartspec"].(map[string]any)
	cols, _ := tableSpec["columns"].(map[string]any)
	status, _ := cols["status"].(map[string]any)
	if tableSpec["type"] != "table" || tableSpec["search"] != "已完成" || status["filter"] != "=done" || status["visible"] != false {
		t.Fatalf("table view not saved: %s", tableRec.Body.String())
	}

	boardBody, _ := json.Marshal(map[string]any{"name": "空看板", "cards": []any{}})
	boardReq := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewReader(boardBody))
	boardReq.AddCookie(cookie)
	boardRec := httptest.NewRecorder()
	handler.ServeHTTP(boardRec, boardReq)
	if boardRec.Code != http.StatusCreated {
		t.Fatalf("create dashboard %d: %s", boardRec.Code, boardRec.Body.String())
	}
	var board map[string]any
	_ = json.Unmarshal(boardRec.Body.Bytes(), &board)
	boardID, _ := board["id"].(string)
	tabs, _ := board["tabs"].([]any)
	if boardID == "" || len(tabs) == 0 {
		t.Fatalf("dashboard missing tabs: %s", boardRec.Body.String())
	}
	tabID, _ := tabs[0].(map[string]any)["id"].(string)

	updateBody, _ := json.Marshal(map[string]any{
		"name": "空看板",
		"tabs": board["tabs"],
		"cards": []map[string]any{{
			"id": "crd_layout1", "type": "question", "question_id": questionID,
			"tab_id": tabID,
			"layout": map[string]int{"x": 2, "y": 3, "w": 6, "h": 5},
			"config": map[string]any{"chartspec": map[string]any{"type": "bar"}},
		}},
		"filters": []any{},
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/api/dashboards/"+boardID, bytes.NewReader(updateBody))
	updateReq.AddCookie(cookie)
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update dashboard %d: %s", updateRec.Code, updateRec.Body.String())
	}
	var updated map[string]any
	_ = json.Unmarshal(updateRec.Body.Bytes(), &updated)
	cards, _ := updated["cards"].([]any)
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %s", updateRec.Body.String())
	}
	card := cards[0].(map[string]any)
	layout := card["layout"].(map[string]any)
	if layout["x"] != float64(2) || layout["y"] != float64(3) || layout["w"] != float64(6) || layout["h"] != float64(5) {
		t.Fatalf("layout not persisted: %+v", layout)
	}
	if card["question_id"] != questionID {
		t.Fatalf("question not attached: %+v", card)
	}
}

func TestViewPagesDoNotShadowStaticScripts(t *testing.T) {
	handler := testServer(t)
	cookie := adminSession(t, handler)

	pageReq := httptest.NewRequest(http.MethodGet, "/questions/qst_demo/", nil)
	pageReq.AddCookie(cookie)
	pageRec := httptest.NewRecorder()
	handler.ServeHTTP(pageRec, pageReq)
	if pageRec.Code != http.StatusOK || !bytes.Contains(pageRec.Body.Bytes(), []byte("/question-view.js")) {
		t.Fatalf("question page %d %s", pageRec.Code, pageRec.Body.String())
	}

	jsReq := httptest.NewRequest(http.MethodGet, "/question-view.js", nil)
	jsRec := httptest.NewRecorder()
	handler.ServeHTTP(jsRec, jsReq)
	if jsRec.Code != http.StatusOK || !bytes.Contains(jsRec.Body.Bytes(), []byte("async function boot")) {
		t.Fatalf("question-view.js %d %s", jsRec.Code, jsRec.Body.String())
	}
	if bytes.Contains(jsRec.Body.Bytes(), []byte("<!doctype html>")) {
		t.Fatal("question-view.js was served as HTML")
	}

	nestedReq := httptest.NewRequest(http.MethodGet, "/questions/view/view.js", nil)
	nestedRec := httptest.NewRecorder()
	handler.ServeHTTP(nestedRec, nestedReq)
	if nestedRec.Code != http.StatusNotFound && bytes.Contains(nestedRec.Body.Bytes(), []byte("<!doctype html>")) {
		t.Fatalf("nested view.js should not be the question HTML page: %d", nestedRec.Code)
	}

	dashJSReq := httptest.NewRequest(http.MethodGet, "/dashboard-view.js", nil)
	dashJSRec := httptest.NewRecorder()
	handler.ServeHTTP(dashJSRec, dashJSReq)
	if dashJSRec.Code != http.StatusOK || !bytes.Contains(dashJSRec.Body.Bytes(), []byte("function renderPalette")) {
		t.Fatalf("dashboard-view.js %d %s", dashJSRec.Code, dashJSRec.Body.String())
	}

	listPage := httptest.NewRequest(http.MethodGet, "/dashboard/", nil)
	listPage.AddCookie(cookie)
	listPageRec := httptest.NewRecorder()
	handler.ServeHTTP(listPageRec, listPage)
	if listPageRec.Code != http.StatusOK || !bytes.Contains(listPageRec.Body.Bytes(), []byte("/dashboard-list.js")) {
		t.Fatalf("dashboard list page %d %s", listPageRec.Code, listPageRec.Body.String())
	}

	listJS := httptest.NewRequest(http.MethodGet, "/dashboard-list.js", nil)
	listJSRec := httptest.NewRecorder()
	handler.ServeHTTP(listJSRec, listJS)
	if listJSRec.Code != http.StatusOK || bytes.Contains(listJSRec.Body.Bytes(), []byte("<!doctype html>")) {
		t.Fatalf("dashboard-list.js %d %s", listJSRec.Code, listJSRec.Body.String())
	}
	if !bytes.Contains(listJSRec.Body.Bytes(), []byte("POST")) && !bytes.Contains(listJSRec.Body.Bytes(), []byte("/api/dashboards")) {
		t.Fatalf("dashboard-list.js missing create call: %s", listJSRec.Body.String())
	}

	cssReq := httptest.NewRequest(http.MethodGet, "/dashboard.css", nil)
	cssRec := httptest.NewRecorder()
	handler.ServeHTTP(cssRec, cssReq)
	if cssRec.Code != http.StatusOK || bytes.Contains(cssRec.Body.Bytes(), []byte("<!doctype html>")) {
		t.Fatalf("dashboard.css %d ctype=%s", cssRec.Code, cssRec.Header().Get("Content-Type"))
	}
	if bytes.Contains(dashJSRec.Body.Bytes(), []byte("<!doctype html>")) {
		t.Fatal("dashboard-view.js was served as HTML")
	}

	oldDashReq := httptest.NewRequest(http.MethodGet, "/dashboard/view/view.js", nil)
	oldDashRec := httptest.NewRecorder()
	handler.ServeHTTP(oldDashRec, oldDashReq)
	if bytes.Contains(oldDashRec.Body.Bytes(), []byte("<!doctype html>")) && bytes.Contains(oldDashRec.Body.Bytes(), []byte("id=\"edit\"")) {
		t.Fatal("/dashboard/view/view.js was intercepted as the dashboard HTML page")
	}
}

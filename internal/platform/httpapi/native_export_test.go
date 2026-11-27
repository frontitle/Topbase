package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frontitle/Topbase/internal/core"
)

func TestNativeQuestionCSVExport(t *testing.T) {
	handler := testServer(t)
	admin := adminSession(t, handler)
	request := func(cookie *http.Cookie, method, path, data string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(data))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s %s = %d, want %d: %s", method, path, rec.Code, status, rec.Body)
		}
		return rec
	}
	database := request(admin, "POST", "/api/databases", `{"engine":"sqlite","name":"Export fixture","database":":memory:"}`, 201)
	var db core.Database
	if err := json.Unmarshal(database.Body.Bytes(), &db); err != nil {
		t.Fatal(err)
	}
	create := func(statement string) core.Question {
		t.Helper()
		body, err := json.Marshal(core.Question{Name: "Saved SQL", DatabaseID: db.ID, QueryType: "native", NativeSQL: statement})
		if err != nil {
			t.Fatal(err)
		}
		rec := request(admin, "POST", "/api/questions", string(body), 201)
		var question core.Question
		if err := json.Unmarshal(rec.Body.Bytes(), &question); err != nil {
			t.Fatal(err)
		}
		return question
	}
	simple := create("select 42 as answer")
	noParams := request(admin, "POST", "/api/questions/"+simple.ID+"/export", "", 200)
	if noParams.Body.String() != "answer\n42\n" {
		t.Fatalf("CSV with no request body: %q", noParams.Body)
	}
	question := create("with recursive n(i) as (select 1 union all select i+1 from n where i<1100) select i, {{tag}} as label from n")
	path := "/api/questions/" + question.ID + "/export"
	bad := request(admin, "POST", path, "{}", 400)
	if !strings.Contains(bad.Body.String(), "missing parameter tag") || bad.Header().Get("Content-Disposition") != "" {
		t.Fatalf("parameter error returned as an attachment: %v %s", bad.Header(), bad.Body)
	}
	input := `{"parameters":{"tag":"a,b"}}`
	previewPath := "/api/questions/" + question.ID + "/native-preview"
	request(admin, "POST", previewPath, "{}", http.StatusBadRequest)
	detailPath := "/api/questions/" + question.ID + "?view=detail"
	checkDetail := func(cookie *http.Cookie, edit, run, materialize bool) {
		t.Helper()
		rec := request(cookie, "GET", detailPath, "", http.StatusOK)
		var info struct {
			CanEdit        *bool  `json:"can_edit"`
			CanRun         *bool  `json:"can_run"`
			CanMaterialize *bool  `json:"can_materialize"`
			CollectionName string `json:"collection_name"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil || info.CanEdit == nil || *info.CanEdit != edit || info.CanRun == nil || *info.CanRun != run ||
			info.CanMaterialize == nil || *info.CanMaterialize != materialize || info.CollectionName == "" {
			t.Fatalf("unexpected question capabilities: %s", rec.Body)
		}
	}
	checkDetail(admin, true, true, true)
	rec := request(admin, "POST", path, input, 200)
	if rec.Header().Get("Content-Disposition") != `attachment; filename="`+question.ID+`.csv"` {
		t.Fatal(rec.Header())
	}
	records, err := csv.NewReader(rec.Body).ReadAll()
	if err != nil || len(records) != 1001 || records[0][1] != "label" || records[1000][0] != "1000" || records[1000][1] != "a,b" {
		t.Fatalf("exported %d records, error %v", len(records), err)
	}
	newUser := request(admin, "POST", "/api/users", `{"name":"Reader","email":"reader@example.com","password":"secret123"}`, 201)
	var reader core.User
	if err := json.Unmarshal(newUser.Body.Bytes(), &reader); err != nil {
		t.Fatal(err)
	}
	request(admin, "PUT", "/api/collections/"+question.CollectionID+"/shares", `{"user_ids":["`+reader.ID+`"]}`, 204)
	request(admin, "PUT", "/api/permissions/graph", `{"revision":1,"data_graph":{"grp_all_users":{"data":"view","sql":"query","admin":"none"}},"collection_graph":{}}`, 200)
	login := request(nil, "POST", "/api/session", `{"email":"reader@example.com","password":"secret123"}`, 200)
	readerSession := cookieNamed(t, login.Result().Cookies(), sessionCookie)
	checkDetail(readerSession, false, false, false)
	request(readerSession, "POST", previewPath, input, 403)
	denied := request(readerSession, "POST", path, input, 403)
	if !strings.Contains(denied.Body.String(), "sql permission required") {
		t.Fatal(denied.Body)
	}
	request(admin, "PUT", "/api/permissions/graph", `{"revision":2,"data_graph":{"grp_all_users":{"data":"view","sql":"native","admin":"none"}},"collection_graph":{}}`, 200)
	checkDetail(readerSession, false, true, false)
	preview := request(readerSession, "POST", previewPath, input, 200)
	var result struct {
		Rows [][]any `json:"rows"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &result); err != nil || len(result.Rows) != 1000 || result.Rows[999][1] != records[1000][1] {
		t.Fatalf("native preview disagrees with CSV: rows %d, error %v", len(result.Rows), err)
	}
	request(readerSession, "PUT", "/api/questions/"+question.ID, `{"name":"Forbidden change"}`, http.StatusForbidden)
	request(readerSession, "DELETE", "/api/questions/"+question.ID, "", http.StatusForbidden)
	destinations := request(readerSession, "GET", "/api/collections?access=edit", "", 200)
	if strings.Contains(destinations.Body.String(), question.CollectionID) {
		t.Fatal("read-only shared collection offered as a move destination")
	}
	request(admin, "PUT", "/api/collections/"+question.CollectionID+"/shares", `{"user_ids":[]}`, 204)
	request(readerSession, "POST", previewPath, input, 403)
}

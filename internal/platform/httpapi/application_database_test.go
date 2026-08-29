package httpapi

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSQLiteSetupStatusAndAdminLogicalBackup(t *testing.T) {
	t.Setenv("TOPBASE_DATA_DIR", t.TempDir())
	t.Setenv("TOPBASE_APP_DB_ENGINE", "sqlite")
	t.Setenv("TOPBASE_CRON", "off")
	handler := NewServer()
	if closer, ok := handler.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK || !bytes.Contains(statusResponse.Body.Bytes(), []byte(`"mode":"development"`)) || !bytes.Contains(statusResponse.Body.Bytes(), []byte(`"development_risk":true`)) {
		t.Fatalf("setup status %d: %s", statusResponse.Code, statusResponse.Body.String())
	}

	cookies := setupCookies(t, handler)
	backupRequest := httptest.NewRequest(http.MethodGet, "/api/admin/application-database/backup", nil)
	for _, cookie := range cookies {
		backupRequest.AddCookie(cookie)
	}
	backupResponse := httptest.NewRecorder()
	handler.ServeHTTP(backupResponse, backupRequest)
	if backupResponse.Code != http.StatusOK || backupResponse.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("backup %d: %s", backupResponse.Code, backupResponse.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(backupResponse.Body.Bytes()), int64(backupResponse.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	foundManifest := false
	for _, file := range archive.File {
		foundManifest = foundManifest || file.Name == "manifest.json"
	}
	if !foundManifest {
		t.Fatal("logical backup does not contain manifest.json")
	}
}

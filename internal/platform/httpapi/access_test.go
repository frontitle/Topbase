package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestMapBoundaryAssetIsPublic(t *testing.T) {
	req := httptest.NewRequest("GET", "/assets/maps/china-provinces.json", nil)
	if !isPublicRequest(req) {
		t.Fatal("map boundary asset must be available to public and embedded dashboards")
	}
}

func TestOtherJSONFilesStayProtected(t *testing.T) {
	req := httptest.NewRequest("GET", "/assets/private.json", nil)
	if isPublicRequest(req) {
		t.Fatal("generic JSON files must not become public static assets")
	}
}

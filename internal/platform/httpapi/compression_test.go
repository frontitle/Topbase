package httpapi

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompressionStreamsTextResponses(t *testing.T) {
	body := strings.Repeat("Topbase performance ", 1000)
	handler := compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Content-Length", "20000")
		_, _ = io.WriteString(w, body)
	}))
	request := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()
	if response.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("content encoding = %q", response.Header.Get("Content-Encoding"))
	}
	if response.Header.Get("Content-Length") != "" {
		t.Fatalf("compressed response retained Content-Length: %q", response.Header.Get("Content-Length"))
	}
	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != body {
		t.Fatal("compressed response did not round trip")
	}
}

func TestCompressionSkipsBinaryRangeAndDisabledGzip(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		encoding string
		rangeHdr string
	}{
		{name: "binary", content: "image/png", encoding: "gzip"},
		{name: "range", content: "text/plain", encoding: "gzip", rangeHdr: "bytes=0-10"},
		{name: "disabled", content: "text/plain", encoding: "gzip;q=0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", test.content)
				_, _ = io.WriteString(w, "payload")
			}))
			request := httptest.NewRequest(http.MethodGet, "/asset", nil)
			request.Header.Set("Accept-Encoding", test.encoding)
			request.Header.Set("Range", test.rangeHdr)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if got := recorder.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("content encoding = %q", got)
			}
		})
	}
}

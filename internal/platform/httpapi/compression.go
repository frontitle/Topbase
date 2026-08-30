package httpapi

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriters = sync.Pool{New: func() any {
	writer, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
	return writer
}}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	wroteHeader bool
	compressed  bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.compressed = status >= 200 && status != http.StatusNoContent && status != http.StatusNotModified && compressibleContentType(w.Header().Get("Content-Type")) && w.Header().Get("Content-Encoding") == ""
	if w.compressed {
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(body))
		}
		w.WriteHeader(http.StatusOK)
	}
	if w.compressed {
		return w.writer.Write(body)
	}
	return w.ResponseWriter.Write(body)
}

func (w *gzipResponseWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.compressed {
		_ = w.writer.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead || r.Header.Get("Range") != "" || !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}
		writer := gzipWriters.Get().(*gzip.Writer)
		writer.Reset(w)
		wrapped := &gzipResponseWriter{ResponseWriter: w, writer: writer}
		defer func() {
			if wrapped.compressed {
				_ = writer.Close()
			}
			writer.Reset(io.Discard)
			gzipWriters.Put(writer)
		}()
		next.ServeHTTP(wrapped, r)
	})
}

func acceptsGzip(value string) bool {
	for _, item := range strings.Split(value, ",") {
		parts := strings.Split(strings.TrimSpace(item), ";")
		if !strings.EqualFold(parts[0], "gzip") {
			continue
		}
		for _, parameter := range parts[1:] {
			if strings.TrimSpace(parameter) == "q=0" {
				return false
			}
		}
		return true
	}
	return false
}

func compressibleContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.HasPrefix(contentType, "text/") ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "json") ||
		strings.Contains(contentType, "xml") ||
		strings.Contains(contentType, "svg")
}

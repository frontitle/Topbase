package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/topbase/topbase/internal/buildinfo"
	"github.com/topbase/topbase/internal/platform/httpapi"
)

func main() {
	address := os.Getenv("TOPBASE_ADDR")
	if address == "" {
		address = ":8080"
	}
	handler := httpapi.NewServer()
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	stopped, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- server.ListenAndServe() }()
	log.Printf("Topbase %s (%s) listening on http://localhost%s", buildinfo.Version, buildinfo.Commit, address)
	var serveErr error
	select {
	case err := <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			serveErr = err
		}
	case <-stopped.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Topbase HTTP shutdown: %v", err)
		}
	}
	if closer, ok := handler.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			log.Printf("Topbase resource shutdown: %v", err)
		}
	}
	if serveErr != nil {
		log.Fatal(serveErr)
	}
}

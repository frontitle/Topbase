package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/topbase/topbase/internal/buildinfo"
	"github.com/topbase/topbase/internal/platform/httpapi"
)

type transportConfig struct {
	Address  string
	Scheme   string
	CertFile string
	KeyFile  string
}

func loadTransportConfig() (transportConfig, error) {
	config := transportConfig{Address: strings.TrimSpace(os.Getenv("TOPBASE_ADDR")), Scheme: "http"}
	if config.Address == "" {
		port := strings.TrimSpace(os.Getenv("TOPBASE_PORT"))
		if port == "" {
			port = "80"
		}
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return transportConfig{}, fmt.Errorf("TOPBASE_PORT must be between 1 and 65535")
		}
		config.Address = ":" + strconv.Itoa(value)
	}
	config.CertFile = strings.TrimSpace(os.Getenv("TOPBASE_TLS_CERT_FILE"))
	config.KeyFile = strings.TrimSpace(os.Getenv("TOPBASE_TLS_KEY_FILE"))
	if (config.CertFile == "") != (config.KeyFile == "") {
		return transportConfig{}, fmt.Errorf("TOPBASE_TLS_CERT_FILE and TOPBASE_TLS_KEY_FILE must be configured together")
	}
	if config.CertFile != "" {
		config.Scheme = "https"
	}
	return config, nil
}

func (config transportConfig) serve(server *http.Server) error {
	if config.Scheme == "https" {
		return server.ListenAndServeTLS(config.CertFile, config.KeyFile)
	}
	return server.ListenAndServe()
}

func (config transportConfig) displayURL() string {
	address := config.Address
	if strings.HasPrefix(address, ":") {
		address = "localhost" + address
	}
	return config.Scheme + "://" + address
}

func main() {
	transport, err := loadTransportConfig()
	if err != nil {
		log.Fatal(err)
	}
	if transport.Scheme == "https" {
		if err := os.Setenv("TOPBASE_SECURE_COOKIES", "true"); err != nil {
			log.Fatal(err)
		}
	}
	handler := httpapi.NewServer()
	server := &http.Server{Addr: transport.Address, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	stopped, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- transport.serve(server) }()
	log.Printf("Topbase %s (%s) listening on %s", buildinfo.Version, buildinfo.Commit, transport.displayURL())
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

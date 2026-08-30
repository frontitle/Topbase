package main

import "testing"

func TestLoadTransportConfigDefaultsToPort80(t *testing.T) {
	t.Setenv("TOPBASE_ADDR", "")
	t.Setenv("TOPBASE_PORT", "")
	t.Setenv("TOPBASE_TLS_CERT_FILE", "")
	t.Setenv("TOPBASE_TLS_KEY_FILE", "")
	config, err := loadTransportConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Address != ":80" || config.Scheme != "http" {
		t.Fatalf("unexpected default transport: %#v", config)
	}
}

func TestLoadTransportConfigSupportsCustomPort(t *testing.T) {
	t.Setenv("TOPBASE_ADDR", "")
	t.Setenv("TOPBASE_PORT", "9443")
	t.Setenv("TOPBASE_TLS_CERT_FILE", "/run/secrets/tls.crt")
	t.Setenv("TOPBASE_TLS_KEY_FILE", "/run/secrets/tls.key")
	config, err := loadTransportConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Address != ":9443" || config.Scheme != "https" {
		t.Fatalf("unexpected custom transport: %#v", config)
	}
	if got := config.displayURL(); got != "https://localhost:9443" {
		t.Fatalf("unexpected display URL: %q", got)
	}
}

func TestLoadTransportConfigRejectsIncompleteTLS(t *testing.T) {
	t.Setenv("TOPBASE_ADDR", ":8080")
	t.Setenv("TOPBASE_TLS_CERT_FILE", "/run/secrets/tls.crt")
	t.Setenv("TOPBASE_TLS_KEY_FILE", "")
	if _, err := loadTransportConfig(); err == nil {
		t.Fatal("expected incomplete TLS configuration to fail")
	}
}

package identity

import "testing"

func TestValidPublicHost(t *testing.T) {
	valid := []string{"bi.example.com", "localhost", "127.0.0.1", "[2001:db8::1]"}
	invalid := []string{"https://bi.example.com", "bi.example.com:8443", "bad_domain.example", "example.com/path", "-bad.example"}
	for _, value := range valid {
		if !validPublicHost(value) {
			t.Errorf("expected %q to be valid", value)
		}
	}
	for _, value := range invalid {
		if validPublicHost(value) {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}

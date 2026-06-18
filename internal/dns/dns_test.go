package dns

import (
	"testing"
)

func TestResolveIPv4_Localhost(t *testing.T) {
	ip, err := ResolveIPv4("localhost")
	if err != nil {
		t.Fatal(err)
	}
	if ip != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %s", ip)
	}
}

func TestResolveIPv4_Invalid(t *testing.T) {
	_, err := ResolveIPv4("this.does.not.exist.invalid")
	if err == nil {
		t.Fatal("expected error for nonexistent domain")
	}
}

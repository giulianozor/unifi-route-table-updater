package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
unifi_site: "mysite"
dns_name: "home.example.com"
route_id: "abc123"
route_cidr: "/32"
check_interval: "1m"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.UnifiBaseURL != "https://192.168.1.1:8443" {
		t.Fatalf("unexpected base URL: %s", cfg.UnifiBaseURL)
	}
	if cfg.UnifiAPIKey != "test-key" {
		t.Fatalf("unexpected api key: %s", cfg.UnifiAPIKey)
	}
	if cfg.UnifiSite != "mysite" {
		t.Fatalf("unexpected site: %s", cfg.UnifiSite)
	}
	if cfg.DNSName != "home.example.com" {
		t.Fatalf("unexpected dns name: %s", cfg.DNSName)
	}
	if cfg.RouteID != "abc123" {
		t.Fatalf("unexpected route id: %s", cfg.RouteID)
	}
	if cfg.RouteCIDR != "/32" {
		t.Fatalf("unexpected cidr: %s", cfg.RouteCIDR)
	}
	if cfg.CheckInterval != 1*time.Minute {
		t.Fatalf("unexpected interval: %s", cfg.CheckInterval)
	}
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.UnifiSite != "default" {
		t.Fatalf("expected default site, got %s", cfg.UnifiSite)
	}
	if cfg.RouteCIDR != "/32" {
		t.Fatalf("expected default cidr, got %s", cfg.RouteCIDR)
	}
	if cfg.CheckInterval != 5*time.Minute {
		t.Fatalf("expected default interval, got %s", cfg.CheckInterval)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`unifi_base_url: "https://192.168.1.1:8443"`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`::: invalid yaml`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

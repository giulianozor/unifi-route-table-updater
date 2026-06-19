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

func TestLoad_EmptyRouteCIDR(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
route_cidr: ""
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty route_cidr")
	}
}

func TestLoad_RouteCIDRMissingSlash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
route_cidr: "32"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for route_cidr without leading slash")
	}
}

func TestLoad_InvalidBaseURLNoScheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for base URL without scheme")
	}
}

func TestLoad_InvalidBaseURLNoHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for base URL without host")
	}
}

func TestLoad_TelegramEnabledMissingToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
telegram_enabled: true
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for telegram_enabled without bot_token")
	}
}

func TestLoad_TelegramEnabledMissingChatID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
telegram_enabled: true
telegram_bot_token: "bot123"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for telegram_enabled without chat_id")
	}
}

func TestLoad_TelegramEnabledValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
telegram_enabled: true
telegram_bot_token: "bot123"
telegram_chat_id: "chat456"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.TelegramEnabled {
		t.Fatal("expected telegram_enabled to be true")
	}
	if cfg.TelegramBotToken != "bot123" {
		t.Fatalf("unexpected bot token: %s", cfg.TelegramBotToken)
	}
	if cfg.TelegramChatID != "chat456" {
		t.Fatalf("unexpected chat id: %s", cfg.TelegramChatID)
	}
}

func TestLoad_ZeroCheckInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
check_interval: "0s"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for zero check_interval")
	}
}

func TestLoad_NegativeCheckInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "test-key"
dns_name: "home.example.com"
route_id: "abc123"
check_interval: "-5m"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for negative check_interval")
	}
}

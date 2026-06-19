package config

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	UnifiBaseURL  string        `yaml:"unifi_base_url"`
	UnifiAPIKey   string        `yaml:"unifi_api_key"`
	UnifiSite     string        `yaml:"unifi_site"`
	RouteID       string        `yaml:"route_id"`
	RouteCIDR     string        `yaml:"route_cidr"`
	DNSName       string        `yaml:"dns_name"`
	CheckInterval time.Duration `yaml:"check_interval"`
	Insecure      bool          `yaml:"insecure"`
	CACert        string        `yaml:"ca_cert"`
	Verbose       bool          `yaml:"verbose"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{
		UnifiSite:     "default",
		RouteCIDR:     "/32",
		CheckInterval: 5 * time.Minute,
	}

	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.UnifiBaseURL == "" || cfg.UnifiAPIKey == "" || cfg.DNSName == "" || cfg.RouteID == "" {
		return nil, fmt.Errorf("unifi_base_url, unifi_api_key, dns_name, and route_id are required")
	}

	u, err := url.Parse(cfg.UnifiBaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("unifi_base_url must be a valid URL with scheme and host, got %q", cfg.UnifiBaseURL)
	}

	if cfg.RouteCIDR == "" || cfg.RouteCIDR[0] != '/' {
		return nil, fmt.Errorf("route_cidr must be a non-empty string starting with '/', got %q", cfg.RouteCIDR)
	}

	if cfg.CheckInterval <= 0 {
		return nil, fmt.Errorf("check_interval must be positive, got %s", cfg.CheckInterval)
	}

	return cfg, nil
}

package unifi

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	site       string
	httpClient *http.Client
	Verbose    bool

	MaxRetries   int
	retryInitial time.Duration
	retryMax     time.Duration
}

type StaticRoute struct {
	ID            string `json:"_id,omitempty"`
	Name          string `json:"name"`
	Destination   string `json:"static-route_network"`
	Interface     string `json:"static-route_interface"`
	RouteType     string `json:"static-route_type"`
	Type          string `json:"type"`
	Enabled       bool   `json:"enabled"`
	GatewayDevice string `json:"gateway_device"`
	GatewayType   string `json:"gateway_type"`
	SiteID        string `json:"site_id"`
}

type unifiResponse struct {
	Data []json.RawMessage `json:"data"`
	Meta struct {
		RC string `json:"rc"`
	} `json:"meta"`
}

func NewClient(baseURL, apiKey, site string, insecure bool, caCertPath string) (*Client, error) {
	tlsCfg := &tls.Config{}

	if insecure {
		tlsCfg.InsecureSkipVerify = true
	} else if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("no certificates found in %s", caCertPath)
		}
		tlsCfg.RootCAs = pool
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsCfg

	return &Client{
		baseURL:      baseURL,
		apiKey:       apiKey,
		site:         site,
		httpClient:   &http.Client{Timeout: 45 * time.Second, Transport: transport},
		MaxRetries:   4,
		retryInitial: time.Second,
		retryMax:     30 * time.Second,
	}, nil
}

func (c *Client) retryDelay(attempt int) time.Duration {
	d := c.retryInitial
	for i := 0; i < attempt; i++ {
		d *= 3
		if d >= c.retryMax {
			return c.retryMax
		}
	}
	return d
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if body != nil {
		if bodyBytes, err = io.ReadAll(body); err != nil {
			return nil, err
		}
	}

	var lastErr error
	for attempt := 0; ; attempt++ {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(method, u, reqBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-KEY", c.apiKey)
		if c.Verbose {
			fmt.Printf("DEBUG: %s %s\n", method, u)
			for k, v := range req.Header {
				val := v[0]
				if k == "X-Api-Key" {
					val = "***"
				}
				fmt.Printf("DEBUG:   %s: %s\n", k, val)
			}
		}
		resp, err := c.httpClient.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt >= c.MaxRetries {
			break
		}
		delay := c.retryDelay(attempt)
		if c.Verbose {
			fmt.Printf("DEBUG: request failed: %v; retrying in %v\n", err, delay)
		}
		time.Sleep(delay)
	}
	return nil, lastErr
}

func (c *Client) sitePath(suffix string) string {
	return fmt.Sprintf("/proxy/network/api/s/%s/rest%s", c.site, suffix)
}

func formatData(data []json.RawMessage) string {
	var parts []string
	for _, d := range data {
		parts = append(parts, string(d))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (c *Client) fetchRoutes() ([]StaticRoute, error) {
	resp, err := c.do("GET", c.sitePath("/routing"), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if c.Verbose {
			if resp.Request != nil {
				fmt.Printf("DEBUG: %s %s -> %s\n", resp.Request.Method, resp.Request.URL, resp.Status)
			}
			for k, v := range resp.Header {
				fmt.Printf("DEBUG:   %s: %s\n", k, v[0])
			}
			if len(body) > 0 {
				fmt.Printf("DEBUG: %s\n", string(body))
			}
		}
		return nil, fmt.Errorf("get routes failed: %s: %s", resp.Status, string(body))
	}

	var result unifiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		io.Copy(io.Discard, resp.Body)
		return nil, err
	}
	io.Copy(io.Discard, resp.Body)
	if result.Meta.RC != "ok" {
		return nil, fmt.Errorf("get routes failed: rc=%s data=%s", result.Meta.RC, formatData(result.Data))
	}

	var routes []StaticRoute
	for _, raw := range result.Data {
		var route StaticRoute
		if err := json.Unmarshal(raw, &route); err != nil {
			log.Printf("unifi: skipping route unmarshal error: %v", err)
			continue
		}
		routes = append(routes, route)
	}
	return routes, nil
}

func (c *Client) GetInfo() (string, error) {
	resp, err := c.do("GET", "/proxy/network/integration/v1/info", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("info request failed: %s: %s", resp.Status, string(body))
	}
	return string(body), nil
}

func (c *Client) ListSites() (string, error) {
	resp, err := c.do("GET", "/proxy/network/api/self/sites", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("list sites failed: %s: %s", resp.Status, string(body))
	}
	return string(body), nil
}

func (c *Client) ListStaticRoutes() ([]StaticRoute, error) {
	return c.fetchRoutes()
}

func (c *Client) ListStaticRoutesRaw() (string, error) {
	resp, err := c.do("GET", c.sitePath("/routing"), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("list routes failed: %s: %s", resp.Status, string(body))
	}
	return string(body), nil
}

func (c *Client) GetStaticRoute(id string) (*StaticRoute, error) {
	resp, err := c.do("GET", c.sitePath("/routing/"+id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get route failed: %s: %s", resp.Status, string(body))
	}

	var result unifiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		io.Copy(io.Discard, resp.Body)
		return nil, err
	}
	io.Copy(io.Discard, resp.Body)
	if result.Meta.RC != "ok" {
		return nil, fmt.Errorf("get route failed: rc=%s data=%s", result.Meta.RC, formatData(result.Data))
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("route %s not found", id)
	}

	var route StaticRoute
	if err := json.Unmarshal(result.Data[0], &route); err != nil {
		return nil, err
	}
	return &route, nil
}

func (c *Client) GetStaticRouteRaw(id string) (string, error) {
	resp, err := c.do("GET", c.sitePath("/routing/"+id), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get route failed: %s: %s", resp.Status, string(body))
	}
	return string(body), nil
}

func (c *Client) UpdateStaticRoute(route *StaticRoute) error {
	b, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("marshaling route: %w", err)
	}
	resp, err := c.do("PUT", c.sitePath("/routing/"+route.ID), bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update route failed: %s: %s", resp.Status, string(body))
	}

	var result unifiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		io.Copy(io.Discard, resp.Body)
		return err
	}
	io.Copy(io.Discard, resp.Body)
	if result.Meta.RC != "ok" {
		return fmt.Errorf("update route failed: rc=%s data=%s", result.Meta.RC, formatData(result.Data))
	}
	return nil
}

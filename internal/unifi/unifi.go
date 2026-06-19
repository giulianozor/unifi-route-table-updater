package unifi

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	site       string
	httpClient *http.Client
	Verbose    bool
}

type StaticRoute struct {
	ID              string `json:"_id,omitempty"`
	Name            string `json:"name"`
	Destination     string `json:"static-route_network"`
	Interface       string `json:"static-route_interface"`
	RouteType       string `json:"static-route_type"`
	Type            string `json:"type"`
	Enabled         bool   `json:"enabled"`
	GatewayDevice   string `json:"gateway_device"`
	GatewayType     string `json:"gateway_type"`
	SiteID          string `json:"site_id"`
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

	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		site:       site,
		httpClient: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: tlsCfg}},
	}, nil
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header["X-API-KEY"] = []string{c.apiKey}
	if c.Verbose {
		fmt.Printf("DEBUG: %s %s\n", method, u)
		for k, v := range req.Header {
			val := v[0]
			if k == "X-Api-Key" || k == "X-API-KEY" {
				val = "***"
			}
			fmt.Printf("DEBUG:   %s: %s\n", k, val)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) sitePath(suffix string) string {
	return fmt.Sprintf("/proxy/network/api/s/%s/rest%s", c.site, suffix)
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
			fmt.Printf("DEBUG: %s %s -> %s\n", resp.Request.Method, resp.Request.URL, resp.Status)
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
		return nil, err
	}
	if result.Meta.RC != "ok" {
		return nil, fmt.Errorf("unexpected response code: %s", result.Meta.RC)
	}

	var routes []StaticRoute
	for _, raw := range result.Data {
		var route StaticRoute
		if err := json.Unmarshal(raw, &route); err != nil {
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
		return nil, err
	}
	if result.Meta.RC != "ok" {
		return nil, fmt.Errorf("unexpected response code: %s", result.Meta.RC)
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
	b, _ := json.Marshal(route)
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
		return err
	}
	if result.Meta.RC != "ok" {
		return fmt.Errorf("update route unexpected response: %s", result.Meta.RC)
	}
	return nil
}

package unifi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(baseURL, apiKey, site string) *Client {
	c, _ := NewClient(baseURL, apiKey, site, false, "")
	return c
}

func testRoute(id, name, dest string) StaticRoute {
	return StaticRoute{
		ID:            id,
		Name:          name,
		Destination:   dest,
		Interface:     "6a0ba933836aa09b8d15f089",
		RouteType:     "interface-route",
		Type:          "static-route",
		Enabled:       true,
		GatewayDevice: "94:2a:6f:52:cb:e8",
		GatewayType:   "default",
		SiteID:        "69b146bc8c38ba0b451b832d",
	}
}

func marshalRoute(r StaticRoute) json.RawMessage {
	b, _ := json.Marshal(r)
	return json.RawMessage(b)
}

func TestGetStaticRoute_Found(t *testing.T) {
	routes := []json.RawMessage{
		marshalRoute(testRoute("1", "foo", "10.0.0.1/32")),
		marshalRoute(testRoute("2", "bar", "10.0.0.2/32")),
	}
	resp := unifiResponse{Data: routes, Meta: struct {
		RC string `json:"rc"`
	}{RC: "ok"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "GET" {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "test-key", "default")
	route, err := client.GetStaticRoute("1")
	if err != nil {
		t.Fatal(err)
	}
	if route.ID != "1" || route.Destination != "10.0.0.1/32" {
		t.Fatalf("got %+v", route)
	}
}

func TestGetStaticRoute_NotFound(t *testing.T) {
	resp := unifiResponse{Data: nil, Meta: struct {
		RC string `json:"rc"`
	}{RC: "ok"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "key", "default")
	_, err := client.GetStaticRoute("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing route")
	}
}

func TestGetStaticRoute_NotOK(t *testing.T) {
	resp := unifiResponse{Data: nil, Meta: struct {
		RC string `json:"rc"`
	}{RC: "error"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "key", "default")
	_, err := client.GetStaticRoute("1")
	if err == nil {
		t.Fatal("expected error for rc != ok")
	}
}

func TestGetStaticRoute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "key", "default")
	_, err := client.GetStaticRoute("1")
	if err == nil {
		t.Fatal("expected error on bad HTTP status")
	}
}

func TestUpdateStaticRoute_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-Api-Key") != "key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "key", "default")
	route := testRoute("r1", "test", "10.0.0.1/32")
	if err := client.UpdateStaticRoute(&route); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateStaticRoute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "key", "default")
	route := testRoute("r1", "test", "10.0.0.1/32")
	err := client.UpdateStaticRoute(&route)
	if err == nil {
		t.Fatal("expected error on bad HTTP status")
	}
}

func TestListStaticRoutes(t *testing.T) {
	routes := []json.RawMessage{
		marshalRoute(testRoute("r1", "foo", "10.0.0.1/32")),
		marshalRoute(testRoute("r2", "bar", "10.0.0.2/32")),
	}
	resp := unifiResponse{Data: routes, Meta: struct {
		RC string `json:"rc"`
	}{RC: "ok"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "key", "default")
	list, err := client.ListStaticRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(list))
	}
}

func TestUpdateStaticRoute_NotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"meta":{"rc":"error"},"data":[]}`)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL, "key", "default")
	route := testRoute("r1", "test", "10.0.0.1/32")
	err := client.UpdateStaticRoute(&route)
	if err == nil {
		t.Fatal("expected error for rc != ok")
	}
}

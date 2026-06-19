package telegram

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	originalURL := apiURL
	apiURL = func(_ string) string { return srv.URL + "/sendMessage" }
	defer func() { apiURL = originalURL }()

	if err := Send("bot:token", "chat123", "hello"); err != nil {
		t.Fatal(err)
	}
}

func TestSend_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":false,"description":"chat not found"}`)
	}))
	defer srv.Close()

	originalURL := apiURL
	apiURL = func(_ string) string { return srv.URL + "/sendMessage" }
	defer func() { apiURL = originalURL }()

	err := Send("bot:token", "badchat", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "telegram API error: chat not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"ok":false,"description":"Unauthorized"}`)
	}))
	defer srv.Close()

	originalURL := apiURL
	apiURL = func(_ string) string { return srv.URL + "/sendMessage" }
	defer func() { apiURL = originalURL }()

	err := Send("bad:token", "chat123", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

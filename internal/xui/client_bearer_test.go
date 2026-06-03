package xui

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientAPITokenSkipsLoginAndCSRF(t *testing.T) {
	t.Parallel()

	var loginCalls int32
	var listCalls int32

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&loginCalls, 1)
	})
	mux.HandleFunc("/ui/panel/api/inbounds/list", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&listCalls, 1)
		if r.Header.Get("Authorization") != "Bearer panel-api-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-CSRF-Token") != "" {
			t.Error("unexpected CSRF header with bearer auth")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":[]}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(ClientConfig{
		BaseURL:            srv.URL + "/ui/",
		APIToken:           "panel-api-token",
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := c.ListInbounds(); err != nil {
		t.Fatalf("ListInbounds() error = %v", err)
	}
	if atomic.LoadInt32(&loginCalls) != 0 {
		t.Fatalf("login must not run with api_token, got %d calls", loginCalls)
	}
	if atomic.LoadInt32(&listCalls) != 1 {
		t.Fatalf("expected one list call, got %d", listCalls)
	}
}

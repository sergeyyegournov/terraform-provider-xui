package xui

import (
	"net/http"
	"testing"
)

// registerMockSessionRoutes adds /csrf-token and /login handlers expected by 3x-ui v3.
func registerMockSessionRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/csrf-token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":"test-csrf-token"}`))
	})
	mux.HandleFunc(prefix+"/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") == "" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"success":false,"msg":"csrf required"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
}

func newTestSessionClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := NewClient(ClientConfig{
		BaseURL:            baseURL,
		Username:           "u",
		Password:           "p",
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c
}

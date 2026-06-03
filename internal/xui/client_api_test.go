package xui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientAPIGetAddUpdateDelete(t *testing.T) {
	t.Parallel()

	var loginCalls int32
	store := map[string]PanelClientRecord{
		"user@example.com": {
			UUID:   "550e8400-e29b-41d4-a716-446655440000",
			Email:  "user@example.com",
			Enable: true,
		},
	}
	attachments := map[string][]int{
		"user@example.com": {3},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/csrf-token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":"test-csrf-token"}`))
	})
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&loginCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/api/clients/get/", func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimPrefix(r.URL.Path, "/ui/panel/api/clients/get/")
		c, ok := store[email]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":false,"msg":"not found","obj":null}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		payload, _ := json.Marshal(map[string]any{
			"client":     c,
			"inboundIds": attachments[email],
		})
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":` + string(payload) + `}`))
	})
	mux.HandleFunc("/ui/panel/api/clients/add", func(w http.ResponseWriter, r *http.Request) {
		var req ClientCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode add: %v", err)
		}
		store[req.Client.Email] = PanelClientRecord{
			UUID:   req.Client.ID,
			Email:  req.Client.Email,
			Enable: req.Client.Enable,
			LimitIP: req.Client.LimitIP,
		}
		attachments[req.Client.Email] = req.InboundIDs
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/api/clients/update/", func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimPrefix(r.URL.Path, "/ui/panel/api/clients/update/")
		var updated PanelClientInput
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			t.Fatalf("decode update: %v", err)
		}
		prev := store[email]
		prev.Email = email
		prev.Enable = updated.Enable
		prev.LimitIP = updated.LimitIP
		store[email] = prev
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/api/clients/del/", func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimPrefix(r.URL.Path, "/ui/panel/api/clients/del/")
		delete(store, email)
		delete(attachments, email)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newTestSessionClient(t, srv.URL+"/ui/")

	got, err := c.GetClientByEmail("user@example.com")
	if err != nil {
		t.Fatalf("GetClientByEmail: %v", err)
	}
	if PanelClientUUID(got.Client) != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("uuid = %q", PanelClientUUID(got.Client))
	}
	if len(got.InboundIDs) != 1 || got.InboundIDs[0] != 3 {
		t.Fatalf("inboundIds = %v", got.InboundIDs)
	}

	if err := c.AddClient(ClientCreateRequest{
		Client: PanelClientInput{
			ID:     "11111111-1111-1111-1111-111111111111",
			Email:  "new@example.com",
			Enable: true,
		},
		InboundIDs: []int{5},
	}); err != nil {
		t.Fatalf("AddClient: %v", err)
	}

	if err := c.UpdateClient("user@example.com", PanelClientInput{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Email:   "user@example.com",
		Enable:  false,
		LimitIP: 2,
	}); err != nil {
		t.Fatalf("UpdateClient: %v", err)
	}
	got, err = c.GetClientByEmail("user@example.com")
	if err != nil {
		t.Fatalf("GetClientByEmail after update: %v", err)
	}
	if got.Client.Enable {
		t.Fatal("expected enable=false after update")
	}
	if got.Client.LimitIP != 2 {
		t.Fatalf("limitIp = %d", got.Client.LimitIP)
	}

	if err := c.DeleteClient("new@example.com", false); err != nil {
		t.Fatalf("DeleteClient: %v", err)
	}
	if _, err := c.GetClientByEmail("new@example.com"); err == nil {
		t.Fatal("expected error after delete")
	}
}

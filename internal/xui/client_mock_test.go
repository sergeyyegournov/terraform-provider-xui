package xui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientListInboundsWithMockedAPI(t *testing.T) {
	t.Parallel()

	var loginCalls int32
	var listCalls int32

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&loginCalls, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected login method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/api/inbounds/list", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&listCalls, 1)
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected list method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":[{"id":11,"remark":"tf-test","settings":"{\"clients\":[]}"}]}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL+"/ui/", "u", "p", true)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	raw, err := c.ListInbounds()
	if err != nil {
		t.Fatalf("ListInbounds() error = %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal list obj: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 inbound, got %d", len(got))
	}
	if got[0]["remark"] != "tf-test" {
		t.Fatalf("unexpected remark: %v", got[0]["remark"])
	}
	if atomic.LoadInt32(&loginCalls) < 1 {
		t.Fatalf("expected login to be called")
	}
	if atomic.LoadInt32(&listCalls) != 1 {
		t.Fatalf("expected one list call, got %d", listCalls)
	}
}

func TestClientRequestJSONRetriesAfter404(t *testing.T) {
	t.Parallel()

	var loginCalls int32
	var listCalls int32

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&loginCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/api/inbounds/list", func(w http.ResponseWriter, _ *http.Request) {
		call := atomic.AddInt32(&listCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		if call == 1 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"success":false,"msg":"not found","obj":null}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":[]}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL+"/ui/", "u", "p", true)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := c.ListInbounds(); err != nil {
		t.Fatalf("ListInbounds() error = %v", err)
	}

	if atomic.LoadInt32(&listCalls) != 2 {
		t.Fatalf("expected 2 list calls due to retry, got %d", listCalls)
	}
	if atomic.LoadInt32(&loginCalls) < 2 {
		t.Fatalf("expected login to run at least twice, got %d", loginCalls)
	}
}

func TestGetXrayTemplateSupportsStringWrappedObj(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/xray", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected xray method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":"{\"xraySetting\":{\"log\":{\"loglevel\":\"warning\"}}}"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL+"/ui/", "u", "p", true)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	got, err := c.GetXrayTemplate()
	if err != nil {
		t.Fatalf("GetXrayTemplate() error = %v", err)
	}
	if !strings.Contains(got, `"loglevel":"warning"`) {
		t.Fatalf("expected xraySetting payload, got %s", got)
	}
}

func TestUpdateXrayTemplateUsesFormEndpoint(t *testing.T) {
	t.Parallel()

	var updateCalls int32
	var gotBody string

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/xray/update", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&updateCalls, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
			t.Fatalf("unexpected content-type: %s", ct)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":null}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL+"/ui/", "u", "p", true)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := c.UpdateXrayTemplate(`{"log":{"loglevel":"warning"}}`); err != nil {
		t.Fatalf("UpdateXrayTemplate() error = %v", err)
	}
	if atomic.LoadInt32(&updateCalls) != 1 {
		t.Fatalf("expected one update call, got %d", updateCalls)
	}
	if !strings.Contains(gotBody, "xraySetting=") {
		t.Fatalf("expected xraySetting form field, got %s", gotBody)
	}
}

func TestUpdatePanelSettingsPostsJSON(t *testing.T) {
	t.Parallel()

	var updateCalls int32
	var got map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/setting/update", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&updateCalls, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("unexpected content-type: %s", ct)
		}
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":null}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL+"/ui/", "u", "p", true)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := c.UpdatePanelSettings(map[string]any{"webPort": 2053, "tgBotEnable": true}); err != nil {
		t.Fatalf("UpdatePanelSettings() error = %v", err)
	}
	if atomic.LoadInt32(&updateCalls) != 1 {
		t.Fatalf("expected one update call, got %d", updateCalls)
	}
	if got["webPort"] != float64(2053) || got["tgBotEnable"] != true {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestRestartXrayServiceUsesFormAndChecksXrayResult(t *testing.T) {
	t.Parallel()

	var restartCalls int32
	var resultCalls int32

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/api/server/restartXrayService", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&restartCalls, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
			t.Fatalf("unexpected content-type: %s", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"Xray has been successfully relaunched.","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/xray/getXrayResult", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&resultCalls, 1)
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":""}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL+"/ui/", "u", "p", true)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := c.RestartXrayService(); err != nil {
		t.Fatalf("RestartXrayService() error = %v", err)
	}
	if atomic.LoadInt32(&restartCalls) != 1 {
		t.Fatalf("expected one restart call, got %d", restartCalls)
	}
	if atomic.LoadInt32(&resultCalls) < 1 {
		t.Fatalf("expected getXrayResult to be called")
	}
}

func TestRestartXrayServiceReturnsErrorWhenXrayResultNotEmpty(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/ui/login", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/api/server/restartXrayService", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"Xray has been successfully relaunched.","obj":null}`))
	})
	mux.HandleFunc("/ui/panel/xray/getXrayResult", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":"some startup error"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL+"/ui/", "u", "p", true)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := c.RestartXrayService(); err == nil {
		t.Fatalf("expected error when xray result stays non-empty")
	}
}

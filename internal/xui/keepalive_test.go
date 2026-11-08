package xui

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestKeepAliveWireMarshal(t *testing.T) {
	t.Parallel()
	zero := KeepAliveWire("0")
	raw, err := json.Marshal(PanelClientInput{Email: "a@example.com", KeepAlive: &zero})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"keepAlive":0`) {
		t.Fatalf("expected numeric zero, got %s", raw)
	}
	ranged := KeepAliveWire("15-25")
	raw, err = json.Marshal(PanelClientInput{Email: "a@example.com", KeepAlive: &ranged})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"keepAlive":"15-25"`) {
		t.Fatalf("expected range string, got %s", raw)
	}
	raw, err = json.Marshal(PanelClientInput{Email: "a@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "keepAlive") {
		t.Fatalf("expected omitted keepAlive, got %s", raw)
	}
}

func TestKeepAliveWireUnmarshal(t *testing.T) {
	t.Parallel()
	var rec PanelClientRecord
	if err := json.Unmarshal([]byte(`{"email":"a","keepAlive":25}`), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.KeepAlive != "25" {
		t.Fatalf("got %q", rec.KeepAlive)
	}
	if err := json.Unmarshal([]byte(`{"email":"a","keepAlive":"15-25"}`), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.KeepAlive != "15-25" {
		t.Fatalf("got %q", rec.KeepAlive)
	}
}

func TestPanelClientInputHostRuleClear(t *testing.T) {
	t.Parallel()
	empty := []int{}
	raw, err := json.Marshal(PanelClientInput{Email: "a@example.com", ClientHostRuleIDs: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"clientHostRuleIds":[]`) {
		t.Fatalf("expected empty list, got %s", raw)
	}
	raw, err = json.Marshal(PanelClientInput{Email: "a@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "clientHostRuleIds") {
		t.Fatalf("expected omitted host rules, got %s", raw)
	}
}

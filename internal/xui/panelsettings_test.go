package xui

import (
	"strings"
	"testing"
)

func TestMergePanelSettingsPreservesServerOnlyFields(t *testing.T) {
	t.Parallel()

	current := map[string]any{
		"webPort":           float64(2053),
		"smtpPort":          float64(587),
		"smtpEnable":        false,
		"tgMemory":          float64(80),
		"panelOutbound":     "direct",
		"remarkTemplate":    "{{INBOUND}}",
		"hasTgBotToken":     true,
		"hasSmtpPassword":   false,
	}
	updates := map[string]any{
		"webPort":      int64(8443),
		"panelProxy":   "should-map",
		"remarkModel":  "legacy-remark",
		"tgBotLoginNotify": true,
		"subShowInfo":  true,
	}

	merged := mergePanelSettings(current, updates)

	if merged["webPort"] != int64(8443) {
		t.Fatalf("webPort = %v, want 8443", merged["webPort"])
	}
	if merged["smtpPort"] != float64(587) {
		t.Fatalf("smtpPort = %v, want preserved 587", merged["smtpPort"])
	}
	if merged["tgMemory"] != float64(80) {
		t.Fatalf("tgMemory = %v, want preserved 80", merged["tgMemory"])
	}
	if _, ok := merged["panelProxy"]; !ok {
		t.Fatal("panelProxy mirror should be present for legacy panels")
	}
	if _, ok := merged["remarkModel"]; !ok {
		t.Fatal("remarkModel mirror should be present for legacy panels")
	}
	if merged["panelOutbound"] != "should-map" {
		t.Fatalf("panelOutbound = %v, want alias from panelProxy", merged["panelOutbound"])
	}
	if merged["remarkTemplate"] != "legacy-remark" {
		t.Fatalf("remarkTemplate = %v, want alias from remarkModel", merged["remarkTemplate"])
	}
	if _, ok := merged["tgBotLoginNotify"]; !ok {
		t.Fatal("tgBotLoginNotify should be preserved in merged payload")
	}
	if _, ok := merged["hasTgBotToken"]; ok {
		t.Fatal("hasTgBotToken should be stripped from merged payload")
	}
}

func TestFormatAPIErrorIncludesValidationIssues(t *testing.T) {
	t.Parallel()

	err := formatAPIError("POST", "https://example/panel/api/setting/update", &APIResponse{
		Success: false,
		Msg:     "request body failed validation",
		Obj:     []byte(`{"message":"request body failed validation","issues":[{"field":"smtpPort","rule":"gte","message":"smtpPort must be gte 1"}]}`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	if want := "smtpPort (gte)"; !strings.Contains(got, want) {
		t.Fatalf("error %q should mention %q", got, want)
	}
}

package provider

import "testing"

func TestValidateOptionalJSONString(t *testing.T) {
	t.Parallel()

	if err := validateOptionalJSONString("", "sub_json_fragment"); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if err := validateOptionalJSONString("  ", "sub_json_noises"); err != nil {
		t.Fatalf("whitespace: %v", err)
	}
	if err := validateOptionalJSONString(`{"a":1}`, "sub_json_mux"); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := validateOptionalJSONString(`{bad`, "sub_json_rules"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestPanelJSONWireValue(t *testing.T) {
	t.Parallel()

	if got := panelJSONWireValue(normalizedJSONStringValue("")); got != "" {
		t.Fatalf("empty panel value = %q, want \"\"", got)
	}
	if got := panelJSONWireValue(normalizedJSONStringValue("{}")); got != "" {
		t.Fatalf("empty sentinel = %q, want \"\"", got)
	}
	in := `{"b":2,"a":1}`
	want := `{"a":1,"b":2}`
	if got := panelJSONWireValue(normalizedJSONStringValue(in)); got != want {
		t.Fatalf("compact = %q, want %q", got, want)
	}
}

package provider

import "testing"

func TestValidateOptionalJSONString(t *testing.T) {
	t.Parallel()

	if err := validateOptionalJSONString("", "sub_json_final_mask"); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if err := validateOptionalJSONString("  ", "sub_json_mux"); err != nil {
		t.Fatalf("whitespace: %v", err)
	}
	if err := validateOptionalJSONString(`{"a":1}`, "sub_json_mux"); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := validateOptionalJSONString(`{bad`, "sub_json_rules"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONSemanticEqual(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b  string
		equal bool
	}{
		{"", "", true},
		{"", "{}", true},
		{"{}", "", true},
		{`{"b":2,"a":1}`, `{"a":1,"b":2}`, true},
		{`{"a":1}`, `{"a":2}`, false},
		{"", `{"a":1}`, false},
	}
	for _, tc := range cases {
		got := jsonSemanticEqual(tc.a, tc.b)
		if got != tc.equal {
			t.Fatalf("jsonSemanticEqual(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.equal)
		}
	}
}

func TestPanelJSONWireValue(t *testing.T) {
	t.Parallel()

	if got := panelJSONWireValue(""); got != "" {
		t.Fatalf("empty = %q, want \"\"", got)
	}
	if got := panelJSONWireValue("{}"); got != "" {
		t.Fatalf("{} = %q, want \"\"", got)
	}
	in := `{"b":2,"a":1}`
	want := `{"a":1,"b":2}`
	if got := panelJSONWireValue(in); got != want {
		t.Fatalf("compact = %q, want %q", got, want)
	}
}

func TestPanelJSONStateValue(t *testing.T) {
	t.Parallel()

	if got := panelJSONStateValue(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := panelJSONStateValue(`{ "x" : 1 }`); got != `{"x":1}` {
		t.Fatalf("compact state = %q", got)
	}
}

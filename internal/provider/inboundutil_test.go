package provider

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		t.Fatalf("parse %q: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		t.Fatalf("parse %q: %v", b, err)
	}
	return reflect.DeepEqual(av, bv)
}

func TestMergeInboundSettingsPreservingClients(t *testing.T) {
	t.Parallel()

	server := `{"clients":[{"id":"existing","email":"a"}],"decryption":"none","fallbacks":[]}`
	user := `{"clients":[{"id":"new","email":"b"}],"decryption":"none","udp":true}`

	merged, err := mergeInboundSettingsPreservingClients(server, user)
	if err != nil {
		t.Fatalf("mergeInboundSettingsPreservingClients() error = %v", err)
	}

	if !strings.Contains(merged, `"udp":true`) {
		t.Fatalf("expected user key to be merged, got %s", merged)
	}
	if !strings.Contains(merged, `"id":"existing"`) {
		t.Fatalf("expected server clients to be preserved, got %s", merged)
	}
	if strings.Contains(merged, `"id":"new"`) {
		t.Fatalf("expected user clients to be ignored, got %s", merged)
	}
}

func TestCanonicalizeInboundSettings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "strips empty string client fields",
			in:       `{"clients":[{"id":"x","email":"a","flow":"","password":"","security":"","subId":""}],"decryption":"none"}`,
			expected: `{"clients":[{"email":"a","id":"x"}],"decryption":"none"}`,
		},
		{
			name:     "strips panel timestamp fields",
			in:       `{"clients":[{"id":"x","email":"a","created_at":1,"updated_at":2}]}`,
			expected: `{"clients":[{"email":"a","id":"x"}]}`,
		},
		{
			name:     "preserves non-empty string and numeric fields",
			in:       `{"clients":[{"id":"x","email":"a","flow":"xtls","limitIp":5,"totalGB":0}]}`,
			expected: `{"clients":[{"email":"a","flow":"xtls","id":"x","limitIp":5,"totalGB":0}]}`,
		},
		{
			name:     "no clients key is left untouched",
			in:       `{"decryption":"none","fallbacks":[]}`,
			expected: `{"decryption":"none","fallbacks":[]}`,
		},
		{
			name:     "multiple clients each normalized",
			in:       `{"clients":[{"id":"a","email":"1","flow":""},{"id":"b","email":"2","subId":"","comment":"hi"}]}`,
			expected: `{"clients":[{"email":"1","id":"a"},{"comment":"hi","email":"2","id":"b"}]}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := canonicalizeInboundSettings(tc.in)
			// Compare semantically (order-agnostic) rather than as strings.
			if !jsonEqual(t, got, tc.expected) {
				t.Fatalf("canonicalizeInboundSettings(%s) = %s, want %s", tc.in, got, tc.expected)
			}
		})
	}
}

func TestCanonicalizeInboundSettingsIsIdempotent(t *testing.T) {
	t.Parallel()
	in := `{"clients":[{"id":"x","email":"a","flow":"","password":"","created_at":1,"updated_at":2}],"decryption":"none"}`
	once := canonicalizeInboundSettings(in)
	twice := canonicalizeInboundSettings(once)
	if once != twice {
		t.Fatalf("canonicalize not idempotent:\n once: %s\ntwice: %s", once, twice)
	}
}

func TestJSONStringFromMap_objectAndString(t *testing.T) {
	t.Parallel()

	obj := map[string]any{
		"settings": map[string]any{
			"clients":    []any{},
			"decryption": "none",
		},
		"streamSettings": `{"network":"tcp"}`,
	}
	got := jsonStringFromMap(obj, "settings")
	if got == "" {
		t.Fatal("expected marshaled settings object")
	}
	if got[0] != '{' {
		t.Fatalf("expected JSON object, got %q", got)
	}
	if jsonStringFromMap(obj, "streamSettings") != `{"network":"tcp"}` {
		t.Fatalf("expected string passthrough")
	}
}

package provider

// JSON normalization strategy.
//
// The 3x-ui panel stores and serves several JSON-string attributes.
// Different flavours need different treatment, so the provider uses three
// distinct normalizers; everything JSON-related in this provider should go
// through one of them.
//
//  1. jsontypes.Normalized (github.com/hashicorp/terraform-plugin-framework-jsontypes)
//
//     The default choice for any opaque JSON-blob attribute where only
//     semantic equality matters — whitespace / key-order drift between the
//     user's HCL and the panel's stored form should never cause a plan or
//     refresh diff. Terraform's framework handles the semantic-equality
//     check for us, so in Create / Update / Read the provider just stores
//     the raw string and the framework does the rest.
//
//     Used for: xui_xray_template.json, xui_inbound.stream_settings,
//     xui_inbound.sniffing, and the matching data-source attributes.
//
//     xui_panel_settings sub_json_* fields use plain strings plus
//     jsonSemanticString plan modifiers instead (optional null/empty config).
//
//  2. canonicalizeInboundSettings (this file)
//
//     A structural normalizer for xui_inbound.settings. Unlike the blobs
//     above, the provider actively manipulates this JSON: it maintains a
//     sentinel client, and on update leaves the panel's client list alone.
//     The panel also mutates its own representation across endpoints —
//     dropping empty-string client fields (`password`, `security`, …)
//     between insert and update, and adding `created_at` / `updated_at`
//     timestamps that tick independently of user intent. We therefore
//     cannot rely on plain semantic equality. canonicalizeInboundSettings
//     strips those panel-added fluctuations from every client object and
//     produces a compact, byte-stable form.
//
//     Used both as state projection in resource_inbound.go and as the
//     output of the settings plan modifier in inboundutil.go.
//
//  3. compactJSON (this file)
//
//     Tiny re-marshaling helper. Only used for canonicalizing JSON
//     payloads sent to the panel so over-the-wire requests don't carry
//     the user's HCL whitespace. Never used for state storage or plan
//     comparisons: that is what (1) and (2) are for.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// compactJSON re-encodes a JSON string into compact form. If the input is
// not valid JSON, it is returned unchanged so the caller can surface a
// more specific validation error downstream.
func compactJSON(s string) string {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	out, err := json.Marshal(v)
	if err != nil {
		return s
	}
	return string(out)
}

// canonicalizeInboundSettings returns a stable compact JSON representation
// of 3x-ui inbound settings. See the package-level doc above for why this
// is not a plain jsontypes.Normalized.
//
// Concretely it drops from every element of settings.clients[]:
//   - keys with a nil value
//   - keys with an empty-string value
//   - the panel-added "created_at" / "updated_at" timestamps
//
// Non-client keys are left untouched; user-authored clients keep any
// non-empty field values.
func canonicalizeInboundSettings(s string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return compactJSON(s)
	}
	clients, ok := m["clients"].([]any)
	if ok {
		for i, c := range clients {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			for k, v := range cm {
				if v == nil {
					delete(cm, k)
					continue
				}
				if str, isStr := v.(string); isStr && str == "" {
					delete(cm, k)
					continue
				}
				if k == "created_at" || k == "updated_at" {
					delete(cm, k)
					continue
				}
			}
			clients[i] = cm
		}
		m["clients"] = clients
	}
	out, err := json.Marshal(m)
	if err != nil {
		return compactJSON(s)
	}
	return string(out)
}

// validateOptionalJSONString accepts empty/whitespace (panel default) but requires
// valid JSON when a non-empty value is set.
func validateOptionalJSONString(s, name string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// panelJSONIsEmpty reports whether a panel subscription JSON field is unset.
func panelJSONIsEmpty(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "{}"
}

// jsonSemanticEqual compares two JSON strings semantically. Empty string and
// "{}" are treated as equivalent (3x-ui stores unset fields as "").
func jsonSemanticEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if panelJSONIsEmpty(a) && panelJSONIsEmpty(b) {
		return true
	}
	if panelJSONIsEmpty(a) || panelJSONIsEmpty(b) {
		return false
	}
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}
	ca, err := json.Marshal(va)
	if err != nil {
		return false
	}
	cb, err := json.Marshal(vb)
	if err != nil {
		return false
	}
	return string(ca) == string(cb)
}

// panelJSONStateValue returns a compact JSON string for state, or "" when unset.
func panelJSONStateValue(s string) string {
	s = strings.TrimSpace(s)
	if panelJSONIsEmpty(s) {
		return ""
	}
	return compactJSON(s)
}

// panelJSONWireValue compacts non-empty JSON for panel API payloads.
func panelJSONWireValue(s string) string {
	if panelJSONIsEmpty(s) {
		return ""
	}
	return compactJSON(s)
}

type jsonSemanticStringPlanModifier struct{}

func (jsonSemanticStringPlanModifier) Description(_ context.Context) string {
	return "Treats semantically equal JSON values as unchanged (whitespace, key order, and empty vs `{}`)."
}

func (jsonSemanticStringPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Treats semantically equal JSON values as unchanged (whitespace, key order, and empty vs `{}`)."
}

func (jsonSemanticStringPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}
	if req.StateValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}
	if jsonSemanticEqual(req.PlanValue.ValueString(), req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

func jsonSemanticString() planmodifier.String {
	return jsonSemanticStringPlanModifier{}
}

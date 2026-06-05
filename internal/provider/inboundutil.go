package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func inboundMapFromJSON(raw []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func intFromMap(m map[string]any, key string) (int, error) {
	v, ok := m[key]
	if !ok {
		return 0, fmt.Errorf("missing %q", key)
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("invalid type for %q", key)
	}
}

func stringFromMap(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// jsonStringFromMap returns a JSON text value for an inbound field that 3x-ui
// may return either as a string (legacy wire shape) or as a nested object (v3+).
func jsonStringFromMap(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	out, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(out)
}

func int64FromMap(m map[string]any, key string) int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

func boolFromMap(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)
	if ok {
		return b
	}
	return false
}

// mergeInboundSettingsPreservingClients applies non-`clients` keys from userJSON onto serverJSON.
// If the server JSON has a `clients` key, its value is kept (so clients managed via API / xui_vless_client stay).
// If the server has no `clients` key (e.g. some protocols), the merged object has no `clients` key unless user-only merge added nothing there — user `clients` are never applied on update.
func mergeInboundSettingsPreservingClients(serverJSON, userJSON string) (string, error) {
	var server map[string]any
	if err := json.Unmarshal([]byte(serverJSON), &server); err != nil {
		return "", fmt.Errorf("parse server settings: %w", err)
	}
	if server == nil {
		server = map[string]any{}
	}
	existingClients, hadClients := server["clients"]

	var user map[string]any
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return "", fmt.Errorf("parse settings: %w", err)
	}
	if user == nil {
		user = map[string]any{}
	}
	for k, v := range user {
		if k == "clients" {
			continue
		}
		server[k] = v
	}
	if hadClients {
		server["clients"] = existingClients
	} else {
		delete(server, "clients")
	}
	// Keep settings JSON compact to avoid string-format drift between plan/apply/read.
	out, err := json.Marshal(server)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// settingsIgnoreClientsPlanModifier projects the planned value of the
// inbound `settings` attribute into the exact canonical form that the
// resource will store in state after apply. Concretely, on Update it
// takes the non-`clients` keys from the user's plan and grafts them onto
// the current state's `clients` array (managed via xui_*_client resources
// or the panel), then canonicalizes the result. This gives Terraform a
// planned value that is byte-equal to what the post-apply refresh will
// observe, which is what the framework's "inconsistent result after apply"
// check requires.
//
// On Create (state is null) the modifier is a no-op.
type settingsIgnoreClientsPlanModifier struct{}

func (m settingsIgnoreClientsPlanModifier) Description(_ context.Context) string {
	return "Projects the clients array from current state onto the planned settings JSON so user-managed fields don't fight with the panel-managed client list."
}

func (m settingsIgnoreClientsPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m settingsIgnoreClientsPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	projected, err := mergeInboundSettingsPreservingClients(req.StateValue.ValueString(), req.PlanValue.ValueString())
	if err != nil {
		return
	}
	resp.PlanValue = types.StringValue(canonicalizeInboundSettings(projected))
}

func settingsIgnoreClients() planmodifier.String {
	return settingsIgnoreClientsPlanModifier{}
}

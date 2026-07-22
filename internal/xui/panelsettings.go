// Package xui includes panel settings merge helpers used when talking to 3x-ui.
package xui

import (
	"encoding/json"
	"fmt"
	"strings"
)

var panelSettingsReadOnlyKeys = map[string]struct{}{
	"hasTgBotToken":     {},
	"hasTwoFactorToken": {},
	"hasLdapPassword":   {},
	"hasApiToken":       {},
	"hasWarpSecret":     {},
	"hasNordSecret":     {},
	"hasSmtpPassword":   {},
}

// deprecatedPanelSettingsKeys are no longer accepted by current AllSetting
// handlers (removed in 3x-ui v3.4+) and must not be sent on update.
var deprecatedPanelSettingsKeys = map[string]struct{}{
	"subEmailInRemark": {},
	"subShowInfo":      {},
	"tgBotLoginNotify": {},
	"subJsonFragment":  {},
	"subJsonNoises":    {},
}

func mergePanelSettings(current, updates map[string]any) map[string]any {
	merged := make(map[string]any, len(current)+len(updates))
	for k, v := range current {
		if _, ro := panelSettingsReadOnlyKeys[k]; ro {
			continue
		}
		merged[k] = v
	}
	for k, v := range updates {
		if _, ro := panelSettingsReadOnlyKeys[k]; ro {
			continue
		}
		if _, deprecated := deprecatedPanelSettingsKeys[k]; deprecated {
			continue
		}
		merged[k] = v
	}
	return merged
}

func formatAPIError(method, endpoint string, msg *APIResponse) error {
	if msg == nil {
		return fmt.Errorf("%s %s: unknown error", method, endpoint)
	}
	detail := strings.TrimSpace(msg.Msg)
	if len(msg.Obj) > 0 {
		var payload struct {
			Message string `json:"message"`
			Issues  []struct {
				Field   string `json:"field"`
				Rule    string `json:"rule"`
				Message string `json:"message"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(msg.Obj, &payload); err == nil {
			if payload.Message != "" {
				detail = payload.Message
			}
			if len(payload.Issues) > 0 {
				parts := make([]string, 0, len(payload.Issues))
				for _, issue := range payload.Issues {
					if issue.Field == "" {
						continue
					}
					part := issue.Field
					if issue.Rule != "" {
						part += " (" + issue.Rule + ")"
					}
					parts = append(parts, part)
				}
				if len(parts) > 0 {
					if detail == "" {
						detail = "validation failed"
					}
					detail += ": " + strings.Join(parts, ", ")
				}
			}
		}
	}
	if detail == "" {
		detail = "request failed"
	}
	return fmt.Errorf("%s %s: %s", method, endpoint, detail)
}

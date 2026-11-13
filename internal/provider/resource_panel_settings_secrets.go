package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// panelSecretSlot is a settings secret the panel redacts on read.
// A blank value in POST /panel/api/setting/all means "stored, not shown"
// when the matching has* flag is true, and "unchanged" on update unless
// clearKey is sent.
type panelSecretSlot struct {
	hasKey   string
	clearKey string
	value    *types.String
}

func panelSecretSlots(m *panelSettingsModel) []panelSecretSlot {
	return []panelSecretSlot{
		{hasKey: "hasTgBotToken", clearKey: "clearTgBotToken", value: &m.TgBotToken},
		{hasKey: "hasSmtpPassword", clearKey: "clearSmtpPassword", value: &m.SmtpPassword},
		{hasKey: "hasLdapPassword", clearKey: "clearLdapPassword", value: &m.LdapPassword},
		{hasKey: "hasDiscordBotToken", clearKey: "clearDiscordBotToken", value: &m.DiscordBotToken},
		{hasKey: "hasTwoFactorToken", value: &m.TwoFactorToken},
	}
}

func panelSecretBlank(v types.String) bool {
	return v.IsNull() || v.IsUnknown() || strings.TrimSpace(v.ValueString()) == ""
}

// restoreRedactedPanelSecrets keeps a secret already stored in Terraform when
// the panel reports it as configured but returns the value blank.
func restoreRedactedPanelSecrets(api map[string]any, state, prior *panelSettingsModel) {
	priorSlots := panelSecretSlots(prior)
	for i, slot := range panelSecretSlots(state) {
		if !boolFromMap(api, slot.hasKey) {
			continue
		}
		if !panelSecretBlank(*slot.value) {
			continue
		}
		prev := priorSlots[i].value
		if panelSecretBlank(*prev) {
			continue
		}
		*slot.value = *prev
	}
}

// applyPanelSecretClears asks the panel to delete a secret this resource
// previously stored. A blank secret without the flag is preserved server-side,
// and an omitted attribute defaults to blank, so the flag is sent only when
// state had a value and the plan removes it.
func applyPanelSecretClears(payload map[string]any, prior, plan *panelSettingsModel) {
	priorSlots := panelSecretSlots(prior)
	for i, slot := range panelSecretSlots(plan) {
		if slot.clearKey == "" {
			continue
		}
		if panelSecretBlank(*priorSlots[i].value) || !panelSecretBlank(*slot.value) {
			continue
		}
		payload[slot.clearKey] = true
	}
}

// panelSettingsSecretRestoreOnly reports that every difference between prior
// state and the plan is a redacted secret whose state was blank. That happens
// after an older provider refresh stored the API's empty string. Rewriting the
// configured value does not need the panel process to restart.
func panelSettingsSecretRestoreOnly(prior, plan panelSettingsModel) bool {
	priorSlots := panelSecretSlots(&prior)
	planSlots := panelSecretSlots(&plan)
	for i := range priorSlots {
		if priorSlots[i].value.IsUnknown() || planSlots[i].value.IsUnknown() {
			return false
		}
		prev := strings.TrimSpace(priorSlots[i].value.ValueString())
		next := strings.TrimSpace(planSlots[i].value.ValueString())
		if prev == next {
			continue
		}
		if prev != "" || next == "" {
			return false
		}
	}
	clearedPrior := prior
	clearedPlan := plan
	for _, slot := range panelSecretSlots(&clearedPrior) {
		*slot.value = types.StringValue("")
	}
	for _, slot := range panelSecretSlots(&clearedPlan) {
		*slot.value = types.StringValue("")
	}
	return clearedPrior == clearedPlan
}

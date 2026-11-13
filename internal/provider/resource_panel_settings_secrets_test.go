package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAPIToModelKeepsConfiguredSecretWhenPanelRedactsIt(t *testing.T) {
	t.Parallel()

	state := &panelSettingsModel{
		TgBotToken:      types.StringValue("telegram-secret"),
		SmtpPassword:    types.StringValue("smtp-secret"),
		LdapPassword:    types.StringValue("ldap-secret"),
		DiscordBotToken: types.StringValue("discord-secret"),
		TwoFactorToken:  types.StringValue("totp-secret"),
		WebPort:         types.Int64Value(2053),
	}
	api := map[string]any{
		"webPort":            float64(8443),
		"tgBotToken":         "",
		"hasTgBotToken":      true,
		"smtpPassword":       "",
		"hasSmtpPassword":    true,
		"ldapPassword":       "   ",
		"hasLdapPassword":    true,
		"discordBotToken":    "",
		"hasDiscordBotToken": true,
		"twoFactorToken":     "",
		"hasTwoFactorToken":  true,
	}

	(&panelSettingsResource{}).apiToModel(api, state)

	if state.WebPort.ValueInt64() != 8443 {
		t.Fatalf("webPort = %d, want 8443", state.WebPort.ValueInt64())
	}
	assertSecret := func(name, got, want string) {
		t.Helper()
		if got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	assertSecret("tgBotToken", state.TgBotToken.ValueString(), "telegram-secret")
	assertSecret("smtpPassword", state.SmtpPassword.ValueString(), "smtp-secret")
	assertSecret("ldapPassword", state.LdapPassword.ValueString(), "ldap-secret")
	assertSecret("discordBotToken", state.DiscordBotToken.ValueString(), "discord-secret")
	assertSecret("twoFactorToken", state.TwoFactorToken.ValueString(), "totp-secret")
}

func TestAPIToModelDropsSecretWhenPanelReportsItUnset(t *testing.T) {
	t.Parallel()

	state := &panelSettingsModel{TgBotToken: types.StringValue("telegram-secret")}
	(&panelSettingsResource{}).apiToModel(map[string]any{
		"tgBotToken":    "",
		"hasTgBotToken": false,
	}, state)

	if got := state.TgBotToken.ValueString(); got != "" {
		t.Fatalf("tgBotToken = %q, want empty", got)
	}
}

func TestAPIToModelUsesSecretReturnedByPanel(t *testing.T) {
	t.Parallel()

	state := &panelSettingsModel{TgBotToken: types.StringValue("old-secret")}
	(&panelSettingsResource{}).apiToModel(map[string]any{
		"tgBotToken":    "panel-secret",
		"hasTgBotToken": true,
	}, state)

	if got := state.TgBotToken.ValueString(); got != "panel-secret" {
		t.Fatalf("tgBotToken = %q, want panel-secret", got)
	}
}

func TestAPIToModelLeavesRedactedSecretEmptyWhenStateHasNone(t *testing.T) {
	t.Parallel()

	state := &panelSettingsModel{}
	(&panelSettingsResource{}).apiToModel(map[string]any{
		"tgBotToken":    "",
		"hasTgBotToken": true,
	}, state)

	if got := state.TgBotToken.ValueString(); got != "" {
		t.Fatalf("tgBotToken = %q, want empty", got)
	}
}

func TestApplyPanelSecretClearsOnlyWhenConfiguredSecretRemoved(t *testing.T) {
	t.Parallel()

	prior := &panelSettingsModel{
		TgBotToken:      types.StringValue("telegram-secret"),
		SmtpPassword:    types.StringValue("smtp-secret"),
		DiscordBotToken: types.StringValue("discord-secret"),
	}
	plan := &panelSettingsModel{
		TgBotToken:      types.StringValue(""),
		SmtpPassword:    types.StringValue("smtp-secret"),
		DiscordBotToken: types.StringValue(""),
		LdapPassword:    types.StringValue(""),
	}
	payload := map[string]any{}
	applyPanelSecretClears(payload, prior, plan)

	if payload["clearTgBotToken"] != true {
		t.Fatalf("clearTgBotToken = %#v, want true", payload["clearTgBotToken"])
	}
	if payload["clearDiscordBotToken"] != true {
		t.Fatalf("clearDiscordBotToken = %#v, want true", payload["clearDiscordBotToken"])
	}
	if _, ok := payload["clearSmtpPassword"]; ok {
		t.Fatal("clearSmtpPassword should be absent when the password is unchanged")
	}
	if _, ok := payload["clearLdapPassword"]; ok {
		t.Fatal("clearLdapPassword should be absent when state never stored a password")
	}
	if _, ok := payload["clearTwoFactorToken"]; ok {
		t.Fatal("two-factor token has no clear flag")
	}
}

func TestPanelSettingsSecretRestoreOnly(t *testing.T) {
	t.Parallel()

	base := panelSettingsModel{
		ID:           types.StringValue("panel-settings"),
		WebPort:      types.Int64Value(2053),
		RestartPanel: types.BoolValue(true),
		TgBotToken:   types.StringValue(""),
	}
	restored := base
	restored.TgBotToken = types.StringValue("telegram-secret")
	if !panelSettingsSecretRestoreOnly(base, restored) {
		t.Fatal("restoring a blank redacted token should skip panel restart")
	}

	rotated := base
	rotated.TgBotToken = types.StringValue("telegram-secret")
	changed := rotated
	changed.TgBotToken = types.StringValue("other-secret")
	if panelSettingsSecretRestoreOnly(rotated, changed) {
		t.Fatal("replacing a stored token should still restart when requested")
	}

	withPort := restored
	withPort.WebPort = types.Int64Value(8443)
	if panelSettingsSecretRestoreOnly(base, withPort) {
		t.Fatal("a non-secret change should still restart when requested")
	}

	cleared := base
	cleared.TgBotToken = types.StringValue("telegram-secret")
	if panelSettingsSecretRestoreOnly(cleared, base) {
		t.Fatal("clearing a token should still restart when requested")
	}
}

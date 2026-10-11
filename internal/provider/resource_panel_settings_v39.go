package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// panelSettingsV39Attributes are AllSetting fields added after v3.7.
// sub_profile_mode has no static default: an omitted value is echoed from the panel.
func panelSettingsV39Attributes() map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"sub_profile_mode": schema.StringAttribute{
			MarkdownDescription: "Subscription profile page mode (`subProfileMode`): `none`, `builtin`, or `custom`. Omit this attribute to keep the value stored on the panel. An upgraded panel that already has a profile URL is `custom`; a factory install is `none`. Setting `none` keeps a stored URL but stops sending it.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				preserveUnsetSubProfileMode{},
			},
			Validators: []validator.String{
				stringvalidator.OneOf("none", "builtin", "custom"),
			},
		},
		"reality_scan_candidates": schema.StringAttribute{
			MarkdownDescription: "Comma-separated REALITY scan targets (`realityScanCandidates`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("www.cloudflare.com:443,www.microsoft.com:443,www.amazon.com:443,aws.amazon.com:443,www.samsung.com:443,www.nvidia.com:443,www.amd.com:443,www.intel.com:443,www.sony.com:443,dl.google.com:443"),
		},
		"sub_info_node_enable": schema.BoolAttribute{
			MarkdownDescription: "Include node information in subscription user info (`subInfoNodeEnable`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_calendar_expire_inclusive": schema.BoolAttribute{
			MarkdownDescription: "Treat the expiry calendar day as still valid (`subCalendarExpireInclusive`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_expired_template": schema.StringAttribute{
			MarkdownDescription: "Remark template used after a client expires (`subExpiredTemplate`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("⛔ {{EMAIL}} | Expired: {{EXPIRE_DATE}}"),
		},
		"sub_traffic_depleted_template": schema.StringAttribute{
			MarkdownDescription: "Remark template used after a client uses its traffic (`subTrafficDepletedTemplate`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("🚫 {{EMAIL}} | Traffic Depleted | {{TRAFFIC_USED}}/{{TRAFFIC_TOTAL}}"),
		},
		"external_sub_user_agent": schema.StringAttribute{
			MarkdownDescription: "User-Agent sent when the panel fetches an external subscription (`externalSubUserAgent`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("v2rayNG/1.8.5"),
		},
		"sub_json_routing_rules": schema.StringAttribute{
			MarkdownDescription: "Remote routing source for JSON subscriptions (`subJsonRoutingRules`). A URL or empty, not a JSON object.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_json_dns": schema.StringAttribute{
			MarkdownDescription: "DNS object merged into JSON subscriptions (`subJsonDns`). Accepts empty/null (unset). Semantic JSON equality avoids whitespace and key-order drift.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			PlanModifiers: []planmodifier.String{
				jsonSemanticString(),
			},
		},
		"happ_link_enable": schema.BoolAttribute{
			MarkdownDescription: "Include Happ deep links in subscription output (`happLinkEnable`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"discord_bot_enable": schema.BoolAttribute{
			MarkdownDescription: "Enable the Discord bot (`discordBotEnable`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"discord_bot_token": schema.StringAttribute{
			MarkdownDescription: "Discord bot token (`discordBotToken`).",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			Default:             stringdefault.StaticString(""),
		},
		"discord_channel_id": schema.StringAttribute{
			MarkdownDescription: "Discord channel id for bot messages (`discordChannelId`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"discord_admin_ids": schema.StringAttribute{
			MarkdownDescription: "Comma-separated Discord admin user ids (`discordAdminIds`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"discord_run_time": schema.StringAttribute{
			MarkdownDescription: "Cron schedule for Discord bot jobs (`discordRunTime`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("@daily"),
		},
		"discord_bot_backup": schema.BoolAttribute{
			MarkdownDescription: "Send panel backups to Discord (`discordBotBackup`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"discord_cpu": schema.Int64Attribute{
			MarkdownDescription: "CPU percent that triggers a Discord alert (`discordCpu`).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(80),
		},
		"discord_memory": schema.Int64Attribute{
			MarkdownDescription: "Memory percent that triggers a Discord alert (`discordMemory`).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(80),
		},
		"discord_lang": schema.StringAttribute{
			MarkdownDescription: "Discord bot language (`discordLang`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("en-US"),
		},
		"discord_enabled_events": schema.StringAttribute{
			MarkdownDescription: "Comma-separated Discord notification events (`discordEnabledEvents`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("login.attempt,cpu.high"),
		},
		"sub_happ_auto_detect": schema.BoolAttribute{
			MarkdownDescription: "Auto-detect Happ clients from the subscription User-Agent (`subHappAutoDetect`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_happ_provider_id": schema.StringAttribute{
			MarkdownDescription: "Happ provider id (`subHappProviderId`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_new_url": schema.StringAttribute{
			MarkdownDescription: "Happ new-subscription URL (`subHappNewUrl`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_fallback_url": schema.StringAttribute{
			MarkdownDescription: "Happ fallback URL (`subHappFallbackUrl`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_sub_info_color": schema.StringAttribute{
			MarkdownDescription: "Happ subscription info color (`subHappSubInfoColor`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("blue"),
		},
		"sub_happ_sub_info_text": schema.StringAttribute{
			MarkdownDescription: "Happ subscription info text (`subHappSubInfoText`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_sub_info_button_text": schema.StringAttribute{
			MarkdownDescription: "Happ subscription info button text (`subHappSubInfoButtonText`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_sub_info_button_link": schema.StringAttribute{
			MarkdownDescription: "Happ subscription info button link (`subHappSubInfoButtonLink`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_sub_expire": schema.BoolAttribute{
			MarkdownDescription: "Show Happ expiry information (`subHappSubExpire`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_happ_sub_expire_button_link": schema.StringAttribute{
			MarkdownDescription: "Happ expiry button link (`subHappSubExpireButtonLink`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_notification_expire": schema.BoolAttribute{
			MarkdownDescription: "Send Happ expiry notifications (`subHappNotificationExpire`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_happ_no_limit": schema.BoolAttribute{
			MarkdownDescription: "Disable Happ subscription limits (`subHappNoLimit`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_happ_always_hwid": schema.BoolAttribute{
			MarkdownDescription: "Always require a Happ hardware id (`subHappAlwaysHwid`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_happ_tun_mode": schema.StringAttribute{
			MarkdownDescription: "Happ TUN mode (`subHappTunMode`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_tun_type": schema.StringAttribute{
			MarkdownDescription: "Happ TUN type (`subHappTunType`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_exclude_routes": schema.StringAttribute{
			MarkdownDescription: "Happ excluded routes (`subHappExcludeRoutes`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_exclude_apns": schema.BoolAttribute{
			MarkdownDescription: "Exclude Apple push routes from Happ (`subHappExcludeApns`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_happ_color_profile": schema.StringAttribute{
			MarkdownDescription: "Happ color profile (`subHappColorProfile`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_ping_type": schema.StringAttribute{
			MarkdownDescription: "Happ ping type (`subHappPingType`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_auto_connect": schema.BoolAttribute{
			MarkdownDescription: "Enable Happ auto-connect (`subHappAutoConnect`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_happ_auto_connect_type": schema.StringAttribute{
			MarkdownDescription: "Happ auto-connect strategy (`subHappAutoConnectType`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("lowestdelay"),
		},
		"sub_happ_per_app_mode": schema.StringAttribute{
			MarkdownDescription: "Happ per-app proxy mode (`subHappPerAppMode`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("off"),
		},
		"sub_happ_per_app_list": schema.StringAttribute{
			MarkdownDescription: "Happ per-app package list (`subHappPerAppList`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_happ_local_proxy_auth": schema.StringAttribute{
			MarkdownDescription: "Happ local proxy authentication (`subHappLocalProxyAuth`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("auto"),
		},
		"sub_incy_app_auto_detect": schema.BoolAttribute{
			MarkdownDescription: "Auto-detect Incy clients from the subscription User-Agent (`subIncyAppAutoDetect`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"sub_incy_profile_description": schema.StringAttribute{
			MarkdownDescription: "Incy profile description (`subIncyProfileDescription`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_sort_order": schema.StringAttribute{
			MarkdownDescription: "Incy profile sort order (`subIncySortOrder`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_support_email": schema.StringAttribute{
			MarkdownDescription: "Incy support email (`subIncySupportEmail`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_announce_url": schema.StringAttribute{
			MarkdownDescription: "Incy announcement URL (`subIncyAnnounceUrl`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_premium_url": schema.StringAttribute{
			MarkdownDescription: "Incy premium URL (`subIncyPremiumUrl`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_banner_text": schema.StringAttribute{
			MarkdownDescription: "Incy banner text (`subIncyBannerText`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_banner_button_text": schema.StringAttribute{
			MarkdownDescription: "Incy banner button text (`subIncyBannerButtonText`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_banner_button_url": schema.StringAttribute{
			MarkdownDescription: "Incy banner button URL (`subIncyBannerButtonUrl`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_banner_bg_color": schema.StringAttribute{
			MarkdownDescription: "Incy banner background color (`subIncyBannerBgColor`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_banner_button_color": schema.StringAttribute{
			MarkdownDescription: "Incy banner button color (`subIncyBannerButtonColor`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_hide_url": schema.StringAttribute{
			MarkdownDescription: "Incy hide-url flag (`subIncyHideUrl`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_hide_check": schema.StringAttribute{
			MarkdownDescription: "Incy hide-check flag (`subIncyHideCheck`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_no_limit_enabled": schema.StringAttribute{
			MarkdownDescription: "Incy no-limit flag (`subIncyNoLimitEnabled`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_per_app_enable": schema.StringAttribute{
			MarkdownDescription: "Incy per-app enable flag (`subIncyPerAppEnable`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_per_app_mode": schema.StringAttribute{
			MarkdownDescription: "Incy per-app mode (`subIncyPerAppMode`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_per_app_list": schema.StringAttribute{
			MarkdownDescription: "Incy per-app list (`subIncyPerAppList`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_fragmentation_enable": schema.StringAttribute{
			MarkdownDescription: "Incy fragmentation enable flag (`subIncyFragmentationEnable`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_fragment_length": schema.StringAttribute{
			MarkdownDescription: "Incy fragment length (`subIncyFragmentLength`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_fragment_interval": schema.StringAttribute{
			MarkdownDescription: "Incy fragment interval (`subIncyFragmentInterval`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_fragment_packets": schema.StringAttribute{
			MarkdownDescription: "Incy fragment packets (`subIncyFragmentPackets`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_noises_enable": schema.StringAttribute{
			MarkdownDescription: "Incy noises enable flag (`subIncyNoisesEnable`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_noises_type": schema.StringAttribute{
			MarkdownDescription: "Incy noises type (`subIncyNoisesType`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_noises_packet": schema.StringAttribute{
			MarkdownDescription: "Incy noises packet (`subIncyNoisesPacket`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_noises_delay": schema.StringAttribute{
			MarkdownDescription: "Incy noises delay (`subIncyNoisesDelay`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_resolve_enable": schema.StringAttribute{
			MarkdownDescription: "Incy DNS resolve enable flag (`subIncyResolveEnable`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_resolve_dns_domain": schema.StringAttribute{
			MarkdownDescription: "Incy resolve DNS domain (`subIncyResolveDnsDomain`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sub_incy_resolve_dns_ip": schema.StringAttribute{
			MarkdownDescription: "Incy resolve DNS IP (`subIncyResolveDnsIp`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
	}
	return attrs
}

func init() {
	for _, key := range panelSettingsV39OptionalKeys {
		panelSettingsOptionalKeys[key] = struct{}{}
	}
}

var panelSettingsV39OptionalKeys = []string{
	"subProfileMode",
	"realityScanCandidates",
	"subInfoNodeEnable",
	"subCalendarExpireInclusive",
	"subExpiredTemplate",
	"subTrafficDepletedTemplate",
	"externalSubUserAgent",
	"subJsonRoutingRules",
	"subJsonDns",
	"happLinkEnable",
	"discordBotEnable",
	"discordBotToken",
	"discordChannelId",
	"discordAdminIds",
	"discordRunTime",
	"discordBotBackup",
	"discordCpu",
	"discordMemory",
	"discordLang",
	"discordEnabledEvents",
	"subHappAutoDetect",
	"subHappProviderId",
	"subHappNewUrl",
	"subHappFallbackUrl",
	"subHappSubInfoColor",
	"subHappSubInfoText",
	"subHappSubInfoButtonText",
	"subHappSubInfoButtonLink",
	"subHappSubExpire",
	"subHappSubExpireButtonLink",
	"subHappNotificationExpire",
	"subHappNoLimit",
	"subHappAlwaysHwid",
	"subHappTunMode",
	"subHappTunType",
	"subHappExcludeRoutes",
	"subHappExcludeApns",
	"subHappColorProfile",
	"subHappPingType",
	"subHappAutoConnect",
	"subHappAutoConnectType",
	"subHappPerAppMode",
	"subHappPerAppList",
	"subHappLocalProxyAuth",
	"subIncyAppAutoDetect",
	"subIncyProfileDescription",
	"subIncySortOrder",
	"subIncySupportEmail",
	"subIncyAnnounceUrl",
	"subIncyPremiumUrl",
	"subIncyBannerText",
	"subIncyBannerButtonText",
	"subIncyBannerButtonUrl",
	"subIncyBannerBgColor",
	"subIncyBannerButtonColor",
	"subIncyHideUrl",
	"subIncyHideCheck",
	"subIncyNoLimitEnabled",
	"subIncyPerAppEnable",
	"subIncyPerAppMode",
	"subIncyPerAppList",
	"subIncyFragmentationEnable",
	"subIncyFragmentLength",
	"subIncyFragmentInterval",
	"subIncyFragmentPackets",
	"subIncyNoisesEnable",
	"subIncyNoisesType",
	"subIncyNoisesPacket",
	"subIncyNoisesDelay",
	"subIncyResolveEnable",
	"subIncyResolveDnsDomain",
	"subIncyResolveDnsIp",
}

func appendPanelSettingsV39(m *panelSettingsModel, payload map[string]any) {
	if !m.SubProfileMode.IsNull() && !m.SubProfileMode.IsUnknown() && m.SubProfileMode.ValueString() != "" {
		payload["subProfileMode"] = m.SubProfileMode.ValueString()
	}
	payload["realityScanCandidates"] = m.RealityScanCandidates.ValueString()
	payload["subInfoNodeEnable"] = m.SubInfoNodeEnable.ValueBool()
	payload["subCalendarExpireInclusive"] = m.SubCalendarExpireInclusive.ValueBool()
	payload["subExpiredTemplate"] = m.SubExpiredTemplate.ValueString()
	payload["subTrafficDepletedTemplate"] = m.SubTrafficDepletedTemplate.ValueString()
	payload["externalSubUserAgent"] = m.ExternalSubUserAgent.ValueString()
	payload["subJsonRoutingRules"] = m.SubJSONRoutingRules.ValueString()
	payload["subJsonDns"] = panelJSONWireValue(m.SubJSONDNS.ValueString())
	payload["happLinkEnable"] = m.HappLinkEnable.ValueBool()
	payload["discordBotEnable"] = m.DiscordBotEnable.ValueBool()
	payload["discordBotToken"] = m.DiscordBotToken.ValueString()
	payload["discordChannelId"] = m.DiscordChannelID.ValueString()
	payload["discordAdminIds"] = m.DiscordAdminIDs.ValueString()
	payload["discordRunTime"] = m.DiscordRunTime.ValueString()
	payload["discordBotBackup"] = m.DiscordBotBackup.ValueBool()
	payload["discordCpu"] = m.DiscordCPU.ValueInt64()
	payload["discordMemory"] = m.DiscordMemory.ValueInt64()
	payload["discordLang"] = m.DiscordLang.ValueString()
	payload["discordEnabledEvents"] = m.DiscordEnabledEvents.ValueString()
	payload["subHappAutoDetect"] = m.SubHappAutoDetect.ValueBool()
	payload["subHappProviderId"] = m.SubHappProviderID.ValueString()
	payload["subHappNewUrl"] = m.SubHappNewURL.ValueString()
	payload["subHappFallbackUrl"] = m.SubHappFallbackURL.ValueString()
	payload["subHappSubInfoColor"] = m.SubHappSubInfoColor.ValueString()
	payload["subHappSubInfoText"] = m.SubHappSubInfoText.ValueString()
	payload["subHappSubInfoButtonText"] = m.SubHappSubInfoButtonText.ValueString()
	payload["subHappSubInfoButtonLink"] = m.SubHappSubInfoButtonLink.ValueString()
	payload["subHappSubExpire"] = m.SubHappSubExpire.ValueBool()
	payload["subHappSubExpireButtonLink"] = m.SubHappSubExpireButtonLink.ValueString()
	payload["subHappNotificationExpire"] = m.SubHappNotificationExpire.ValueBool()
	payload["subHappNoLimit"] = m.SubHappNoLimit.ValueBool()
	payload["subHappAlwaysHwid"] = m.SubHappAlwaysHwid.ValueBool()
	payload["subHappTunMode"] = m.SubHappTunMode.ValueString()
	payload["subHappTunType"] = m.SubHappTunType.ValueString()
	payload["subHappExcludeRoutes"] = m.SubHappExcludeRoutes.ValueString()
	payload["subHappExcludeApns"] = m.SubHappExcludeApns.ValueBool()
	payload["subHappColorProfile"] = m.SubHappColorProfile.ValueString()
	payload["subHappPingType"] = m.SubHappPingType.ValueString()
	payload["subHappAutoConnect"] = m.SubHappAutoConnect.ValueBool()
	payload["subHappAutoConnectType"] = m.SubHappAutoConnectType.ValueString()
	payload["subHappPerAppMode"] = m.SubHappPerAppMode.ValueString()
	payload["subHappPerAppList"] = m.SubHappPerAppList.ValueString()
	payload["subHappLocalProxyAuth"] = m.SubHappLocalProxyAuth.ValueString()
	payload["subIncyAppAutoDetect"] = m.SubIncyAppAutoDetect.ValueBool()
	payload["subIncyProfileDescription"] = m.SubIncyProfileDescription.ValueString()
	payload["subIncySortOrder"] = m.SubIncySortOrder.ValueString()
	payload["subIncySupportEmail"] = m.SubIncySupportEmail.ValueString()
	payload["subIncyAnnounceUrl"] = m.SubIncyAnnounceURL.ValueString()
	payload["subIncyPremiumUrl"] = m.SubIncyPremiumURL.ValueString()
	payload["subIncyBannerText"] = m.SubIncyBannerText.ValueString()
	payload["subIncyBannerButtonText"] = m.SubIncyBannerButtonText.ValueString()
	payload["subIncyBannerButtonUrl"] = m.SubIncyBannerButtonURL.ValueString()
	payload["subIncyBannerBgColor"] = m.SubIncyBannerBgColor.ValueString()
	payload["subIncyBannerButtonColor"] = m.SubIncyBannerButtonColor.ValueString()
	payload["subIncyHideUrl"] = m.SubIncyHideURL.ValueString()
	payload["subIncyHideCheck"] = m.SubIncyHideCheck.ValueString()
	payload["subIncyNoLimitEnabled"] = m.SubIncyNoLimitEnabled.ValueString()
	payload["subIncyPerAppEnable"] = m.SubIncyPerAppEnable.ValueString()
	payload["subIncyPerAppMode"] = m.SubIncyPerAppMode.ValueString()
	payload["subIncyPerAppList"] = m.SubIncyPerAppList.ValueString()
	payload["subIncyFragmentationEnable"] = m.SubIncyFragmentationEnable.ValueString()
	payload["subIncyFragmentLength"] = m.SubIncyFragmentLength.ValueString()
	payload["subIncyFragmentInterval"] = m.SubIncyFragmentInterval.ValueString()
	payload["subIncyFragmentPackets"] = m.SubIncyFragmentPackets.ValueString()
	payload["subIncyNoisesEnable"] = m.SubIncyNoisesEnable.ValueString()
	payload["subIncyNoisesType"] = m.SubIncyNoisesType.ValueString()
	payload["subIncyNoisesPacket"] = m.SubIncyNoisesPacket.ValueString()
	payload["subIncyNoisesDelay"] = m.SubIncyNoisesDelay.ValueString()
	payload["subIncyResolveEnable"] = m.SubIncyResolveEnable.ValueString()
	payload["subIncyResolveDnsDomain"] = m.SubIncyResolveDNSDomain.ValueString()
	payload["subIncyResolveDnsIp"] = m.SubIncyResolveDNSIP.ValueString()
}

func applyPanelSettingsV39(raw map[string]any, m *panelSettingsModel) {
	if panelSettingsHasKey(raw, "subProfileMode") {
		mode := stringFromMap(raw, "subProfileMode")
		if mode == "" {
			mode = "none"
		}
		m.SubProfileMode = types.StringValue(mode)
	} else if m.SubProfileMode.IsNull() || m.SubProfileMode.IsUnknown() {
		m.SubProfileMode = types.StringValue("none")
	}
	assignOptionalString(raw, "realityScanCandidates", &m.RealityScanCandidates, "www.cloudflare.com:443,www.microsoft.com:443,www.amazon.com:443,aws.amazon.com:443,www.samsung.com:443,www.nvidia.com:443,www.amd.com:443,www.intel.com:443,www.sony.com:443,dl.google.com:443")
	assignOptionalBool(raw, "subInfoNodeEnable", &m.SubInfoNodeEnable, false)
	assignOptionalBool(raw, "subCalendarExpireInclusive", &m.SubCalendarExpireInclusive, false)
	assignOptionalString(raw, "subExpiredTemplate", &m.SubExpiredTemplate, "⛔ {{EMAIL}} | Expired: {{EXPIRE_DATE}}")
	assignOptionalString(raw, "subTrafficDepletedTemplate", &m.SubTrafficDepletedTemplate, "🚫 {{EMAIL}} | Traffic Depleted | {{TRAFFIC_USED}}/{{TRAFFIC_TOTAL}}")
	assignOptionalString(raw, "externalSubUserAgent", &m.ExternalSubUserAgent, "v2rayNG/1.8.5")
	assignOptionalString(raw, "subJsonRoutingRules", &m.SubJSONRoutingRules, "")
	assignOptionalString(raw, "subJsonDns", &m.SubJSONDNS, "")
	assignOptionalBool(raw, "happLinkEnable", &m.HappLinkEnable, false)
	assignOptionalBool(raw, "discordBotEnable", &m.DiscordBotEnable, false)
	assignOptionalString(raw, "discordBotToken", &m.DiscordBotToken, "")
	assignOptionalString(raw, "discordChannelId", &m.DiscordChannelID, "")
	assignOptionalString(raw, "discordAdminIds", &m.DiscordAdminIDs, "")
	assignOptionalString(raw, "discordRunTime", &m.DiscordRunTime, "@daily")
	assignOptionalBool(raw, "discordBotBackup", &m.DiscordBotBackup, false)
	assignOptionalInt64(raw, "discordCpu", &m.DiscordCPU, 80)
	assignOptionalInt64(raw, "discordMemory", &m.DiscordMemory, 80)
	assignOptionalString(raw, "discordLang", &m.DiscordLang, "en-US")
	assignOptionalString(raw, "discordEnabledEvents", &m.DiscordEnabledEvents, "login.attempt,cpu.high")
	assignOptionalBool(raw, "subHappAutoDetect", &m.SubHappAutoDetect, false)
	assignOptionalString(raw, "subHappProviderId", &m.SubHappProviderID, "")
	assignOptionalString(raw, "subHappNewUrl", &m.SubHappNewURL, "")
	assignOptionalString(raw, "subHappFallbackUrl", &m.SubHappFallbackURL, "")
	assignOptionalString(raw, "subHappSubInfoColor", &m.SubHappSubInfoColor, "blue")
	assignOptionalString(raw, "subHappSubInfoText", &m.SubHappSubInfoText, "")
	assignOptionalString(raw, "subHappSubInfoButtonText", &m.SubHappSubInfoButtonText, "")
	assignOptionalString(raw, "subHappSubInfoButtonLink", &m.SubHappSubInfoButtonLink, "")
	assignOptionalBool(raw, "subHappSubExpire", &m.SubHappSubExpire, false)
	assignOptionalString(raw, "subHappSubExpireButtonLink", &m.SubHappSubExpireButtonLink, "")
	assignOptionalBool(raw, "subHappNotificationExpire", &m.SubHappNotificationExpire, false)
	assignOptionalBool(raw, "subHappNoLimit", &m.SubHappNoLimit, false)
	assignOptionalBool(raw, "subHappAlwaysHwid", &m.SubHappAlwaysHwid, false)
	assignOptionalString(raw, "subHappTunMode", &m.SubHappTunMode, "")
	assignOptionalString(raw, "subHappTunType", &m.SubHappTunType, "")
	assignOptionalString(raw, "subHappExcludeRoutes", &m.SubHappExcludeRoutes, "")
	assignOptionalBool(raw, "subHappExcludeApns", &m.SubHappExcludeApns, false)
	assignOptionalString(raw, "subHappColorProfile", &m.SubHappColorProfile, "")
	assignOptionalString(raw, "subHappPingType", &m.SubHappPingType, "")
	assignOptionalBool(raw, "subHappAutoConnect", &m.SubHappAutoConnect, false)
	assignOptionalString(raw, "subHappAutoConnectType", &m.SubHappAutoConnectType, "lowestdelay")
	assignOptionalString(raw, "subHappPerAppMode", &m.SubHappPerAppMode, "off")
	assignOptionalString(raw, "subHappPerAppList", &m.SubHappPerAppList, "")
	assignOptionalString(raw, "subHappLocalProxyAuth", &m.SubHappLocalProxyAuth, "auto")
	assignOptionalBool(raw, "subIncyAppAutoDetect", &m.SubIncyAppAutoDetect, false)
	assignOptionalString(raw, "subIncyProfileDescription", &m.SubIncyProfileDescription, "")
	assignOptionalString(raw, "subIncySortOrder", &m.SubIncySortOrder, "")
	assignOptionalString(raw, "subIncySupportEmail", &m.SubIncySupportEmail, "")
	assignOptionalString(raw, "subIncyAnnounceUrl", &m.SubIncyAnnounceURL, "")
	assignOptionalString(raw, "subIncyPremiumUrl", &m.SubIncyPremiumURL, "")
	assignOptionalString(raw, "subIncyBannerText", &m.SubIncyBannerText, "")
	assignOptionalString(raw, "subIncyBannerButtonText", &m.SubIncyBannerButtonText, "")
	assignOptionalString(raw, "subIncyBannerButtonUrl", &m.SubIncyBannerButtonURL, "")
	assignOptionalString(raw, "subIncyBannerBgColor", &m.SubIncyBannerBgColor, "")
	assignOptionalString(raw, "subIncyBannerButtonColor", &m.SubIncyBannerButtonColor, "")
	assignOptionalString(raw, "subIncyHideUrl", &m.SubIncyHideURL, "")
	assignOptionalString(raw, "subIncyHideCheck", &m.SubIncyHideCheck, "")
	assignOptionalString(raw, "subIncyNoLimitEnabled", &m.SubIncyNoLimitEnabled, "")
	assignOptionalString(raw, "subIncyPerAppEnable", &m.SubIncyPerAppEnable, "")
	assignOptionalString(raw, "subIncyPerAppMode", &m.SubIncyPerAppMode, "")
	assignOptionalString(raw, "subIncyPerAppList", &m.SubIncyPerAppList, "")
	assignOptionalString(raw, "subIncyFragmentationEnable", &m.SubIncyFragmentationEnable, "")
	assignOptionalString(raw, "subIncyFragmentLength", &m.SubIncyFragmentLength, "")
	assignOptionalString(raw, "subIncyFragmentInterval", &m.SubIncyFragmentInterval, "")
	assignOptionalString(raw, "subIncyFragmentPackets", &m.SubIncyFragmentPackets, "")
	assignOptionalString(raw, "subIncyNoisesEnable", &m.SubIncyNoisesEnable, "")
	assignOptionalString(raw, "subIncyNoisesType", &m.SubIncyNoisesType, "")
	assignOptionalString(raw, "subIncyNoisesPacket", &m.SubIncyNoisesPacket, "")
	assignOptionalString(raw, "subIncyNoisesDelay", &m.SubIncyNoisesDelay, "")
	assignOptionalString(raw, "subIncyResolveEnable", &m.SubIncyResolveEnable, "")
	assignOptionalString(raw, "subIncyResolveDnsDomain", &m.SubIncyResolveDNSDomain, "")
	assignOptionalString(raw, "subIncyResolveDnsIp", &m.SubIncyResolveDNSIP, "")
	if panelSettingsHasKey(raw, "subJsonDns") {
		m.SubJSONDNS = types.StringValue(panelJSONStateValue(m.SubJSONDNS.ValueString()))
	}
}

func (r *panelSettingsResource) finalizeSubProfileMode(m *panelSettingsModel) error {
	if !m.SubProfileMode.IsNull() && !m.SubProfileMode.IsUnknown() {
		return nil
	}
	raw, err := r.client.GetPanelSettings()
	if err != nil {
		return err
	}
	mode := "none"
	if panelSettingsHasKey(raw, "subProfileMode") {
		if s := stringFromMap(raw, "subProfileMode"); s != "" {
			mode = s
		}
	}
	m.SubProfileMode = types.StringValue(mode)
	return nil
}

type preserveUnsetSubProfileMode struct{}

func (preserveUnsetSubProfileMode) Description(context.Context) string {
	return "Keeps the stored subscription profile mode when the attribute is omitted from configuration."
}

func (preserveUnsetSubProfileMode) MarkdownDescription(context.Context) string {
	return "Keeps the stored subscription profile mode when the attribute is omitted from configuration."
}

func (preserveUnsetSubProfileMode) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		return
	}
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		return
	}
	resp.PlanValue = types.StringUnknown()
}

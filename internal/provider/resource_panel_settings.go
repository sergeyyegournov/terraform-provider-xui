package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ resource.Resource = (*panelSettingsResource)(nil)
var _ resource.ResourceWithImportState = (*panelSettingsResource)(nil)

const panelSettingsJSONAttrNote = " Accepts empty/null (unset). Semantic JSON equality avoids whitespace and key-order drift."

type panelSettingsResource struct {
	client *xui.Client
}

func NewPanelSettingsResource() resource.Resource {
	return &panelSettingsResource{}
}

func (r *panelSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_panel_settings"
}

func (r *panelSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages 3x-ui panel settings (`/panel/api/setting/update`). This is a singleton resource — only one instance should exist per panel. All attributes are optional and default to the panel's built-in defaults. Set `restart_panel` to true if you want to restart the panel after applying changes (required for web listen/port/cert changes to take effect). Attributes mirror the panel's `AllSetting` model (Telegram, SMTP, LDAP, subscription, etc.).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Static resource id (`panel-settings`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Web server
			"web_listen": schema.StringAttribute{
				MarkdownDescription: "Web panel listen IP address (empty = all interfaces).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"web_domain": schema.StringAttribute{
				MarkdownDescription: "Web panel domain for validation.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"web_port": schema.Int64Attribute{
				MarkdownDescription: "Web panel port.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2053),
			},
			"web_cert_file": schema.StringAttribute{
				MarkdownDescription: "Path to SSL certificate file for the web panel.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"web_key_file": schema.StringAttribute{
				MarkdownDescription: "Path to SSL private key file for the web panel.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"web_base_path": schema.StringAttribute{
				MarkdownDescription: "Base path for panel URLs (e.g. `/<uuid>/`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/"),
			},
			"trusted_proxy_cidrs": schema.StringAttribute{
				MarkdownDescription: "Trusted reverse-proxy CIDRs used for forwarded headers.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("127.0.0.1/32,::1/128"),
			},
			"panel_proxy": schema.StringAttribute{
				MarkdownDescription: "Proxy URL used by the panel for outbound requests (`panelOutbound`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"session_max_age": schema.Int64Attribute{
				MarkdownDescription: "Session maximum age in minutes.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
			},

			// UI
			"page_size": schema.Int64Attribute{
				MarkdownDescription: "Number of items per page in lists.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(50),
			},
			"expire_diff": schema.Int64Attribute{
				MarkdownDescription: "Expiration warning threshold in days.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"traffic_diff": schema.Int64Attribute{
				MarkdownDescription: "Traffic warning threshold percentage.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"remark_model": schema.StringAttribute{
				MarkdownDescription: "Remark template for inbounds (`remarkTemplate`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"datepicker": schema.StringAttribute{
				MarkdownDescription: "Date picker format.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("gregorian"),
			},

			// Telegram bot
			"tg_bot_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable Telegram bot notifications.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"tg_bot_token": schema.StringAttribute{
				MarkdownDescription: "Telegram bot token.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Default:             stringdefault.StaticString(""),
			},
			"tg_bot_proxy": schema.StringAttribute{
				MarkdownDescription: "Proxy URL for Telegram bot.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"tg_bot_api_server": schema.StringAttribute{
				MarkdownDescription: "Custom API server for Telegram bot.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"tg_bot_chat_id": schema.StringAttribute{
				MarkdownDescription: "Telegram chat ID for notifications.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"tg_run_time": schema.StringAttribute{
				MarkdownDescription: "Cron schedule for Telegram notifications (e.g. `@daily`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("@daily"),
			},
			"tg_bot_backup": schema.BoolAttribute{
				MarkdownDescription: "Enable database backup via Telegram.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"tg_cpu": schema.Int64Attribute{
				MarkdownDescription: "CPU usage threshold percentage for Telegram alerts.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(80),
			},
			"tg_memory": schema.Int64Attribute{
				MarkdownDescription: "Memory usage threshold percentage for Telegram alerts.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(80),
			},
			"tg_lang": schema.StringAttribute{
				MarkdownDescription: "Telegram bot language.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("en-US"),
			},
			"tg_enabled_events": schema.StringAttribute{
				MarkdownDescription: "Comma-separated Telegram notification events (e.g. `login.attempt,cpu.high`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("login.attempt,cpu.high"),
			},

			// SMTP
			"smtp_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable SMTP email notifications.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"smtp_host": schema.StringAttribute{
				MarkdownDescription: "SMTP server host.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"smtp_port": schema.Int64Attribute{
				MarkdownDescription: "SMTP server port.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(587),
			},
			"smtp_username": schema.StringAttribute{
				MarkdownDescription: "SMTP username.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"smtp_password": schema.StringAttribute{
				MarkdownDescription: "SMTP password.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Default:             stringdefault.StaticString(""),
			},
			"smtp_from": schema.StringAttribute{
				MarkdownDescription: "SMTP From address.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"smtp_from_name": schema.StringAttribute{
				MarkdownDescription: "SMTP From display name.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"smtp_to": schema.StringAttribute{
				MarkdownDescription: "SMTP recipient address(es).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"smtp_encryption_type": schema.StringAttribute{
				MarkdownDescription: "SMTP encryption type (`no`, `starttls`, or `tls`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("starttls"),
			},
			"smtp_enabled_events": schema.StringAttribute{
				MarkdownDescription: "Comma-separated SMTP notification events (e.g. `login.attempt,cpu.high`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("login.attempt,cpu.high"),
			},
			"smtp_cpu": schema.Int64Attribute{
				MarkdownDescription: "CPU usage threshold percentage for SMTP alerts.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(80),
			},
			"smtp_memory": schema.Int64Attribute{
				MarkdownDescription: "Memory usage threshold percentage for SMTP alerts.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(80),
			},

			"outbound_down_threshold": schema.Int64Attribute{
				MarkdownDescription: "Consecutive failed observatory probes before an outbound.down event fires.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
			},

			// Security
			"time_location": schema.StringAttribute{
				MarkdownDescription: "Time zone location (e.g. `UTC`, `Asia/Tehran`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
			},
			"two_factor_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable two-factor authentication for the panel.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"two_factor_token": schema.StringAttribute{
				MarkdownDescription: "Two-factor authentication secret / token.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Default:             stringdefault.StaticString(""),
			},

			// LDAP
			"ldap_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable LDAP authentication.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ldap_host": schema.StringAttribute{
				MarkdownDescription: "LDAP server host.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_port": schema.Int64Attribute{
				MarkdownDescription: "LDAP server port.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(389),
			},
			"ldap_use_tls": schema.BoolAttribute{
				MarkdownDescription: "Use TLS for LDAP connections.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ldap_insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS certificate verification for LDAP connections.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ldap_bind_dn": schema.StringAttribute{
				MarkdownDescription: "LDAP bind DN.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_password": schema.StringAttribute{
				MarkdownDescription: "LDAP bind password.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_base_dn": schema.StringAttribute{
				MarkdownDescription: "LDAP base DN for searches.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_user_filter": schema.StringAttribute{
				MarkdownDescription: "LDAP user search filter.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_user_attr": schema.StringAttribute{
				MarkdownDescription: "LDAP attribute for username (e.g. `mail` or `uid`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_vless_field": schema.StringAttribute{
				MarkdownDescription: "LDAP attribute mapped to VLESS identity.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_sync_cron": schema.StringAttribute{
				MarkdownDescription: "Cron schedule for LDAP sync.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_flag_field": schema.StringAttribute{
				MarkdownDescription: "LDAP attribute used as a generic flag.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_truthy_values": schema.StringAttribute{
				MarkdownDescription: "Comma-separated values treated as true for the flag field.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_invert_flag": schema.BoolAttribute{
				MarkdownDescription: "Invert LDAP flag interpretation.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ldap_inbound_tags": schema.StringAttribute{
				MarkdownDescription: "Inbound tags for LDAP-provisioned users.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ldap_auto_create": schema.BoolAttribute{
				MarkdownDescription: "Automatically create clients from LDAP.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ldap_auto_delete": schema.BoolAttribute{
				MarkdownDescription: "Automatically delete clients removed from LDAP.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ldap_default_total_gb": schema.Int64Attribute{
				MarkdownDescription: "Default traffic limit in GB for LDAP-created clients (0 = unlimited).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"ldap_default_expiry_days": schema.Int64Attribute{
				MarkdownDescription: "Default account expiry in days for LDAP-created clients (0 = never).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"ldap_default_limit_ip": schema.Int64Attribute{
				MarkdownDescription: "Default IP limit for LDAP-created clients.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},

			// Subscription server
			"sub_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable subscription server.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_json_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable JSON subscription endpoint.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_json_auto_detect": schema.BoolAttribute{
				MarkdownDescription: "Auto-detect clients that should receive JSON subscriptions.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_json_always_array": schema.BoolAttribute{
				MarkdownDescription: "Always return JSON subscriptions as an array.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_json_user_agent_regex": schema.StringAttribute{
				MarkdownDescription: "User-Agent regex for JSON subscription auto-detect.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_clash_auto_detect": schema.BoolAttribute{
				MarkdownDescription: "Auto-detect clients that should receive Clash/Mihomo subscriptions.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_clash_user_agent_regex": schema.StringAttribute{
				MarkdownDescription: "User-Agent regex for Clash/Mihomo subscription auto-detect.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_title": schema.StringAttribute{
				MarkdownDescription: "Subscription title.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_support_url": schema.StringAttribute{
				MarkdownDescription: "Subscription support URL.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_profile_url": schema.StringAttribute{
				MarkdownDescription: "Subscription profile URL.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_announce": schema.StringAttribute{
				MarkdownDescription: "Subscription announcement.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_listen": schema.StringAttribute{
				MarkdownDescription: "Subscription server listen IP.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_port": schema.Int64Attribute{
				MarkdownDescription: "Subscription server port.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2096),
			},
			"sub_path": schema.StringAttribute{
				MarkdownDescription: "Base path for subscription URLs.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/sub/"),
			},
			"sub_domain": schema.StringAttribute{
				MarkdownDescription: "Domain for subscription server validation.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_cert_file": schema.StringAttribute{
				MarkdownDescription: "SSL certificate file for subscription server.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_key_file": schema.StringAttribute{
				MarkdownDescription: "SSL private key file for subscription server.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_updates": schema.Int64Attribute{
				MarkdownDescription: "Subscription update interval in minutes.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(12),
			},
			"sub_encrypt": schema.BoolAttribute{
				MarkdownDescription: "Encrypt subscription responses.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_uri": schema.StringAttribute{
				MarkdownDescription: "Subscription server URI.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_json_path": schema.StringAttribute{
				MarkdownDescription: "Path for JSON subscription endpoint.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/json/"),
			},
			"sub_json_uri": schema.StringAttribute{
				MarkdownDescription: "JSON subscription server URI.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_json_mux": schema.StringAttribute{
				MarkdownDescription: "JSON subscription mux configuration." + panelSettingsJSONAttrNote,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					jsonSemanticString(),
				},
			},
			"sub_json_rules": schema.StringAttribute{
				MarkdownDescription: "JSON subscription routing rules." + panelSettingsJSONAttrNote,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					jsonSemanticString(),
				},
			},
			"sub_json_final_mask": schema.StringAttribute{
				MarkdownDescription: "JSON subscription FinalMask configuration." + panelSettingsJSONAttrNote,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					jsonSemanticString(),
				},
			},
			"sub_enable_routing": schema.BoolAttribute{
				MarkdownDescription: "Enable routing for subscription.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_routing_rules": schema.StringAttribute{
				MarkdownDescription: "Subscription global routing rules (plain text, not JSON).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_incy_enable_routing": schema.BoolAttribute{
				MarkdownDescription: "Enable Incy routing for subscription.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_incy_routing_rules": schema.StringAttribute{
				MarkdownDescription: "Incy routing rules (plain text).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_clash_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable Clash/Mihomo subscription endpoint.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"sub_clash_path": schema.StringAttribute{
				MarkdownDescription: "Path for Clash/Mihomo subscription endpoint.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/clash/"),
			},
			"sub_clash_uri": schema.StringAttribute{
				MarkdownDescription: "Clash/Mihomo subscription URI.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_clash_enable_routing": schema.BoolAttribute{
				MarkdownDescription: "Enable Clash/Mihomo subscription routing rules.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sub_clash_rules": schema.StringAttribute{
				MarkdownDescription: "Clash/Mihomo subscription routing rules (plain text).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_theme_dir": schema.StringAttribute{
				MarkdownDescription: "Directory for custom subscription theme assets.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sub_hide_settings": schema.BoolAttribute{
				MarkdownDescription: "Hide settings section in subscription pages.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"restart_xray_on_client_disable": schema.BoolAttribute{
				MarkdownDescription: "Restart Xray when clients are auto-disabled by expiry/traffic limits.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"external_traffic_inform_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable external traffic reporting.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"external_traffic_inform_uri": schema.StringAttribute{
				MarkdownDescription: "URI for external traffic reporting.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"warp_update_interval": schema.Int64Attribute{
				MarkdownDescription: "WARP account update interval in hours (0 = disabled).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},

			// Restart
			"restart_panel": schema.BoolAttribute{
				MarkdownDescription: "If true, restart the panel after applying settings changes. Required for web listen/port/cert changes to take effect.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *panelSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cli, ok := req.ProviderData.(*xui.Client)
	if !ok {
		resp.Diagnostics.AddError("Internal error", "invalid provider data type")
		return
	}
	r.client = cli
}

type panelSettingsModel struct {
	ID types.String `tfsdk:"id"`

	// Web
	WebListen         types.String `tfsdk:"web_listen"`
	WebDomain         types.String `tfsdk:"web_domain"`
	WebPort           types.Int64  `tfsdk:"web_port"`
	WebCertFile       types.String `tfsdk:"web_cert_file"`
	WebKeyFile        types.String `tfsdk:"web_key_file"`
	WebBasePath       types.String `tfsdk:"web_base_path"`
	TrustedProxyCIDRs types.String `tfsdk:"trusted_proxy_cidrs"`
	PanelProxy        types.String `tfsdk:"panel_proxy"`
	SessionMaxAge     types.Int64  `tfsdk:"session_max_age"`

	// UI
	PageSize    types.Int64  `tfsdk:"page_size"`
	ExpireDiff  types.Int64  `tfsdk:"expire_diff"`
	TrafficDiff types.Int64  `tfsdk:"traffic_diff"`
	RemarkModel types.String `tfsdk:"remark_model"`
	Datepicker  types.String `tfsdk:"datepicker"`

	// Telegram
	TgBotEnable     types.Bool   `tfsdk:"tg_bot_enable"`
	TgBotToken      types.String `tfsdk:"tg_bot_token"`
	TgBotProxy      types.String `tfsdk:"tg_bot_proxy"`
	TgBotAPIServer  types.String `tfsdk:"tg_bot_api_server"`
	TgBotChatID     types.String `tfsdk:"tg_bot_chat_id"`
	TgRunTime       types.String `tfsdk:"tg_run_time"`
	TgBotBackup     types.Bool   `tfsdk:"tg_bot_backup"`
	TgCPU           types.Int64  `tfsdk:"tg_cpu"`
	TgMemory        types.Int64  `tfsdk:"tg_memory"`
	TgLang          types.String `tfsdk:"tg_lang"`
	TgEnabledEvents types.String `tfsdk:"tg_enabled_events"`

	// SMTP
	SmtpEnable         types.Bool   `tfsdk:"smtp_enable"`
	SmtpHost           types.String `tfsdk:"smtp_host"`
	SmtpPort           types.Int64  `tfsdk:"smtp_port"`
	SmtpUsername       types.String `tfsdk:"smtp_username"`
	SmtpPassword       types.String `tfsdk:"smtp_password"`
	SmtpFrom           types.String `tfsdk:"smtp_from"`
	SmtpFromName       types.String `tfsdk:"smtp_from_name"`
	SmtpTo             types.String `tfsdk:"smtp_to"`
	SmtpEncryptionType types.String `tfsdk:"smtp_encryption_type"`
	SmtpEnabledEvents  types.String `tfsdk:"smtp_enabled_events"`
	SmtpCPU            types.Int64  `tfsdk:"smtp_cpu"`
	SmtpMemory         types.Int64  `tfsdk:"smtp_memory"`

	OutboundDownThreshold types.Int64 `tfsdk:"outbound_down_threshold"`

	// Security
	TimeLocation           types.String `tfsdk:"time_location"`
	TwoFactorEnable        types.Bool   `tfsdk:"two_factor_enable"`
	TwoFactorToken         types.String `tfsdk:"two_factor_token"`
	LdapEnable             types.Bool   `tfsdk:"ldap_enable"`
	LdapHost               types.String `tfsdk:"ldap_host"`
	LdapPort               types.Int64  `tfsdk:"ldap_port"`
	LdapUseTLS             types.Bool   `tfsdk:"ldap_use_tls"`
	LdapInsecureSkipVerify types.Bool   `tfsdk:"ldap_insecure_skip_verify"`
	LdapBindDN             types.String `tfsdk:"ldap_bind_dn"`
	LdapPassword           types.String `tfsdk:"ldap_password"`
	LdapBaseDN             types.String `tfsdk:"ldap_base_dn"`
	LdapUserFilter         types.String `tfsdk:"ldap_user_filter"`
	LdapUserAttr           types.String `tfsdk:"ldap_user_attr"`
	LdapVlessField         types.String `tfsdk:"ldap_vless_field"`
	LdapSyncCron           types.String `tfsdk:"ldap_sync_cron"`
	LdapFlagField          types.String `tfsdk:"ldap_flag_field"`
	LdapTruthyValues       types.String `tfsdk:"ldap_truthy_values"`
	LdapInvertFlag         types.Bool   `tfsdk:"ldap_invert_flag"`
	LdapInboundTags        types.String `tfsdk:"ldap_inbound_tags"`
	LdapAutoCreate         types.Bool   `tfsdk:"ldap_auto_create"`
	LdapAutoDelete         types.Bool   `tfsdk:"ldap_auto_delete"`
	LdapDefaultTotalGB     types.Int64  `tfsdk:"ldap_default_total_gb"`
	LdapDefaultExpiryDays  types.Int64  `tfsdk:"ldap_default_expiry_days"`
	LdapDefaultLimitIP     types.Int64  `tfsdk:"ldap_default_limit_ip"`

	// Subscription
	SubEnable                   types.Bool   `tfsdk:"sub_enable"`
	SubJSONEnable               types.Bool   `tfsdk:"sub_json_enable"`
	SubJSONAutoDetect           types.Bool   `tfsdk:"sub_json_auto_detect"`
	SubJSONAlwaysArray          types.Bool   `tfsdk:"sub_json_always_array"`
	SubJSONUserAgentRegex       types.String `tfsdk:"sub_json_user_agent_regex"`
	SubClashAutoDetect          types.Bool   `tfsdk:"sub_clash_auto_detect"`
	SubClashUserAgentRegex      types.String `tfsdk:"sub_clash_user_agent_regex"`
	SubTitle                    types.String `tfsdk:"sub_title"`
	SubSupportURL               types.String `tfsdk:"sub_support_url"`
	SubProfileURL               types.String `tfsdk:"sub_profile_url"`
	SubAnnounce                 types.String `tfsdk:"sub_announce"`
	SubListen                   types.String `tfsdk:"sub_listen"`
	SubPort                     types.Int64  `tfsdk:"sub_port"`
	SubPath                     types.String `tfsdk:"sub_path"`
	SubDomain                   types.String `tfsdk:"sub_domain"`
	SubCertFile                 types.String `tfsdk:"sub_cert_file"`
	SubKeyFile                  types.String `tfsdk:"sub_key_file"`
	SubUpdates                  types.Int64  `tfsdk:"sub_updates"`
	SubEncrypt                  types.Bool   `tfsdk:"sub_encrypt"`
	SubURI                      types.String `tfsdk:"sub_uri"`
	SubJSONPath                 types.String `tfsdk:"sub_json_path"`
	SubJSONURI                  types.String `tfsdk:"sub_json_uri"`
	SubJSONMux                  types.String `tfsdk:"sub_json_mux"`
	SubJSONRules                types.String `tfsdk:"sub_json_rules"`
	SubJSONFinalMask            types.String `tfsdk:"sub_json_final_mask"`
	SubEnableRouting            types.Bool   `tfsdk:"sub_enable_routing"`
	SubRoutingRules             types.String `tfsdk:"sub_routing_rules"`
	SubIncyEnableRouting        types.Bool   `tfsdk:"sub_incy_enable_routing"`
	SubIncyRoutingRules         types.String `tfsdk:"sub_incy_routing_rules"`
	SubClashEnable              types.Bool   `tfsdk:"sub_clash_enable"`
	SubClashPath                types.String `tfsdk:"sub_clash_path"`
	SubClashURI                 types.String `tfsdk:"sub_clash_uri"`
	SubClashEnableRouting       types.Bool   `tfsdk:"sub_clash_enable_routing"`
	SubClashRules               types.String `tfsdk:"sub_clash_rules"`
	SubThemeDir                 types.String `tfsdk:"sub_theme_dir"`
	SubHideSettings             types.Bool   `tfsdk:"sub_hide_settings"`
	RestartXrayOnClientDisable  types.Bool   `tfsdk:"restart_xray_on_client_disable"`
	ExternalTrafficInformEnable types.Bool   `tfsdk:"external_traffic_inform_enable"`
	ExternalTrafficInformURI    types.String `tfsdk:"external_traffic_inform_uri"`
	WarpUpdateInterval          types.Int64  `tfsdk:"warp_update_interval"`

	// Restart
	RestartPanel types.Bool `tfsdk:"restart_panel"`
}

func (r *panelSettingsResource) modelToPayload(m *panelSettingsModel) map[string]any {
	return map[string]any{
		"webListen":                   m.WebListen.ValueString(),
		"webDomain":                   m.WebDomain.ValueString(),
		"webPort":                     m.WebPort.ValueInt64(),
		"webCertFile":                 m.WebCertFile.ValueString(),
		"webKeyFile":                  m.WebKeyFile.ValueString(),
		"webBasePath":                 m.WebBasePath.ValueString(),
		"trustedProxyCIDRs":           m.TrustedProxyCIDRs.ValueString(),
		"panelOutbound":               m.PanelProxy.ValueString(),
		"sessionMaxAge":               m.SessionMaxAge.ValueInt64(),
		"pageSize":                    m.PageSize.ValueInt64(),
		"expireDiff":                  m.ExpireDiff.ValueInt64(),
		"trafficDiff":                 m.TrafficDiff.ValueInt64(),
		"remarkTemplate":              m.RemarkModel.ValueString(),
		"datepicker":                  m.Datepicker.ValueString(),
		"tgBotEnable":                 m.TgBotEnable.ValueBool(),
		"tgBotToken":                  m.TgBotToken.ValueString(),
		"tgBotProxy":                  m.TgBotProxy.ValueString(),
		"tgBotAPIServer":              m.TgBotAPIServer.ValueString(),
		"tgBotChatId":                 m.TgBotChatID.ValueString(),
		"tgRunTime":                   m.TgRunTime.ValueString(),
		"tgBotBackup":                 m.TgBotBackup.ValueBool(),
		"tgCpu":                       m.TgCPU.ValueInt64(),
		"tgMemory":                    m.TgMemory.ValueInt64(),
		"tgLang":                      m.TgLang.ValueString(),
		"tgEnabledEvents":             m.TgEnabledEvents.ValueString(),
		"smtpEnable":                  m.SmtpEnable.ValueBool(),
		"smtpHost":                    m.SmtpHost.ValueString(),
		"smtpPort":                    m.SmtpPort.ValueInt64(),
		"smtpUsername":                m.SmtpUsername.ValueString(),
		"smtpPassword":                m.SmtpPassword.ValueString(),
		"smtpFrom":                    m.SmtpFrom.ValueString(),
		"smtpFromName":                m.SmtpFromName.ValueString(),
		"smtpTo":                      m.SmtpTo.ValueString(),
		"smtpEncryptionType":          m.SmtpEncryptionType.ValueString(),
		"smtpEnabledEvents":           m.SmtpEnabledEvents.ValueString(),
		"smtpCpu":                     m.SmtpCPU.ValueInt64(),
		"smtpMemory":                  m.SmtpMemory.ValueInt64(),
		"outboundDownThreshold":       m.OutboundDownThreshold.ValueInt64(),
		"timeLocation":                m.TimeLocation.ValueString(),
		"twoFactorEnable":             m.TwoFactorEnable.ValueBool(),
		"twoFactorToken":              m.TwoFactorToken.ValueString(),
		"ldapEnable":                  m.LdapEnable.ValueBool(),
		"ldapHost":                    m.LdapHost.ValueString(),
		"ldapPort":                    m.LdapPort.ValueInt64(),
		"ldapUseTLS":                  m.LdapUseTLS.ValueBool(),
		"ldapInsecureSkipVerify":      m.LdapInsecureSkipVerify.ValueBool(),
		"ldapBindDN":                  m.LdapBindDN.ValueString(),
		"ldapPassword":                m.LdapPassword.ValueString(),
		"ldapBaseDN":                  m.LdapBaseDN.ValueString(),
		"ldapUserFilter":              m.LdapUserFilter.ValueString(),
		"ldapUserAttr":                m.LdapUserAttr.ValueString(),
		"ldapVlessField":              m.LdapVlessField.ValueString(),
		"ldapSyncCron":                m.LdapSyncCron.ValueString(),
		"ldapFlagField":               m.LdapFlagField.ValueString(),
		"ldapTruthyValues":            m.LdapTruthyValues.ValueString(),
		"ldapInvertFlag":              m.LdapInvertFlag.ValueBool(),
		"ldapInboundTags":             m.LdapInboundTags.ValueString(),
		"ldapAutoCreate":              m.LdapAutoCreate.ValueBool(),
		"ldapAutoDelete":              m.LdapAutoDelete.ValueBool(),
		"ldapDefaultTotalGB":          m.LdapDefaultTotalGB.ValueInt64(),
		"ldapDefaultExpiryDays":       m.LdapDefaultExpiryDays.ValueInt64(),
		"ldapDefaultLimitIP":          m.LdapDefaultLimitIP.ValueInt64(),
		"subEnable":                   m.SubEnable.ValueBool(),
		"subJsonEnable":               m.SubJSONEnable.ValueBool(),
		"subJsonAutoDetect":           m.SubJSONAutoDetect.ValueBool(),
		"subJsonAlwaysArray":          m.SubJSONAlwaysArray.ValueBool(),
		"subJsonUserAgentRegex":       m.SubJSONUserAgentRegex.ValueString(),
		"subClashAutoDetect":          m.SubClashAutoDetect.ValueBool(),
		"subClashUserAgentRegex":      m.SubClashUserAgentRegex.ValueString(),
		"subTitle":                    m.SubTitle.ValueString(),
		"subSupportUrl":               m.SubSupportURL.ValueString(),
		"subProfileUrl":               m.SubProfileURL.ValueString(),
		"subAnnounce":                 m.SubAnnounce.ValueString(),
		"subListen":                   m.SubListen.ValueString(),
		"subPort":                     m.SubPort.ValueInt64(),
		"subPath":                     m.SubPath.ValueString(),
		"subDomain":                   m.SubDomain.ValueString(),
		"subCertFile":                 m.SubCertFile.ValueString(),
		"subKeyFile":                  m.SubKeyFile.ValueString(),
		"subUpdates":                  m.SubUpdates.ValueInt64(),
		"subEncrypt":                  m.SubEncrypt.ValueBool(),
		"subURI":                      m.SubURI.ValueString(),
		"subJsonPath":                 m.SubJSONPath.ValueString(),
		"subJsonURI":                  m.SubJSONURI.ValueString(),
		"subJsonMux":                  panelJSONWireValue(m.SubJSONMux.ValueString()),
		"subJsonRules":                panelJSONWireValue(m.SubJSONRules.ValueString()),
		"subJsonFinalMask":            panelJSONWireValue(m.SubJSONFinalMask.ValueString()),
		"subEnableRouting":            m.SubEnableRouting.ValueBool(),
		"subRoutingRules":             m.SubRoutingRules.ValueString(),
		"subIncyEnableRouting":        m.SubIncyEnableRouting.ValueBool(),
		"subIncyRoutingRules":         m.SubIncyRoutingRules.ValueString(),
		"subClashEnable":              m.SubClashEnable.ValueBool(),
		"subClashPath":                m.SubClashPath.ValueString(),
		"subClashURI":                 m.SubClashURI.ValueString(),
		"subClashEnableRouting":       m.SubClashEnableRouting.ValueBool(),
		"subClashRules":               m.SubClashRules.ValueString(),
		"subThemeDir":                 m.SubThemeDir.ValueString(),
		"subHideSettings":             m.SubHideSettings.ValueBool(),
		"restartXrayOnClientDisable":  m.RestartXrayOnClientDisable.ValueBool(),
		"externalTrafficInformEnable": m.ExternalTrafficInformEnable.ValueBool(),
		"externalTrafficInformURI":    m.ExternalTrafficInformURI.ValueString(),
		"warpUpdateInterval":          m.WarpUpdateInterval.ValueInt64(),
	}
}

// panelSettingsOptionalKeys were added after 3x-ui v3.5.0. Older panels omit
// them from /setting/all; keep prior Terraform state instead of zeroing.
var panelSettingsOptionalKeys = map[string]struct{}{
	"outboundDownThreshold":  {},
	"smtpFrom":               {},
	"smtpFromName":           {},
	"subJsonAutoDetect":      {},
	"subJsonAlwaysArray":     {},
	"subJsonUserAgentRegex":  {},
	"subClashAutoDetect":     {},
	"subClashUserAgentRegex": {},
}

func panelSettingsHasKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

func (r *panelSettingsResource) apiToModel(m map[string]any, state *panelSettingsModel) {
	state.WebListen = types.StringValue(stringFromMap(m, "webListen"))
	state.WebDomain = types.StringValue(stringFromMap(m, "webDomain"))
	state.WebPort = types.Int64Value(int64FromMap(m, "webPort"))
	state.WebCertFile = types.StringValue(stringFromMap(m, "webCertFile"))
	state.WebKeyFile = types.StringValue(stringFromMap(m, "webKeyFile"))
	state.WebBasePath = types.StringValue(stringFromMap(m, "webBasePath"))
	state.TrustedProxyCIDRs = types.StringValue(stringFromMap(m, "trustedProxyCIDRs"))
	state.PanelProxy = types.StringValue(stringFromMap(m, "panelOutbound"))
	state.SessionMaxAge = types.Int64Value(int64FromMap(m, "sessionMaxAge"))
	state.PageSize = types.Int64Value(int64FromMap(m, "pageSize"))
	state.ExpireDiff = types.Int64Value(int64FromMap(m, "expireDiff"))
	state.TrafficDiff = types.Int64Value(int64FromMap(m, "trafficDiff"))
	state.RemarkModel = types.StringValue(stringFromMap(m, "remarkTemplate"))
	state.Datepicker = types.StringValue(stringFromMap(m, "datepicker"))
	state.TgBotEnable = types.BoolValue(boolFromMap(m, "tgBotEnable"))
	state.TgBotToken = types.StringValue(stringFromMap(m, "tgBotToken"))
	state.TgBotProxy = types.StringValue(stringFromMap(m, "tgBotProxy"))
	state.TgBotAPIServer = types.StringValue(stringFromMap(m, "tgBotAPIServer"))
	state.TgBotChatID = types.StringValue(stringFromMap(m, "tgBotChatId"))
	state.TgRunTime = types.StringValue(stringFromMap(m, "tgRunTime"))
	state.TgBotBackup = types.BoolValue(boolFromMap(m, "tgBotBackup"))
	state.TgCPU = types.Int64Value(int64FromMap(m, "tgCpu"))
	state.TgMemory = types.Int64Value(int64FromMap(m, "tgMemory"))
	state.TgLang = types.StringValue(stringFromMap(m, "tgLang"))
	state.TgEnabledEvents = types.StringValue(stringFromMap(m, "tgEnabledEvents"))
	state.SmtpEnable = types.BoolValue(boolFromMap(m, "smtpEnable"))
	state.SmtpHost = types.StringValue(stringFromMap(m, "smtpHost"))
	state.SmtpPort = types.Int64Value(int64FromMap(m, "smtpPort"))
	state.SmtpUsername = types.StringValue(stringFromMap(m, "smtpUsername"))
	state.SmtpPassword = types.StringValue(stringFromMap(m, "smtpPassword"))
	assignOptionalString(m, "smtpFrom", &state.SmtpFrom, "")
	assignOptionalString(m, "smtpFromName", &state.SmtpFromName, "")
	state.SmtpTo = types.StringValue(stringFromMap(m, "smtpTo"))
	state.SmtpEncryptionType = types.StringValue(stringFromMap(m, "smtpEncryptionType"))
	state.SmtpEnabledEvents = types.StringValue(stringFromMap(m, "smtpEnabledEvents"))
	state.SmtpCPU = types.Int64Value(int64FromMap(m, "smtpCpu"))
	state.SmtpMemory = types.Int64Value(int64FromMap(m, "smtpMemory"))
	assignOptionalInt64(m, "outboundDownThreshold", &state.OutboundDownThreshold, 3)
	state.TimeLocation = types.StringValue(stringFromMap(m, "timeLocation"))
	state.TwoFactorEnable = types.BoolValue(boolFromMap(m, "twoFactorEnable"))
	state.TwoFactorToken = types.StringValue(stringFromMap(m, "twoFactorToken"))
	state.LdapEnable = types.BoolValue(boolFromMap(m, "ldapEnable"))
	state.LdapHost = types.StringValue(stringFromMap(m, "ldapHost"))
	state.LdapPort = types.Int64Value(int64FromMap(m, "ldapPort"))
	state.LdapUseTLS = types.BoolValue(boolFromMap(m, "ldapUseTLS"))
	state.LdapInsecureSkipVerify = types.BoolValue(boolFromMap(m, "ldapInsecureSkipVerify"))
	state.LdapBindDN = types.StringValue(stringFromMap(m, "ldapBindDN"))
	state.LdapPassword = types.StringValue(stringFromMap(m, "ldapPassword"))
	state.LdapBaseDN = types.StringValue(stringFromMap(m, "ldapBaseDN"))
	state.LdapUserFilter = types.StringValue(stringFromMap(m, "ldapUserFilter"))
	state.LdapUserAttr = types.StringValue(stringFromMap(m, "ldapUserAttr"))
	state.LdapVlessField = types.StringValue(stringFromMap(m, "ldapVlessField"))
	state.LdapSyncCron = types.StringValue(stringFromMap(m, "ldapSyncCron"))
	state.LdapFlagField = types.StringValue(stringFromMap(m, "ldapFlagField"))
	state.LdapTruthyValues = types.StringValue(stringFromMap(m, "ldapTruthyValues"))
	state.LdapInvertFlag = types.BoolValue(boolFromMap(m, "ldapInvertFlag"))
	state.LdapInboundTags = types.StringValue(stringFromMap(m, "ldapInboundTags"))
	state.LdapAutoCreate = types.BoolValue(boolFromMap(m, "ldapAutoCreate"))
	state.LdapAutoDelete = types.BoolValue(boolFromMap(m, "ldapAutoDelete"))
	state.LdapDefaultTotalGB = types.Int64Value(int64FromMap(m, "ldapDefaultTotalGB"))
	state.LdapDefaultExpiryDays = types.Int64Value(int64FromMap(m, "ldapDefaultExpiryDays"))
	state.LdapDefaultLimitIP = types.Int64Value(int64FromMap(m, "ldapDefaultLimitIP"))
	state.SubEnable = types.BoolValue(boolFromMap(m, "subEnable"))
	state.SubJSONEnable = types.BoolValue(boolFromMap(m, "subJsonEnable"))
	assignOptionalBool(m, "subJsonAutoDetect", &state.SubJSONAutoDetect, false)
	assignOptionalBool(m, "subJsonAlwaysArray", &state.SubJSONAlwaysArray, false)
	assignOptionalString(m, "subJsonUserAgentRegex", &state.SubJSONUserAgentRegex, "")
	assignOptionalBool(m, "subClashAutoDetect", &state.SubClashAutoDetect, false)
	assignOptionalString(m, "subClashUserAgentRegex", &state.SubClashUserAgentRegex, "")
	state.SubTitle = types.StringValue(stringFromMap(m, "subTitle"))
	state.SubSupportURL = types.StringValue(stringFromMap(m, "subSupportUrl"))
	state.SubProfileURL = types.StringValue(stringFromMap(m, "subProfileUrl"))
	state.SubAnnounce = types.StringValue(stringFromMap(m, "subAnnounce"))
	state.SubListen = types.StringValue(stringFromMap(m, "subListen"))
	state.SubPort = types.Int64Value(int64FromMap(m, "subPort"))
	state.SubPath = types.StringValue(stringFromMap(m, "subPath"))
	state.SubDomain = types.StringValue(stringFromMap(m, "subDomain"))
	state.SubCertFile = types.StringValue(stringFromMap(m, "subCertFile"))
	state.SubKeyFile = types.StringValue(stringFromMap(m, "subKeyFile"))
	state.SubUpdates = types.Int64Value(int64FromMap(m, "subUpdates"))
	state.SubEncrypt = types.BoolValue(boolFromMap(m, "subEncrypt"))
	state.SubURI = types.StringValue(stringFromMap(m, "subURI"))
	state.SubJSONPath = types.StringValue(stringFromMap(m, "subJsonPath"))
	state.SubJSONURI = types.StringValue(stringFromMap(m, "subJsonURI"))
	state.SubJSONMux = types.StringValue(panelJSONStateValue(stringFromMap(m, "subJsonMux")))
	state.SubJSONRules = types.StringValue(panelJSONStateValue(stringFromMap(m, "subJsonRules")))
	state.SubJSONFinalMask = types.StringValue(panelJSONStateValue(stringFromMap(m, "subJsonFinalMask")))
	state.SubEnableRouting = types.BoolValue(boolFromMap(m, "subEnableRouting"))
	state.SubRoutingRules = types.StringValue(stringFromMap(m, "subRoutingRules"))
	state.SubIncyEnableRouting = types.BoolValue(boolFromMap(m, "subIncyEnableRouting"))
	state.SubIncyRoutingRules = types.StringValue(stringFromMap(m, "subIncyRoutingRules"))
	state.SubClashEnable = types.BoolValue(boolFromMap(m, "subClashEnable"))
	state.SubClashPath = types.StringValue(stringFromMap(m, "subClashPath"))
	state.SubClashURI = types.StringValue(stringFromMap(m, "subClashURI"))
	state.SubClashEnableRouting = types.BoolValue(boolFromMap(m, "subClashEnableRouting"))
	state.SubClashRules = types.StringValue(stringFromMap(m, "subClashRules"))
	state.SubThemeDir = types.StringValue(stringFromMap(m, "subThemeDir"))
	state.SubHideSettings = types.BoolValue(boolFromMap(m, "subHideSettings"))
	state.RestartXrayOnClientDisable = types.BoolValue(boolFromMap(m, "restartXrayOnClientDisable"))
	state.ExternalTrafficInformEnable = types.BoolValue(boolFromMap(m, "externalTrafficInformEnable"))
	state.ExternalTrafficInformURI = types.StringValue(stringFromMap(m, "externalTrafficInformURI"))
	state.WarpUpdateInterval = types.Int64Value(int64FromMap(m, "warpUpdateInterval"))
}

func assignOptionalString(m map[string]any, key string, dest *types.String, absentDefault string) {
	if _, optional := panelSettingsOptionalKeys[key]; optional && !panelSettingsHasKey(m, key) {
		if dest.IsNull() || dest.IsUnknown() {
			*dest = types.StringValue(absentDefault)
		}
		return
	}
	*dest = types.StringValue(stringFromMap(m, key))
}

func assignOptionalInt64(m map[string]any, key string, dest *types.Int64, absentDefault int64) {
	if _, optional := panelSettingsOptionalKeys[key]; optional && !panelSettingsHasKey(m, key) {
		if dest.IsNull() || dest.IsUnknown() {
			*dest = types.Int64Value(absentDefault)
		}
		return
	}
	*dest = types.Int64Value(int64FromMap(m, key))
}

func assignOptionalBool(m map[string]any, key string, dest *types.Bool, absentDefault bool) {
	if _, optional := panelSettingsOptionalKeys[key]; optional && !panelSettingsHasKey(m, key) {
		if dest.IsNull() || dest.IsUnknown() {
			*dest = types.BoolValue(absentDefault)
		}
		return
	}
	*dest = types.BoolValue(boolFromMap(m, key))
}

func validatePanelSettingsJSON(m *panelSettingsModel) error {
	for name, val := range map[string]types.String{
		"sub_json_mux":        m.SubJSONMux,
		"sub_json_rules":      m.SubJSONRules,
		"sub_json_final_mask": m.SubJSONFinalMask,
	} {
		if err := validateOptionalJSONString(val.ValueString(), name); err != nil {
			return err
		}
	}
	return nil
}

func (r *panelSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan panelSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validatePanelSettingsJSON(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", err.Error())
		return
	}
	payload := r.modelToPayload(&plan)
	tflog.Info(ctx, "xui_panel_settings create prepared payload")
	if err := r.client.UpdatePanelSettings(payload); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	tflog.Info(ctx, "xui_panel_settings create updatePanelSettings succeeded")
	tflog.Info(ctx, "xui_panel_settings create restart_panel evaluated", map[string]any{
		"restart_panel_value": plan.RestartPanel.ValueBool(),
	})
	if plan.RestartPanel.ValueBool() {
		tflog.Info(ctx, "xui_panel_settings create attempting panel restart after settings apply")
		if err := r.client.RestartPanel(); err != nil {
			tflog.Error(ctx, "xui_panel_settings create panel restart failed", map[string]any{"error": err.Error()})
			resp.Diagnostics.AddError("Panel restart failed", fmt.Sprintf("`restart_panel = true` was requested but panel restart failed: %s", err.Error()))
			return
		}
		tflog.Info(ctx, "xui_panel_settings create panel restart succeeded")
	}
	plan.ID = types.StringValue("panel-settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *panelSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state panelSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.GetPanelSettings()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	r.apiToModel(m, &state)
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		state.ID = types.StringValue("panel-settings")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *panelSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan panelSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validatePanelSettingsJSON(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", err.Error())
		return
	}
	payload := r.modelToPayload(&plan)
	tflog.Info(ctx, "xui_panel_settings update prepared payload")
	if err := r.client.UpdatePanelSettings(payload); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	tflog.Info(ctx, "xui_panel_settings update updatePanelSettings succeeded")
	tflog.Info(ctx, "xui_panel_settings update restart_panel evaluated", map[string]any{
		"restart_panel_value": plan.RestartPanel.ValueBool(),
	})
	if plan.RestartPanel.ValueBool() {
		tflog.Info(ctx, "xui_panel_settings update attempting panel restart after settings apply")
		if err := r.client.RestartPanel(); err != nil {
			tflog.Error(ctx, "xui_panel_settings update panel restart failed", map[string]any{"error": err.Error()})
			resp.Diagnostics.AddError("Panel restart failed", fmt.Sprintf("`restart_panel = true` was requested but panel restart failed: %s", err.Error()))
			return
		}
		tflog.Info(ctx, "xui_panel_settings update panel restart succeeded")
	}
	plan.ID = types.StringValue("panel-settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *panelSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Panel settings cannot be deleted; removing from state is sufficient.
}

func (r *panelSettingsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	m, err := r.client.GetPanelSettings()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	var state panelSettingsModel
	state.ID = types.StringValue("panel-settings")
	state.RestartPanel = types.BoolValue(false)
	r.apiToModel(m, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

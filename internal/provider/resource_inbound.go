package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ resource.Resource = (*inboundResource)(nil)
var _ resource.ResourceWithImportState = (*inboundResource)(nil)

type inboundResource struct {
	client *xui.Client
}

func NewInboundResource() resource.Resource {
	return &inboundResource{}
}

func (r *inboundResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_inbound"
}

func (r *inboundResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A single 3x-ui / Xray inbound. Scalar fields mirror the panel export; `settings`, `stream_settings`, and `sniffing` are the same JSON strings as in an export (usually via `jsonencode()` or a heredoc). To work around panel APIs that require at least one client, this resource maintains a reserved sentinel client (`__xui_tf_do_not_delete__`) inside `settings.clients`. On **update**, any `clients` array inside `settings` is **ignored** and the panel’s current clients are preserved so separate client resources (e.g. `xui_vless_client`) keep working.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Inbound id assigned by the panel (use with `terraform import xui_inbound.NAME ID`).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Xray inbound protocol: `vless`, `vmess`, `trojan`, `shadowsocks`, `mixed`, etc. (same as export `protocol`).",
				Required:            true,
			},
			"remark": schema.StringAttribute{
				MarkdownDescription: "Inbound remark / display name.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"listen": schema.StringAttribute{
				MarkdownDescription: "Listen address; empty means all interfaces (panel default).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Inbound port (must be unique on the server).",
				Required:            true,
			},
			"enable": schema.BoolAttribute{
				MarkdownDescription: "Whether the inbound is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"expiry_time": schema.Int64Attribute{
				MarkdownDescription: "Expiry time in milliseconds since Unix epoch (0 = never); export field `expiryTime`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"traffic_reset": schema.StringAttribute{
				MarkdownDescription: "Traffic reset schedule (`never`, `daily`, `weekly`, `monthly`, …).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("never"),
			},
			"total": schema.Int64Attribute{
				MarkdownDescription: "Total traffic limit for the inbound in bytes (0 = unlimited); export `total`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"settings": schema.StringAttribute{
				MarkdownDescription: "Protocol-specific `settings` JSON string, same as export `settings` (panel stores escaped JSON; in Terraform use `jsonencode()` on an object). On **create**, use the full object from an export or a minimal valid shape for your protocol. This resource always ensures a reserved sentinel client (`__xui_tf_do_not_delete__`) exists in `settings.clients`. On **update**, keys other than `clients` are applied; `clients` always come from the server.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					settingsIgnoreClients(),
				},
			},
			"stream_settings": schema.StringAttribute{
				MarkdownDescription: "`streamSettings` JSON string from export (transport + TLS/REALITY). See Xray [StreamSettingsObject](https://xtls.github.io/config/inbounds.html#streamsettingsobject). Compared with JSON semantic equality, so whitespace and key-order differences between your config and the panel do not show as drift.",
				Required:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
			"sniffing": schema.StringAttribute{
				MarkdownDescription: "`sniffing` JSON string from export. Compared with JSON semantic equality.",
				Optional:            true,
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
				Default:             stringdefault.StaticString("{}"),
			},
			"tag": schema.StringAttribute{
				MarkdownDescription: "Inbound tag assigned by the panel (e.g. `inbound-443`). Read-only.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dummy_client_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of provider-managed reserved inbound client (`__xui_tf_do_not_delete__`) used to satisfy panel APIs that require at least one client.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_ipv4": schema.StringAttribute{
				MarkdownDescription: "Public IPv4 reported by x-ui status endpoint.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_ipv6": schema.StringAttribute{
				MarkdownDescription: "Public IPv6 reported by x-ui status endpoint.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *inboundResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type inboundModel struct {
	ID             types.Int64          `tfsdk:"id"`
	Protocol       types.String         `tfsdk:"protocol"`
	Remark         types.String         `tfsdk:"remark"`
	Listen         types.String         `tfsdk:"listen"`
	Port           types.Int64          `tfsdk:"port"`
	Enable         types.Bool           `tfsdk:"enable"`
	ExpiryTime     types.Int64          `tfsdk:"expiry_time"`
	TrafficReset   types.String         `tfsdk:"traffic_reset"`
	Total          types.Int64          `tfsdk:"total"`
	Settings       types.String         `tfsdk:"settings"`
	StreamSettings jsontypes.Normalized `tfsdk:"stream_settings"`
	Sniffing       jsontypes.Normalized `tfsdk:"sniffing"`
	Tag            types.String         `tfsdk:"tag"`
	DummyClientID  types.String         `tfsdk:"dummy_client_uuid"`
	PublicIPv4     types.String         `tfsdk:"public_ipv4"`
	PublicIPv6     types.String         `tfsdk:"public_ipv6"`
}

func (r *inboundResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan inboundModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateJSONString(plan.Settings.ValueString(), "settings"); err != nil {
		resp.Diagnostics.AddError("Invalid settings", err.Error())
		return
	}
	if err := validateJSONString(plan.StreamSettings.ValueString(), "stream_settings"); err != nil {
		resp.Diagnostics.AddError("Invalid stream_settings", err.Error())
		return
	}
	if err := validateJSONString(plan.Sniffing.ValueString(), "sniffing"); err != nil {
		resp.Diagnostics.AddError("Invalid sniffing", err.Error())
		return
	}
	settingsPayload, dummyUUID, err := ensureDummyInboundClient(plan.Settings.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError("Invalid settings", err.Error())
		return
	}
	plan.DummyClientID = types.StringValue(dummyUUID)
	payload := map[string]any{
		"remark":         plan.Remark.ValueString(),
		"listen":         plan.Listen.ValueString(),
		"port":           plan.Port.ValueInt64(),
		"protocol":       plan.Protocol.ValueString(),
		"settings":       canonicalizeInboundSettings(settingsPayload),
		"streamSettings": compactJSON(plan.StreamSettings.ValueString()),
		"sniffing":       compactJSON(plan.Sniffing.ValueString()),
		"enable":         plan.Enable.ValueBool(),
		"expiryTime":     plan.ExpiryTime.ValueInt64(),
		"trafficReset":   plan.TrafficReset.ValueString(),
		"total":          plan.Total.ValueInt64(),
		"up":             0,
		"down":           0,
	}
	raw, err := r.client.AddInbound(payload)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	m, err := inboundMapFromJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	id, err := intFromMap(m, "id")
	if err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(id))
	plan.Tag = types.StringValue(stringFromMap(m, "tag"))
	if dummyUUID, err := findDummyClientUUID(jsonStringFromMap(m, "settings")); err == nil && dummyUUID != "" {
		plan.DummyClientID = types.StringValue(dummyUUID)
	}
	status, err := r.client.GetStatusPublicIP()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.PublicIPv4 = types.StringValue(status.IPv4)
	plan.PublicIPv6 = types.StringValue(status.IPv6)
	// Deliberately leave plan.Settings untouched: Terraform's post-apply
	// consistency check requires state to equal the planned value, and the
	// planned value is the user's config (no sentinel client). The
	// follow-up Read auto-refresh canonicalizes state to include the
	// sentinel, and the settings plan modifier reconciles on subsequent
	// plans.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func validateJSONString(s, name string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%s must be non-empty valid JSON", name)
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (r *inboundResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state inboundModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.client.GetInbound(int(state.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	m, err := inboundMapFromJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if m, err = r.ensureDummyClientPresent(m, state.DummyClientID.ValueString()); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.Protocol = types.StringValue(stringFromMap(m, "protocol"))
	state.Remark = types.StringValue(stringFromMap(m, "remark"))
	state.Listen = types.StringValue(stringFromMap(m, "listen"))
	port, err := intFromMap(m, "port")
	if err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	state.Port = types.Int64Value(int64(port))
	state.Enable = types.BoolValue(boolFromMap(m, "enable"))
	exp, _ := intFromMap(m, "expiryTime")
	state.ExpiryTime = types.Int64Value(int64(exp))
	state.TrafficReset = types.StringValue(stringFromMap(m, "trafficReset"))
	state.Total = types.Int64Value(int64FromMap(m, "total"))
	state.Settings = types.StringValue(canonicalizeInboundSettings(jsonStringFromMap(m, "settings")))
	state.StreamSettings = jsontypes.NewNormalizedValue(jsonStringFromMap(m, "streamSettings"))
	state.Sniffing = jsontypes.NewNormalizedValue(jsonStringFromMap(m, "sniffing"))
	state.Tag = types.StringValue(stringFromMap(m, "tag"))
	if dummyUUID, err := findDummyClientUUID(jsonStringFromMap(m, "settings")); err == nil && dummyUUID != "" {
		state.DummyClientID = types.StringValue(dummyUUID)
	}
	status, err := r.client.GetStatusPublicIP()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.PublicIPv4 = types.StringValue(status.IPv4)
	state.PublicIPv6 = types.StringValue(status.IPv6)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *inboundResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state inboundModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateJSONString(plan.Settings.ValueString(), "settings"); err != nil {
		resp.Diagnostics.AddError("Invalid settings", err.Error())
		return
	}
	if err := validateJSONString(plan.StreamSettings.ValueString(), "stream_settings"); err != nil {
		resp.Diagnostics.AddError("Invalid stream_settings", err.Error())
		return
	}
	if err := validateJSONString(plan.Sniffing.ValueString(), "sniffing"); err != nil {
		resp.Diagnostics.AddError("Invalid sniffing", err.Error())
		return
	}
	// Serialize with any concurrent xui_vless_client RMW on the same inbound:
	// we merge the existing client list with the planned settings below, so a
	// parallel client upsert could otherwise land between our GET and our
	// UpdateInbound and be lost.
	unlock := r.client.LockInbound(int(state.ID.ValueInt64()))
	defer unlock()
	raw, err := r.client.GetInbound(int(state.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	cur, err := inboundMapFromJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if !inboundUserManagedFieldsChanged(plan, state) {
		state.Tag = types.StringValue(stringFromMap(cur, "tag"))
		state.Settings = types.StringValue(canonicalizeInboundSettings(jsonStringFromMap(cur, "settings")))
		state.StreamSettings = jsontypes.NewNormalizedValue(jsonStringFromMap(cur, "streamSettings"))
		state.Sniffing = jsontypes.NewNormalizedValue(jsonStringFromMap(cur, "sniffing"))
		if dummyUUID, err := findDummyClientUUID(jsonStringFromMap(cur, "settings")); err == nil {
			state.DummyClientID = types.StringValue(dummyUUID)
		}
		status, err := r.client.GetStatusPublicIP()
		if err != nil {
			resp.Diagnostics.AddError("API error", err.Error())
			return
		}
		state.PublicIPv4 = types.StringValue(status.IPv4)
		state.PublicIPv6 = types.StringValue(status.IPv6)
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}
	settingsJSON := jsonStringFromMap(cur, "settings")
	settingsMerged, err := mergeInboundSettingsPreservingClients(settingsJSON, plan.Settings.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("settings", err.Error())
		return
	}
	var dummyUUID string
	settingsMerged, dummyUUID, err = ensureDummyInboundClient(settingsMerged, state.DummyClientID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("settings", err.Error())
		return
	}
	state.DummyClientID = types.StringValue(dummyUUID)
	payload := map[string]any{
		"id":             int(state.ID.ValueInt64()),
		"remark":         plan.Remark.ValueString(),
		"listen":         plan.Listen.ValueString(),
		"port":           plan.Port.ValueInt64(),
		"protocol":       plan.Protocol.ValueString(),
		"settings":       canonicalizeInboundSettings(settingsMerged),
		"streamSettings": compactJSON(plan.StreamSettings.ValueString()),
		"sniffing":       compactJSON(plan.Sniffing.ValueString()),
		"enable":         plan.Enable.ValueBool(),
		"expiryTime":     plan.ExpiryTime.ValueInt64(),
		"trafficReset":   plan.TrafficReset.ValueString(),
		"total":          plan.Total.ValueInt64(),
		"up":             int64FromMap(cur, "up"),
		"down":           int64FromMap(cur, "down"),
		"allTime":        int64FromMap(cur, "allTime"),
	}
	if _, err := r.client.UpdateInbound(int(state.ID.ValueInt64()), payload); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.Protocol = plan.Protocol
	state.Remark = plan.Remark
	state.Listen = plan.Listen
	state.Port = plan.Port
	state.Enable = plan.Enable
	state.ExpiryTime = plan.ExpiryTime
	state.TrafficReset = plan.TrafficReset
	state.Total = plan.Total
	state.Settings = types.StringValue(canonicalizeInboundSettings(settingsMerged))
	state.StreamSettings = plan.StreamSettings
	state.Sniffing = plan.Sniffing
	if rawAfter, err := r.client.GetInbound(int(state.ID.ValueInt64())); err == nil {
		if m, err := inboundMapFromJSON(rawAfter); err == nil {
			state.Tag = types.StringValue(stringFromMap(m, "tag"))
			state.Settings = types.StringValue(canonicalizeInboundSettings(jsonStringFromMap(m, "settings")))
			state.StreamSettings = jsontypes.NewNormalizedValue(jsonStringFromMap(m, "streamSettings"))
			state.Sniffing = jsontypes.NewNormalizedValue(jsonStringFromMap(m, "sniffing"))
			if dummyUUID, err := findDummyClientUUID(jsonStringFromMap(m, "settings")); err == nil && dummyUUID != "" {
				state.DummyClientID = types.StringValue(dummyUUID)
			}
		}
	}
	status, err := r.client.GetStatusPublicIP()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.PublicIPv4 = types.StringValue(status.IPv4)
	state.PublicIPv6 = types.StringValue(status.IPv6)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func inboundUserManagedFieldsChanged(plan, state inboundModel) bool {
	if plan.Protocol.ValueString() != state.Protocol.ValueString() {
		return true
	}
	if plan.Remark.ValueString() != state.Remark.ValueString() {
		return true
	}
	if plan.Listen.ValueString() != state.Listen.ValueString() {
		return true
	}
	if plan.Port.ValueInt64() != state.Port.ValueInt64() {
		return true
	}
	if plan.Enable.ValueBool() != state.Enable.ValueBool() {
		return true
	}
	if plan.ExpiryTime.ValueInt64() != state.ExpiryTime.ValueInt64() {
		return true
	}
	if plan.TrafficReset.ValueString() != state.TrafficReset.ValueString() {
		return true
	}
	if plan.Total.ValueInt64() != state.Total.ValueInt64() {
		return true
	}
	if canonicalizeInboundSettings(plan.Settings.ValueString()) != canonicalizeInboundSettings(state.Settings.ValueString()) {
		return true
	}
	if compactJSON(plan.StreamSettings.ValueString()) != compactJSON(state.StreamSettings.ValueString()) {
		return true
	}
	if compactJSON(plan.Sniffing.ValueString()) != compactJSON(state.Sniffing.ValueString()) {
		return true
	}
	return false
}

func (r *inboundResource) ensureDummyClientPresent(cur map[string]any, preferredDummyUUID string) (map[string]any, error) {
	settingsJSON := jsonStringFromMap(cur, "settings")
	dummyUUID, err := findDummyClientUUID(settingsJSON)
	if err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	if dummyUUID != "" {
		return cur, nil
	}

	id, err := intFromMap(cur, "id")
	if err != nil {
		return nil, err
	}
	// Lock the inbound for the full re-read / mutate / write cycle. A
	// concurrent xui_vless_client mutation would otherwise push its own
	// settings.clients list concurrently and clobber the sentinel we're
	// about to add. After taking the lock, re-fetch the inbound so we
	// patch the current state, not a copy that may have been mutated
	// while we were waiting for the lock.
	unlock := r.client.LockInbound(id)
	defer unlock()
	raw, err := r.client.GetInbound(id)
	if err != nil {
		return nil, err
	}
	cur, err = inboundMapFromJSON(raw)
	if err != nil {
		return nil, err
	}
	settingsJSON = jsonStringFromMap(cur, "settings")
	if dummyUUID, err := findDummyClientUUID(settingsJSON); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	} else if dummyUUID != "" {
		return cur, nil
	}

	settingsWithDummy, _, err := ensureDummyInboundClient(settingsJSON, preferredDummyUUID)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"id":             id,
		"remark":         stringFromMap(cur, "remark"),
		"listen":         stringFromMap(cur, "listen"),
		"port":           int64FromMap(cur, "port"),
		"protocol":       stringFromMap(cur, "protocol"),
		"settings":       canonicalizeInboundSettings(settingsWithDummy),
		"streamSettings": compactJSON(jsonStringFromMap(cur, "streamSettings")),
		"sniffing":       compactJSON(jsonStringFromMap(cur, "sniffing")),
		"enable":         boolFromMap(cur, "enable"),
		"expiryTime":     int64FromMap(cur, "expiryTime"),
		"trafficReset":   stringFromMap(cur, "trafficReset"),
		"total":          int64FromMap(cur, "total"),
		"up":             int64FromMap(cur, "up"),
		"down":           int64FromMap(cur, "down"),
		"allTime":        int64FromMap(cur, "allTime"),
	}
	if _, err := r.client.UpdateInbound(id, payload); err != nil {
		return nil, err
	}
	rawAfter, err := r.client.GetInbound(id)
	if err != nil {
		return nil, err
	}
	m, err := inboundMapFromJSON(rawAfter)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *inboundResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state inboundModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInbound(int(state.ID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *inboundResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idVal, err := strconv.ParseInt(strings.TrimSpace(req.ID), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id", "Expected numeric inbound id from the panel export.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.Int64Value(idVal))...)
}

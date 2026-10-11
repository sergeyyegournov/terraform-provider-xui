package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ resource.Resource = (*amneziawgClientResource)(nil)
var _ resource.ResourceWithImportState = (*amneziawgClientResource)(nil)
var _ resource.ResourceWithValidateConfig = (*amneziawgClientResource)(nil)

type amneziawgClientResource struct {
	client *xui.Client
}

type amneziawgClientModel struct {
	ID              types.String `tfsdk:"id"`
	InboundID       types.Int64  `tfsdk:"inbound_id"`
	Email           types.String `tfsdk:"email"`
	PrivateKey      types.String `tfsdk:"private_key"`
	PublicKey       types.String `tfsdk:"public_key"`
	PreSharedKey    types.String `tfsdk:"pre_shared_key"`
	AllowedIPs      types.List   `tfsdk:"allowed_ips"`
	KeepAlive       types.Int64  `tfsdk:"keep_alive"`
	ForwardedPorts  types.String `tfsdk:"forwarded_ports"`
	Enable          types.Bool   `tfsdk:"enable"`
	LimitIP         types.Int64  `tfsdk:"limit_ip"`
	LimitHwid       types.Int64  `tfsdk:"limit_hwid"`
	TotalGB         types.Int64  `tfsdk:"total_gb"`
	ExpiryTime      types.Int64  `tfsdk:"expiry_time"`
	TgID            types.Int64  `tfsdk:"tg_id"`
	SubID           types.String `tfsdk:"sub_id"`
	Comment         types.String `tfsdk:"comment"`
	Reset           types.Int64  `tfsdk:"reset"`
	ResetDay        types.Int64  `tfsdk:"reset_day"`
	ResetMax        types.Int64  `tfsdk:"reset_max"`
	ResetWeekday    types.Int64  `tfsdk:"reset_weekday"`
	Group           types.String `tfsdk:"group"`
	TrafficReset    types.String `tfsdk:"traffic_reset"`
	TrafficResetDay types.Int64  `tfsdk:"traffic_reset_day"`
}

func NewAmneziaWGClientResource() resource.Resource {
	return &amneziawgClientResource{}
}

func (r *amneziawgClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_amneziawg_client"
}

func (r *amneziawgClientResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateClientRenewalConfig(ctx, req.Config, &resp.Diagnostics)
}

func (r *amneziawgClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := clientCommonSchemaAttributes("Client public key from the panel (server-generated unless keys are set).")
	attrs["private_key"] = schema.StringAttribute{
		MarkdownDescription: "AmneziaWG client private key. If omitted with `public_key`, the panel generates a keypair on create.",
		Optional:            true,
		Computed:            true,
		Sensitive:           true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["public_key"] = schema.StringAttribute{
		MarkdownDescription: "AmneziaWG client public key.",
		Optional:            true,
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["pre_shared_key"] = schema.StringAttribute{
		MarkdownDescription: "Optional AmneziaWG pre-shared key.",
		Optional:            true,
		Computed:            true,
		Sensitive:           true,
		Default:             stringdefault.StaticString(""),
	}
	attrs["allowed_ips"] = schema.ListAttribute{
		MarkdownDescription: "Peer AllowedIPs (CIDRs). Empty lets the panel allocate from the inbound subnet.",
		ElementType:         types.StringType,
		Optional:            true,
		Computed:            true,
		Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
	}
	attrs["keep_alive"] = schema.Int64Attribute{
		MarkdownDescription: "Persistent keepalive interval in seconds (`keepAlive`).",
		Optional:            true,
		Computed:            true,
		Default:             int64default.StaticInt64(0),
	}
	attrs["forwarded_ports"] = schema.StringAttribute{
		MarkdownDescription: "AmneziaWG port-forwarding spec, e.g. `80,443,8000-8100` (`forwardedPorts`).",
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "AmneziaWG peer (client) on an existing `amneziawg` inbound. Managed via `/panel/api/clients/*`.",
		Attributes:          attrs,
	}
}

func (r *amneziawgClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m amneziawgClientModel) toPanelPlan(ctx context.Context, priv, pub, psk, forwarded string) panelClientPlan {
	var allowed []string
	if !m.AllowedIPs.IsNull() && !m.AllowedIPs.IsUnknown() {
		var elems []types.String
		_ = m.AllowedIPs.ElementsAs(ctx, &elems, false)
		for _, e := range elems {
			if s := strings.TrimSpace(e.ValueString()); s != "" {
				allowed = append(allowed, s)
			}
		}
	}
	return panelClientPlan{
		Email: m.Email.ValueString(), Enable: m.Enable, LimitIP: m.LimitIP, LimitHwid: m.LimitHwid,
		TotalGB: m.TotalGB, ExpiryTime: m.ExpiryTime, TgID: m.TgID, Reset: m.Reset,
		ResetDay: m.ResetDay, ResetWeekday: m.ResetWeekday, ResetMax: m.ResetMax, Group: m.Group, TrafficReset: m.TrafficReset, TrafficResetDay: m.TrafficResetDay,
		Flow: types.StringNull(), SubID: m.SubID, Comment: m.Comment,
		PrivateKey: priv, PublicKey: pub, PreSharedKey: psk, AllowedIPs: allowed,
		KeepAlive: m.KeepAlive, ForwardedPorts: forwarded,
	}
}

func (r *amneziawgClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan amneziawgClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	priv, pub, psk, forwarded := awgSecretsFromPlan(plan)
	rec, err := createPanelClient(r.client, plan.Email.ValueString(), int(plan.InboundID.ValueInt64()), planToPanelClientInput(plan.toPanelPlan(ctx, priv, pub, psk, forwarded)))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	applyAmneziaWGClientSecretsFromRecord(&plan, *rec)
	finalizeClientSubID(&plan.SubID, *rec)
	if !plan.Comment.IsNull() && plan.Comment.ValueString() == "" {
		plan.Comment = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *amneziawgClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state amneziawgClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rec, err := readPanelClientRecord(r.client, state.Email.ValueString(), int(state.InboundID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	if rec == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	applyAmneziaWGRecord(&state, *rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *amneziawgClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state amneziawgClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	priv := state.PrivateKey.ValueString()
	pub := state.PublicKey.ValueString()
	psk := state.PreSharedKey.ValueString()
	forwarded := state.ForwardedPorts.ValueString()
	if !plan.PrivateKey.IsNull() && plan.PrivateKey.ValueString() != "" {
		priv = plan.PrivateKey.ValueString()
	}
	if !plan.PublicKey.IsNull() && plan.PublicKey.ValueString() != "" {
		pub = plan.PublicKey.ValueString()
	}
	if !plan.PreSharedKey.IsNull() {
		psk = plan.PreSharedKey.ValueString()
	}
	if !plan.ForwardedPorts.IsNull() {
		forwarded = plan.ForwardedPorts.ValueString()
	}
	if err := r.client.UpdateClient(state.Email.ValueString(), planToPanelClientInput(plan.toPanelPlan(ctx, priv, pub, psk, forwarded))); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	rec, err := readPanelClientRecord(r.client, state.Email.ValueString(), int(state.InboundID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	if rec == nil {
		resp.Diagnostics.AddError("API error", "client not found after update")
		return
	}
	applyAmneziaWGRecord(&state, *rec)
	state.Enable = plan.Enable
	state.LimitIP = plan.LimitIP
	state.LimitHwid = plan.LimitHwid
	state.TotalGB = plan.TotalGB
	state.ExpiryTime = plan.ExpiryTime
	state.TgID = plan.TgID
	state.SubID = plan.SubID
	state.Comment = plan.Comment
	state.Reset = plan.Reset
	state.ResetDay = plan.ResetDay
	state.ResetMax = plan.ResetMax
	state.ResetWeekday = plan.ResetWeekday
	state.Group = plan.Group
	state.TrafficReset = plan.TrafficReset
	state.TrafficResetDay = plan.TrafficResetDay
	state.KeepAlive = plan.KeepAlive
	state.AllowedIPs = plan.AllowedIPs
	state.ForwardedPorts = plan.ForwardedPorts
	if !plan.PrivateKey.IsNull() && plan.PrivateKey.ValueString() != "" {
		state.PrivateKey = plan.PrivateKey
	}
	if !plan.PublicKey.IsNull() && plan.PublicKey.ValueString() != "" {
		state.PublicKey = plan.PublicKey
	}
	if !plan.PreSharedKey.IsNull() {
		state.PreSharedKey = plan.PreSharedKey
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *amneziawgClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state amneziawgClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteClient(state.Email.ValueString(), false); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *amneziawgClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	inboundID, email, err := parseClientImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), types.Int64Value(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), types.StringValue(email))...)
}

func awgSecretsFromPlan(plan amneziawgClientModel) (priv, pub, psk, forwarded string) {
	if !plan.PrivateKey.IsNull() {
		priv = strings.TrimSpace(plan.PrivateKey.ValueString())
	}
	if !plan.PublicKey.IsNull() {
		pub = strings.TrimSpace(plan.PublicKey.ValueString())
	}
	if !plan.PreSharedKey.IsNull() {
		psk = strings.TrimSpace(plan.PreSharedKey.ValueString())
	}
	if !plan.ForwardedPorts.IsNull() {
		forwarded = strings.TrimSpace(plan.ForwardedPorts.ValueString())
	}
	return priv, pub, psk, forwarded
}

func applyAmneziaWGClientSecretsFromRecord(m *amneziawgClientModel, rec xui.PanelClientRecord) {
	m.ID = types.StringValue(panelClientIDFromRecord(rec))
	m.PrivateKey = types.StringValue(rec.PrivateKey)
	m.PublicKey = types.StringValue(rec.PublicKey)
	m.PreSharedKey = types.StringValue(rec.PreSharedKey)
	m.ForwardedPorts = types.StringValue(rec.ForwardedPorts)
	m.KeepAlive = types.Int64Value(rec.KeepAlive)
	vals := make([]attr.Value, 0)
	for _, ip := range splitCSV(rec.AllowedIPs) {
		vals = append(vals, types.StringValue(ip))
	}
	m.AllowedIPs = types.ListValueMust(types.StringType, vals)
}

func applyAmneziaWGRecord(m *amneziawgClientModel, rec xui.PanelClientRecord) {
	applyAmneziaWGClientSecretsFromRecord(m, rec)
	applyCommonClientFields(&m.Enable, &m.LimitIP, &m.LimitHwid, &m.TotalGB, &m.ExpiryTime, &m.TgID, &m.Reset, &m.ResetDay, &m.ResetMax, &m.ResetWeekday, &m.TrafficResetDay, &m.TrafficReset, &m.SubID, &m.Comment, &m.Group, rec)
}

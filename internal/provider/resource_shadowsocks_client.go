package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ resource.Resource = (*shadowsocksClientResource)(nil)
var _ resource.ResourceWithImportState = (*shadowsocksClientResource)(nil)
var _ resource.ResourceWithValidateConfig = (*shadowsocksClientResource)(nil)

type shadowsocksClientResource struct {
	client *xui.Client
}

type shadowsocksClientModel struct {
	ID              types.String `tfsdk:"id"`
	InboundID       types.Int64  `tfsdk:"inbound_id"`
	Email           types.String `tfsdk:"email"`
	Password        types.String `tfsdk:"password"`
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

func NewShadowsocksClientResource() resource.Resource {
	return &shadowsocksClientResource{}
}

func (r *shadowsocksClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_shadowsocks_client"
}

func (r *shadowsocksClientResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateClientRenewalConfig(ctx, req.Config, &resp.Diagnostics)
}

func (r *shadowsocksClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := clientCommonSchemaAttributes("Client password from the panel (server-generated unless `password` is set).")
	attrs["password"] = schema.StringAttribute{
		MarkdownDescription: "Shadowsocks client password. If omitted, the panel generates one on create.",
		Optional:            true,
		Computed:            true,
		Sensitive:           true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Shadowsocks user (client) on an existing 3x-ui inbound. Managed via `/panel/api/clients/*` (add, get, update, del).",
		Attributes:          attrs,
	}
}

func (r *shadowsocksClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m shadowsocksClientModel) toPanelPlan(password string) panelClientPlan {
	return panelClientPlan{
		Email: m.Email.ValueString(), Enable: m.Enable, LimitIP: m.LimitIP, LimitHwid: m.LimitHwid,
		TotalGB: m.TotalGB, ExpiryTime: m.ExpiryTime, TgID: m.TgID, Reset: m.Reset,
		ResetDay: m.ResetDay, ResetWeekday: m.ResetWeekday, ResetMax: m.ResetMax, Group: m.Group, TrafficReset: m.TrafficReset, TrafficResetDay: m.TrafficResetDay,
		Flow: types.StringNull(), SubID: m.SubID, Comment: m.Comment, Password: password,
	}
}

func (r *shadowsocksClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan shadowsocksClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	password := ""
	if !plan.Password.IsNull() {
		password = strings.TrimSpace(plan.Password.ValueString())
	}
	rec, err := createPanelClient(r.client, plan.Email.ValueString(), int(plan.InboundID.ValueInt64()), planToPanelClientInput(plan.toPanelPlan(password)))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	applyShadowsocksClientSecretsFromRecord(&plan, *rec)
	finalizeClientSubID(&plan.SubID, *rec)
	if !plan.Comment.IsNull() && plan.Comment.ValueString() == "" {
		plan.Comment = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *shadowsocksClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state shadowsocksClientModel
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
	applyShadowsocksRecord(&state, *rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *shadowsocksClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state shadowsocksClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	password := state.Password.ValueString()
	if !plan.Password.IsNull() && plan.Password.ValueString() != "" {
		password = plan.Password.ValueString()
	}
	if err := r.client.UpdateClient(state.Email.ValueString(), planToPanelClientInput(plan.toPanelPlan(password))); err != nil {
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
	applyShadowsocksRecord(&state, *rec)
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
	if !plan.Password.IsNull() && plan.Password.ValueString() != "" {
		state.Password = plan.Password
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *shadowsocksClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state shadowsocksClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteClient(state.Email.ValueString(), false); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *shadowsocksClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	inboundID, email, err := parseClientImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), types.Int64Value(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), types.StringValue(email))...)
}

func applyShadowsocksClientSecretsFromRecord(m *shadowsocksClientModel, rec xui.PanelClientRecord) {
	m.ID = types.StringValue(panelClientIDFromRecord(rec))
	m.Password = types.StringValue(rec.Password)
}

func applyShadowsocksRecord(m *shadowsocksClientModel, rec xui.PanelClientRecord) {
	applyShadowsocksClientSecretsFromRecord(m, rec)
	applyCommonClientFields(&m.Enable, &m.LimitIP, &m.LimitHwid, &m.TotalGB, &m.ExpiryTime, &m.TgID, &m.Reset, &m.ResetDay, &m.ResetMax, &m.ResetWeekday, &m.TrafficResetDay, &m.TrafficReset, &m.SubID, &m.Comment, &m.Group, rec)
}

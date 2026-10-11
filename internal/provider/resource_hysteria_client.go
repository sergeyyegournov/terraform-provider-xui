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

var _ resource.Resource = (*hysteriaClientResource)(nil)
var _ resource.ResourceWithImportState = (*hysteriaClientResource)(nil)
var _ resource.ResourceWithValidateConfig = (*hysteriaClientResource)(nil)

type hysteriaClientResource struct {
	client *xui.Client
}

type hysteriaClientModel struct {
	ID              types.String `tfsdk:"id"`
	InboundID       types.Int64  `tfsdk:"inbound_id"`
	Email           types.String `tfsdk:"email"`
	Auth            types.String `tfsdk:"auth"`
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

func NewHysteriaClientResource() resource.Resource {
	return &hysteriaClientResource{}
}

func (r *hysteriaClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_hysteria_client"
}

func (r *hysteriaClientResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateClientRenewalConfig(ctx, req.Config, &resp.Diagnostics)
}

func (r *hysteriaClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := clientCommonSchemaAttributes("Client auth token from the panel (server-generated unless `auth` is set).")
	attrs["auth"] = schema.StringAttribute{
		MarkdownDescription: "Hysteria client auth password. If omitted, the panel generates one on create.",
		Optional:            true,
		Computed:            true,
		Sensitive:           true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Hysteria user (client) on an existing 3x-ui inbound. Managed via `/panel/api/clients/*` (add, get, update, del).",
		Attributes:          attrs,
	}
}

func (r *hysteriaClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m hysteriaClientModel) toPanelPlan(auth string) panelClientPlan {
	return panelClientPlan{
		Email: m.Email.ValueString(), Enable: m.Enable, LimitIP: m.LimitIP, LimitHwid: m.LimitHwid,
		TotalGB: m.TotalGB, ExpiryTime: m.ExpiryTime, TgID: m.TgID, Reset: m.Reset,
		ResetDay: m.ResetDay, ResetWeekday: m.ResetWeekday, ResetMax: m.ResetMax, Group: m.Group, TrafficReset: m.TrafficReset, TrafficResetDay: m.TrafficResetDay,
		Flow: types.StringNull(), SubID: m.SubID, Comment: m.Comment, Auth: auth,
	}
}

func (r *hysteriaClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hysteriaClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	auth := ""
	if !plan.Auth.IsNull() {
		auth = strings.TrimSpace(plan.Auth.ValueString())
	}
	rec, err := createPanelClient(r.client, plan.Email.ValueString(), int(plan.InboundID.ValueInt64()), planToPanelClientInput(plan.toPanelPlan(auth)))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	applyHysteriaClientSecretsFromRecord(&plan, *rec)
	finalizeClientSubID(&plan.SubID, *rec)
	if !plan.Comment.IsNull() && plan.Comment.ValueString() == "" {
		plan.Comment = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *hysteriaClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hysteriaClientModel
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
	applyHysteriaRecord(&state, *rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *hysteriaClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state hysteriaClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	auth := state.Auth.ValueString()
	if !plan.Auth.IsNull() && plan.Auth.ValueString() != "" {
		auth = plan.Auth.ValueString()
	}
	if err := r.client.UpdateClient(state.Email.ValueString(), planToPanelClientInput(plan.toPanelPlan(auth))); err != nil {
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
	applyHysteriaRecord(&state, *rec)
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
	if !plan.Auth.IsNull() && plan.Auth.ValueString() != "" {
		state.Auth = plan.Auth
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *hysteriaClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hysteriaClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteClient(state.Email.ValueString(), false); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *hysteriaClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	inboundID, email, err := parseClientImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), types.Int64Value(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), types.StringValue(email))...)
}

func applyHysteriaClientSecretsFromRecord(m *hysteriaClientModel, rec xui.PanelClientRecord) {
	m.ID = types.StringValue(panelClientIDFromRecord(rec))
	m.Auth = types.StringValue(rec.Auth)
}

func applyHysteriaRecord(m *hysteriaClientModel, rec xui.PanelClientRecord) {
	applyHysteriaClientSecretsFromRecord(m, rec)
	applyCommonClientFields(&m.Enable, &m.LimitIP, &m.LimitHwid, &m.TotalGB, &m.ExpiryTime, &m.TgID, &m.Reset, &m.ResetDay, &m.ResetMax, &m.ResetWeekday, &m.TrafficResetDay, &m.TrafficReset, &m.SubID, &m.Comment, &m.Group, rec)
}

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

var _ resource.Resource = (*masqueClientResource)(nil)
var _ resource.ResourceWithImportState = (*masqueClientResource)(nil)
var _ resource.ResourceWithValidateConfig = (*masqueClientResource)(nil)

type masqueClientResource struct {
	client *xui.Client
}

type masqueClientModel struct {
	ID                types.String `tfsdk:"id"`
	InboundID         types.Int64  `tfsdk:"inbound_id"`
	Email             types.String `tfsdk:"email"`
	Password          types.String `tfsdk:"password"`
	Enable            types.Bool   `tfsdk:"enable"`
	LimitIP           types.Int64  `tfsdk:"limit_ip"`
	LimitHwid         types.Int64  `tfsdk:"limit_hwid"`
	TotalGB           types.Int64  `tfsdk:"total_gb"`
	ExpiryTime        types.Int64  `tfsdk:"expiry_time"`
	TgID              types.Int64  `tfsdk:"tg_id"`
	SubID             types.String `tfsdk:"sub_id"`
	Comment           types.String `tfsdk:"comment"`
	Reset             types.Int64  `tfsdk:"reset"`
	ResetDay          types.Int64  `tfsdk:"reset_day"`
	ResetMax          types.Int64  `tfsdk:"reset_max"`
	ResetWeekday      types.Int64  `tfsdk:"reset_weekday"`
	Group             types.String `tfsdk:"group"`
	ClientHostRuleIDs types.List   `tfsdk:"client_host_rule_ids"`
	TrafficReset      types.String `tfsdk:"traffic_reset"`
	TrafficResetDay   types.Int64  `tfsdk:"traffic_reset_day"`
}

func NewMasqueClientResource() resource.Resource {
	return &masqueClientResource{}
}

func (r *masqueClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_masque_client"
}

func (r *masqueClientResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateClientRenewalConfig(ctx, req.Config, &resp.Diagnostics)
}

func (r *masqueClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := clientCommonSchemaAttributes("Client password from the panel (server-generated unless `password` is set).")
	attrs["password"] = schema.StringAttribute{
		MarkdownDescription: "MASQUE client password. If omitted, the panel generates one on create.",
		Optional:            true,
		Computed:            true,
		Sensitive:           true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "MASQUE user (client) on an existing `masque` inbound. Managed via `/panel/api/clients/*` (add, get, update, del).",
		Attributes:          attrs,
	}
}

func (r *masqueClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m masqueClientModel) toPanelPlan(password string) panelClientPlan {
	return panelClientPlan{
		Email: m.Email.ValueString(), Enable: m.Enable, LimitIP: m.LimitIP, LimitHwid: m.LimitHwid,
		TotalGB: m.TotalGB, ExpiryTime: m.ExpiryTime, TgID: m.TgID, Reset: m.Reset,
		ResetDay: m.ResetDay, ResetWeekday: m.ResetWeekday, ResetMax: m.ResetMax, Group: m.Group, ClientHostRuleIDs: m.ClientHostRuleIDs, TrafficReset: m.TrafficReset, TrafficResetDay: m.TrafficResetDay,
		Flow: types.StringNull(), SubID: m.SubID, Comment: m.Comment, Password: password,
	}
}

func (r *masqueClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan masqueClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	password := ""
	if !plan.Password.IsNull() {
		password = strings.TrimSpace(plan.Password.ValueString())
	}
	rec, err := createPanelClient(r.client, plan.Email.ValueString(), int(plan.InboundID.ValueInt64()), planToPanelClientInput(ctx, plan.toPanelPlan(password)))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	applyMasqueClientSecretsFromRecord(&plan, *rec)
	finalizeClientSubID(&plan.SubID, *rec)
	finalizeClientHostRuleIDs(&plan.ClientHostRuleIDs, *rec)
	if !plan.Comment.IsNull() && plan.Comment.ValueString() == "" {
		plan.Comment = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *masqueClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state masqueClientModel
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
	applyMasqueRecord(&state, *rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *masqueClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state masqueClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	password := state.Password.ValueString()
	if !plan.Password.IsNull() && plan.Password.ValueString() != "" {
		password = plan.Password.ValueString()
	}
	if err := r.client.UpdateClient(state.Email.ValueString(), planToPanelClientInput(ctx, plan.toPanelPlan(password))); err != nil {
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
	applyMasqueRecord(&state, *rec)
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
	state.ClientHostRuleIDs = plan.ClientHostRuleIDs
	state.TrafficReset = plan.TrafficReset
	state.TrafficResetDay = plan.TrafficResetDay
	if !plan.Password.IsNull() && plan.Password.ValueString() != "" {
		state.Password = plan.Password
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *masqueClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state masqueClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteClient(state.Email.ValueString(), false); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *masqueClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	inboundID, email, err := parseClientImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), types.Int64Value(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), types.StringValue(email))...)
}

func applyMasqueClientSecretsFromRecord(m *masqueClientModel, rec xui.PanelClientRecord) {
	m.ID = types.StringValue(panelClientIDFromRecord(rec))
	m.Password = types.StringValue(rec.Password)
}

func applyMasqueRecord(m *masqueClientModel, rec xui.PanelClientRecord) {
	applyMasqueClientSecretsFromRecord(m, rec)
	applyCommonClientFields(&m.Enable, &m.LimitIP, &m.LimitHwid, &m.TotalGB, &m.ExpiryTime, &m.TgID, &m.Reset, &m.ResetDay, &m.ResetMax, &m.ResetWeekday, &m.TrafficResetDay, &m.TrafficReset, &m.SubID, &m.Comment, &m.Group, &m.ClientHostRuleIDs, rec)
}

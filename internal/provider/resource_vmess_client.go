package provider

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ resource.Resource = (*vmessClientResource)(nil)
var _ resource.ResourceWithImportState = (*vmessClientResource)(nil)
var _ resource.ResourceWithValidateConfig = (*vmessClientResource)(nil)

type vmessClientResource struct {
	client *xui.Client
}

type vmessClientModel struct {
	ID              types.String `tfsdk:"id"`
	InboundID       types.Int64  `tfsdk:"inbound_id"`
	Email           types.String `tfsdk:"email"`
	UUID            types.String `tfsdk:"uuid"`
	Security        types.String `tfsdk:"security"`
	Flow            types.String `tfsdk:"flow"`
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

func NewVMessClientResource() resource.Resource {
	return &vmessClientResource{}
}

func (r *vmessClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_vmess_client"
}

func (r *vmessClientResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateClientRenewalConfig(ctx, req.Config, &resp.Diagnostics)
}

func (r *vmessClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := clientCommonSchemaAttributes("Client UUID from the panel (server-generated unless `uuid` is set).")
	attrs["uuid"] = schema.StringAttribute{
		MarkdownDescription: "Static VMess UUID. If omitted, the panel generates one on create.",
		Optional:            true,
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["security"] = schema.StringAttribute{
		MarkdownDescription: "Encryption method (e.g. `auto`, `aes-128-gcm`). Defaults to `auto` when set in config.",
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString("auto"),
	}
	attrs["flow"] = schema.StringAttribute{
		MarkdownDescription: "Flow control when the inbound supports TLS flow (usually empty for VMess).",
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "VMess user (client) on an existing 3x-ui inbound. Managed via `/panel/api/clients/*` (add, get, update, del).",
		Attributes:          attrs,
	}
}

func (r *vmessClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m vmessClientModel) toPanelPlan(id, security string) panelClientPlan {
	return panelClientPlan{
		Email: m.Email.ValueString(), Enable: m.Enable, LimitIP: m.LimitIP, LimitHwid: m.LimitHwid,
		TotalGB: m.TotalGB, ExpiryTime: m.ExpiryTime, TgID: m.TgID, Reset: m.Reset,
		ResetDay: m.ResetDay, ResetWeekday: m.ResetWeekday, ResetMax: m.ResetMax, Group: m.Group, TrafficReset: m.TrafficReset, TrafficResetDay: m.TrafficResetDay,
		Flow: m.Flow, SubID: m.SubID, Comment: m.Comment, ID: id, Security: security,
	}
}

func (r *vmessClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmessClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var id string
	if uid := strings.TrimSpace(plan.UUID.ValueString()); uid != "" {
		if _, err := uuid.Parse(uid); err != nil {
			resp.Diagnostics.AddError("Invalid uuid", err.Error())
			return
		}
		id = uid
	}
	security := ""
	if !plan.Security.IsNull() && plan.Security.ValueString() != "" {
		security = plan.Security.ValueString()
	}
	wantEmptyFlow := !plan.Flow.IsNull() && plan.Flow.ValueString() == ""
	wantEmptyComment := !plan.Comment.IsNull() && plan.Comment.ValueString() == ""
	rec, err := createPanelClient(r.client, plan.Email.ValueString(), int(plan.InboundID.ValueInt64()), planToPanelClientInput(plan.toPanelPlan(id, security)))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	applyVMessClientSecretsFromRecord(&plan, *rec)
	finalizeClientSubID(&plan.SubID, *rec)
	if wantEmptyFlow {
		plan.Flow = types.StringValue("")
	}
	if wantEmptyComment {
		plan.Comment = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *vmessClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmessClientModel
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
	applyVMessRecord(&state, *rec)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vmessClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vmessClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	security := plan.Security.ValueString()
	if err := r.client.UpdateClient(state.Email.ValueString(), planToPanelClientInput(plan.toPanelPlan(state.ID.ValueString(), security))); err != nil {
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
	applyVMessRecord(&state, *rec)
	state.Flow = plan.Flow
	state.Security = plan.Security
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
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vmessClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmessClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteClient(state.Email.ValueString(), false); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *vmessClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	inboundID, email, err := parseClientImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), types.Int64Value(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), types.StringValue(email))...)
}

func applyVMessClientSecretsFromRecord(m *vmessClientModel, rec xui.PanelClientRecord) {
	uid := xui.PanelClientUUID(rec)
	if uid == "" {
		return
	}
	m.ID = types.StringValue(uid)
	m.UUID = types.StringValue(uid)
}

func applyVMessRecord(m *vmessClientModel, rec xui.PanelClientRecord) {
	applyVMessClientSecretsFromRecord(m, rec)
	m.Security = types.StringValue(rec.Security)
	m.Flow = types.StringValue(rec.Flow)
	applyCommonClientFields(&m.Enable, &m.LimitIP, &m.LimitHwid, &m.TotalGB, &m.ExpiryTime, &m.TgID, &m.Reset, &m.ResetDay, &m.ResetMax, &m.ResetWeekday, &m.TrafficResetDay, &m.TrafficReset, &m.SubID, &m.Comment, &m.Group, rec)
}

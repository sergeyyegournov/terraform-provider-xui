package provider

import (
	"context"
	"strings"

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

var _ resource.Resource = (*shadowsocksClientResource)(nil)
var _ resource.ResourceWithImportState = (*shadowsocksClientResource)(nil)

type shadowsocksClientResource struct {
	client *xui.Client
}

type shadowsocksClientModel struct {
	ID         types.String `tfsdk:"id"`
	InboundID  types.Int64  `tfsdk:"inbound_id"`
	Email      types.String `tfsdk:"email"`
	Password   types.String `tfsdk:"password"`
	Enable     types.Bool   `tfsdk:"enable"`
	LimitIP    types.Int64  `tfsdk:"limit_ip"`
	TotalGB    types.Int64  `tfsdk:"total_gb"`
	ExpiryTime types.Int64  `tfsdk:"expiry_time"`
	TgID       types.Int64  `tfsdk:"tg_id"`
	SubID      types.String `tfsdk:"sub_id"`
	Comment    types.String `tfsdk:"comment"`
	Reset      types.Int64  `tfsdk:"reset"`
}

func NewShadowsocksClientResource() resource.Resource {
	return &shadowsocksClientResource{}
}

func (r *shadowsocksClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_shadowsocks_client"
}

func (r *shadowsocksClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Shadowsocks user (client) on an existing 3x-ui inbound. Managed via `/panel/api/clients/*` (add, get, update, del).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Client password from the panel (server-generated unless `password` is set).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inbound_id": schema.Int64Attribute{
				MarkdownDescription: "Panel inbound id (number from URL / API).",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Unique client email / label in the panel.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Shadowsocks client password. If omitted, the panel generates one on create.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"limit_ip": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
			},
			"total_gb": schema.Int64Attribute{
				MarkdownDescription: "Traffic limit in **bytes** (panel field `totalGB`; 0 = unlimited).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"expiry_time": schema.Int64Attribute{
				MarkdownDescription: "Expiry in milliseconds since Unix epoch (0 = never).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"tg_id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
			},
			"sub_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"reset": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
			},
		},
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

func (r *shadowsocksClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan shadowsocksClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateClientEmail(plan.Email.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid email", err.Error())
		return
	}
	password := ""
	if !plan.Password.IsNull() {
		password = strings.TrimSpace(plan.Password.ValueString())
	}
	input := planToPanelClientInput(
		plan.Email.ValueString(), plan.Enable, plan.LimitIP, plan.TotalGB, plan.ExpiryTime, plan.TgID, plan.Reset,
		types.StringNull(), plan.SubID, plan.Comment, "", password, "", "",
	)
	rec, err := createPanelClient(r.client, plan.Email.ValueString(), int(plan.InboundID.ValueInt64()), input)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	applyShadowsocksRecord(&plan, *rec)
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
	input := planToPanelClientInput(
		plan.Email.ValueString(), plan.Enable, plan.LimitIP, plan.TotalGB, plan.ExpiryTime, plan.TgID, plan.Reset,
		types.StringNull(), plan.SubID, plan.Comment, "", password, "", "",
	)
	if err := r.client.UpdateClient(state.Email.ValueString(), input); err != nil {
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
	state.TotalGB = plan.TotalGB
	state.ExpiryTime = plan.ExpiryTime
	state.TgID = plan.TgID
	state.SubID = plan.SubID
	state.Comment = plan.Comment
	state.Reset = plan.Reset
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

func applyShadowsocksRecord(m *shadowsocksClientModel, rec xui.PanelClientRecord) {
	secret := rec.Password
	if secret == "" {
		secret = rec.Email
	}
	m.ID = types.StringValue(secret)
	m.Password = types.StringValue(rec.Password)
	m.Enable = types.BoolValue(rec.Enable)
	m.LimitIP = types.Int64Value(rec.LimitIP)
	m.TotalGB = types.Int64Value(rec.TotalGB)
	m.ExpiryTime = types.Int64Value(rec.ExpiryTime)
	m.TgID = types.Int64Value(rec.TgID)
	m.SubID = types.StringValue(rec.SubID)
	m.Comment = types.StringValue(rec.Comment)
	m.Reset = types.Int64Value(rec.Reset)
}

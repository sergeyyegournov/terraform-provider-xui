package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
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

var _ resource.Resource = (*vlessClientResource)(nil)
var _ resource.ResourceWithImportState = (*vlessClientResource)(nil)

type vlessClientResource struct {
	client *xui.Client
}

func NewVLESSClientResource() resource.Resource {
	return &vlessClientResource{}
}

func (r *vlessClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_vless_client"
}

func (r *vlessClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VLESS user (client) on an existing 3x-ui inbound. Managed via `/panel/api/clients/*` (add, get, update, del).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Client UUID (`id` in Xray VLESS settings).",
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
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Static VLESS UUID. If empty, one is generated on create.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"flow": schema.StringAttribute{
				MarkdownDescription: "e.g. `xtls-rprx-vision` for XTLS Vision.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
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

func (r *vlessClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type vlessClientModel struct {
	ID         types.String `tfsdk:"id"`
	InboundID  types.Int64  `tfsdk:"inbound_id"`
	Email      types.String `tfsdk:"email"`
	UUID       types.String `tfsdk:"uuid"`
	Flow       types.String `tfsdk:"flow"`
	Enable     types.Bool   `tfsdk:"enable"`
	LimitIP    types.Int64  `tfsdk:"limit_ip"`
	TotalGB    types.Int64  `tfsdk:"total_gb"`
	ExpiryTime types.Int64  `tfsdk:"expiry_time"`
	TgID       types.Int64  `tfsdk:"tg_id"`
	SubID      types.String `tfsdk:"sub_id"`
	Comment    types.String `tfsdk:"comment"`
	Reset      types.Int64  `tfsdk:"reset"`
}

func (r *vlessClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vlessClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Email.ValueString() == inboundDummyClientEmail {
		resp.Diagnostics.AddError("Invalid email", fmt.Sprintf("email %q is reserved for provider-managed inbound sentinel client", inboundDummyClientEmail))
		return
	}
	uid := strings.TrimSpace(plan.UUID.ValueString())
	if uid == "" {
		uid = uuid.New().String()
	}
	if _, err := uuid.Parse(uid); err != nil {
		resp.Diagnostics.AddError("Invalid uuid", err.Error())
		return
	}
	inboundID := int(plan.InboundID.ValueInt64())
	if err := r.client.AddClient(xui.ClientCreateRequest{
		Client:     planToPanelClientInput(plan, uid),
		InboundIDs: []int{inboundID},
	}); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	got, err := r.client.GetClientByEmail(plan.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("read client after create: %v", err))
		return
	}
	if !clientAttachedToInbound(got, inboundID) {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("client %q not attached to inbound %d after create", plan.Email.ValueString(), inboundID))
		return
	}
	flow, comment := plan.Flow.ValueString(), plan.Comment.ValueString()
	applyPanelClientToModel(&plan, got.Client)
	if flow == "" {
		plan.Flow = types.StringValue("")
	}
	if comment == "" {
		plan.Comment = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *vlessClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vlessClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetClientByEmail(state.Email.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if !clientAttachedToInbound(got, int(state.InboundID.ValueInt64())) {
		resp.State.RemoveResource(ctx)
		return
	}
	applyPanelClientToModel(&state, got.Client)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vlessClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vlessClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdateClient(state.Email.ValueString(), planToPanelClientInput(plan, state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	got, err := r.client.GetClientByEmail(state.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("read client after update: %v", err))
		return
	}
	applyPanelClientToModel(&state, got.Client)
	state.Flow = plan.Flow
	state.Enable = plan.Enable
	state.LimitIP = plan.LimitIP
	state.TotalGB = plan.TotalGB
	state.ExpiryTime = plan.ExpiryTime
	state.TgID = plan.TgID
	state.SubID = plan.SubID
	state.Comment = plan.Comment
	state.Reset = plan.Reset
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vlessClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vlessClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteClient(state.Email.ValueString(), false); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *vlessClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid id", "Expected `inbound_id:email` (e.g. `3:user@example.com`).")
		return
	}
	inboundID, err := parseInt64Trim(parts[0])
	if err != nil {
		resp.Diagnostics.AddError("Invalid inbound_id", err.Error())
		return
	}
	email := strings.TrimSpace(parts[1])
	if email == "" {
		resp.Diagnostics.AddError("Invalid email", "Empty email in import id")
		return
	}
	if email == inboundDummyClientEmail {
		resp.Diagnostics.AddError("Invalid email", fmt.Sprintf("email %q is reserved for provider-managed inbound sentinel client", inboundDummyClientEmail))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), types.Int64Value(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), types.StringValue(email))...)
}

func parseInt64Trim(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func planToPanelClientInput(plan vlessClientModel, uid string) xui.PanelClientInput {
	c := xui.PanelClientInput{
		ID:         uid,
		Email:      plan.Email.ValueString(),
		Enable:     plan.Enable.ValueBool(),
		LimitIP:    plan.LimitIP.ValueInt64(),
		TotalGB:    plan.TotalGB.ValueInt64(),
		ExpiryTime: plan.ExpiryTime.ValueInt64(),
		TgID:       plan.TgID.ValueInt64(),
		Reset:      plan.Reset.ValueInt64(),
	}
	if !plan.Flow.IsNull() {
		c.Flow = plan.Flow.ValueString()
	}
	if !plan.SubID.IsNull() {
		c.SubID = plan.SubID.ValueString()
	}
	if !plan.Comment.IsNull() {
		c.Comment = plan.Comment.ValueString()
	}
	return c
}

func applyPanelClientToModel(m *vlessClientModel, c xui.PanelClientRecord) {
	uid := xui.PanelClientUUID(c)
	m.ID = types.StringValue(uid)
	m.UUID = types.StringValue(uid)
	m.Flow = types.StringValue(c.Flow)
	m.Enable = types.BoolValue(c.Enable)
	m.LimitIP = types.Int64Value(c.LimitIP)
	m.TotalGB = types.Int64Value(c.TotalGB)
	m.ExpiryTime = types.Int64Value(c.ExpiryTime)
	m.TgID = types.Int64Value(c.TgID)
	m.SubID = types.StringValue(c.SubID)
	m.Comment = types.StringValue(c.Comment)
	m.Reset = types.Int64Value(c.Reset)
}

func clientAttachedToInbound(got *xui.ClientGetResult, inboundID int) bool {
	for _, id := range got.InboundIDs {
		if id == inboundID {
			return true
		}
	}
	return false
}

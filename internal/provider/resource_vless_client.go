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
				MarkdownDescription: "Client UUID from the panel (server-generated unless `uuid` is set).",
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
				MarkdownDescription: "Static VLESS UUID. If omitted, the panel generates one on create.",
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

func (r *vlessClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vlessClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateClientEmail(plan.Email.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid email", err.Error())
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
	input := planToPanelClientInput(
		plan.Email.ValueString(), plan.Enable, plan.LimitIP, plan.TotalGB, plan.ExpiryTime, plan.TgID, plan.Reset,
		plan.Flow, plan.SubID, plan.Comment, id, "", "", "",
	)
	wantEmptyFlow := !plan.Flow.IsNull() && plan.Flow.ValueString() == ""
	wantEmptyComment := !plan.Comment.IsNull() && plan.Comment.ValueString() == ""
	rec, err := createPanelClient(r.client, plan.Email.ValueString(), int(plan.InboundID.ValueInt64()), input)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	uid := xui.PanelClientUUID(*rec)
	plan.ID = types.StringValue(uid)
	plan.UUID = types.StringValue(uid)
	finalizeClientSubID(&plan.SubID, *rec)
	// Keep plan values for enable, limits, flow, etc. so post-apply state matches the
	// plan (the panel GET right after add may still report enable=true).
	if wantEmptyFlow {
		plan.Flow = types.StringValue("")
	}
	if wantEmptyComment {
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
	rec, err := readPanelClientRecord(r.client, state.Email.ValueString(), int(state.InboundID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	if rec == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	uid := xui.PanelClientUUID(*rec)
	state.ID = types.StringValue(uid)
	state.UUID = types.StringValue(uid)
	state.Flow = types.StringValue(rec.Flow)
	state.Enable = types.BoolValue(rec.Enable)
	state.LimitIP = types.Int64Value(rec.LimitIP)
	state.TotalGB = types.Int64Value(rec.TotalGB)
	state.ExpiryTime = types.Int64Value(rec.ExpiryTime)
	state.TgID = types.Int64Value(rec.TgID)
	state.SubID = types.StringValue(rec.SubID)
	state.Comment = types.StringValue(rec.Comment)
	state.Reset = types.Int64Value(rec.Reset)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vlessClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vlessClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := planToPanelClientInput(
		plan.Email.ValueString(), plan.Enable, plan.LimitIP, plan.TotalGB, plan.ExpiryTime, plan.TgID, plan.Reset,
		plan.Flow, plan.SubID, plan.Comment, state.ID.ValueString(), "", "", "",
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
	uid := xui.PanelClientUUID(*rec)
	state.ID = types.StringValue(uid)
	state.UUID = types.StringValue(uid)
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
	inboundID, email, err := parseClientImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), types.Int64Value(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), types.StringValue(email))...)
}

func parseClientImportID(id string) (int64, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("expected `inbound_id:email` (e.g. `3:user@example.com`)")
	}
	inboundID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid inbound_id: %w", err)
	}
	email := strings.TrimSpace(parts[1])
	if email == "" {
		return 0, "", fmt.Errorf("empty email in import id")
	}
	if err := validateClientEmail(email); err != nil {
		return 0, "", err
	}
	return inboundID, email, nil
}

func validateClientEmail(email string) error {
	if email == inboundDummyClientEmail {
		return fmt.Errorf("email %q is reserved for provider-managed inbound sentinel client", inboundDummyClientEmail)
	}
	return nil
}

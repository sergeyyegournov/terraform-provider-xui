package provider

import (
	"context"
	"encoding/json"
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
		MarkdownDescription: "VLESS user (client) on an existing 3x-ui inbound. Because the panel's `addClient` / `updateClient` / `delClient` endpoints are stubs in current 3x-ui releases (they either no-op or crash), this resource manages clients by reading the parent inbound, patching its `settings.clients` array, and pushing the whole inbound back through `/panel/api/inbounds/update`. The provider holds an in-process, per-inbound mutex for the duration of each mutation so parallel `for_each` instances can't race.",
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
				Default:  stringdefault.StaticString(""),
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
	clientObj := r.clientMapFromPlan(plan, uid)
	inboundID := plan.InboundID.ValueInt64()
	cm, err := r.upsertVLESSClient(inboundID, plan.Email.ValueString(), clientObj)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	createdUUID := clientUUID(cm)
	if createdUUID == "" {
		createdUUID = uid
	}
	plan.ID = types.StringValue(createdUUID)
	plan.UUID = types.StringValue(createdUUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *vlessClientResource) clientMapFromPlan(plan vlessClientModel, uid string) map[string]any {
	m := map[string]any{
		"id":         uid,
		"email":      plan.Email.ValueString(),
		"enable":     plan.Enable.ValueBool(),
		"limitIp":    plan.LimitIP.ValueInt64(),
		"totalGB":    plan.TotalGB.ValueInt64(),
		"expiryTime": plan.ExpiryTime.ValueInt64(),
		"tgId":       plan.TgID.ValueInt64(),
		"reset":      plan.Reset.ValueInt64(),
	}
	if !plan.Flow.IsNull() && plan.Flow.ValueString() != "" {
		m["flow"] = plan.Flow.ValueString()
	} else {
		m["flow"] = ""
	}
	if !plan.SubID.IsNull() {
		m["subId"] = plan.SubID.ValueString()
	} else {
		m["subId"] = ""
	}
	if !plan.Comment.IsNull() {
		m["comment"] = plan.Comment.ValueString()
	} else {
		m["comment"] = ""
	}
	return m
}

func (r *vlessClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vlessClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.client.GetInbound(int(state.InboundID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	m, err := inboundMapFromJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	settingsJSON := jsonStringFromMap(m, "settings")
	cm, err := findVLESSClientByEmail(settingsJSON, state.Email.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	uid := clientUUID(cm)
	state.ID = types.StringValue(uid)
	state.UUID = types.StringValue(uid)
	if v, ok := cm["flow"].(string); ok && v != "" {
		state.Flow = types.StringValue(v)
	} else {
		state.Flow = types.StringValue("")
	}
	if v, ok := cm["enable"].(bool); ok {
		state.Enable = types.BoolValue(v)
	}
	state.LimitIP = types.Int64Value(int64FromAny(cm["limitIp"]))
	state.TotalGB = types.Int64Value(int64FromAny(cm["totalGB"]))
	state.ExpiryTime = types.Int64Value(int64FromAny(cm["expiryTime"]))
	state.TgID = types.Int64Value(int64FromAny(cm["tgId"]))
	if v, ok := cm["subId"].(string); ok && v != "" {
		state.SubID = types.StringValue(v)
	} else {
		state.SubID = types.StringValue("")
	}
	if v, ok := cm["comment"].(string); ok && v != "" {
		state.Comment = types.StringValue(v)
	} else {
		state.Comment = types.StringValue("")
	}
	state.Reset = types.Int64Value(int64FromAny(cm["reset"]))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func int64FromAny(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

func (r *vlessClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vlessClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uid := state.ID.ValueString()
	clientObj := r.clientMapFromPlan(plan, uid)
	inboundID := plan.InboundID.ValueInt64()
	if _, err := r.upsertVLESSClient(inboundID, state.Email.ValueString(), clientObj); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
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
	if err := r.removeVLESSClient(state.InboundID.ValueInt64(), state.Email.ValueString(), state.ID.ValueString()); err != nil {
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

// upsertVLESSClient writes clientObj into the inbound's settings.clients list,
// keyed by email, and pushes the whole inbound back to the panel. The
// matchEmail argument is the email currently stored in Terraform state (for
// Update it may differ from clientObj["email"] if the user renames, though the
// schema marks email as RequiresReplace, so in practice they are equal). It
// returns the final client map as stored on the panel after the write.
//
// 3x-ui's addClient / updateClient endpoints are stubs that do not reliably
// mutate persistent state, so every client mutation goes through
// /panel/api/inbounds/update. That is a read-modify-write sequence, which
// Terraform's per-resource parallelism (for_each, -parallelism) would
// otherwise race on; r.client.LockInbound serializes RMWs per inbound id
// within the provider process.
func (r *vlessClientResource) upsertVLESSClient(inboundID int64, matchEmail string, clientObj map[string]any) (map[string]any, error) {
	unlock := r.client.LockInbound(int(inboundID))
	defer unlock()
	inbound, settings, err := r.fetchInboundSettings(inboundID)
	if err != nil {
		return nil, err
	}
	clients, _ := settings["clients"].([]any)
	newEmail, _ := clientObj["email"].(string)
	replaced := false
	for i, c := range clients {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		em, _ := cm["email"].(string)
		if em == matchEmail || (matchEmail == "" && em == newEmail) {
			clients[i] = clientObj
			replaced = true
			break
		}
	}
	if !replaced {
		clients = append(clients, clientObj)
	}
	settings["clients"] = clients
	if err := r.pushInbound(inboundID, inbound, settings); err != nil {
		return nil, err
	}
	rawAfter, err := r.client.GetInbound(int(inboundID))
	if err != nil {
		return nil, err
	}
	after, err := inboundMapFromJSON(rawAfter)
	if err != nil {
		return nil, err
	}
	cm, err := findVLESSClientByEmail(jsonStringFromMap(after, "settings"), newEmail)
	if err != nil {
		return nil, fmt.Errorf("client %q not found after inbound update: %w", newEmail, err)
	}
	return cm, nil
}

// removeVLESSClient deletes the client matching clientUUID (or, as a
// fallback, email) from the inbound's settings.clients list and pushes the
// inbound back. Runs under the per-inbound lock for the same reasons as
// upsertVLESSClient.
func (r *vlessClientResource) removeVLESSClient(inboundID int64, email, clientUUID string) error {
	unlock := r.client.LockInbound(int(inboundID))
	defer unlock()
	inbound, settings, err := r.fetchInboundSettings(inboundID)
	if err != nil {
		return err
	}
	clients, _ := settings["clients"].([]any)
	filtered := clients[:0]
	removed := false
	for _, c := range clients {
		cm, ok := c.(map[string]any)
		if !ok {
			filtered = append(filtered, c)
			continue
		}
		id, _ := cm["id"].(string)
		em, _ := cm["email"].(string)
		if (clientUUID != "" && id == clientUUID) || (clientUUID == "" && em == email) {
			removed = true
			continue
		}
		filtered = append(filtered, c)
	}
	if !removed {
		return nil
	}
	settings["clients"] = filtered
	return r.pushInbound(inboundID, inbound, settings)
}

// fetchInboundSettings gets the inbound and parses its settings JSON into a
// map so callers can patch it in place before pushing back via pushInbound.
func (r *vlessClientResource) fetchInboundSettings(inboundID int64) (map[string]any, map[string]any, error) {
	raw, err := r.client.GetInbound(int(inboundID))
	if err != nil {
		return nil, nil, err
	}
	inbound, err := inboundMapFromJSON(raw)
	if err != nil {
		return nil, nil, err
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(jsonStringFromMap(inbound, "settings")), &settings); err != nil {
		return nil, nil, fmt.Errorf("parse inbound settings: %w", err)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	if _, ok := settings["clients"].([]any); !ok {
		settings["clients"] = []any{}
	}
	return inbound, settings, nil
}

// pushInbound serializes settings back onto inbound and sends the full
// inbound object to /panel/api/inbounds/update. All other inbound fields
// (port, stream settings, sniffing, traffic counters, …) are passed through
// verbatim from the GET we just did so we don't clobber them.
func (r *vlessClientResource) pushInbound(inboundID int64, inbound, settings map[string]any) error {
	settingsRaw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	port, err := intFromMap(inbound, "port")
	if err != nil {
		return err
	}
	payload := map[string]any{
		"id":             int(inboundID),
		"remark":         stringFromMap(inbound, "remark"),
		"listen":         stringFromMap(inbound, "listen"),
		"port":           port,
		"protocol":       stringFromMap(inbound, "protocol"),
		"settings":       canonicalizeInboundSettings(string(settingsRaw)),
		"streamSettings": compactJSON(jsonStringFromMap(inbound, "streamSettings")),
		"sniffing":       compactJSON(jsonStringFromMap(inbound, "sniffing")),
		"enable":         boolFromMap(inbound, "enable"),
		"expiryTime":     int64FromMap(inbound, "expiryTime"),
		"trafficReset":   stringFromMap(inbound, "trafficReset"),
		"total":          int64FromMap(inbound, "total"),
		"up":             int64FromMap(inbound, "up"),
		"down":           int64FromMap(inbound, "down"),
		"allTime":        int64FromMap(inbound, "allTime"),
	}
	_, err = r.client.UpdateInbound(int(inboundID), payload)
	return err
}

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ resource.Resource = (*xrayTemplateResource)(nil)
var _ resource.ResourceWithImportState = (*xrayTemplateResource)(nil)

type xrayTemplateResource struct {
	client *xui.Client
}

func NewXrayTemplateResource() resource.Resource {
	return &xrayTemplateResource{}
}

func (r *xrayTemplateResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "xui_xray_template"
}

func (r *xrayTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages full Xray template config JSON (`/panel/xray/update`). This is intentionally unopinionated: provide the full template JSON body and optionally restart Xray after apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Static resource id (`xray-template`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"json": schema.StringAttribute{
				Required:            true,
				CustomType:          jsontypes.NormalizedType{},
				MarkdownDescription: "Full Xray template JSON body sent to 3x-ui `POST /panel/xray/update` as `xraySetting`. The attribute uses semantic JSON equality, so whitespace and key-order differences between the config and the panel's stored value are not reported as drift.",
			},
			"restart_xray": schema.BoolAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "If true, call `POST /panel/api/server/restartXrayService` after updating template.",
			},
			"public_ipv4": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Public IPv4 reported by x-ui status endpoint.",
			},
			"public_ipv6": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Public IPv6 reported by x-ui status endpoint.",
			},
		},
	}
}

func (r *xrayTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type xrayTemplateModel struct {
	ID          types.String         `tfsdk:"id"`
	JSON        jsontypes.Normalized `tfsdk:"json"`
	RestartXray types.Bool           `tfsdk:"restart_xray"`
	PublicIPv4  types.String         `tfsdk:"public_ipv4"`
	PublicIPv6  types.String         `tfsdk:"public_ipv6"`
}

func (r *xrayTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan xrayTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateJSONString(plan.JSON.ValueString(), "json"); err != nil {
		resp.Diagnostics.AddError("Invalid json", err.Error())
		return
	}
	if err := r.client.UpdateXrayTemplate(compactJSON(plan.JSON.ValueString())); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	if !plan.RestartXray.IsNull() && plan.RestartXray.ValueBool() {
		if err := r.client.RestartXrayService(); err != nil {
			resp.Diagnostics.AddError("API error", err.Error())
			return
		}
	}
	plan.ID = types.StringValue("xray-template")
	plan.RestartXray = types.BoolNull()
	status, err := r.client.GetStatusPublicIP()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.PublicIPv4 = types.StringValue(status.IPv4)
	plan.PublicIPv6 = types.StringValue(status.IPv6)
	// plan.JSON is stored verbatim; the attribute uses jsontypes.Normalized
	// which compares values by semantic JSON equality, so whitespace /
	// ordering differences between the user's config and what the panel
	// serves on refresh don't surface as drift.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *xrayTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state xrayTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.client.GetXrayTemplate()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.JSON = jsontypes.NewNormalizedValue(raw)
	state.RestartXray = types.BoolNull()
	status, err := r.client.GetStatusPublicIP()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.PublicIPv4 = types.StringValue(status.IPv4)
	state.PublicIPv6 = types.StringValue(status.IPv6)
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		state.ID = types.StringValue("xray-template")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *xrayTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan xrayTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateJSONString(plan.JSON.ValueString(), "json"); err != nil {
		resp.Diagnostics.AddError("Invalid json", err.Error())
		return
	}
	if err := r.client.UpdateXrayTemplate(compactJSON(plan.JSON.ValueString())); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	if !plan.RestartXray.IsNull() && plan.RestartXray.ValueBool() {
		if err := r.client.RestartXrayService(); err != nil {
			resp.Diagnostics.AddError("API error", err.Error())
			return
		}
	}
	plan.ID = types.StringValue("xray-template")
	plan.RestartXray = types.BoolNull()
	status, err := r.client.GetStatusPublicIP()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.PublicIPv4 = types.StringValue(status.IPv4)
	plan.PublicIPv6 = types.StringValue(status.IPv6)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *xrayTemplateResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No delete endpoint on 3x-ui side; Terraform state removal is sufficient.
}

func (r *xrayTemplateResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetXrayTemplate()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	if err := validateJSONString(raw, "json"); err != nil {
		resp.Diagnostics.AddError("Invalid json", err.Error())
		return
	}
	state := xrayTemplateModel{
		ID:          types.StringValue("xray-template"),
		JSON:        jsontypes.NewNormalizedValue(raw),
		RestartXray: types.BoolNull(),
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

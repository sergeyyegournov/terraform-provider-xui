package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ datasource.DataSource = (*inboundDataSource)(nil)

type inboundDataSource struct {
	client *xui.Client
}

func NewInboundDataSource() datasource.DataSource {
	return &inboundDataSource{}
}

func (d *inboundDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "xui_inbound"
}

func (d *inboundDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single inbound by id (`/panel/api/inbounds/get/:id`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Inbound id.",
				Required:            true,
			},
			"remark": schema.StringAttribute{Computed: true},
			"listen": schema.StringAttribute{Computed: true},
			"port":   schema.Int64Attribute{Computed: true},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Xray protocol (e.g. vless, vmess).",
				Computed:            true,
			},
			"enable": schema.BoolAttribute{Computed: true},
			"settings": schema.StringAttribute{
				MarkdownDescription: "Raw VLESS/Trojan/etc. `settings` JSON string from the panel.",
				Computed:            true,
			},
			"stream_settings": schema.StringAttribute{
				MarkdownDescription: "Raw `streamSettings` JSON string. Compared with JSON semantic equality.",
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
			"sniffing": schema.StringAttribute{
				MarkdownDescription: "Raw `sniffing` JSON string. Compared with JSON semantic equality.",
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
			"json": schema.StringAttribute{
				MarkdownDescription: "Full inbound object as JSON (for advanced fields).",
				Computed:            true,
			},
		},
	}
}

func (d *inboundDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cli, ok := req.ProviderData.(*xui.Client)
	if !ok {
		resp.Diagnostics.AddError("Internal error", "invalid provider data type")
		return
	}
	d.client = cli
}

type inboundDSModel struct {
	ID             types.Int64          `tfsdk:"id"`
	Remark         types.String         `tfsdk:"remark"`
	Listen         types.String         `tfsdk:"listen"`
	Port           types.Int64          `tfsdk:"port"`
	Protocol       types.String         `tfsdk:"protocol"`
	Enable         types.Bool           `tfsdk:"enable"`
	Settings       types.String         `tfsdk:"settings"`
	StreamSettings jsontypes.Normalized `tfsdk:"stream_settings"`
	Sniffing       jsontypes.Normalized `tfsdk:"sniffing"`
	JSON           types.String         `tfsdk:"json"`
}

func (d *inboundDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg inboundDSModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := d.client.GetInbound(int(cfg.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	if err := fillInboundDSModelFromRaw(raw, &cfg); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}

func fillInboundDSModelFromRaw(raw []byte, cfg *inboundDSModel) error {
	m, err := inboundMapFromJSON(raw)
	if err != nil {
		return err
	}
	port, err := intFromMap(m, "port")
	if err != nil {
		return err
	}
	cfg.Remark = types.StringValue(stringFromMap(m, "remark"))
	cfg.Listen = types.StringValue(stringFromMap(m, "listen"))
	cfg.Port = types.Int64Value(int64(port))
	cfg.Protocol = types.StringValue(stringFromMap(m, "protocol"))
	cfg.Enable = types.BoolValue(boolFromMap(m, "enable"))
	cfg.Settings = types.StringValue(jsonStringFromMap(m, "settings"))
	cfg.StreamSettings = jsontypes.NewNormalizedValue(jsonStringFromMap(m, "streamSettings"))
	cfg.Sniffing = jsontypes.NewNormalizedValue(jsonStringFromMap(m, "sniffing"))
	cfg.JSON = types.StringValue(string(raw))
	return nil
}

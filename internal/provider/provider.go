package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

var _ provider.Provider = (*xuiProvider)(nil)

type xuiProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &xuiProvider{version: version}
	}
}

func (p *xuiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "xui"
}

func (p *xuiProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage [3x-ui](https://github.com/MHSanaei/3x-ui/) v3+ panel resources via the HTTP API. Authenticate with a panel API token (`api_token`) or username/password session (CSRF-protected). Both modes apply to `/panel/api/*`.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Panel root URL including random path prefix, e.g. `https://host:port/<uuid>/`.",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Panel login username. Required when using password session auth.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Panel login password.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: "Panel API token (Bearer). When set, `/panel/api/*` requests use `Authorization: Bearer <token>` and skip CSRF. Create tokens in the panel under Settings → API tokens.",
				Optional:            true,
				Sensitive:           true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS verification (e.g. self-signed panel certificate).",
				Optional:            true,
			},
		},
	}
}

type providerModel struct {
	BaseURL            types.String `tfsdk:"base_url"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	APIToken           types.String `tfsdk:"api_token"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

func (p *xuiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasUserPass := !cfg.Username.IsNull() && !cfg.Password.IsNull() &&
		cfg.Username.ValueString() != "" && cfg.Password.ValueString() != ""
	hasToken := !cfg.APIToken.IsNull() && cfg.APIToken.ValueString() != ""

	if !hasUserPass && !hasToken {
		resp.Diagnostics.AddError(
			"Missing authentication",
			"Set `api_token`, or both `username` and `password`.",
		)
		return
	}

	insecure := false
	if !cfg.InsecureSkipVerify.IsNull() {
		insecure = cfg.InsecureSkipVerify.ValueBool()
	}

	user, pass, token := "", "", ""
	if hasUserPass {
		user = cfg.Username.ValueString()
		pass = cfg.Password.ValueString()
	}
	if hasToken {
		token = cfg.APIToken.ValueString()
	}

	cli, err := xui.NewClient(xui.ClientConfig{
		BaseURL:            cfg.BaseURL.ValueString(),
		Username:           user,
		Password:           pass,
		APIToken:           token,
		InsecureSkipVerify: insecure,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client error", err.Error())
		return
	}
	resp.DataSourceData = cli
	resp.ResourceData = cli
}

func (p *xuiProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInboundResource,
		NewVLESSClientResource,
		NewVMessClientResource,
		NewTrojanClientResource,
		NewShadowsocksClientResource,
		NewHysteriaClientResource,
		NewXrayTemplateResource,
		NewPanelSettingsResource,
	}
}

func (p *xuiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewInboundsDataSource,
		NewInboundDataSource,
	}
}

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func init() {
	for _, key := range []string{"subHostSortMode", "nodeDownThreshold", "subClashRemoteDNS"} {
		panelSettingsOptionalKeys[key] = struct{}{}
	}
}

func panelSettingsV311Attributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"sub_host_sort_mode": schema.StringAttribute{
			MarkdownDescription: "How subscription formats order host rules relative to inbounds (`subHostSortMode`): `inbound-first` or `hosts-first`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("inbound-first"),
			Validators: []validator.String{
				stringvalidator.OneOf("inbound-first", "hosts-first"),
			},
		},
		"node_down_threshold": schema.Int64Attribute{
			MarkdownDescription: "Consecutive failed node checks before a `node.down` alert (`nodeDownThreshold`; 1–100).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(1),
		},
		"sub_clash_remote_dns": schema.BoolAttribute{
			MarkdownDescription: "Also take `dns` and `sniffer` blocks from remote Clash rule templates (`subClashRemoteDNS`).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
}

func appendPanelSettingsV311(m *panelSettingsModel, payload map[string]any) {
	payload["subHostSortMode"] = m.SubHostSortMode.ValueString()
	payload["nodeDownThreshold"] = m.NodeDownThreshold.ValueInt64()
	payload["subClashRemoteDNS"] = m.SubClashRemoteDNS.ValueBool()
}

func applyPanelSettingsV311(raw map[string]any, m *panelSettingsModel) {
	assignOptionalString(raw, "subHostSortMode", &m.SubHostSortMode, "inbound-first")
	assignOptionalInt64(raw, "nodeDownThreshold", &m.NodeDownThreshold, 1)
	assignOptionalBool(raw, "subClashRemoteDNS", &m.SubClashRemoteDNS, false)
}

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func clientCommonSchemaAttributes(idDesc string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: idDesc,
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
	}
}

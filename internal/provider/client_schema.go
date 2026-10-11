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
		"limit_hwid": schema.Int64Attribute{
			MarkdownDescription: "Per-client subscription HWID device limit (`limitHwid`; 0 = unlimited).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(0),
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
			MarkdownDescription: "Rolling auto-renew interval in days (`reset`). Cannot be combined with `reset_weekday`. The panel ignores it when `reset_day` is set.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(0),
		},
		"reset_day": schema.Int64Attribute{
			MarkdownDescription: "Calendar renewal day of month 1–31 (`resetDay`). `0` disables monthly renewal. Cannot be combined with `reset_weekday`.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(0),
		},
		"reset_weekday": schema.Int64Attribute{
			MarkdownDescription: "Calendar renewal weekday 1–7, Monday–Sunday (`resetWeekday`). `0` disables weekly renewal. Cannot be combined with `reset` or `reset_day`.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(0),
		},
		"group": schema.StringAttribute{
			MarkdownDescription: "Client group label (`group`). Empty means the client is not in a group.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"reset_max": schema.Int64Attribute{
			MarkdownDescription: "Maximum number of auto-renewals (`resetMax`; 0 = unlimited).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(0),
		},
		"traffic_reset": schema.StringAttribute{
			MarkdownDescription: "Per-client traffic reset cycle: `never`, `hourly`, `daily`, `weekly`, or `monthly`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("never"),
		},
		"traffic_reset_day": schema.Int64Attribute{
			MarkdownDescription: "Day of month for monthly per-client traffic reset (`trafficResetDay`).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(1),
		},
	}
}

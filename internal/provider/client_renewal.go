package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateClientRenewalConfig(ctx context.Context, config tfsdk.Config, diags *diag.Diagnostics) {
	var reset, resetDay, resetWeekday types.Int64
	diags.Append(config.GetAttribute(ctx, path.Root("reset"), &reset)...)
	diags.Append(config.GetAttribute(ctx, path.Root("reset_day"), &resetDay)...)
	diags.Append(config.GetAttribute(ctx, path.Root("reset_weekday"), &resetWeekday)...)
	if diags.HasError() {
		return
	}
	msg := clientRenewalConflict(int64OrZero(reset), int64OrZero(resetDay), int64OrZero(resetWeekday))
	if msg == "" {
		return
	}
	diags.AddAttributeError(path.Root("reset_weekday"), "Invalid client renewal", msg)
}

func clientRenewalConflict(reset, resetDay, resetWeekday int64) string {
	if resetWeekday < 0 || resetWeekday > 7 {
		return fmt.Sprintf("reset_weekday must be between 0 and 7, got %d", resetWeekday)
	}
	if resetWeekday > 0 && (reset > 0 || resetDay > 0) {
		return "reset_weekday cannot be combined with reset or reset_day"
	}
	return ""
}

func int64OrZero(v types.Int64) int64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	return v.ValueInt64()
}

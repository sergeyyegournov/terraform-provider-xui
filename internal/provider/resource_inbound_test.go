package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestInboundUserManagedFieldsChanged(t *testing.T) {
	t.Parallel()

	base := inboundModel{
		Protocol:       types.StringValue("vless"),
		Remark:         types.StringValue("r"),
		Listen:         types.StringValue(""),
		Port:           types.Int64Value(443),
		Enable:         types.BoolValue(true),
		ExpiryTime:     types.Int64Value(0),
		TrafficReset:   types.StringValue("never"),
		Total:          types.Int64Value(0),
		Settings:       types.StringValue(`{"clients":[{"id":"1"}],"decryption":"none"}`),
		StreamSettings: jsontypes.NewNormalizedValue(`{"network":"tcp"}`),
		Sniffing:       jsontypes.NewNormalizedValue(`{"enabled":true}`),
	}

	sameDifferentFormatting := base
	sameDifferentFormatting.Settings = types.StringValue("{\n  \"decryption\": \"none\",\n  \"clients\": [{\"id\":\"1\"}]\n}")
	if inboundUserManagedFieldsChanged(sameDifferentFormatting, base) {
		t.Fatalf("expected no change for equivalent JSON formatting")
	}

	changed := base
	changed.Remark = types.StringValue("changed")
	if !inboundUserManagedFieldsChanged(changed, base) {
		t.Fatalf("expected change when user-managed field differs")
	}

	sub := base
	sub.ExcludeFromSub = types.BoolValue(true)
	sub.SubSortIndex = types.Int64Value(-2)
	if !inboundUserManagedFieldsChanged(sub, base) {
		t.Fatalf("expected change when subscription fields differ")
	}
}

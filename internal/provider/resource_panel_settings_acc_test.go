package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPanelSettingsConfig(remark string, restart bool) string {
	return fmt.Sprintf(`%s

resource "xui_panel_settings" "test" {
  remark_model  = %q
  restart_panel = %t
}
`, providerConfig(), remark, restart)
}

func TestAccPanelSettings_emptySubscriptionJSON(t *testing.T) {
	testAccPreCheck(t)
	remark := fmt.Sprintf("tf-acc-panel-json-%d", time.Now().UnixNano())
	cfg := testAccPanelSettingsEmptyJSONConfig(remark)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_json_rules", ""),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_json_fragment", ""),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccPanelSettingsEmptyJSONConfig(remark string) string {
	return fmt.Sprintf(`%s

resource "xui_panel_settings" "test" {
  remark_model      = %q
  sub_json_rules    = ""
  sub_json_fragment = null
}
`, providerConfig(), remark)
}

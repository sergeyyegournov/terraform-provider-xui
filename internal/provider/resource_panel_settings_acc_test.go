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
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_json_final_mask", ""),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "smtp_enable", "false"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "tg_memory", "80"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func TestAccPanelSettings_newAttributes(t *testing.T) {
	testAccPreCheck(t)
	remark := fmt.Sprintf("tf-acc-panel-new-%d", time.Now().UnixNano())
	cfg := testAccPanelSettingsNewAttrsConfig(remark)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_panel_settings.test", "tg_memory", "90"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "tg_enabled_events", "cpu.high"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "smtp_port", "2525"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "outbound_down_threshold", "5"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_clash_enable_routing", "true"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_hide_settings", "true"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "ldap_insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "warp_update_interval", "24"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccPanelSettingsEmptyJSONConfig(remark string) string {
	return fmt.Sprintf(`%s

resource "xui_panel_settings" "test" {
  remark_model        = %q
  sub_json_rules      = ""
  sub_json_final_mask = null
}
`, providerConfig(), remark)
}

func testAccPanelSettingsNewAttrsConfig(remark string) string {
	return fmt.Sprintf(`%s

resource "xui_panel_settings" "test" {
  remark_model              = %q
  tg_memory                 = 90
  tg_enabled_events         = "cpu.high"
  smtp_port                 = 2525
  outbound_down_threshold   = 5
  sub_clash_enable_routing  = true
  sub_hide_settings         = true
  ldap_insecure_skip_verify = true
  warp_update_interval      = 24
}
`, providerConfig(), remark)
}

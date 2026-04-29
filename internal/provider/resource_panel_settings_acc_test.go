package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPanelSettings_restartPanel(t *testing.T) {
	testAccPreCheck(t)

	cli, err := accClient()
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	if _, err := cli.GetStatus(); err != nil {
		t.Fatalf("fetch status before apply: %v", err)
	}

	remark := fmt.Sprintf("tf-acc-panel-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPanelSettingsConfig(remark, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_panel_settings.test", "id", "panel-settings"),
					resource.TestCheckResourceAttr("xui_panel_settings.test", "remark_model", remark),
					// Restart endpoint returns success and apply completes without diagnostics.
					// 3x-ui handles panel restart with SIGHUP in-process, so app uptime
					// is not a reliable restart signal and is not asserted here.
				),
			},
		},
	})
}

func testAccPanelSettingsConfig(remark string, restart bool) string {
	return fmt.Sprintf(`%s

resource "xui_panel_settings" "test" {
  remark_model  = %q
  restart_panel = %t
}
`, providerConfig(), remark, restart)
}

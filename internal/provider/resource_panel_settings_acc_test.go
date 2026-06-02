package provider

import "fmt"

func testAccPanelSettingsConfig(remark string, restart bool) string {
	return fmt.Sprintf(`%s

resource "xui_panel_settings" "test" {
  remark_model  = %q
  restart_panel = %t
}
`, providerConfig(), remark, restart)
}

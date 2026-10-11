package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccInbound_enableAndSubFields(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-sub-%d", port)
	updated := remark + "-upd"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccInboundSubConfig(remark, port, false, true, -3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_inbound.test", "enable", "false"),
					resource.TestCheckResourceAttr("xui_inbound.test", "exclude_from_sub", "true"),
					resource.TestCheckResourceAttr("xui_inbound.test", "sub_sort_index", "-3"),
					checkInboundAPI("xui_inbound.test", false, true, -3),
				),
			},
			{
				Config: testAccInboundSubConfig(updated, port, false, true, -3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_inbound.test", "remark", updated),
					resource.TestCheckResourceAttr("xui_inbound.test", "enable", "false"),
					checkInboundAPI("xui_inbound.test", false, true, -3),
				),
			},
			accClientNoDriftStep(testAccInboundSubConfig(updated, port, false, true, -3)),
		},
	})
}

func testAccInboundSubConfig(remark string, port int, enable, exclude bool, subSort int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol         = "vless"
  remark           = %q
  port             = %d
  enable           = %t
  exclude_from_sub = %t
  sub_sort_index   = %d
  settings = jsonencode({
    clients    = []
    decryption = "none"
  })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "none"
    tcpSettings = {
      acceptProxyProtocol = false
      header              = { type = "none" }
    }
  })
  sniffing = "{}"
}
`, providerConfig(), remark, port, enable, exclude, subSort)
}

func checkInboundAPI(resourceName string, enable, exclude bool, subSort int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		cli, err := accClient()
		if err != nil {
			return err
		}
		raw, err := cli.GetInbound(id)
		if err != nil {
			return err
		}
		m, err := inboundMapFromJSON(raw)
		if err != nil {
			return err
		}
		if boolFromMap(m, "enable") != enable {
			return fmt.Errorf("panel enable = %v, want %v", boolFromMap(m, "enable"), enable)
		}
		if boolFromMap(m, "excludeFromSub") != exclude {
			return fmt.Errorf("panel excludeFromSub = %v, want %v", boolFromMap(m, "excludeFromSub"), exclude)
		}
		if got := subSortIndexFromMap(m); got != subSort {
			return fmt.Errorf("panel subSortIndex = %d, want %d", got, subSort)
		}
		return nil
	}
}

func TestAccVLESSClient_resetWeekday(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-weekday-%d", port)
	email := fmt.Sprintf("tf-acc-weekday-%d@example.com", port)
	cfg := testAccVLESSWeekdayConfig(remark, port, email, 3, "weekly", 0, 0)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vless_client.test", "reset_weekday", "3"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "group", "weekly"),
					checkClientRenewalAPI(email, 3, "weekly"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func TestAccVLESSClient_resetWeekdayConflict(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-weekday-bad-%d", port)
	email := fmt.Sprintf("tf-acc-weekday-bad-%d@example.com", port)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccVLESSWeekdayConfig(remark, port, email, 1, "", 7, 0),
				ExpectError: regexp.MustCompile("cannot be combined"),
			},
		},
	})
}

func testAccVLESSWeekdayConfig(remark string, port int, email string, weekday int, group string, reset, resetDay int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol = "vless"
  remark   = %q
  port     = %d
  settings = jsonencode({
    clients    = []
    decryption = "none"
  })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "none"
    tcpSettings = {
      acceptProxyProtocol = false
      header              = { type = "none" }
    }
  })
  sniffing = "{}"
}

resource "xui_vless_client" "test" {
  inbound_id    = xui_inbound.test.id
  email         = %q
  reset_weekday = %d
  group         = %q
  reset         = %d
  reset_day     = %d
}
`, providerConfig(), remark, port, email, weekday, group, reset, resetDay)
}

func checkClientRenewalAPI(email string, weekday int64, group string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cli, err := accClient()
		if err != nil {
			return err
		}
		got, err := cli.GetClientByEmail(email)
		if err != nil {
			return err
		}
		if got.Client.ResetWeekday != weekday {
			return fmt.Errorf("panel resetWeekday = %d, want %d", got.Client.ResetWeekday, weekday)
		}
		if got.Client.Group != group {
			return fmt.Errorf("panel group = %q, want %q", got.Client.Group, group)
		}
		return nil
	}
}

func TestAccAmneziaWGClient_keepAliveZero(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-awg-%d", port)
	email := fmt.Sprintf("tf-acc-awg-%d@example.com", port)
	// The panel expands AmneziaWG server settings on save, so this inbound is
	// created through the API. The client resource is what the test manages.
	inboundID, err := createAmneziaWGInbound(remark, port)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cli, err := accClient()
		if err != nil {
			return
		}
		_ = cli.DeleteInbound(inboundID)
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAmneziaWGKeepAliveConfig(inboundID, email, 25),
				Check:  checkClientKeepAliveAPI(email, 25),
			},
			{
				Config: testAccAmneziaWGKeepAliveConfig(inboundID, email, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_amneziawg_client.test", "keep_alive", "0"),
					checkClientKeepAliveAPI(email, 0),
				),
			},
			accClientNoDriftStep(testAccAmneziaWGKeepAliveConfig(inboundID, email, 0)),
		},
	})
}

func createAmneziaWGInbound(remark string, port int) (int, error) {
	cli, err := accClient()
	if err != nil {
		return 0, err
	}
	raw, err := cli.AddInbound(map[string]any{
		"remark":         remark,
		"listen":         "",
		"port":           port,
		"protocol":       "amneziawg",
		"settings":       `{"server":{"subnetIp":"10.8.1.0","subnetCidr":24},"clients":[]}`,
		"streamSettings": "{}",
		"sniffing":       "{}",
		"enable":         true,
		"expiryTime":     0,
		"trafficReset":   "never",
		"total":          0,
		"up":             0,
		"down":           0,
	})
	if err != nil {
		return 0, err
	}
	m, err := inboundMapFromJSON(raw)
	if err != nil {
		return 0, err
	}
	id, err := intFromMap(m, "id")
	if err != nil {
		return 0, err
	}
	return id, nil
}

func testAccAmneziaWGKeepAliveConfig(inboundID int, email string, keepAlive int) string {
	return fmt.Sprintf(`%s

resource "xui_amneziawg_client" "test" {
  inbound_id  = %d
  email       = %q
  keep_alive  = %d
  allowed_ips = ["10.8.1.2/32"]
}
`, providerConfig(), inboundID, email, keepAlive)
}

func checkClientKeepAliveAPI(email string, keepAlive int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cli, err := accClient()
		if err != nil {
			return err
		}
		got, err := cli.GetClientByEmail(email)
		if err != nil {
			return err
		}
		if got.Client.KeepAlive != keepAlive {
			return fmt.Errorf("panel keepAlive = %d, want %d", got.Client.KeepAlive, keepAlive)
		}
		return nil
	}
}

func TestAccTUICClient_basic(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-tuic-%d", port)
	email := fmt.Sprintf("tf-acc-tuic-%d@example.com", port)
	cfg := testAccTUICClientConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_tuic_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_tuic_client.test", "uuid"),
					resource.TestCheckResourceAttrSet("xui_tuic_client.test", "password"),
					resource.TestCheckResourceAttr("xui_tuic_client.test", "email", email),
					accClientCommonDefaultsChecks("xui_tuic_client.test"),
				),
			},
			{
				ResourceName:            "xui_tuic_client.test",
				ImportState:             true,
				ImportStateIdFunc:       importClientIDFunc("xui_tuic_client.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "sub_id", "comment"},
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccTUICClientConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol = "tuic"
  remark   = %q
  port     = %d
  settings         = jsonencode({ clients = [] })
  stream_settings = "{}"
  sniffing        = "{}"
}

resource "xui_tuic_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
}
`, providerConfig(), remark, port, email)
}

func TestAccPanelSettings_subProfileMode(t *testing.T) {
	testAccPreCheck(t)
	remark := fmt.Sprintf("tf-acc-profile-%d", nextPort())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPanelSettingsProfileConfig(remark, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_profile_mode", "none"),
				),
			},
			accClientNoDriftStep(testAccPanelSettingsProfileConfig(remark, "")),
			{
				Config: testAccPanelSettingsProfileConfig(remark, "custom"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_profile_mode", "custom"),
				),
			},
			accClientNoDriftStep(testAccPanelSettingsProfileConfig(remark, "custom")),
			{
				Config: testAccPanelSettingsProfileConfig(remark, "none"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_panel_settings.test", "sub_profile_mode", "none"),
				),
			},
		},
	})
}

func testAccPanelSettingsProfileConfig(remark, mode string) string {
	modeLine := ""
	if mode != "" {
		modeLine = fmt.Sprintf("\n  sub_profile_mode = %q", mode)
	}
	return fmt.Sprintf(`%s

resource "xui_panel_settings" "test" {
  remark_model = %q%s
}
`, providerConfig(), remark, modeLine)
}

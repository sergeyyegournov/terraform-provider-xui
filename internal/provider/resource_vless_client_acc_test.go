package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccVLESSClient_basic(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	inboundRemark := fmt.Sprintf("tf-acc-vless-%d", port)
	email := fmt.Sprintf("tf-acc-user-%d", port)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccVLESSClientConfig(inboundRemark, port, email, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_vless_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_vless_client.test", "uuid"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "email", email),
					resource.TestCheckResourceAttr("xui_vless_client.test", "enable", "true"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "limit_ip", "0"),
					resource.TestCheckResourceAttrPair(
						"xui_vless_client.test", "inbound_id",
						"xui_inbound.test", "id",
					),
				),
			},
			{
				Config: testAccVLESSClientConfig(inboundRemark, port, email, 3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vless_client.test", "limit_ip", "3"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "email", email),
				),
			},
			{
				ResourceName:      "xui_vless_client.test",
				ImportState:       true,
				ImportStateIdFunc: importClientIDFunc("xui_vless_client.test"),
				ImportStateVerify: true,
				// Optional panel fields round-trip from the panel as null
				// when they were never set; the user sees them as null in
				// imported state even though they passed empty-string
				// defaults at create time.
				ImportStateVerifyIgnore: []string{"flow", "sub_id", "comment"},
			},
		},
	})
}

// TestAccVLESSClient_parallelFanout applies many xui_vless_client resources on
// the same inbound in parallel (Terraform default -parallelism). With the
// dedicated /panel/api/clients API, concurrent creates must not clobber each
// other; the second step expects an empty refresh plan (no drift).
//
// This test uses count rather than for_each because plugin-testing v1's
// legacy state shim doesn't accept string for_each keys at destroy time.
func TestAccVLESSClient_parallelFanout(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	inboundRemark := fmt.Sprintf("tf-acc-parallel-%d", port)

	cfg := testAccVLESSClientParallelConfig(inboundRemark, port, 5)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xui_vless_client.users[0]",
						tfjsonpath.New("flow"),
						knownvalue.StringExact("xtls-rprx-vision"),
					),
					statecheck.ExpectKnownValue(
						"xui_vless_client.users[4]",
						tfjsonpath.New("email"),
						knownvalue.StringExact(fmt.Sprintf("tf-acc-parallel-%d-4", port)),
					),
				},
			},
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func testAccVLESSClientParallelConfig(remark string, port, count int) string {
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

resource "xui_vless_client" "users" {
  count      = %d
  inbound_id = xui_inbound.test.id
  email      = "%s-${count.index}"
  flow       = "xtls-rprx-vision"
  sub_id     = "sub-${count.index}"
  comment    = "user ${count.index}"
}
`, providerConfig(), remark, port, count, remark)
}

// TestAccVLESSClient_fullAttributes exercises a client with every optional
// attribute set to a non-empty value and re-applies the same config to
// catch the "resource always shows update-in-place" regression — the bug
// the user reported that motivated switching all client mutations to the
// inbound-update RMW path.
func TestAccVLESSClient_fullAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	inboundRemark := fmt.Sprintf("tf-acc-vlessfull-%d", port)
	email := fmt.Sprintf("tf-acc-userfull-%d", port)

	cfg := testAccVLESSClientFullConfig(inboundRemark, port, email)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vless_client.test", "flow", "xtls-rprx-vision"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "sub_id", "sub-abc"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "comment", "primary user"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "limit_ip", "2"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "total_gb", "1073741824"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "tg_id", "42"),
				),
			},
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func testAccVLESSClientFullConfig(remark string, port int, email string) string {
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
  inbound_id  = xui_inbound.test.id
  email       = %q
  flow        = "xtls-rprx-vision"
  sub_id      = "sub-abc"
  comment     = "primary user"
  limit_ip    = 2
  total_gb    = 1073741824
  expiry_time = 0
  tg_id       = 42
}
`, providerConfig(), remark, port, email)
}

func testAccVLESSClientConfig(remark string, port int, email string, limitIP int) string {
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
  inbound_id = xui_inbound.test.id
  email      = %q
  limit_ip   = %d
}
`, providerConfig(), remark, port, email, limitIP)
}

// TestAccVLESSClient_emptyStringsNoDrift verifies that an explicit empty
// comment does not cause perpetual no-op updates when flow is set.
func TestAccVLESSClient_emptyStringsNoDrift(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	inboundRemark := fmt.Sprintf("tf-acc-vlessempty-%d", port)
	email := fmt.Sprintf("tf-acc-userempty-%d", port)

	cfg := testAccVLESSClientEmptyStringsConfig(inboundRemark, port, email)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vless_client.test", "flow", "xtls-rprx-vision"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "comment", ""),
				),
			},
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func testAccVLESSClientEmptyStringsConfig(remark string, port int, email string) string {
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
  inbound_id = xui_inbound.test.id
  email      = %q
  flow       = "xtls-rprx-vision"
  comment    = ""
}
`, providerConfig(), remark, port, email)
}

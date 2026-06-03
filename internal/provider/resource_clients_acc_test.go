package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVMessClient_basic(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-vmess-%d", port)
	email := fmt.Sprintf("tf-acc-vmess-user-%d", port)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccVMessClientConfig(remark, port, email, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_vmess_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_vmess_client.test", "uuid"),
					resource.TestCheckResourceAttr("xui_vmess_client.test", "email", email),
					resource.TestCheckResourceAttr("xui_vmess_client.test", "security", "auto"),
				),
			},
			{
				ResourceName:            "xui_vmess_client.test",
				ImportState:             true,
				ImportStateIdFunc:       importClientIDFunc("xui_vmess_client.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"flow", "sub_id", "comment"},
			},
		},
	})
}

func TestAccTrojanClient_basic(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-trojan-%d", port)
	email := fmt.Sprintf("tf-acc-trojan-user-%d", port)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccTrojanClientConfig(remark, port, email, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_trojan_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_trojan_client.test", "password"),
					resource.TestCheckResourceAttr("xui_trojan_client.test", "email", email),
				),
			},
			{
				ResourceName:            "xui_trojan_client.test",
				ImportState:             true,
				ImportStateIdFunc:       importClientIDFunc("xui_trojan_client.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "sub_id", "comment"},
			},
		},
	})
}

func TestAccHysteriaClient_basic(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-hysteria-%d", port)
	email := fmt.Sprintf("tf-acc-hysteria-user-%d", port)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccHysteriaClientConfig(remark, port, email, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_hysteria_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_hysteria_client.test", "auth"),
					resource.TestCheckResourceAttr("xui_hysteria_client.test", "email", email),
				),
			},
			{
				ResourceName:            "xui_hysteria_client.test",
				ImportState:             true,
				ImportStateIdFunc:       importClientIDFunc("xui_hysteria_client.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth", "sub_id", "comment"},
			},
		},
	})
}

func testAccVMessClientConfig(remark string, port int, email string, limitIP int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol = "vmess"
  remark   = %q
  port     = %d
  settings = jsonencode({
    clients = []
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

resource "xui_vmess_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
  security   = "auto"
  limit_ip   = %d
}
`, providerConfig(), remark, port, email, limitIP)
}

func testAccTrojanClientConfig(remark string, port int, email string, limitIP int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol = "trojan"
  remark   = %q
  port     = %d
  settings = jsonencode({
    clients   = []
    fallbacks = []
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

resource "xui_trojan_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
  limit_ip   = %d
}
`, providerConfig(), remark, port, email, limitIP)
}

func testAccHysteriaClientConfig(remark string, port int, email string, limitIP int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol = "hysteria"
  remark   = %q
  port     = %d
  settings = jsonencode({
    version = 1
    clients = []
  })
  stream_settings = jsonencode({
    network  = "udp"
    security = "none"
  })
  sniffing = "{}"
}

resource "xui_hysteria_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
  limit_ip   = %d
}
`, providerConfig(), remark, port, email, limitIP)
}

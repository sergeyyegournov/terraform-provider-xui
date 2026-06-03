package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShadowsocksClient_basic(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	inboundRemark := fmt.Sprintf("tf-acc-ss-%d", port)
	email := fmt.Sprintf("tf-acc-ss-user-%d", port)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccShadowsocksClientConfig(inboundRemark, port, email, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_shadowsocks_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_shadowsocks_client.test", "password"),
					resource.TestCheckResourceAttr("xui_shadowsocks_client.test", "email", email),
					resource.TestCheckResourceAttrPair(
						"xui_shadowsocks_client.test", "inbound_id",
						"xui_inbound.test", "id",
					),
				),
			},
			{
				Config: testAccShadowsocksClientConfig(inboundRemark, port, email, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_shadowsocks_client.test", "limit_ip", "2"),
				),
			},
			{
				ResourceName:      "xui_shadowsocks_client.test",
				ImportState:       true,
				ImportStateIdFunc: importVLESSClientIDFunc("xui_shadowsocks_client.test"),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"password", "sub_id", "comment"},
			},
		},
	})
}

func testAccShadowsocksClientConfig(remark string, port int, email string, limitIP int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol = "shadowsocks"
  remark   = %q
  port     = %d
  settings = jsonencode({
    method    = "aes-256-gcm"
    password  = "inbound-shared-secret"
    network   = "tcp,udp"
    clients   = []
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

resource "xui_shadowsocks_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
  limit_ip   = %d
}
`, providerConfig(), remark, port, email, limitIP)
}

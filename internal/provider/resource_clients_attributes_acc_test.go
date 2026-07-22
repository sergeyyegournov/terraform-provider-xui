package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	testAccExplicitClientUUID     = "11111111-2222-4333-8444-555555555555"
	testAccExplicitClientPassword = "tf-acc-explicit-password"
	testAccExplicitClientAuth     = "tf-acc-explicit-auth"
)

func accClientNoDriftStep(cfg string) resource.TestStep {
	return resource.TestStep{
		Config: cfg,
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		},
	}
}

func accClientCommonDefaultsChecks(resourceName string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(resourceName, "enable", "true"),
		resource.TestCheckResourceAttr(resourceName, "limit_ip", "0"),
		resource.TestCheckResourceAttr(resourceName, "total_gb", "0"),
		resource.TestCheckResourceAttr(resourceName, "expiry_time", "0"),
		resource.TestCheckResourceAttr(resourceName, "tg_id", "0"),
		resource.TestCheckResourceAttr(resourceName, "reset", "0"),
		resource.TestCheckResourceAttr(resourceName, "comment", ""),
	)
}

func accClientExplicitCommonChecks(resourceName string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(resourceName, "sub_id", "sub-explicit"),
		resource.TestCheckResourceAttr(resourceName, "comment", "explicit attrs"),
		resource.TestCheckResourceAttr(resourceName, "limit_ip", "2"),
		resource.TestCheckResourceAttr(resourceName, "total_gb", "1073741824"),
		resource.TestCheckResourceAttr(resourceName, "tg_id", "42"),
		resource.TestCheckResourceAttr(resourceName, "enable", "true"),
	)
}

// --- VLESS ---

func TestAccVLESSClient_minimalAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-vless-min-%d", port)
	email := fmt.Sprintf("tf-acc-vless-min-%d", port)
	cfg := testAccVLESSClientMinimalConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_vless_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_vless_client.test", "uuid"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "email", email),
					resource.TestCheckResourceAttr("xui_vless_client.test", "flow", ""),
					accClientCommonDefaultsChecks("xui_vless_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func TestAccVLESSClient_enableFalse(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-vless-disabled-%d", port)
	email := fmt.Sprintf("tf-acc-vless-disabled-%d", port)
	cfg := testAccVLESSClientEnableFalseConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				// 3x-ui add may report enable=true on refresh even when false was sent.
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vless_client.test", "enable", "false"),
					resource.TestCheckResourceAttr("xui_vless_client.test", "email", email),
				),
			},
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vless_client.test", "enable", "false"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccVLESSClientEnableFalseConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_vless_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
  enable     = false
}
`, providerConfig(), testAccVLESSInboundBlock(remark, port), email)
}

func TestAccVLESSClient_explicitAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-vless-exp-%d", port)
	email := fmt.Sprintf("tf-acc-vless-exp-%d", port)
	cfg := testAccVLESSClientExplicitConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vless_client.test", "uuid", testAccExplicitClientUUID),
					resource.TestCheckResourceAttr("xui_vless_client.test", "id", testAccExplicitClientUUID),
					accClientExplicitCommonChecks("xui_vless_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccVLESSClientMinimalConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_vless_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
}
`, providerConfig(), testAccVLESSInboundBlock(remark, port), email)
}

func testAccVLESSClientExplicitConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_vless_client" "test" {
  inbound_id  = xui_inbound.test.id
  email       = %q
  uuid        = %q
  sub_id      = "sub-explicit"
  comment     = "explicit attrs"
  limit_ip    = 2
  total_gb    = 1073741824
  expiry_time = 0
  tg_id       = 42
}
`, providerConfig(), testAccVLESSInboundBlock(remark, port), email, testAccExplicitClientUUID)
}

func testAccVLESSInboundBlock(remark string, port int) string {
	return fmt.Sprintf(`
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
}`, remark, port)
}

// --- VMess ---

func TestAccVMessClient_minimalAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-vmess-min-%d", port)
	email := fmt.Sprintf("tf-acc-vmess-min-%d", port)
	cfg := testAccVMessClientMinimalConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_vmess_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_vmess_client.test", "uuid"),
					resource.TestCheckResourceAttr("xui_vmess_client.test", "email", email),
					resource.TestCheckResourceAttr("xui_vmess_client.test", "security", "auto"),
					resource.TestCheckResourceAttr("xui_vmess_client.test", "flow", ""),
					accClientCommonDefaultsChecks("xui_vmess_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func TestAccVMessClient_explicitAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-vmess-exp-%d", port)
	email := fmt.Sprintf("tf-acc-vmess-exp-%d", port)
	cfg := testAccVMessClientExplicitConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_vmess_client.test", "uuid", testAccExplicitClientUUID),
					resource.TestCheckResourceAttr("xui_vmess_client.test", "id", testAccExplicitClientUUID),
					resource.TestCheckResourceAttr("xui_vmess_client.test", "security", "aes-128-gcm"),
					accClientExplicitCommonChecks("xui_vmess_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccVMessClientMinimalConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_vmess_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
}
`, providerConfig(), testAccVMessInboundBlock(remark, port), email)
}

func testAccVMessClientExplicitConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_vmess_client" "test" {
  inbound_id  = xui_inbound.test.id
  email       = %q
  uuid        = %q
  security    = "aes-128-gcm"
  sub_id      = "sub-explicit"
  comment     = "explicit attrs"
  limit_ip    = 2
  total_gb    = 1073741824
  expiry_time = 0
  tg_id       = 42
}
`, providerConfig(), testAccVMessInboundBlock(remark, port), email, testAccExplicitClientUUID)
}

func testAccVMessInboundBlock(remark string, port int) string {
	return fmt.Sprintf(`
resource "xui_inbound" "test" {
  protocol = "vmess"
  remark   = %q
  port     = %d
  settings = jsonencode({ clients = [] })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "none"
    tcpSettings = {
      acceptProxyProtocol = false
      header              = { type = "none" }
    }
  })
  sniffing = "{}"
}`, remark, port)
}

// --- Trojan ---

func TestAccTrojanClient_minimalAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-trojan-min-%d", port)
	email := fmt.Sprintf("tf-acc-trojan-min-%d", port)
	cfg := testAccTrojanClientMinimalConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_trojan_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_trojan_client.test", "password"),
					resource.TestCheckResourceAttr("xui_trojan_client.test", "email", email),
					accClientCommonDefaultsChecks("xui_trojan_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func TestAccTrojanClient_explicitAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-trojan-exp-%d", port)
	email := fmt.Sprintf("tf-acc-trojan-exp-%d", port)
	cfg := testAccTrojanClientExplicitConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_trojan_client.test", "password", testAccExplicitClientPassword),
					resource.TestCheckResourceAttr("xui_trojan_client.test", "id", testAccExplicitClientPassword),
					accClientExplicitCommonChecks("xui_trojan_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccTrojanClientMinimalConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_trojan_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
}
`, providerConfig(), testAccTrojanInboundBlock(remark, port), email)
}

func testAccTrojanClientExplicitConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_trojan_client" "test" {
  inbound_id  = xui_inbound.test.id
  email       = %q
  password    = %q
  sub_id      = "sub-explicit"
  comment     = "explicit attrs"
  limit_ip    = 2
  total_gb    = 1073741824
  expiry_time = 0
  tg_id       = 42
}
`, providerConfig(), testAccTrojanInboundBlock(remark, port), email, testAccExplicitClientPassword)
}

func testAccTrojanInboundBlock(remark string, port int) string {
	return fmt.Sprintf(`
resource "xui_inbound" "test" {
  protocol = "trojan"
  remark   = %q
  port     = %d
  settings = jsonencode({ clients = [], fallbacks = [] })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "none"
    tcpSettings = {
      acceptProxyProtocol = false
      header              = { type = "none" }
    }
  })
  sniffing = "{}"
}`, remark, port)
}

// --- Shadowsocks ---

func TestAccShadowsocksClient_minimalAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-ss-min-%d", port)
	email := fmt.Sprintf("tf-acc-ss-min-%d", port)
	cfg := testAccShadowsocksClientMinimalConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_shadowsocks_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_shadowsocks_client.test", "password"),
					resource.TestCheckResourceAttr("xui_shadowsocks_client.test", "email", email),
					accClientCommonDefaultsChecks("xui_shadowsocks_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func TestAccShadowsocksClient_explicitAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-ss-exp-%d", port)
	email := fmt.Sprintf("tf-acc-ss-exp-%d", port)
	cfg := testAccShadowsocksClientExplicitConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_shadowsocks_client.test", "password", testAccExplicitClientPassword),
					resource.TestCheckResourceAttr("xui_shadowsocks_client.test", "id", testAccExplicitClientPassword),
					accClientExplicitCommonChecks("xui_shadowsocks_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccShadowsocksClientMinimalConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_shadowsocks_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
}
`, providerConfig(), testAccShadowsocksInboundBlock(remark, port), email)
}

func testAccShadowsocksClientExplicitConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_shadowsocks_client" "test" {
  inbound_id  = xui_inbound.test.id
  email       = %q
  password    = %q
  sub_id      = "sub-explicit"
  comment     = "explicit attrs"
  limit_ip    = 2
  total_gb    = 1073741824
  expiry_time = 0
  tg_id       = 42
}
`, providerConfig(), testAccShadowsocksInboundBlock(remark, port), email, testAccExplicitClientPassword)
}

func testAccShadowsocksInboundBlock(remark string, port int) string {
	return fmt.Sprintf(`
resource "xui_inbound" "test" {
  protocol = "shadowsocks"
  remark   = %q
  port     = %d
  settings = jsonencode({
    method   = "aes-256-gcm"
    password = "inbound-shared-secret"
    network  = "tcp,udp"
    clients  = []
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
}`, remark, port)
}

// --- Hysteria ---

func TestAccHysteriaClient_minimalAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-hysteria-min-%d", port)
	email := fmt.Sprintf("tf-acc-hysteria-min-%d", port)
	cfg := testAccHysteriaClientMinimalConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_hysteria_client.test", "id"),
					resource.TestCheckResourceAttrSet("xui_hysteria_client.test", "auth"),
					resource.TestCheckResourceAttr("xui_hysteria_client.test", "email", email),
					accClientCommonDefaultsChecks("xui_hysteria_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func TestAccHysteriaClient_explicitAttributes(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-hysteria-exp-%d", port)
	email := fmt.Sprintf("tf-acc-hysteria-exp-%d", port)
	cfg := testAccHysteriaClientExplicitConfig(remark, port, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_hysteria_client.test", "auth", testAccExplicitClientAuth),
					resource.TestCheckResourceAttr("xui_hysteria_client.test", "id", testAccExplicitClientAuth),
					accClientExplicitCommonChecks("xui_hysteria_client.test"),
				),
			},
			accClientNoDriftStep(cfg),
		},
	})
}

func testAccHysteriaClientMinimalConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_hysteria_client" "test" {
  inbound_id = xui_inbound.test.id
  email      = %q
}
`, providerConfig(), testAccHysteriaInboundBlock(remark, port), email)
}

func testAccHysteriaClientExplicitConfig(remark string, port int, email string) string {
	return fmt.Sprintf(`%s
%s
resource "xui_hysteria_client" "test" {
  inbound_id  = xui_inbound.test.id
  email       = %q
  auth        = %q
  sub_id      = "sub-explicit"
  comment     = "explicit attrs"
  limit_ip    = 2
  total_gb    = 1073741824
  expiry_time = 0
  tg_id       = 42
}
`, providerConfig(), testAccHysteriaInboundBlock(remark, port), email, testAccExplicitClientAuth)
}

func testAccHysteriaInboundBlock(remark string, port int) string {
	return fmt.Sprintf(`
resource "xui_inbound" "test" {
  protocol = "hysteria"
  remark   = %q
  port     = %d
  settings = jsonencode({ version = 1, clients = [] })
  stream_settings = jsonencode({
    network  = "udp"
    security = "none"
  })
  sniffing = "{}"
}`, remark, port)
}

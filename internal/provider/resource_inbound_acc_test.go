package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccInbound_basic(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-basic-%d", port)
	updatedRemark := remark + "-upd"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccInboundConfig(remark, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_inbound.test", "port", fmt.Sprintf("%d", port)),
					resource.TestCheckResourceAttr("xui_inbound.test", "remark", remark),
					resource.TestCheckResourceAttr("xui_inbound.test", "protocol", "vless"),
					resource.TestCheckResourceAttr("xui_inbound.test", "enable", "true"),
					resource.TestCheckResourceAttrSet("xui_inbound.test", "id"),
					resource.TestCheckResourceAttrSet("xui_inbound.test", "tag"),
					testCheckResourceAttrPresent("xui_inbound.test", "public_ipv4"),
					testCheckResourceAttrPresent("xui_inbound.test", "public_ipv6"),
				),
			},
			{
				Config: testAccInboundConfig(updatedRemark, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_inbound.test", "remark", updatedRemark),
					resource.TestCheckResourceAttr("xui_inbound.test", "port", fmt.Sprintf("%d", port)),
				),
			},
			{
				ResourceName:            "xui_inbound.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"public_ipv4", "public_ipv6"},
			},
		},
	})
}

// TestAccInbound_emptyClients verifies that a VLESS inbound can be created
// with settings.clients = [] and stays empty on the panel across refresh
// and non-client setting updates.
func TestAccInbound_emptyClients(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-empty-%d", port)
	updatedRemark := remark + "-upd"

	var inboundID int

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccInboundConfig(remark, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_inbound.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["xui_inbound.test"]
						if !ok {
							return fmt.Errorf("resource not found: xui_inbound.test")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return err
						}
						inboundID = id
						count, err := inboundClientCount(id)
						if err != nil {
							return err
						}
						if count != 0 {
							return fmt.Errorf("expected 0 clients on panel after create, got %d", count)
						}
						return nil
					},
				),
			},
			{
				Config: testAccInboundConfig(updatedRemark, port),
				Check: func(s *terraform.State) error {
					count, err := inboundClientCount(inboundID)
					if err != nil {
						return err
					}
					if count != 0 {
						return fmt.Errorf("expected 0 clients on panel after update, got %d", count)
					}
					return nil
				},
			},
		},
	})
}

// TestAccInbound_importAdopted covers importing a panel-native inbound that
// was created outside Terraform.
func TestAccInbound_importAdopted(t *testing.T) {
	testAccPreCheck(t)
	port := nextPort()
	remark := fmt.Sprintf("tf-acc-adopt-%d", port)

	var preCreatedID int

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             checkInboundDestroyed,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					id, err := createInboundBypassTerraform(remark, port)
					if err != nil {
						t.Fatalf("pre-create inbound via panel API: %v", err)
					}
					preCreatedID = id
				},
				Config:             testAccInboundAdoptedConfig(remark, port),
				ResourceName:       "xui_inbound.adopted",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc:  func(_ *terraform.State) (string, error) { return strconv.Itoa(preCreatedID), nil },
			},
			{
				Config: testAccInboundAdoptedConfig(remark, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_inbound.adopted", "remark", remark),
					resource.TestCheckResourceAttr("xui_inbound.adopted", "port", strconv.Itoa(port)),
					testCheckResourceAttrPresent("xui_inbound.adopted", "public_ipv4"),
					testCheckResourceAttrPresent("xui_inbound.adopted", "public_ipv6"),
				),
			},
		},
	})
}

func testAccInboundAdoptedConfig(remark string, port int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "adopted" {
  protocol = "vless"
  remark   = %q
  listen   = ""
  port     = %d
  settings = jsonencode({
    clients    = []
    decryption = "none"
    fallbacks  = []
  })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "none"
    tcpSettings = {
      acceptProxyProtocol = false
      header              = { type = "none" }
    }
  })
  sniffing = jsonencode({})
}
`, providerConfig(), remark, port)
}

func testAccInboundConfig(remark string, port int) string {
	return fmt.Sprintf(`%s

resource "xui_inbound" "test" {
  protocol = "vless"
  remark   = %q
  listen   = ""
  port     = %d
  settings = jsonencode({
    clients    = []
    decryption = "none"
    fallbacks  = []
  })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "none"
    tcpSettings = {
      acceptProxyProtocol = false
      header = {
        type = "none"
      }
    }
  })
  sniffing = jsonencode({ enabled = false, destOverride = ["http", "tls"] })
}
`, providerConfig(), remark, port)
}

func testCheckResourceAttrPresent(resourceName, attr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if _, ok := rs.Primary.Attributes[attr]; !ok {
			return fmt.Errorf("attribute %q not present", attr)
		}
		return nil
	}
}

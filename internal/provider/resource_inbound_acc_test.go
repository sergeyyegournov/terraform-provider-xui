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
					resource.TestCheckResourceAttrSet("xui_inbound.test", "dummy_client_uuid"),
					testCheckResourceAttrPresent("xui_inbound.test", "public_ipv4"),
					testCheckResourceAttrPresent("xui_inbound.test", "public_ipv6"),
				),
			},
			{
				Config: testAccInboundConfig(updatedRemark, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_inbound.test", "remark", updatedRemark),
					resource.TestCheckResourceAttr("xui_inbound.test", "port", fmt.Sprintf("%d", port)),
					resource.TestCheckResourceAttrSet("xui_inbound.test", "dummy_client_uuid"),
				),
			},
			{
				ResourceName:            "xui_inbound.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"dummy_client_uuid"},
			},
		},
	})
}

// TestAccInbound_importWithoutSentinel covers the scenario where a user
// adopts a panel-native inbound that was created outside of Terraform and
// therefore has no provider-managed sentinel client. On import, the
// provider's Read must detect the missing sentinel and inject it via the
// panel update API; after that the resource behaves like any other inbound.
func TestAccInbound_importWithoutSentinel(t *testing.T) {
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
					// Sanity: the seeded inbound does NOT yet have a sentinel.
					has, err := inboundHasSentinelClient(id)
					if err != nil {
						t.Fatalf("check sentinel absence: %v", err)
					}
					if has {
						t.Fatalf("test setup corrupted: seeded inbound %d already has a sentinel", id)
					}
				},
				Config:             testAccInboundAdoptedConfig(remark, port),
				ResourceName:       "xui_inbound.adopted",
				ImportState:        true,
				ImportStatePersist: true, // carry imported state into subsequent steps so test teardown destroys the seeded inbound
				ImportStateIdFunc:  func(_ *terraform.State) (string, error) { return strconv.Itoa(preCreatedID), nil },
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) == 0 {
						return fmt.Errorf("import returned no instance state")
					}
					for _, s := range states {
						uid := s.Attributes["dummy_client_uuid"]
						if uid == "" {
							return fmt.Errorf("expected sentinel dummy_client_uuid to be set after import, got empty")
						}
					}
					return nil
				},
			},
			{
				Config: testAccInboundAdoptedConfig(remark, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("xui_inbound.adopted", "dummy_client_uuid"),
					resource.TestCheckResourceAttr("xui_inbound.adopted", "remark", remark),
					resource.TestCheckResourceAttr("xui_inbound.adopted", "port", strconv.Itoa(port)),
					testCheckResourceAttrPresent("xui_inbound.adopted", "public_ipv4"),
					testCheckResourceAttrPresent("xui_inbound.adopted", "public_ipv6"),
					// Assert the sentinel landed on the panel too (not only in TF state).
					func(*terraform.State) error {
						has, err := inboundHasSentinelClient(preCreatedID)
						if err != nil {
							return err
						}
						if !has {
							return fmt.Errorf("panel inbound %d still missing sentinel client after adopt+refresh", preCreatedID)
						}
						return nil
					},
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

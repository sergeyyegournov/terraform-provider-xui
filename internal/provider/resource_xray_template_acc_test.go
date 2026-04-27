package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccXrayTemplate_basic fetches the panel's current xray template and
// applies it back via the resource. This validates the full round-trip
// (read, normalize, write) without actually mutating xray behaviour.
// Note: xui_xray_template has no delete endpoint on the panel; its Delete
// just drops state. We do NOT re-apply a scrubbed template because changing
// xray config can knock out the panel mid-test-suite.
func TestAccXrayTemplate_basic(t *testing.T) {
	testAccPreCheck(t)
	cli, err := accClient()
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	templateJSON, err := cli.GetXrayTemplate()
	if err != nil {
		t.Fatalf("fetch xray template: %v", err)
	}
	if templateJSON == "" {
		t.Fatalf("panel returned empty xray template")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccXrayTemplateConfig(templateJSON),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xui_xray_template.test", "id", "xray-template"),
					resource.TestCheckResourceAttrSet("xui_xray_template.test", "json"),
				),
			},
			{
				ResourceName:      "xui_xray_template.test",
				ImportState:       true,
				ImportStateId:     "xray-template",
				ImportStateVerify: true,
				// `restart_xray` is provider-local (no equivalent on the panel).
				// `json` uses jsontypes.Normalized and compares with JSON
				// semantic equality in production paths (plan, refresh,
				// apply-consistency). terraform-plugin-testing's
				// ImportStateVerify, however, does a plain byte-level
				// reflect.DeepEqual on the flattened attribute map, which
				// ignores the custom type's StringSemanticEquals — so
				// whitespace differences between the heredoc config and the
				// panel's compact response show up as test-only drift.
				ImportStateVerifyIgnore: []string{"restart_xray", "json"},
			},
		},
	})
}

func testAccXrayTemplateConfig(templateJSON string) string {
	// Wrap the template JSON in an HCL heredoc to avoid quoting issues.
	// The provider accepts this string verbatim as `xraySetting`.
	return fmt.Sprintf(`%s

resource "xui_xray_template" "test" {
  json         = <<-EOT
%s
EOT
  restart_xray = false
}
`, providerConfig(), templateJSON)
}

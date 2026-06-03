package provider

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

// accClient builds a fresh xui.Client pointing at the shared test panel.
// Use from CheckDestroy / CheckExist helpers; do not reuse across tests.
func accClient() (*xui.Client, error) {
	if accPanel == nil {
		return nil, fmt.Errorf("accPanel not initialized; TF_ACC missing?")
	}
	return xui.NewClient(xui.ClientConfig{
		BaseURL:            accPanel.BaseURL,
		Username:           accPanel.Username,
		Password:           accPanel.Password,
		InsecureSkipVerify: true,
	})
}

// listInbounds returns all inbounds currently on the panel as a slice of maps.
func listInbounds() ([]map[string]any, error) {
	cli, err := accClient()
	if err != nil {
		return nil, err
	}
	raw, err := cli.ListInbounds()
	if err != nil {
		return nil, fmt.Errorf("list inbounds: %w", err)
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("decode inbounds: %w", err)
	}
	return list, nil
}

// inboundExists returns whether an inbound with the given id is present.
func inboundExists(id int) (bool, error) {
	list, err := listInbounds()
	if err != nil {
		return false, err
	}
	for _, in := range list {
		switch v := in["id"].(type) {
		case float64:
			if int(v) == id {
				return true, nil
			}
		case int:
			if v == id {
				return true, nil
			}
		}
	}
	return false, nil
}

// checkInboundDestroyed returns a CheckDestroy function asserting that every
// xui_inbound resource recorded in state has been removed from the panel.
func checkInboundDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "xui_inbound" {
			continue
		}
		idStr := rs.Primary.ID
		if idStr == "" {
			continue
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("inbound id %q: %w", idStr, err)
		}
		exists, err := inboundExists(id)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("inbound %d still present on panel after destroy", id)
		}
	}
	return nil
}

// importVLESSClientIDFunc builds an ImportStateIdFunc that emits
// `inbound_id:email` by reading the resource's current state.
func importVLESSClientIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %q not found in state", resourceName)
		}
		inboundID := rs.Primary.Attributes["inbound_id"]
		email := rs.Primary.Attributes["email"]
		if inboundID == "" || email == "" {
			return "", fmt.Errorf("resource %q missing inbound_id or email in state", resourceName)
		}
		return fmt.Sprintf("%s:%s", inboundID, email), nil
	}
}

// createInboundBypassTerraform creates an inbound directly via the xui
// client, bypassing Terraform entirely. This is used to seed acceptance
// tests that need to verify import of an inbound that was NOT created by
// this provider and therefore lacks the sentinel client. The returned id
// is the panel-assigned inbound id; the caller is responsible for cleanup
// (usually Terraform Destroy after import).
func createInboundBypassTerraform(remark string, port int) (int, error) {
	cli, err := accClient()
	if err != nil {
		return 0, err
	}
	// Include one real client so the panel accepts the inbound: panel APIs
	// reject empty client lists on vless.
	settings := fmt.Sprintf(`{"clients":[{"id":"11111111-1111-1111-1111-111111111111","email":%q,"flow":"","enable":true}],"decryption":"none","fallbacks":[]}`, remark+"-seed")
	streamSettings := `{"network":"tcp","security":"none","tcpSettings":{"acceptProxyProtocol":false,"header":{"type":"none"}}}`
	sniffing := `{}`
	payload := map[string]any{
		"remark":         remark,
		"listen":         "",
		"port":           port,
		"protocol":       "vless",
		"settings":       settings,
		"streamSettings": streamSettings,
		"sniffing":       sniffing,
		"enable":         true,
		"expiryTime":     0,
		"trafficReset":   "never",
		"total":          0,
		"up":             0,
		"down":           0,
	}
	raw, err := cli.AddInbound(payload)
	if err != nil {
		return 0, fmt.Errorf("add inbound: %w", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return 0, fmt.Errorf("decode add response: %w", err)
	}
	idF, ok := obj["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("no id in add response: %s", string(raw))
	}
	return int(idF), nil
}

// inboundHasSentinelClient returns whether the given inbound on the panel
// contains the provider-managed sentinel client.
func inboundHasSentinelClient(inboundID int) (bool, error) {
	cli, err := accClient()
	if err != nil {
		return false, err
	}
	raw, err := cli.GetInbound(inboundID)
	if err != nil {
		return false, err
	}
	var inbound map[string]any
	if err := json.Unmarshal(raw, &inbound); err != nil {
		return false, err
	}
	settingsStr := jsonStringFromMap(inbound, "settings")
	uid, err := findDummyClientUUID(settingsStr)
	if err != nil {
		return false, err
	}
	return uid != "", nil
}

// findClientUUIDByEmail returns the VLESS client UUID for the given email on
// the given inbound, or an empty string if not found.
func findClientUUIDByEmail(inboundID int, email string) (string, error) {
	cli, err := accClient()
	if err != nil {
		return "", err
	}
	got, err := cli.GetClientByEmail(email)
	if err != nil {
		return "", fmt.Errorf("get client %q: %w", email, err)
	}
	for _, id := range got.InboundIDs {
		if id == inboundID {
			return xui.PanelClientUUID(got.Client), nil
		}
	}
	return "", nil
}

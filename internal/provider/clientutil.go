package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

func planToPanelClientInput(email string, enable types.Bool, limitIP, totalGB, expiry, tgID, reset types.Int64, flow, subID, comment types.String, id, password string) xui.PanelClientInput {
	c := xui.PanelClientInput{
		Email:      email,
		Enable:     enable.ValueBool(),
		LimitIP:    limitIP.ValueInt64(),
		TotalGB:    totalGB.ValueInt64(),
		ExpiryTime: expiry.ValueInt64(),
		TgID:       tgID.ValueInt64(),
		Reset:      reset.ValueInt64(),
	}
	if id != "" {
		c.ID = id
	}
	if password != "" {
		c.Password = password
	}
	if !flow.IsNull() {
		c.Flow = flow.ValueString()
	}
	if !subID.IsNull() && subID.ValueString() != "" {
		c.SubID = subID.ValueString()
	}
	if !comment.IsNull() {
		c.Comment = comment.ValueString()
	}
	return c
}

func clientAttachedToInbound(got *xui.ClientGetResult, inboundID int) bool {
	for _, id := range got.InboundIDs {
		if id == inboundID {
			return true
		}
	}
	return false
}

func readPanelClientRecord(cli *xui.Client, email string, inboundID int) (*xui.PanelClientRecord, error) {
	got, err := cli.GetClientByEmail(email)
	if err != nil {
		return nil, err
	}
	if !clientAttachedToInbound(got, inboundID) {
		return nil, nil
	}
	rec := got.Client
	return &rec, nil
}

func createPanelClient(cli *xui.Client, email string, inboundID int, input xui.PanelClientInput) (*xui.PanelClientRecord, error) {
	if err := cli.AddClient(xui.ClientCreateRequest{
		Client:     input,
		InboundIDs: []int{inboundID},
	}); err != nil {
		return nil, err
	}
	rec, err := readPanelClientRecord(cli, email, inboundID)
	if err != nil {
		return nil, fmt.Errorf("read client after create: %w", err)
	}
	if rec == nil {
		return nil, fmt.Errorf("client %q not attached to inbound %d after create", email, inboundID)
	}
	return rec, nil
}

package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/syegournov/xkeen-gen/terraform-provider-xui/internal/xui"
)

// panelClientPlan holds the shared Terraform values used to build a panel client payload.
type panelClientPlan struct {
	Email           string
	Enable          types.Bool
	LimitIP         types.Int64
	LimitHwid       types.Int64
	TotalGB         types.Int64
	ExpiryTime      types.Int64
	TgID            types.Int64
	Reset           types.Int64
	ResetDay        types.Int64
	ResetWeekday    types.Int64
	ResetMax        types.Int64
	TrafficReset    types.String
	TrafficResetDay types.Int64
	Group           types.String
	Flow            types.String
	SubID           types.String
	Comment         types.String
	ID              string
	Password        string
	Auth            string
	Security        string
	PrivateKey      string
	PublicKey       string
	PreSharedKey    string
	AllowedIPs      []string
	KeepAlive       types.Int64
	ForwardedPorts  string
}

func planToPanelClientInput(p panelClientPlan) xui.PanelClientInput {
	c := xui.PanelClientInput{
		Email:           p.Email,
		Enable:          p.Enable.ValueBool(),
		LimitIP:         p.LimitIP.ValueInt64(),
		LimitHwid:       p.LimitHwid.ValueInt64(),
		TotalGB:         p.TotalGB.ValueInt64(),
		ExpiryTime:      p.ExpiryTime.ValueInt64(),
		TgID:            p.TgID.ValueInt64(),
		Reset:           p.Reset.ValueInt64(),
		ResetDay:        p.ResetDay.ValueInt64(),
		ResetWeekday:    p.ResetWeekday.ValueInt64(),
		ResetMax:        p.ResetMax.ValueInt64(),
		TrafficResetDay: p.TrafficResetDay.ValueInt64(),
		Group:           p.Group.ValueString(),
	}
	if !p.TrafficReset.IsNull() && p.TrafficReset.ValueString() != "" {
		c.TrafficReset = p.TrafficReset.ValueString()
	}
	if p.ID != "" {
		c.ID = p.ID
	}
	if p.Password != "" {
		c.Password = p.Password
	}
	if p.Auth != "" {
		c.Auth = p.Auth
	}
	if p.Security != "" {
		c.Security = p.Security
	}
	if !p.Flow.IsNull() {
		c.Flow = p.Flow.ValueString()
	}
	if !p.SubID.IsNull() && p.SubID.ValueString() != "" {
		c.SubID = p.SubID.ValueString()
	}
	if !p.Comment.IsNull() {
		c.Comment = p.Comment.ValueString()
	}
	if p.PrivateKey != "" {
		c.PrivateKey = p.PrivateKey
	}
	if p.PublicKey != "" {
		c.PublicKey = p.PublicKey
	}
	if p.PreSharedKey != "" {
		c.PreSharedKey = p.PreSharedKey
	}
	if len(p.AllowedIPs) > 0 {
		c.AllowedIPs = p.AllowedIPs
	}
	if !p.KeepAlive.IsNull() && !p.KeepAlive.IsUnknown() {
		v := p.KeepAlive.ValueInt64()
		c.KeepAlive = &v
	}
	if p.ForwardedPorts != "" {
		c.ForwardedPorts = p.ForwardedPorts
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

func applyCommonClientFields(
	enable *types.Bool,
	limitIP, limitHwid, totalGB, expiry, tgID, reset, resetDay, resetMax, resetWeekday, trafficResetDay *types.Int64,
	trafficReset, subID, comment, group *types.String,
	rec xui.PanelClientRecord,
) {
	*enable = types.BoolValue(rec.Enable)
	*limitIP = types.Int64Value(rec.LimitIP)
	*limitHwid = types.Int64Value(rec.LimitHwid)
	*totalGB = types.Int64Value(rec.TotalGB)
	*expiry = types.Int64Value(rec.ExpiryTime)
	*tgID = types.Int64Value(rec.TgID)
	*subID = types.StringValue(rec.SubID)
	*comment = types.StringValue(rec.Comment)
	*group = types.StringValue(rec.Group)
	*reset = types.Int64Value(rec.Reset)
	*resetDay = types.Int64Value(rec.ResetDay)
	*resetWeekday = types.Int64Value(rec.ResetWeekday)
	*resetMax = types.Int64Value(rec.ResetMax)
	tr := rec.TrafficReset
	if tr == "" {
		tr = "never"
	}
	*trafficReset = types.StringValue(tr)
	*trafficResetDay = types.Int64Value(rec.TrafficResetDay)
}

// finalizeClientSubID sets sub_id from the panel when the plan left it unknown
// (UseStateForUnknown on create). Explicit plan values are kept.
func finalizeClientSubID(plan *types.String, rec xui.PanelClientRecord) {
	if plan.IsUnknown() {
		*plan = types.StringValue(rec.SubID)
	}
}

func panelClientIDFromRecord(rec xui.PanelClientRecord) string {
	if uid := xui.PanelClientUUID(rec); uid != "" {
		return uid
	}
	if rec.Password != "" {
		return rec.Password
	}
	if rec.Auth != "" {
		return rec.Auth
	}
	if rec.PublicKey != "" {
		return rec.PublicKey
	}
	return rec.Email
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

package xui

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// PanelClientRecord is the client shape returned by GET /panel/api/clients/get/:email.
// The JSON "id" field is the panel database row id; VLESS UUID is in "uuid".
type PanelClientRecord struct {
	UUID       string `json:"uuid"`
	Password   string `json:"password"`
	Auth       string `json:"auth"`
	Security   string `json:"security"`
	Email      string `json:"email"`
	Flow       string `json:"flow,omitempty"`
	Enable     bool   `json:"enable"`
	LimitIP    int64  `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	TgID       int64  `json:"tgId"`
	SubID      string `json:"subId,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Reset      int64  `json:"reset"`
}

// PanelClientInput is the client body for POST add and update (model.Client wire shape).
type PanelClientInput struct {
	ID         string `json:"id,omitempty"`
	Password   string `json:"password,omitempty"`
	Auth       string `json:"auth,omitempty"`
	Security   string `json:"security,omitempty"`
	Email      string `json:"email"`
	Flow       string `json:"flow,omitempty"`
	Enable     bool   `json:"enable"`
	LimitIP    int64  `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	TgID       int64  `json:"tgId"`
	SubID      string `json:"subId,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Reset      int64  `json:"reset"`
}

// ClientGetResult is the payload from GET /panel/api/clients/get/:email.
type ClientGetResult struct {
	Client     PanelClientRecord
	InboundIDs []int
}

// ClientCreateRequest is the body for POST /panel/api/clients/add.
type ClientCreateRequest struct {
	Client     PanelClientInput `json:"client"`
	InboundIDs []int            `json:"inboundIds"`
}

// PanelClientUUID returns the VLESS client identifier from a panel client record.
func PanelClientUUID(c PanelClientRecord) string {
	return c.UUID
}

// GetClientByEmail returns a client and the inbound ids it is attached to.
func (c *Client) GetClientByEmail(email string) (*ClientGetResult, error) {
	msg, err := c.get([]string{"panel", "api", "clients", "get", email})
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Client     json.RawMessage `json:"client"`
		InboundIDs []int           `json:"inboundIds"`
	}
	if err := json.Unmarshal(msg.Obj, &wrapper); err != nil {
		return nil, fmt.Errorf("decode client get: %w", err)
	}
	var client PanelClientRecord
	if err := json.Unmarshal(wrapper.Client, &client); err != nil {
		return nil, fmt.Errorf("decode client record: %w", err)
	}
	return &ClientGetResult{
		Client:     client,
		InboundIDs: wrapper.InboundIDs,
	}, nil
}

// AddClient creates a client and attaches it to the given inbounds.
func (c *Client) AddClient(req ClientCreateRequest) error {
	_, err := c.postJSON([]string{"panel", "api", "clients", "add"}, req)
	return err
}

// UpdateClient updates a client identified by email (path parameter).
func (c *Client) UpdateClient(email string, client PanelClientInput) error {
	_, err := c.postJSON([]string{"panel", "api", "clients", "update", email}, client)
	return err
}

// DeleteClient removes a client by email.
func (c *Client) DeleteClient(email string, keepTraffic bool) error {
	endpoint, err := c.join("panel", "api", "clients", "del", email)
	if err != nil {
		return err
	}
	if keepTraffic {
		endpoint += "?keepTraffic=1"
	}
	_, err = c.requestJSON(http.MethodPost, endpoint, nil)
	return err
}

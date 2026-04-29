package xui

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
)

// Client talks to a 3x-ui panel using the same cookie session as the web UI.
type Client struct {
	baseURL *url.URL
	http    *http.Client
	user    string
	pass    string

	// inboundMus serializes read-modify-write sequences against a single
	// inbound's settings. 3x-ui's per-client endpoints (addClient,
	// updateClient, delClient) are stubs / unreliable in current releases,
	// so every client mutation this package performs is done by fetching
	// the inbound, patching settings.clients, and pushing the whole inbound
	// back via /panel/api/inbounds/update. Terraform runs sibling resources
	// concurrently (for_each, -parallelism), so without this lock two
	// parallel client upserts on the same inbound would clobber each other.
	inboundMuMu sync.Mutex
	inboundMus  map[int]*sync.Mutex
}

// StatusPublicIP is a subset of /panel/api/server/status payload.
type StatusPublicIP struct {
	IPv4 string
	IPv6 string
}

// Status is a subset of /panel/api/server/status payload.
type Status struct {
	Uptime    int64
	AppUptime int64
	PublicIP  StatusPublicIP
}

// NewClient builds an HTTP client; baseURL must include the panel path prefix (e.g. https://host:port/<uuid>/).
func NewClient(rawBaseURL, username, password string, insecureSkipVerify bool) (*Client, error) {
	u, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base_url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("base_url must include scheme and host")
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}
	return &Client{
		baseURL: u,
		http: &http.Client{
			Jar:       jar,
			Transport: tr,
		},
		user: username,
		pass: password,
	}, nil
}

func (c *Client) join(elem ...string) (string, error) {
	return c.baseURL.JoinPath(elem...).String(), nil
}

type loginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login establishes a session cookie. Call before other API methods.
func (c *Client) Login() error {
	endpoint, err := c.join("login")
	if err != nil {
		return err
	}
	body, err := json.Marshal(loginBody{Username: c.user, Password: c.pass})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	var msg APIResponse
	if err := json.Unmarshal(b, &msg); err != nil {
		return fmt.Errorf("login: decode response: %w; body=%s", err, truncate(b, 512))
	}
	if !msg.Success {
		return fmt.Errorf("login failed: %s", msg.Msg)
	}
	return nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func (c *Client) postJSON(path []string, payload any) (*APIResponse, error) {
	endpoint, err := c.join(path...)
	if err != nil {
		return nil, err
	}
	var body []byte
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	return c.requestJSON(http.MethodPost, endpoint, body)
}

func (c *Client) get(path []string) (*APIResponse, error) {
	endpoint, err := c.join(path...)
	if err != nil {
		return nil, err
	}
	return c.requestJSON(http.MethodGet, endpoint, nil)
}

func (c *Client) requestJSON(method, endpoint string, body []byte) (*APIResponse, error) {
	if err := c.Login(); err != nil {
		return nil, err
	}
	doOnce := func() ([]byte, int, error) {
		var rdr io.Reader
		if body != nil {
			rdr = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, endpoint, rdr)
		if err != nil {
			return nil, 0, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		return b, resp.StatusCode, err
	}
	b, status, err := doOnce()
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		if err := c.Login(); err != nil {
			return nil, err
		}
		b, _, err = doOnce()
		if err != nil {
			return nil, err
		}
	}
	var msg APIResponse
	if err := json.Unmarshal(b, &msg); err != nil {
		return nil, fmt.Errorf("%s %s: %w; body=%s", method, endpoint, err, truncate(b, 512))
	}
	if !msg.Success {
		return nil, fmt.Errorf("%s %s: %s", method, endpoint, msg.Msg)
	}
	return &msg, nil
}

func (c *Client) postForm(path []string, payload map[string]string) (*APIResponse, error) {
	endpoint, err := c.join(path...)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	for k, v := range payload {
		form.Set(k, v)
	}
	return c.requestForm(endpoint, form)
}

func (c *Client) requestForm(endpoint string, form url.Values) (*APIResponse, error) {
	if err := c.Login(); err != nil {
		return nil, err
	}
	doOnce := func() ([]byte, int, error) {
		req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		return b, resp.StatusCode, err
	}
	b, status, err := doOnce()
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		if err := c.Login(); err != nil {
			return nil, err
		}
		b, _, err = doOnce()
		if err != nil {
			return nil, err
		}
	}
	var msg APIResponse
	if err := json.Unmarshal(b, &msg); err != nil {
		return nil, fmt.Errorf("POST %s: %w; body=%s", endpoint, err, truncate(b, 512))
	}
	if !msg.Success {
		return nil, fmt.Errorf("POST %s: %s", endpoint, msg.Msg)
	}
	return &msg, nil
}

func toJSONString(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		return "", err
	}
	return compact.String(), nil
}

// GetXrayTemplate returns current Xray template JSON from /panel/xray/.
func (c *Client) GetXrayTemplate() (string, error) {
	msg, err := c.postJSON([]string{"panel", "xray"}, map[string]any{})
	if err != nil {
		return "", err
	}
	var objString string
	if err := json.Unmarshal(msg.Obj, &objString); err == nil {
		var wrap map[string]any
		if err := json.Unmarshal([]byte(objString), &wrap); err != nil {
			return "", fmt.Errorf("decode /panel/xray payload: %w", err)
		}
		if x, ok := wrap["xraySetting"]; ok {
			return toJSONString(x)
		}
		return "", fmt.Errorf("decode /panel/xray payload: xraySetting missing")
	}
	var wrap map[string]any
	if err := json.Unmarshal(msg.Obj, &wrap); err != nil {
		return "", fmt.Errorf("decode /panel/xray payload: %w", err)
	}
	x, ok := wrap["xraySetting"]
	if !ok {
		return "", fmt.Errorf("decode /panel/xray payload: xraySetting missing")
	}
	return toJSONString(x)
}

// UpdateXrayTemplate saves Xray template JSON via /panel/xray/update.
func (c *Client) UpdateXrayTemplate(templateJSON string) error {
	_, err := c.postForm([]string{"panel", "xray", "update"}, map[string]string{
		"xraySetting": templateJSON,
	})
	return err
}

// RestartXrayService triggers /panel/api/server/restartXrayService.
func (c *Client) RestartXrayService() error {
	_, err := c.postForm([]string{"panel", "api", "server", "restartXrayService"}, map[string]string{})
	if err == nil {
		return nil
	}
	_, ctErr := c.postJSON([]string{"panel", "api", "server", "restartXrayService"}, map[string]any{})
	if ctErr == nil {
		return nil
	}
	if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "405") || strings.Contains(err.Error(), "415") {
		return ctErr
	}
	return err
}

// ListInbounds returns raw obj JSON (array of inbounds).
func (c *Client) ListInbounds() (json.RawMessage, error) {
	msg, err := c.get([]string{"panel", "api", "inbounds", "list"})
	if err != nil {
		return nil, err
	}
	return msg.Obj, nil
}

// GetInbound returns one inbound as JSON object.
func (c *Client) GetInbound(id int) (json.RawMessage, error) {
	msg, err := c.get([]string{"panel", "api", "inbounds", "get", fmt.Sprintf("%d", id)})
	if err != nil {
		return nil, err
	}
	return msg.Obj, nil
}

// AddInbound creates an inbound; returns created inbound JSON in Obj.
func (c *Client) AddInbound(payload map[string]any) (json.RawMessage, error) {
	msg, err := c.postJSON([]string{"panel", "api", "inbounds", "add"}, payload)
	if err != nil {
		return nil, err
	}
	return msg.Obj, nil
}

// UpdateInbound updates inbound by id (id in URL and body).
func (c *Client) UpdateInbound(id int, payload map[string]any) (json.RawMessage, error) {
	msg, err := c.postJSON([]string{"panel", "api", "inbounds", "update", fmt.Sprintf("%d", id)}, payload)
	if err != nil {
		return nil, err
	}
	return msg.Obj, nil
}

// DeleteInbound removes an inbound.
func (c *Client) DeleteInbound(id int) error {
	_, err := c.postJSON([]string{"panel", "api", "inbounds", "del", fmt.Sprintf("%d", id)}, map[string]any{})
	return err
}

// LockInbound acquires the per-inbound read-modify-write mutex for id and
// returns a release function the caller must defer. 3x-ui's per-client
// endpoints (addClient / updateClient / delClient) don't reliably mutate the
// underlying store, so the provider performs all client-level mutations by
// reading the inbound, patching settings.clients in memory, and pushing the
// full inbound back through /panel/api/inbounds/update. Terraform runs
// sibling resources concurrently (for_each, -parallelism) and without this
// lock two parallel upserts against the same inbound would race and one
// would be lost.
//
// The lock is per-process, per-inbound-id. It does not protect against
// concurrent writes from other clients (the panel web UI, another apply),
// but it is sufficient for a single `terraform apply`, which is the only
// concurrency model Terraform guarantees.
func (c *Client) LockInbound(id int) func() {
	c.inboundMuMu.Lock()
	if c.inboundMus == nil {
		c.inboundMus = map[int]*sync.Mutex{}
	}
	mu, ok := c.inboundMus[id]
	if !ok {
		mu = &sync.Mutex{}
		c.inboundMus[id] = mu
	}
	c.inboundMuMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// GetPanelSettings returns all panel settings as a JSON map.
func (c *Client) GetPanelSettings() (map[string]any, error) {
	msg, err := c.postJSON([]string{"panel", "setting", "all"}, map[string]any{})
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(msg.Obj, &m); err != nil {
		return nil, fmt.Errorf("decode panel settings: %w", err)
	}
	return m, nil
}

// UpdatePanelSettings sends the full settings object to /panel/setting/update.
func (c *Client) UpdatePanelSettings(settings map[string]any) error {
	_, err := c.postJSON([]string{"panel", "setting", "update"}, settings)
	return err
}

// RestartPanel triggers /panel/setting/restartPanel.
func (c *Client) RestartPanel() error {
	// UI uses form body on this endpoint. Keep JSON and /panel/api/server
	// variants as compatibility fallbacks for different panel builds.
	_, err := c.postForm([]string{"panel", "setting", "restartPanel"}, map[string]string{})
	if err == nil {
		return nil
	}
	_, jsonErr := c.postJSON([]string{"panel", "setting", "restartPanel"}, map[string]any{})
	if jsonErr == nil {
		return nil
	}
	_, apiFormErr := c.postForm([]string{"panel", "api", "server", "restartPanel"}, map[string]string{})
	if apiFormErr == nil {
		return nil
	}
	_, apiJSONErr := c.postJSON([]string{"panel", "api", "server", "restartPanel"}, map[string]any{})
	if apiJSONErr == nil {
		return nil
	}
	if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "405") || strings.Contains(err.Error(), "415") {
		return apiJSONErr
	}
	return err
}

// GetStatusPublicIP reads /panel/api/server/status and returns public IPs.
func (c *Client) GetStatusPublicIP() (StatusPublicIP, error) {
	s, err := c.GetStatus()
	if err != nil {
		return StatusPublicIP{}, err
	}
	return s.PublicIP, nil
}

// GetStatus reads /panel/api/server/status and returns selected fields.
func (c *Client) GetStatus() (Status, error) {
	msg, err := c.get([]string{"panel", "api", "server", "status"})
	if err != nil {
		return Status{}, err
	}
	var payload struct {
		Uptime   int64 `json:"uptime"`
		AppStats struct {
			Uptime int64 `json:"uptime"`
		} `json:"appStats"`
		PublicIP struct {
			IPv4 string `json:"ipv4"`
			IPv6 string `json:"ipv6"`
		} `json:"publicIP"`
	}
	if err := json.Unmarshal(msg.Obj, &payload); err != nil {
		return Status{}, fmt.Errorf("decode status payload: %w", err)
	}
	return Status{
		Uptime:    payload.Uptime,
		AppUptime: payload.AppStats.Uptime,
		PublicIP: StatusPublicIP{
			IPv4: payload.PublicIP.IPv4,
			IPv6: payload.PublicIP.IPv6,
		},
	}, nil
}

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
	"time"
)

// Client talks to a 3x-ui panel (v3+) via session cookies or an API bearer token.
type Client struct {
	baseURL  *url.URL
	http     *http.Client
	user     string
	pass     string
	apiToken string
	loginMu  sync.Mutex
	loggedIn bool
	csrfMu   sync.Mutex
	csrf     string

	// inboundMus serializes read-modify-write against a single inbound's
	// settings (e.g. xui_inbound sentinel client maintenance).
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
func NewClient(cfg ClientConfig) (*Client, error) {
	u, err := url.Parse(cfg.BaseURL)
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
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify},
	}
	return &Client{
		baseURL: u,
		http: &http.Client{
			Jar:       jar,
			Transport: tr,
		},
		user:     cfg.Username,
		pass:     cfg.Password,
		apiToken: strings.TrimSpace(cfg.APIToken),
	}, nil
}

func (c *Client) usesBearerAuth(endpoint string) bool {
	return c.apiToken != "" && strings.Contains(endpoint, "/panel/api/")
}

func (c *Client) prepareSession(endpoint string) error {
	if c.usesBearerAuth(endpoint) {
		return nil
	}
	if c.user == "" || c.pass == "" {
		return fmt.Errorf("username and password are required for panel endpoints outside /panel/api/ (got %s)", endpoint)
	}
	return c.Login()
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
	c.loginMu.Lock()
	defer c.loginMu.Unlock()
	if c.loggedIn {
		return nil
	}
	return c.loginLocked()
}

func (c *Client) invalidateSession() {
	c.loginMu.Lock()
	c.loggedIn = false
	c.loginMu.Unlock()
}

func (c *Client) loginLocked() error {
	token, err := c.fetchCSRFToken()
	if err != nil {
		return err
	}
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
	req.Header.Set("X-CSRF-Token", token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	if len(b) == 0 {
		return fmt.Errorf("login: empty response (status %d)", resp.StatusCode)
	}
	var msg APIResponse
	if err := json.Unmarshal(b, &msg); err != nil {
		return fmt.Errorf("login: decode response: %w; body=%s", err, truncate(b, 512))
	}
	if !msg.Success {
		return fmt.Errorf("login failed: %s", msg.Msg)
	}

	// Login rotates the session cookie; refresh the CSRF token bound to the
	// post-login session before any unsafe API calls (3x-ui v3+).
	token, err = c.fetchCSRFToken()
	if err != nil {
		return err
	}
	c.csrfMu.Lock()
	c.csrf = token
	c.csrfMu.Unlock()
	c.loggedIn = true
	return nil
}

func (c *Client) fetchCSRFToken() (string, error) {
	endpoint, err := c.join("csrf-token")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("fetch csrf token: status %d; body=%s", resp.StatusCode, truncate(b, 512))
	}
	var msg APIResponse
	if err := json.Unmarshal(b, &msg); err != nil {
		return "", fmt.Errorf("fetch csrf token: decode response: %w; body=%s", err, truncate(b, 512))
	}
	if !msg.Success {
		return "", fmt.Errorf("fetch csrf token failed: %s", msg.Msg)
	}
	var token string
	if err := json.Unmarshal(msg.Obj, &token); err != nil {
		return "", fmt.Errorf("fetch csrf token: decode obj: %w; body=%s", err, truncate(b, 512))
	}
	return token, nil
}

func (c *Client) csrfToken() string {
	c.csrfMu.Lock()
	defer c.csrfMu.Unlock()
	return c.csrf
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
	if err := c.prepareSession(endpoint); err != nil {
		return nil, err
	}
	return c.requestJSONAuthed(method, endpoint, body, false)
}

func (c *Client) requestJSONAuthed(method, endpoint string, body []byte, retried bool) (*APIResponse, error) {
	bearer := c.usesBearerAuth(endpoint)
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
		if bearer {
			req.Header.Set("Authorization", "Bearer "+c.apiToken)
		} else if method != http.MethodGet && method != http.MethodHead {
			req.Header.Set("X-CSRF-Token", c.csrfToken())
		}
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
	if shouldRetrySession(status) && !retried && !bearer {
		c.invalidateSession()
		if err := c.Login(); err != nil {
			return nil, err
		}
		return c.requestJSONAuthed(method, endpoint, body, true)
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

func shouldRetrySession(status int) bool {
	return status == http.StatusNotFound || status == http.StatusForbidden
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
	if err := c.prepareSession(endpoint); err != nil {
		return nil, err
	}
	return c.requestFormAuthed(endpoint, form, false)
}

func (c *Client) requestFormAuthed(endpoint string, form url.Values, retried bool) (*APIResponse, error) {
	bearer := c.usesBearerAuth(endpoint)
	doOnce := func() ([]byte, int, error) {
		req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		if bearer {
			req.Header.Set("Authorization", "Bearer "+c.apiToken)
		} else {
			req.Header.Set("X-CSRF-Token", c.csrfToken())
		}
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
	if shouldRetrySession(status) && !retried && !bearer {
		c.invalidateSession()
		if err := c.Login(); err != nil {
			return nil, err
		}
		return c.requestFormAuthed(endpoint, form, true)
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
		return c.waitForXrayResultClear(5 * time.Second)
	}
	_, ctErr := c.postJSON([]string{"panel", "api", "server", "restartXrayService"}, map[string]any{})
	if ctErr == nil {
		return c.waitForXrayResultClear(5 * time.Second)
	}
	if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "405") || strings.Contains(err.Error(), "415") {
		return ctErr
	}
	return err
}

// GetXrayResult returns current panel xray result string.
func (c *Client) GetXrayResult() (string, error) {
	msg, err := c.get([]string{"panel", "xray", "getXrayResult"})
	if err != nil {
		return "", err
	}
	var objString string
	if err := json.Unmarshal(msg.Obj, &objString); err == nil {
		return objString, nil
	}
	var objAny any
	if err := json.Unmarshal(msg.Obj, &objAny); err != nil {
		return "", fmt.Errorf("decode /panel/xray/getXrayResult payload: %w", err)
	}
	if objAny == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", objAny), nil
}

func (c *Client) waitForXrayResultClear(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		result, err := c.GetXrayResult()
		if err == nil {
			if strings.TrimSpace(result) == "" {
				return nil
			}
			last = result
		}
		time.Sleep(500 * time.Millisecond)
	}
	if last == "" {
		return fmt.Errorf("xray restart verification timed out")
	}
	return fmt.Errorf("xray restart reported non-empty result: %s", last)
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
// returns a release function the caller must defer.
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

func (c *Client) waitForPanelReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := c.fetchCSRFToken()
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("panel not ready")
	}
	return fmt.Errorf("panel not ready after %s: %w", timeout, lastErr)
}

func (c *Client) finishPanelRestart() error {
	c.invalidateSession()
	return c.waitForPanelReady(30 * time.Second)
}

// RestartPanel triggers /panel/setting/restartPanel.
func (c *Client) RestartPanel() error {
	// UI uses form body on this endpoint. Keep JSON and /panel/api/server
	// variants as compatibility fallbacks for different panel builds.
	_, err := c.postForm([]string{"panel", "setting", "restartPanel"}, map[string]string{})
	if err == nil {
		return c.finishPanelRestart()
	}
	_, jsonErr := c.postJSON([]string{"panel", "setting", "restartPanel"}, map[string]any{})
	if jsonErr == nil {
		return c.finishPanelRestart()
	}
	_, apiFormErr := c.postForm([]string{"panel", "api", "server", "restartPanel"}, map[string]string{})
	if apiFormErr == nil {
		return c.finishPanelRestart()
	}
	_, apiJSONErr := c.postJSON([]string{"panel", "api", "server", "restartPanel"}, map[string]any{})
	if apiJSONErr == nil {
		return c.finishPanelRestart()
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

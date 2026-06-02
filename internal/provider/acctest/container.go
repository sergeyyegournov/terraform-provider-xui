// Package acctest bootstraps a 3x-ui panel running in Docker for acceptance tests.
//
// Typical usage from a TestMain:
//
//	panel, stop, err := acctest.StartPanel(ctx)
//	defer stop()
//
// StartPanel pulls the pinned 3x-ui image, starts it, seeds deterministic
// admin credentials via `docker exec /app/x-ui setting ...`, restarts the
// container so the new settings take effect, and polls the panel URL until
// it serves HTTP 200.
package acctest

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// image3xUI is the pinned panel image. Bumping it is a one-line change plus a
// `make testacc` run. Avoid `:latest` — that breaks test reproducibility.
// Tags are published at https://github.com/mhsanaei/3x-ui/pkgs/container/3x-ui .
const image3xUI = "ghcr.io/mhsanaei/3x-ui:v3.2.0"

// Panel describes a running 3x-ui panel reachable over HTTP(S).
type Panel struct {
	BaseURL  string
	Username string
	Password string
}

const (
	adminUsername = "admin"
	adminPassword = "admin"
	webBasePath   = "tf-acc"
	panelPort     = "2053"
	panelPortTCP  = panelPort + "/tcp"
)

// StartPanel pulls and starts the pinned 3x-ui image, seeds deterministic
// credentials via an in-container `x-ui setting` call, restarts the container
// so the settings take effect, and polls the panel until it responds.
//
// The returned stop function terminates the container and is safe to call
// exactly once; it logs (but does not return) cleanup errors.
func StartPanel(ctx context.Context) (*Panel, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        image3xUI,
		ExposedPorts: []string{panelPortTCP},
		// The panel logs "Web server running HTTP on [::]:2053" after the
		// Gin server binds the port. Match the capitalized prefix so we
		// don't depend on whether HTTPS is configured.
		WaitingFor: wait.ForAll(
			wait.ForLog("Web server running").WithStartupTimeout(90*time.Second),
			wait.ForListeningPort(panelPortTCP).WithStartupTimeout(90*time.Second),
		),
	}
	gcr := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}
	container, err := testcontainers.GenericContainer(ctx, gcr)
	if err != nil {
		return nil, func() {}, fmt.Errorf("start 3x-ui container: %w", err)
	}

	stop := func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(stopCtx); err != nil {
			fmt.Fprintf(os.Stderr, "acctest: terminate 3x-ui container: %v\n", err)
		}
	}

	if err := seedDeterministicCredentials(ctx, container); err != nil {
		stop()
		return nil, func() {}, fmt.Errorf("seed 3x-ui credentials: %w", err)
	}

	if err := restartContainer(ctx, container); err != nil {
		stop()
		return nil, func() {}, fmt.Errorf("restart 3x-ui container: %w", err)
	}

	baseURL, err := resolveBaseURL(ctx, container)
	if err != nil {
		stop()
		return nil, func() {}, err
	}

	if err := waitForHTTPReady(ctx, baseURL, 90*time.Second); err != nil {
		dumpLogsToStderr(ctx, container)
		stop()
		return nil, func() {}, err
	}

	return &Panel{
		BaseURL:  baseURL,
		Username: adminUsername,
		Password: adminPassword,
	}, stop, nil
}

func seedDeterministicCredentials(ctx context.Context, container testcontainers.Container) error {
	cmd := []string{
		"/app/x-ui", "setting",
		"-username", adminUsername,
		"-password", adminPassword,
		"-port", panelPort,
		"-webBasePath", webBasePath,
	}
	code, reader, err := container.Exec(ctx, cmd)
	if err != nil {
		return fmt.Errorf("exec x-ui setting: %w", err)
	}
	out, _ := io.ReadAll(reader)
	if code != 0 {
		return fmt.Errorf("x-ui setting exited %d: %s", code, strings.TrimSpace(string(out)))
	}
	return nil
}

func restartContainer(ctx context.Context, container testcontainers.Container) error {
	timeout := 10 * time.Second
	if err := container.Stop(ctx, &timeout); err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	if err := container.Start(ctx); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	return nil
}

func resolveBaseURL(ctx context.Context, container testcontainers.Container) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("container host: %w", err)
	}
	mapped, err := container.MappedPort(ctx, panelPortTCP)
	if err != nil {
		return "", fmt.Errorf("mapped port: %w", err)
	}
	// 3x-ui serves HTTP by default (no cert configured). Trailing slash is
	// required by the xui.Client URL joining logic.
	return fmt.Sprintf("http://%s:%s/%s/", host, mapped.Port(), webBasePath), nil
}

func waitForHTTPReady(ctx context.Context, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var lastErr error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
		if err != nil {
			return fmt.Errorf("build probe request: %w", err)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		_ = resp.Body.Close()
		// Panel login page returns 200; anything 2xx/3xx means the server is up.
		if resp.StatusCode < 500 {
			return nil
		}
		lastErr = fmt.Errorf("http %d", resp.StatusCode)
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("deadline exceeded")
	}
	return fmt.Errorf("panel %s not ready: %w", baseURL, lastErr)
}

func dumpLogsToStderr(ctx context.Context, container testcontainers.Container) {
	rc, err := container.Logs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "acctest: fetch container logs: %v\n", err)
		return
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	// Print the tail; full logs can be tens of KB of XRay chatter.
	const tail = 4096
	if len(b) > tail {
		b = b[len(b)-tail:]
	}
	fmt.Fprintf(os.Stderr, "acctest: 3x-ui container logs (tail):\n%s\n", string(b))
}

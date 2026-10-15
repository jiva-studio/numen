package plugin

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	errInvalidHandshake = errors.New("invalid or missing handshake from plugin process")
	errHandshakeTimeout = errors.New("timeout waiting for plugin handshake")
)

var defaultPluginClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// Endpoint holds connection parameters and credentials for an active plugin.
type Endpoint struct {
	Host      string
	Port      int
	Token     string
	Transport http.RoundTripper
}

// HTTPClient returns an HTTP client routed to this endpoint.
func (e Endpoint) HTTPClient() *http.Client {
	if e.Transport != nil {
		return &http.Client{Transport: e.Transport}
	}
	return &http.Client{Transport: defaultPluginClient.Transport}
}

// Address returns host and port joined with a colon.
func (e Endpoint) Address() string {
	if e.Port == 0 {
		return e.Host
	}
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

// BaseURL returns the HTTP scheme, host, and port for the plugin server.
func (e Endpoint) BaseURL() string {
	if e.Port == 0 {
		if e.Host != "" {
			return fmt.Sprintf("http://%s", e.Host)
		}
		return "http://inprocess"
	}
	return fmt.Sprintf("http://%s:%d", e.Host, e.Port)
}

// GenerateAuthToken produces a cryptographically secure hex-encoded 32-byte secret token.
func GenerateAuthToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ParseHandshake scans a text line for readiness indicator and allocated port.
func ParseHandshake(line string) (int, bool) {
	trimmed := strings.TrimSpace(line)
	prefix := "NUMEN_READY"
	if !strings.HasPrefix(trimmed, prefix) {
		prefix = "READY"
		if !strings.HasPrefix(trimmed, prefix) {
			return 0, false
		}
	}

	parts := strings.Fields(trimmed)
	for _, part := range parts {
		if strings.HasPrefix(part, "port=") {
			portStr := strings.TrimPrefix(part, "port=")
			port, err := strconv.Atoi(portStr)
			if err == nil && port > 0 {
				return port, true
			}
		}
	}
	return 0, false
}

// WaitForHandshake reads lines from reader until a handshake line is observed or ctx expires.
func WaitForHandshake(ctx context.Context, r io.Reader) (int, error) {
	type result struct {
		port int
		err  error
	}

	resultCh := make(chan result, 1)

	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if port, ok := ParseHandshake(line); ok {
				resultCh <- result{port: port, err: nil}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			resultCh <- result{port: 0, err: fmt.Errorf("scan handshake: %w", err)}
			return
		}
		resultCh <- result{port: 0, err: errInvalidHandshake}
	}()

	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("%w: %w", errHandshakeTimeout, ctx.Err())
	case res := <-resultCh:
		return res.port, res.err
	}
}

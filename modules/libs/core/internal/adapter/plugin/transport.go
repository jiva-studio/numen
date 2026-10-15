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
	"strconv"
	"strings"
)

var (
	errInvalidHandshake = errors.New("invalid or missing handshake from plugin process")
	errHandshakeTimeout = errors.New("timeout waiting for plugin handshake")
)

// Endpoint holds loopback connection parameters and credentials for an active plugin.
type Endpoint struct {
	Host  string
	Port  int
	Token string
}

// Address returns host and port joined with a colon.
func (e Endpoint) Address() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

// BaseURL returns the HTTP scheme, host, and port for the plugin server.
func (e Endpoint) BaseURL() string {
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

// FindAvailablePort reserves an unused TCP port on loopback interface and releases it.
func FindAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen loopback: %w", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("failed to cast net.Addr to *net.TCPAddr")
	}
	return addr.Port, nil
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

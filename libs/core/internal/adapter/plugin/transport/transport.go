package transport

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
	"net/url"
	"strconv"
	"strings"
	"time"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var (
	errInvalidHandshake = errors.New("invalid or missing handshake from plugin process")
	ErrHandshakeTimeout = errors.New("timeout waiting for plugin handshake")
)

// NetworkTransport is the HTTP transport of one plugin process: the connections to its port are
// pooled for the life of the process.
type NetworkTransport struct{ *http.Transport }

// RoundTripTransport marks NetworkTransport as a PluginTransport.
func (NetworkTransport) RoundTripTransport() {}

// NewNetworkTransport returns the transport for a plugin process that listens on a port.
func NewNetworkTransport() NetworkTransport {
	return NetworkTransport{Transport: newHTTPTransport()}
}

func newHTTPTransport() *http.Transport {
	return &http.Transport{
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
	}
}

// PluginTransport is an in-process or custom transport for a plugin endpoint.
type PluginTransport = pluginport.PluginTransport

// GetBaseURL returns the HTTP scheme, host and port of the plugin server at e.
// An endpoint with neither host nor port is the in-process server.
func GetBaseURL(e Endpoint) string {
	host := e.Address()
	if host == "" {
		host = "inprocess"
	}
	return (&url.URL{Scheme: "http", Host: host}).String()
}

// EndpointClient returns an HTTP client routed to this endpoint.
func EndpointClient(e Endpoint) *http.Client {
	if e.Transport != nil {
		e.Transport.RoundTripTransport()
		if rt, ok := e.Transport.(http.RoundTripper); ok && rt != nil {
			return &http.Client{Transport: rt}
		}
	}
	// An endpoint without a transport of its own keeps no connection open after the call.
	unpooled := newHTTPTransport()
	unpooled.DisableKeepAlives = true
	return &http.Client{Transport: unpooled}
}

// Endpoint holds connection parameters and credentials for an active plugin.
type Endpoint = pluginport.Endpoint

// GenerateAuthToken produces a cryptographically secure hex-encoded 32-byte secret token.
func GenerateAuthToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// Handshake is the ready line of a plugin process.
type Handshake struct {
	// Port is the loopback port of the plugin's RPC server.
	Port int
	// Listeners maps the name of each listener the plugin opened to its loopback port.
	Listeners map[string]int
}

const listenerTokenPrefix = "listener."

var errMalformedListenerToken = errors.New("malformed listener token in the ready line")

// ParseHandshake reads a ready line: the port of the plugin's RPC server and one
// listener.<name>=<port> token per listener it opened. ok is false for a line that is not a ready
// line; err is set for a ready line with a malformed listener token.
func ParseHandshake(line string) (handshake Handshake, ok bool, err error) {
	cleanLine := strings.TrimSpace(line)
	if !strings.HasPrefix(cleanLine, "NUMEN_READY") && !strings.HasPrefix(cleanLine, "READY") {
		return Handshake{}, false, nil
	}
	handshake = Handshake{Port: 0, Listeners: map[string]int{}}
	for _, part := range strings.Fields(cleanLine) {
		if rest, isListener := strings.CutPrefix(part, listenerTokenPrefix); isListener {
			if err := addListenerToken(handshake.Listeners, rest); err != nil {
				return Handshake{}, true, fmt.Errorf("%w: %q: %w", errMalformedListenerToken, part, err)
			}
			continue
		}
		value, isPort := strings.CutPrefix(part, "port=")
		if port, err := strconv.Atoi(value); isPort && err == nil && port > 0 && handshake.Port == 0 {
			handshake.Port = port
		}
	}
	return handshake, handshake.Port > 0, nil
}

var (
	errListenerTokenName = errors.New("no listener name or a name given twice")
	errListenerTokenPort = errors.New("no positive integer port")
)

// addListenerToken adds the listener of a token <name>=<port> to listeners.
func addListenerToken(listeners map[string]int, token string) error {
	name, value, hasValue := strings.Cut(token, "=")
	if _, isRepeated := listeners[name]; name == "" || isRepeated {
		return errListenerTokenName
	}
	port, err := strconv.Atoi(value)
	if !hasValue || err != nil || port <= 0 {
		return errListenerTokenPort
	}
	listeners[name] = port
	return nil
}

// WaitForHandshake reads lines from reader until a handshake line is observed or ctx expires. A ready
// line with a malformed listener token returns its error at once.
func WaitForHandshake(ctx context.Context, r io.Reader) (Handshake, error) {
	type result struct {
		handshake Handshake
		err       error
	}

	resultCh := make(chan result, 1)

	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			handshake, ok, err := ParseHandshake(scanner.Text())
			if ok || err != nil {
				resultCh <- result{handshake: handshake, err: err}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			resultCh <- result{handshake: Handshake{}, err: fmt.Errorf("scan handshake: %w", err)}
			return
		}
		resultCh <- result{handshake: Handshake{}, err: errInvalidHandshake}
	}()

	select {
	case <-ctx.Done():
		return Handshake{}, fmt.Errorf("%w: %w", ErrHandshakeTimeout, ctx.Err())
	case res := <-resultCh:
		return res.handshake, res.err
	}
}

// ModelInfoTimeout bounds a request for the model a plugin runs. A plugin answers from what it
// already holds and does not load the model for it, so a plugin that does not answer in this
// time is stuck.
const ModelInfoTimeout = 10 * time.Second

// ModelRefreshTimeout bounds the read of the model that a caller waits for while it does other
// work. A caller that does not get an answer in this time uses the model it read before.
const ModelRefreshTimeout = 2 * time.Second

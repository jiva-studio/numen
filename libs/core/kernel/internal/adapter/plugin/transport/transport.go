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
	"sync"
	"time"
)

var (
	errInvalidHandshake = errors.New("invalid or missing handshake from plugin process")
	ErrHandshakeTimeout = errors.New("timeout waiting for plugin handshake")
)

// NewNetworkTransport returns the HTTP transport of one plugin process that listens on a port: the
// connections to its port are pooled for the life of the process.
func NewNetworkTransport() *http.Transport {
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

// GetBaseURL returns the HTTP scheme, host and port of the plugin server at e.
// An endpoint with neither host nor port is the in-process server.
func GetBaseURL(e Endpoint) string {
	host := e.GetAddress()
	if host == "" {
		host = "inprocess"
	}
	return (&url.URL{Scheme: "http", Host: host}).String()
}

// Clients keeps the HTTP client of each running plugin process, by the token of its endpoint.
type Clients struct {
	mu      sync.Mutex
	byToken map[string]*http.Client
}

// NewClients returns an empty set of clients.
func NewClients() *Clients {
	return &Clients{mu: sync.Mutex{}, byToken: map[string]*http.Client{}}
}

// Add keeps a client that sends through the round tripper for the endpoint with the token.
func (c *Clients) Add(token string, roundTripper http.RoundTripper) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byToken[token] = &http.Client{Transport: roundTripper}
}

// Remove forgets the client of the endpoint with the token and closes its idle connections.
func (c *Clients) Remove(token string) {
	c.mu.Lock()
	client := c.byToken[token]
	delete(c.byToken, token)
	c.mu.Unlock()
	if client != nil {
		client.CloseIdleConnections()
	}
}

// Get returns the client kept for the endpoint, or an unpooled client when none is kept.
func (c *Clients) Get(e Endpoint) *http.Client {
	c.mu.Lock()
	client := c.byToken[e.Token]
	c.mu.Unlock()
	if client != nil {
		return client
	}
	return newUnpooledClient()
}

// ClientLookup finds the client of an endpoint among the clients of a plugin host. The zero value
// finds an unpooled client for every endpoint.
type ClientLookup struct{ clients *Clients }

// NewClientLookup returns the lookup over the clients.
func NewClientLookup(clients *Clients) ClientLookup { return ClientLookup{clients: clients} }

// GetClient returns the client kept for the endpoint, or an unpooled client when none is kept.
func (l ClientLookup) GetClient(e Endpoint) *http.Client {
	if l.clients == nil {
		return newUnpooledClient()
	}
	return l.clients.Get(e)
}

// ClientSource is a plugin host that keeps the client of each endpoint it started.
type ClientSource interface {
	GetClient(e Endpoint) *http.Client
}

// GetClient returns the client of the endpoint from the host when it keeps one, else an unpooled
// client, which keeps no connection open after the call.
func GetClient(host any, e Endpoint) *http.Client {
	if source, ok := host.(ClientSource); ok {
		return source.GetClient(e)
	}
	return newUnpooledClient()
}

func newUnpooledClient() *http.Client {
	unpooled := NewNetworkTransport()
	unpooled.DisableKeepAlives = true
	return &http.Client{Transport: unpooled}
}

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
	Port      int
	Listeners map[string]int
}

const listenerTokenPrefix = "listener."

var errMalformedListenerToken = errors.New("malformed listener token in the ready line")

// ParseHandshake reads a ready line, one that starts with NUMEN_READY: the port of the plugin's RPC
// server and one listener.<name>=<port> token per listener it opened. ok is false for a line that is
// not a ready line; err is set for a ready line with a malformed listener token.
func ParseHandshake(line string) (handshake Handshake, ok bool, err error) {
	cleanLine := strings.TrimSpace(line)
	if !strings.HasPrefix(cleanLine, "NUMEN_READY") {
		return Handshake{}, false, nil
	}
	handshake = Handshake{Port: 0, Listeners: map[string]int{}}
	for part := range strings.FieldsSeq(cleanLine) {
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

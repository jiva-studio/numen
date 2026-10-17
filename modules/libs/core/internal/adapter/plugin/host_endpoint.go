package plugin

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const (
	hostEndpointAddress           = "127.0.0.1:0"
	hostEndpointHeaderTimeout     = 10 * time.Second
	hostEndpointShutdownTolerance = 5 * time.Second
	bearerPrefix                  = "Bearer "
)

var (
	errUnknownCaller = errors.New("the token names no running plugin")
	errNotTCP        = errors.New("the host endpoint listens on no TCP address")
	errNotLoopback   = errors.New("the host endpoint listens on the loopback interface only")
)

// HostCallers is what the host endpoint asks the host about a call: who made
// it, what that plugin may do, and who answers a completion.
type HostCallers interface {
	GetPluginIDByToken(token string) (string, bool)
	CheckPluginPermission(pluginID, permission string) error
	GetCompleter() port.Completer
}

// HostEndpoint answers the calls plugins make to the host. It listens on the
// loopback interface and serves a call only for a token the host gave a
// running plugin.
type HostEndpoint struct {
	callers HostCallers
	server  *http.Server
	port    int
}

// OpenHostEndpoint starts serving the host service on the address, which is a
// free loopback port by default.
func OpenHostEndpoint(ctx context.Context, address string, callers HostCallers) (*HostEndpoint, error) {
	if err := checkLoopback(address); err != nil {
		return nil, err
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen for plugins: %w", err)
	}
	bound, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return nil, fmt.Errorf("%w: %s", errNotTCP, listener.Addr())
	}
	endpoint := &HostEndpoint{callers: callers, port: bound.Port}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostServiceHandler(endpoint))
	endpoint.server = &http.Server{Handler: mux, ReadHeaderTimeout: hostEndpointHeaderTimeout}
	go func() {
		// Serve returns http.ErrServerClosed after Close.
		_ = endpoint.server.Serve(listener)
	}()
	return endpoint, nil
}

// GetPort is the loopback port the endpoint listens on.
func (e *HostEndpoint) GetPort() int {
	return e.port
}

// Close stops the endpoint and waits for the calls being served to end, up to
// the context.
func (e *HostEndpoint) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, hostEndpointShutdownTolerance)
	defer cancel()
	if err := e.server.Shutdown(ctx); err != nil {
		_ = e.server.Close()
		return fmt.Errorf("close host endpoint: %w", err)
	}
	return nil
}

// Complete answers a plugin's question with text from the model the question's
// purpose is routed to. A caller that holds no valid token is refused as
// unauthenticated, and one without the ai:complete permission as denied.
func (e *HostEndpoint) Complete(
	ctx context.Context,
	req *connect.Request[v1.HostServiceCompleteRequest],
) (*connect.Response[v1.HostServiceCompleteResponse], error) {
	if err := e.authorize(req.Header().Get("Authorization"), PermissionAIComplete); err != nil {
		return nil, err
	}
	completer := e.callers.GetCompleter()
	if completer == nil {
		return refuseCompletion(v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE), nil
	}
	text, err := completer.Complete(ctx, readWireQuestion(req.Msg.GetQuestion()))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, connect.NewError(connect.CodeCanceled, ctxErr)
		}
		if code, isRefusal := writeRefusal(err); isRefusal {
			return refuseCompletion(code), nil
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.HostServiceCompleteResponse{
		Answer: &v1.CompletionAnswer{Text: text},
	}), nil
}

// authorize finds the plugin the Authorization header names and checks it holds
// the permission.
func (e *HostEndpoint) authorize(header, permission string) error {
	token, found := strings.CutPrefix(header, bearerPrefix)
	pluginID, known := e.callers.GetPluginIDByToken(token)
	if !found || !known {
		return connect.NewError(connect.CodeUnauthenticated, errUnknownCaller)
	}
	if err := e.callers.CheckPluginPermission(pluginID, permission); err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("plugin %s: %w", pluginID, err))
	}
	return nil
}

func refuseCompletion(code v1.CompletionErrorCode) *connect.Response[v1.HostServiceCompleteResponse] {
	return connect.NewResponse(&v1.HostServiceCompleteResponse{Error: code})
}

// GetPluginIDByToken returns the id of the running plugin the token was given
// to. An empty token names none.
func (h *Host) GetPluginIDByToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	var pluginID string
	var found bool
	for id, endpoint := range h.endpoints {
		if subtle.ConstantTimeCompare([]byte(endpoint.Token), []byte(token)) == 1 {
			pluginID, found = id, true
		}
	}
	return pluginID, found
}

// SetCompleter sets what answers the completions plugins ask for. The host
// endpoint starts serving when the first plugin launches after it is set.
func (h *Host) SetCompleter(completer port.Completer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.completer = completer
}

func (h *Host) GetCompleter() port.Completer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.completer
}

// openHostEndpoint returns the port the host endpoint listens on, starting it
// on the first call after a completer is set. It returns zero while none is set.
func (h *Host) openHostEndpoint(ctx context.Context) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hostEndpoint != nil {
		return h.hostEndpoint.GetPort(), nil
	}
	if h.completer == nil || h.isClosed {
		return 0, nil
	}
	endpoint, err := OpenHostEndpoint(ctx, h.hostAddress, h)
	if err != nil {
		return 0, err
	}
	h.hostEndpoint = endpoint
	return endpoint.GetPort(), nil
}

// closeHostEndpoint stops the host endpoint if it is serving.
func (h *Host) closeHostEndpoint(ctx context.Context) error {
	h.mu.Lock()
	endpoint := h.hostEndpoint
	h.hostEndpoint = nil
	h.mu.Unlock()
	if endpoint == nil {
		return nil
	}
	return endpoint.Close(ctx)
}

// getLaunchVars is the environment variables the host sets for a plugin process
// about to start. The host endpoint is started for it if a completer is set. An
// endpoint that cannot start is reported and leaves the plugin without
// NUMEN_HOST_PORT; the plugins that never call the host launch all the same.
func (h *Host) getLaunchVars(ctx context.Context, pluginID string, processPort int, token string) ([]string, error) {
	hostPort, err := h.openHostEndpoint(ctx)
	if err != nil {
		h.reportError(fmt.Errorf("open host endpoint for plugin %s: %w", pluginID, err))
		hostPort = 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	configJSON, err := h.getPluginConfigJSONLocked(pluginID)
	if err != nil {
		return nil, err
	}
	return getPluginVars(pluginID, launchEnvironment{
		vaultPath: h.vaultPath, port: processPort, hostPort: hostPort, token: token, configJSON: configJSON,
	}), nil
}

// checkLoopback accepts only an address that names a loopback IP: an empty host
// or a name would bind more than the loopback interface.
func checkLoopback(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("host endpoint address %q: %w", address, err)
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !ip.IsLoopback() {
		return fmt.Errorf("%w: %q", errNotLoopback, address)
	}
	return nil
}

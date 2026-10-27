package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const (
	Address                       = "127.0.0.1:0"
	hostEndpointHeaderTimeout     = 10 * time.Second
	hostEndpointShutdownTolerance = 5 * time.Second
	bearerPrefix                  = "Bearer "
)

var (
	errUnknownCaller = errors.New("the token names no running plugin")
	errNotTCP        = errors.New("the host endpoint listens on no TCP address")
	errNotLoopback   = errors.New("the host endpoint listens on the loopback interface only")
)

// Callers is what the host endpoint asks the host about a call: who made
// it, what that plugin may do, whether the person lets it ask a model, and who
// answers a completion.
type Callers interface {
	Services
	GetPluginIDByToken(token string) (string, bool)
	CheckPluginPermission(pluginID, permission string) error
	CheckCompletionConsent(pluginID string) error
	// Acquire keeps a running plugin from being stopped as idle until Release.
	Acquire(pluginID string)
	Release(pluginID string)
}

// Services is what the host lends the plugins that call it.
type Services interface {
	GetCompleter() completionport.Completer
	// GetPluginVault is the vault the host shows, or nil where it has none to offer.
	GetPluginVault() pluginport.PluginVault
	// GetPluginTexts is the text versions the plugins work, or nil.
	GetPluginTexts() pluginport.PluginTexts
	// GetDependencies is what provisions the dependencies, or nil.
	GetDependencies() pluginport.DependencyCatalog
	// GetPluginWork is the queue and the list of tasks the plugins use, or nil.
	GetPluginWork() pluginport.PluginWork
	// GetPluginTools is the tools of the plugins a plugin serves to programs outside Numen, or nil.
	GetPluginTools() PluginTools
}

// Endpoint answers the calls plugins make to the host. It listens on the
// loopback interface and serves a call only for a token the host gave a
// running plugin.
type Endpoint struct {
	callers Callers
	server  *http.Server
	port    int
}

// Open starts serving the host service on the address, which is a
// free loopback port by default.
func Open(ctx context.Context, address string, callers Callers) (*Endpoint, error) {
	if err := checkLoopback(address); err != nil {
		return nil, err
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen for plugins: %w", err)
	}
	linked, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return nil, fmt.Errorf("%w: %s", errNotTCP, listener.Addr())
	}
	endpoint := &Endpoint{callers: callers, port: linked.Port}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostServiceHandler(endpoint))
	mux.Handle(numenv1connect.NewHostVaultServiceHandler(newVaultEndpoint(endpoint)))
	mux.Handle(numenv1connect.NewHostStoreServiceHandler(&storeEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewHostTextServiceHandler(&textEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewDependencyServiceHandler(&dependencyEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewHostQueueServiceHandler(&queueEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(&toolEndpoint{endpoint: endpoint}))
	endpoint.server = &http.Server{Handler: mux, ReadHeaderTimeout: hostEndpointHeaderTimeout}
	go func() {
		// Serve returns http.ErrServerClosed after Close.
		_ = endpoint.server.Serve(listener)
	}()
	return endpoint, nil
}

// GetPort is the loopback port the endpoint listens on.
func (e *Endpoint) GetPort() int {
	return e.port
}

// Close stops the endpoint and waits for the calls being served to end, up to
// the context.
func (e *Endpoint) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, hostEndpointShutdownTolerance)
	defer cancel()
	if err := e.server.Shutdown(ctx); err != nil {
		_ = e.server.Close()
		return fmt.Errorf("close host endpoint: %w", err)
	}
	return nil
}

// Complete answers a plugin's question with text from the agent its setting
// unauthenticated, and one without the ai:complete permission as denied. A
// plugin the person has not let ask a model is answered NO_CONSENT.
func (e *Endpoint) Complete(
	ctx context.Context,
	req *connect.Request[v1.HostServiceCompleteRequest],
) (*connect.Response[v1.HostServiceCompleteResponse], error) {
	pluginID, err := e.authorize(req.Header().Get("Authorization"), plugindir.PermissionAIComplete)
	if err != nil {
		return nil, err
	}
	if err := e.callers.CheckCompletionConsent(pluginID); err != nil {
		return answerFailure(ctx, err)
	}
	completer := e.callers.GetCompleter()
	if completer == nil {
		return refuseCompletion(v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, ""), nil
	}
	text, err := completer.Complete(ctx, pluginID, readWireQuestion(req.Msg.GetQuestion()))
	if err != nil {
		return answerFailure(ctx, err)
	}
	return connect.NewResponse(&v1.HostServiceCompleteResponse{
		Answer: &v1.CompletionAnswer{Text: text},
	}), nil
}

// authorize returns the id of the plugin the Authorization header names once it
// is known to hold the permission.
func (e *Endpoint) authenticate(header string) (string, error) {
	token, found := strings.CutPrefix(header, bearerPrefix)
	pluginID, recognized := e.callers.GetPluginIDByToken(token)
	if !found || !recognized {
		return "", connect.NewError(connect.CodeUnauthenticated, errUnknownCaller)
	}
	return pluginID, nil
}

// authorize returns the id of the plugin the Authorization header names once it
// is known to hold the permission.
func (e *Endpoint) authorize(header, permission string) (string, error) {
	pluginID, err := e.authenticate(header)
	if err != nil {
		return "", err
	}
	if err := e.callers.CheckPluginPermission(pluginID, permission); err != nil {
		return "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("plugin %s: %w", pluginID, err))
	}
	return pluginID, nil
}

// answerFailure answers a call that failed: the end of the caller's context is
// answered with its code, and anything else is internal.
func answerFailure(ctx context.Context, err error) (*connect.Response[v1.HostServiceCompleteResponse], error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, completionport.ErrInvalidQuestion) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if code, isRefusal := writeRefusal(err); isRefusal {
		return refuseCompletion(code, readRefusalDetail(err)), nil
	}
	return nil, connect.NewError(connect.CodeInternal, err)
}

func refuseCompletion(code v1.CompletionErrorCode, detail string) *connect.Response[v1.HostServiceCompleteResponse] {
	return connect.NewResponse(&v1.HostServiceCompleteResponse{Error: code, Detail: detail})
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

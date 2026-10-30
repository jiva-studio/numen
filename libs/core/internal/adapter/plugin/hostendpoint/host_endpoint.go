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

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

const (
	Address                       = "127.0.0.1:0"
	hostEndpointHeaderTimeout     = 10 * time.Second
	hostEndpointShutdownTolerance = 5 * time.Second
	bearerPrefix                  = "Bearer "
)

var (
	errUnknownCaller = errors.New("the token names no running plugin")
	errNotTCP        = errors.New("the host endpoint has no TCP address")
	errNotLoopback   = errors.New("the host endpoint is not bound to the loopback interface")
)

// Callers is what the host endpoint queries the host for about a call: who made
// it, what that plugin may do, whether the user permits it to use a model, and
// who serves a completion.
type callerLease interface {
	Acquire(pluginID string)
	Release(pluginID string)
}

type Callers interface {
	Services
	GetPluginIDByToken(token string) (string, bool)
	CheckPluginPermission(pluginID, permission string) error
	CheckCompletionConsent(pluginID string) error
	callerLease
}

type workServices interface {
	GetPluginWork() pluginport.PluginWork
	GetPluginTools() PluginTools
}

// Services is what the host lends the plugins that call it.
type Services interface {
	GetCompleter() completionport.Completer
	GetPluginVault() pluginport.PluginVault
	GetPluginTexts() pluginport.PluginTexts
	GetDependencies() pluginport.DependencyCatalog
	workServices
}

// Endpoint serves the calls plugins make to the host. It binds the loopback
// interface and serves a call only for a token the host issued to a running
// plugin.
type Endpoint struct {
	callers Callers
	server  *http.Server
	port    int
	// serveResult receives the error that ended the server other than a close, and is closed once the server no longer serves.
	serveResult chan error
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
		return nil, errors.Join(fmt.Errorf("%w: %s", errNotTCP, listener.Addr()), listener.Close())
	}
	endpoint := &Endpoint{callers: callers, port: linked.Port, serveResult: make(chan error, 1)}

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
		defer close(endpoint.serveResult)
		// Serve returns http.ErrServerClosed after Close.
		if err := endpoint.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			endpoint.serveResult <- err
		}
	}()
	return endpoint, nil
}

// GetPort returns the loopback port the endpoint serves on.
func (e *Endpoint) GetPort() int {
	return e.port
}

// Close stops the endpoint and waits for the calls being served to end, up to
// the context.
func (e *Endpoint) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, hostEndpointShutdownTolerance)
	defer cancel()
	if err := e.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("close host endpoint: %w", errors.Join(err, e.server.Close()))
	}
	if err := <-e.serveResult; err != nil {
		return fmt.Errorf("serve host endpoint: %w", err)
	}
	return nil
}

// Complete serves a plugin's completion request with text from the configured
// agent. A request without a valid token is unauthenticated, one without the
// ai:complete permission is denied, and a plugin the user has not allowed to use
// a model receives NO_CONSENT.
func (e *Endpoint) Complete(
	ctx context.Context,
	req *connect.Request[v1.HostServiceCompleteRequest],
) (*connect.Response[v1.HostServiceCompleteResponse], error) {
	pluginID, err := e.authorize(req.Header().Get("Authorization"), manifestfile.PermissionAIComplete)
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

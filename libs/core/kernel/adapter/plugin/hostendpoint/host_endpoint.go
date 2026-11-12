package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/completion"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
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

// Endpoint serves the calls plugins make to the host. It binds the loopback
// interface and serves a call only for a token the host issued to a running
// plugin.
type Endpoint struct {
	callers     Callers
	tasks       taskLeases
	server      *http.Server
	port        int
	serveResult chan error
	callCount   atomic.Int64
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
	mux.Handle(vault.NewHandler(endpoint, callers.Lent))
	mux.Handle(numenv1connect.NewHostStoreServiceHandler(&storeEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewHostTextServiceHandler(&textEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewDependencyServiceHandler(&dependencyEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewHostQueueServiceHandler(&queueEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewHostSearchServiceHandler(&searchEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(&toolEndpoint{endpoint: endpoint}))
	mux.Handle(numenv1connect.NewHostWindowServiceHandler(&windowEndpoint{endpoint: endpoint}))
	countingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint.callCount.Add(1)
		defer endpoint.callCount.Add(-1)
		mux.ServeHTTP(w, r)
	})
	endpoint.server = &http.Server{Handler: countingHandler, ReadHeaderTimeout: hostEndpointHeaderTimeout}
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
		// Shutdown waits up to 5 seconds for a connection that has sent no request.
		isIdle := errors.Is(err, context.DeadlineExceeded) && e.callCount.Load() == 0
		closeErr := e.server.Close()
		if !isIdle || closeErr != nil {
			return fmt.Errorf("close host endpoint: %w", errors.Join(err, closeErr))
		}
	}
	if err := <-e.serveResult; err != nil {
		return fmt.Errorf("serve host endpoint: %w", err)
	}
	return nil
}

// Complete serves a plugin's completion request with text from the configured
// agent. A request without a valid token is unauthenticated, one without the
// ai:complete permission is denied.
func (e *Endpoint) Complete(
	ctx context.Context,
	req *connect.Request[v1.HostServiceCompleteRequest],
) (*connect.Response[v1.HostServiceCompleteResponse], error) {
	pluginID, err := e.Authorize(req.Header().Get("Authorization"), manifest.PermissionAIComplete)
	if err != nil {
		return nil, err
	}
	completer := e.callers.Lent.GetCompleter()
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

func (e *Endpoint) getVault(header http.Header) (*vault.Access, error) {
	return vault.Select(e.callers.Lent.GetPluginVault(), header)
}

func (e *Endpoint) authenticate(header string) (string, error) {
	token, found := strings.CutPrefix(header, bearerPrefix)
	pluginID, recognized := e.callers.Identity.GetPluginIDByToken(token)
	if !found || !recognized {
		return "", connect.NewError(connect.CodeUnauthenticated, errUnknownCaller)
	}
	return pluginID, nil
}

func (e *Endpoint) Authorize(header, permission string) (string, error) {
	pluginID, err := e.authenticate(header)
	if err != nil {
		return "", err
	}
	if err := e.callers.Identity.CheckPluginPermission(pluginID, permission); err != nil {
		return "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("plugin %s: %w", pluginID, err))
	}
	return pluginID, nil
}

func answerFailure(ctx context.Context, err error) (*connect.Response[v1.HostServiceCompleteResponse], error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, completion.ErrInvalidQuestion) {
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

// ClearPluginTasks forgets the tasks a plugin reported once its process has ended.
func (e *Endpoint) ClearPluginTasks(pluginID string) {
	e.tasks.clear(pluginID)
}

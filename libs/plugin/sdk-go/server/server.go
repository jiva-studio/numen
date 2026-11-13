package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// tokenVar names the variable holding the token the host authenticates with.
const tokenVar = hostcall.PluginTokenEnv

func parseDefaultFlags() int {
	port := readFlagPort()
	if port == 0 {
		port = readEnvPort()
	}
	return port
}

func readFlagPort() int {
	if !flag.Parsed() {
		if flag.CommandLine.Lookup("port") == nil {
			flag.Int("port", 0, "Port to listen on")
		}
		flag.Parse()
	}
	f := flag.CommandLine.Lookup("port")
	if f == nil {
		return 0
	}
	port, err := strconv.Atoi(f.Value.String())
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err != nil {
		return 0
	}
	return port
}

func readEnvPort() int {
	port, err := strconv.Atoi(os.Getenv(hostcall.PluginPortEnv))
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err != nil {
		return 0
	}
	return port
}

// Option configures Server instances.
type Option func(*Server)

// Server coordinates the lifecycle, authentication, and HTTP/2 transport of a plugin.
type Server struct {
	port              int
	token             string
	stdout            io.Writer
	stdin             io.Reader
	signals           []os.Signal
	shutdownTimeout   time.Duration
	readHeaderTimeout time.Duration
	onEvent           EventHandler

	closeOnce sync.Once
	closers   []func() error
	closeErr  error

	lifecycle   *LifecycleServer
	interceptor *AuthInterceptor
	mux         *http.ServeMux
	listener    net.Listener
	actualPort  int
	listeners   []listenerPort
}

// New constructs a plugin server using command flags, environment variables, and options.
func New(opts ...Option) *Server {
	port := parseDefaultFlags()
	return newServer(port, "", opts, false)
}

// NewLinkedHandler returns the handler a host that links the plugin into its own process serves it
// through: a server with the services mount registers on it, which authenticates with the token
// vars holds under NUMEN_PLUGIN_TOKEN and reads no flag.
func NewLinkedHandler(vars map[string]string, mount func(*Server) error) (http.Handler, error) {
	token := vars[tokenVar]
	if token == "" {
		return nil, errors.New("empty auth token")
	}
	s := newServer(0, token, nil, true)
	if err := mount(s); err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return s.mux, nil
}

// Run starts a server, mounts services, and serves until shutdown.
// When mounting or serving fails, Run writes the error to standard error and exits with code 1.
func Run(mount func(*Server) error, opts ...Option) {
	if code := run(os.Stderr, mount, opts...); code != 0 {
		os.Exit(code)
	}
}

func run(errOut io.Writer, mount func(*Server) error, opts ...Option) int {
	s := New(opts...)
	err := mount(s)
	if err == nil {
		err = s.Serve()
	}
	if err == nil {
		return 0
	}
	if _, writeErr := fmt.Fprintf(errOut, "server error: %v\n", err); writeErr != nil {
		return 2
	}
	return 1
}

func newServer(port int, token string, opts []Option, linked bool) *Server {
	s := &Server{
		port:              port,
		token:             token,
		stdout:            os.Stdout,
		stdin:             os.Stdin,
		signals:           []os.Signal{os.Interrupt, syscall.SIGTERM},
		shutdownTimeout:   2 * time.Second,
		readHeaderTimeout: 5 * time.Second,
		mux:               http.NewServeMux(),
	}
	for _, opt := range opts {
		opt(s)
	}

	s.interceptor = NewAuthInterceptor(s.token)
	var lifecycleOpts []LifecycleOption
	if linked {
		lifecycleOpts = append(lifecycleOpts, WithLifecycleShutdown(s.Close))
	}
	s.lifecycle = NewLifecycleServer(make(chan struct{}), lifecycleOpts...)

	s.Register(numenv1connect.NewPluginLifecycleServiceHandler(lifecycleHandler{state: s.lifecycle}, s.HandlerOptions()...))
	if s.onEvent != nil {
		s.Register(numenv1connect.NewPluginEventServiceHandler(eventServer{handle: s.onEvent}, s.HandlerOptions()...))
	}

	return s
}

// OnClose adds what Close runs. A server that serves on a port runs it when Serve returns; a linked
// server runs it when the host shuts the plugin down.
func (s *Server) OnClose(fn func() error) {
	s.closers = append(s.closers, fn)
}

// Close runs what OnClose added, once, last added first. It returns the joined errors of the
// functions; a later call returns the same result without running them again.
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		for _, fn := range slices.Backward(s.closers) {
			s.closeErr = errors.Join(s.closeErr, fn())
		}
	})
	return s.closeErr
}

// Register mounts a service handler on the server multiplexer.
func (s *Server) Register(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// RoutePrefix is where the host forwards the plugin route: a window's GET of /plugin/<id>/<rest>
// that names no file of the plugin folder arrives at /plugin/<rest>.
const RoutePrefix = "/plugin/"

// RegisterRoute mounts handler for GET and HEAD requests at RoutePrefix+pattern, a pattern of
// http.ServeMux such as "pages/{page}". A request without the host's token is answered 401.
func (s *Server) RegisterRoute(pattern string, handler http.Handler) {
	s.mux.Handle(http.MethodGet+" "+RoutePrefix+pattern, s.interceptor.RequireToken(handler))
}

// HandlerOptions returns Connect-RPC options enforcing server authentication.
func (s *Server) HandlerOptions() []connect.HandlerOption {
	return []connect.HandlerOption{connect.WithInterceptors(s.interceptor)}
}

func (s *Server) AuthInterceptor() connect.Interceptor {
	return s.interceptor
}

// Lifecycle returns the lifecycle handler.
func (s *Server) Lifecycle() *LifecycleServer {
	return s.lifecycle
}

// SetConfigure registers the callback invoked during Configure RPC calls.
func (s *Server) SetConfigure(fn func(ctx context.Context, values map[string]string) error) {
	s.lifecycle.SetConfigure(fn)
}

// SetUpdateConfig registers a simplified configuration updater callback.
func (s *Server) SetUpdateConfig(fn func(values map[string]string)) {
	s.SetConfigure(func(_ context.Context, values map[string]string) error {
		fn(values)
		return nil
	})
}

// Listen binds the TCP listener and records the allocated port.
func (s *Server) Listen() (int, error) {
	return s.listen(context.Background())
}

func (s *Server) listen(ctx context.Context) (int, error) {
	if s.listener != nil {
		return s.actualPort, nil
	}
	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
	if err != nil {
		return 0, fmt.Errorf("listen on 127.0.0.1:%d: %w", s.port, err)
	}
	addr, isTCP := ln.Addr().(*net.TCPAddr)
	// gomutants:disable-next-line reason="a listener opened on a TCP address always reports a TCP address, so the other branch cannot be reached"
	if !isTCP {
		return 0, errors.Join(fmt.Errorf("listener address %s is not a TCP address", ln.Addr()), ln.Close())
	}
	s.listener = ln
	s.actualPort = addr.Port
	return s.actualPort, nil
}

// Port returns the bound port, or the configured port when unbound.
func (s *Server) Port() int {
	if s.actualPort != 0 {
		return s.actualPort
	}
	return s.port
}

// Serve binds listener and runs the HTTP/2 server until shutdown is triggered.
func (s *Server) Serve() error {
	return s.ServeContext(context.Background())
}

// ServeContext runs the HTTP/2 server until context cancellation or shutdown is triggered.
func (s *Server) ServeContext(ctx context.Context) (err error) {
	defer func() {
		err = errors.Join(err, s.Close())
	}()

	if err := s.authenticate(); err != nil {
		return err
	}
	if _, err := s.listen(ctx); err != nil {
		return err
	}
	defer func() {
		if closeErr := s.closeListener(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	httpSrv := s.newHTTPServer()

	// gomutants:disable-next-line reason="the capacity of the made value does not change what the function returns"
	serveErrCh := make(chan error, 1)
	go serveHTTP(httpSrv, s.listener, serveErrCh)

	if err := s.announceReady(); err != nil {
		return err
	}

	stdinEOF := s.watchStdin()
	sigCh, stopSignals := s.notifySignals()
	defer stopSignals()

	served, err := waitForStop(ctx, s.lifecycle.ShutdownTrigger(), sigCh, stdinEOF, serveErrCh)
	if served {
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}

func (s *Server) newHTTPServer() *http.Server {
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: s.readHeaderTimeout,
		Protocols:         &protocols,
	}
}

func (s *Server) notifySignals() (<-chan os.Signal, func()) {
	// gomutants:disable-next-line reason="the capacity of the made value does not change what the function returns"
	sigCh := make(chan os.Signal, 1)
	// gomutants:disable-next-line reason="a server given no signals installs none, and the test of the shutdown has to send one to prove the opposite"
	if len(s.signals) == 0 {
		return sigCh, func() {}
	}
	signal.Notify(sigCh, s.signals...)
	return sigCh, func() { signal.Stop(sigCh) }
}

func waitForStop(
	ctx context.Context, shutdown <-chan struct{}, sigCh <-chan os.Signal, stdinEOF <-chan struct{}, serveErrCh <-chan error,
) (served bool, err error) {
	select {
	case <-ctx.Done():
	case <-sigCh:
	case <-shutdown:
	case <-stdinEOF:
	case err := <-serveErrCh:
		if err != nil {
			return true, fmt.Errorf("serve: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func serveHTTP(httpSrv *http.Server, listener net.Listener, errs chan<- error) {
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err := httpSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errs <- err
	}
	close(errs)
}

func (s *Server) closeListener() error {
	if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("close listener: %w", err)
	}
	return nil
}

func (s *Server) authenticate() error {
	if s.token == "" && s.stdin != nil {
		line, err := readTokenLine(s.stdin)
		if err != nil {
			return fmt.Errorf("read auth token: %w", err)
		}
		s.token = line
		s.interceptor.SetToken(s.token)
	}
	if s.token == "" {
		return errors.New("empty auth token")
	}
	if err := os.Setenv(tokenVar, s.token); err != nil {
		return fmt.Errorf("set %s: %w", tokenVar, err)
	}
	return nil
}

func (s *Server) announceReady() error {
	if s.stdout == nil {
		return nil
	}
	if _, err := io.WriteString(s.stdout, s.formatReadyLine()); err != nil {
		return fmt.Errorf("write ready line: %w", err)
	}
	return nil
}

func (s *Server) watchStdin() <-chan struct{} {
	stdinEOF := make(chan struct{})
	if s.stdin == nil {
		return stdinEOF
	}
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := s.stdin.Read(buf)
			if err != nil || n == 0 {
				close(stdinEOF)
				return
			}
		}
	}()
	return stdinEOF
}

func readTokenLine(r io.Reader) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				break
			}
			buf = append(buf, b[0])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
	}
	return strings.TrimSpace(string(buf)), nil
}

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

	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// tokenVar names the variable holding the token the host authenticates with.
const tokenVar = "NUMEN_PLUGIN_TOKEN"

var (
	flagMu          sync.Mutex
	defaultPortFlag int
	exitProcess               = os.Exit
	exitOutput      io.Writer = os.Stderr
)

func parseDefaultFlags() int {
	flagMu.Lock()
	defer flagMu.Unlock()

	var port int

	if flag.Parsed() {
		if f := flag.CommandLine.Lookup("port"); f != nil {
			if p, err := strconv.Atoi(f.Value.String()); err == nil {
				port = p
			}
		}
	} else {
		if f := flag.CommandLine.Lookup("port"); f == nil {
			flag.IntVar(&defaultPortFlag, "port", 0, "Port to listen on")
		}
		flag.Parse()
		port = defaultPortFlag
	}

	if port == 0 {
		if envPort := os.Getenv("NUMEN_PLUGIN_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil {
				port = p
			}
		}
	}

	return port
}

// Option configures Server instances.
type Option func(*Server)

// WithPort sets the TCP listen port.
func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

// WithToken sets the expected authentication token.
func WithToken(token string) Option {
	return func(s *Server) {
		s.token = token
	}
}

// WithConfigure registers a callback for Configure RPC calls.
func WithConfigure(fn func(ctx context.Context, values map[string]string) error) Option {
	return func(s *Server) {
		s.onConfigure = fn
	}
}

// WithUpdateConfig registers a simplified callback for Configure RPC calls.
func WithUpdateConfig(fn func(values map[string]string)) Option {
	return func(s *Server) {
		if fn == nil {
			s.onConfigure = nil
			return
		}
		s.onConfigure = func(_ context.Context, values map[string]string) error {
			fn(values)
			return nil
		}
	}
}

// WithStdout sets the handshake output destination.
func WithStdout(w io.Writer) Option {
	return func(s *Server) {
		s.stdout = w
	}
}

// WithStdin sets the input reader monitored for process termination.
func WithStdin(r io.Reader) Option {
	return func(s *Server) {
		s.stdin = r
	}
}

// WithSignals sets the operating system signals that trigger graceful shutdown.
func WithSignals(signals ...os.Signal) Option {
	return func(s *Server) {
		s.signals = signals
	}
}

// WithShutdownTimeout sets the maximum duration permitted for active connections to drain.
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = d
	}
}

// WithReadHeaderTimeout sets the HTTP header read timeout.
func WithReadHeaderTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.readHeaderTimeout = d
	}
}

// Server coordinates the lifecycle, authentication, and HTTP/2 transport of a plugin.
type Server struct {
	port              int
	token             string
	stdout            io.Writer
	stdin             io.Reader
	signals           []os.Signal
	shutdownTimeout   time.Duration
	readHeaderTimeout time.Duration
	onConfigure       func(ctx context.Context, values map[string]string) error
	onEvent           EventHandler

	closeOnce sync.Once
	closers   []func()

	lifecycle   *LifecycleServer
	interceptor *AuthInterceptor
	mux         *http.ServeMux
	listener    net.Listener
	actualPort  int
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
		s.Close()
		return nil, err
	}
	return s.mux, nil
}

// Run starts a server, mounts services, and serves until shutdown.
// When mounting or serving fails, Run writes the error to standard error and exits with code 1.
func Run(mount func(*Server) error, opts ...Option) {
	s := New(opts...)
	if err := mount(s); err != nil {
		fmt.Fprintf(exitOutput, "server error: %v\n", err)
		exitProcess(1)
		return
	}
	if err := s.Serve(); err != nil {
		fmt.Fprintf(exitOutput, "server error: %v\n", err)
		exitProcess(1)
		return
	}
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
	lifecycleOpts := []LifecycleOption{WithLifecycleConfigure(s.onConfigure)}
	if linked {
		lifecycleOpts = append(lifecycleOpts, WithLifecycleShutdown(s.Close))
	}
	s.lifecycle = NewLifecycleServer(make(chan struct{}), lifecycleOpts...)

	s.Register(numenv1connect.NewPluginLifecycleServiceHandler(s.lifecycle, s.HandlerOptions()...))
	if s.onEvent != nil {
		s.Register(numenv1connect.NewPluginEventServiceHandler(eventServer{handle: s.onEvent}, s.HandlerOptions()...))
	}

	return s
}

// OnClose adds what Close runs. A server that serves on a port runs it when Serve returns; a linked
// server runs it when the host shuts the plugin down.
func (s *Server) OnClose(fn func()) {
	s.closers = append(s.closers, fn)
}

// Close runs what OnClose added, once, last added first.
func (s *Server) Close() {
	s.closeOnce.Do(func() {
		for _, fn := range slices.Backward(s.closers) {
			fn()
		}
	})
}

// Register mounts a service handler on the server multiplexer.
func (s *Server) Register(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// RoutePrefix is where the host forwards the plugin route: a window's GET of /plugin/<id>/<rest>
// arrives at /plugin/<rest>.
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

// AuthInterceptor returns the authentication interceptor.
func (s *Server) AuthInterceptor() connect.Interceptor {
	return s.interceptor
}

// Lifecycle returns the lifecycle handler.
func (s *Server) Lifecycle() *LifecycleServer {
	return s.lifecycle
}

// SetConfigure registers the callback invoked during Configure RPC calls.
func (s *Server) SetConfigure(fn func(ctx context.Context, values map[string]string) error) {
	s.onConfigure = fn
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
	if s.listener != nil {
		return s.actualPort, nil
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
	if err != nil {
		return 0, fmt.Errorf("listen on 127.0.0.1:%d: %w", s.port, err)
	}
	s.listener = ln
	s.actualPort = ln.Addr().(*net.TCPAddr).Port
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
func (s *Server) ServeContext(ctx context.Context) error {
	defer s.Close()

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
	_ = os.Setenv(tokenVar, s.token)

	if _, err := s.Listen(); err != nil {
		return err
	}
	defer s.listener.Close()

	// plain TCP on loopback.
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	httpSrv := &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: s.readHeaderTimeout,
		Protocols:         &protocols,
	}

	serveErrCh := make(chan error, 1)
	go func() {
		if err := httpSrv.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErrCh <- err
		}
		close(serveErrCh)
	}()

	if s.stdout != nil {
		fmt.Fprintf(s.stdout, "NUMEN_READY port=%d\n", s.actualPort)
		if syncer, ok := s.stdout.(interface{ Sync() error }); ok {
			_ = syncer.Sync()
		}
	}

	stdinEOF := make(chan struct{})
	if s.stdin != nil {
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
	}

	sigCh := make(chan os.Signal, 1)
	if len(s.signals) > 0 {
		signal.Notify(sigCh, s.signals...)
		defer signal.Stop(sigCh)
	}

	select {
	case <-ctx.Done():
	case <-sigCh:
	case <-s.lifecycle.ShutdownTrigger():
	case <-stdinEOF:
	case err := <-serveErrCh:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
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

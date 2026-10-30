package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server coordinates the MCP server, HTTP transport, and authentication.
type Server struct {
	config     Config
	mcpServer  *sdk.Server
	httpServer *http.Server
	handler    http.Handler
	aggregator *ToolAggregator
	listener   net.Listener
	actualAddr string
	isClosed   bool
	mu         sync.Mutex
	// serveErr holds the errors of the HTTP servers other than their shutdown. Close returns it.
	serveErr error
}

// NewServer creates an MCP server with the provided configuration and host tool client.
func NewServer(cfg Config, client ToolClient) (*Server, error) {
	if cfg.Token == "" {
		token, err := createRandomToken()
		if err != nil {
			return nil, fmt.Errorf("create auth token: %w", err)
		}
		cfg.Token = token
	}

	mcpServer := sdk.NewServer(
		&sdk.Implementation{
			Name:        "numen",
			Title:       "Numen MCP Server",
			Description: "Exposes Numén tools to external MCP clients",
			Version:     "1.0.0",
			WebsiteURL:  "",
			Icons:       nil,
		},
		newServerOptions(),
	)

	aggregator := NewToolAggregator(client, mcpServer)

	mcpServer.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if method == "tools/list" {
				reportSyncError(aggregator.SyncTools(ctx))
			}
			return next(ctx, method, req)
		}
	})

	reportSyncError(aggregator.SyncTools(context.Background()))

	srv := &Server{
		config:     cfg,
		mcpServer:  mcpServer,
		httpServer: nil,
		handler:    nil,
		aggregator: aggregator,
		listener:   nil,
		actualAddr: "",
		isClosed:   false,
		mu:         sync.Mutex{},
	}
	authHandler := srv.createHTTPHandler(mcpServer)
	srv.handler = authHandler
	srv.httpServer = newHTTPServer(authHandler)

	return srv, nil
}

// newServerOptions creates the options of the MCP server.
func newServerOptions() *sdk.ServerOptions {
	opts := new(sdk.ServerOptions)
	opts.Instructions = "Numén notes and plugins tools server."
	return opts
}

// newHTTPServer creates an HTTP server that serves handler.
func newHTTPServer(handler http.Handler) *http.Server {
	httpSrv := new(http.Server)
	httpSrv.Handler = handler
	httpSrv.ReadHeaderTimeout = 10 * time.Second
	return httpSrv
}

// reportSyncError writes a failure to sync the host tools to the standard error stream of the
// plugin process. The tools are synced again on the next listing and on the next configuration.
func reportSyncError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync the host tools: %v\n", err)
	}
}

// createHTTPHandler sets up HTTP routes and token authentication.
func (s *Server) createHTTPHandler(mcpServer *sdk.Server) http.Handler {
	mux := http.NewServeMux()

	sseHandler := sdk.NewSSEHandler(
		func(*http.Request) *sdk.Server { return mcpServer },
		&sdk.SSEOptions{
			DisableLocalhostProtection: true,
			MaxRequestBodyBytes:        0,
		},
	)

	streamHandler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return mcpServer },
		newStreamableOptions(),
	)

	mux.Handle("/sse", sseHandler)
	mux.Handle("/sse/", sseHandler)
	mux.Handle("/messages", sseHandler)
	mux.Handle("/messages/", sseHandler)
	mux.Handle("/mcp", streamHandler)
	mux.Handle("/mcp/", streamHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "" {
			http.NotFound(w, r)
			return
		}
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") || r.URL.Query().Get("sessionid") != "" {
			sseHandler.ServeHTTP(w, r)
			return
		}
		streamHandler.ServeHTTP(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireToken(s.GetToken(), mux).ServeHTTP(w, r)
	})
}

// newStreamableOptions creates the options of the streamable HTTP handler.
func newStreamableOptions() *sdk.StreamableHTTPOptions {
	opts := new(sdk.StreamableHTTPOptions)
	opts.SessionTimeout = 30 * time.Minute
	return opts
}

// Rebind updates the server address and token, restarting the listener when the address changes.
func (s *Server) Rebind(ctx context.Context, cfg Config) error {
	s.mu.Lock()
	s.config.Token = cfg.Token
	if cfg.Addr == "" || cfg.Addr == s.config.Addr || s.isClosed {
		s.mu.Unlock()
		return nil
	}
	oldListener := s.listener
	s.config.Addr = cfg.Addr
	s.mu.Unlock()

	if oldListener != nil {
		if err := oldListener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("close the listener on %s: %w", oldListener.Addr(), err)
		}
	}
	ln, err := new(net.ListenConfig).Listen(context.WithoutCancel(ctx), "tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Addr, err)
	}
	s.mu.Lock()
	if s.isClosed {
		s.mu.Unlock()
		return ln.Close()
	}
	s.listener = ln
	s.actualAddr = ln.Addr().String()
	s.httpServer = newHTTPServer(s.handler)
	httpSrv := s.httpServer
	s.mu.Unlock()

	go s.serveHTTP(httpSrv, ln)
	return nil
}

// serveHTTP serves srv on ln until it is shut down or ln is closed, and keeps any other error for
// Close to return.
func (s *Server) serveHTTP(srv *http.Server, ln net.Listener) {
	err := srv.Serve(ln)
	if err == nil || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return
	}
	s.mu.Lock()
	s.serveErr = errors.Join(s.serveErr, err)
	s.mu.Unlock()
}

// Start opens the network listener and begins serving requests.
func (s *Server) Start() error {
	addr := s.config.Addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	return s.Serve(ln)
}

// Serve begins serving on the provided listener.
func (s *Server) Serve(ln net.Listener) error {
	s.mu.Lock()
	if s.isClosed {
		s.mu.Unlock()
		return errors.New("server is closed")
	}
	s.listener = ln
	s.actualAddr = ln.Addr().String()
	s.mu.Unlock()

	go s.serveHTTP(s.httpServer, ln)

	return nil
}

// Close gracefully stops the server.
func (s *Server) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.isClosed {
		s.mu.Unlock()
		return nil
	}
	s.isClosed = true
	s.mu.Unlock()

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}

	shutdownErr := s.httpServer.Shutdown(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	return errors.Join(shutdownErr, s.serveErr)
}

// Handler returns the HTTP handler of the server.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// GetURL returns the base server URL.
func (s *Server) GetURL() string {
	return s.GetSSEURL()
}

// GetSSEURL returns the SSE endpoint URL.
func (s *Server) GetSSEURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.actualAddr == "" {
		return ""
	}
	return "http://" + s.actualAddr + "/sse"
}

// GetHTTPURL returns the streamable HTTP endpoint URL.
func (s *Server) GetHTTPURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.actualAddr == "" {
		return ""
	}
	return "http://" + s.actualAddr + "/mcp"
}

// GetToken returns the authentication token.
func (s *Server) GetToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config.Token
}

// GetAggregator returns the tool aggregator.
func (s *Server) GetAggregator() *ToolAggregator {
	return s.aggregator
}

// GetMCPServer returns the underlying MCP SDK server.
func (s *Server) GetMCPServer() *sdk.Server {
	return s.mcpServer
}

// RunStdio runs the server over standard input and output streams.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &sdk.StdioTransport{MaxLineLength: 0})
}

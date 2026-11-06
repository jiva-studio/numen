package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
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
		// gomutants:disable-next-line BRANCH_IF reason="createRandomToken fails only when crypto/rand.Read fails"
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

	mcpServer.AddReceivingMiddleware(newSyncMiddleware(aggregator))

	reportSyncError(aggregator.SyncTools(context.Background()))

	srv := &Server{
		config:     cfg,
		mcpServer:  mcpServer,
		httpServer: nil,
		handler:    nil,
		aggregator: aggregator,
		actualAddr: "",
		isClosed:   false,
		mu:         sync.Mutex{},
		serveErr:   nil,
	}
	authHandler := srv.createHTTPHandler(mcpServer)
	srv.handler = authHandler
	srv.httpServer = newHTTPServer(authHandler)

	return srv, nil
}

// newSyncMiddleware syncs the host tools before a listing of the tools and before the result that
// carries the instructions, and writes the instructions of the latest sync into that result. The
// SDK answers initialize, and server/discover of the stateless protocol, with the fixed
// instructions of the server options.
func newSyncMiddleware(aggregator *ToolAggregator) sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			switch method {
			case "tools/list":
				reportSyncError(aggregator.SyncTools(ctx))
			case "initialize", "server/discover":
				reportSyncError(aggregator.SyncTools(ctx))
				result, err := next(ctx, method, req)
				if err == nil {
					writeInstructions(result, joinInstructions(aggregator.GetInstructions()))
				}
				return result, err
			}
			return next(ctx, method, req)
		}
	}
}

// writeInstructions sets the instructions of an initialize or a discover result. The SDK returns a
// typed nil result with an error, as for a second initialize on a session.
func writeInstructions(result sdk.Result, text string) {
	switch res := result.(type) {
	case *sdk.InitializeResult:
		if res != nil {
			res.Instructions = text
		}
	case *sdk.DiscoverResult:
		if res != nil {
			res.Instructions = text
		}
	}
}

// newServerOptions creates the options of the MCP server.
func newServerOptions() *sdk.ServerOptions {
	opts := new(sdk.ServerOptions)
	opts.Instructions = instructions
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
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="stderr diagnostic output does not affect program state"
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync the host tools: %v\n", err)
	}
}

// createHTTPHandler sets up HTTP routes and token authentication.
func (s *Server) createHTTPHandler(mcpServer *sdk.Server) http.Handler {
	mux := http.NewServeMux()

	streamHandler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return mcpServer },
		newStreamableOptions(),
	)

	mux.Handle("/mcp", streamHandler)
	mux.Handle("/mcp/", streamHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "" {
			http.NotFound(w, r)
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

// SetToken replaces the token a client authenticates with.
func (s *Server) SetToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Token = token
}

// serveHTTP serves srv on ln until it is shut down or ln is closed, and keeps any other error for
// Close to return.
func (s *Server) serveHTTP(srv *http.Server, ln net.Listener) {
	err := srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return
	}
	s.mu.Lock()
	s.serveErr = errors.Join(s.serveErr, err)
	s.mu.Unlock()
}

// Serve begins serving on the provided listener.
func (s *Server) Serve(ln net.Listener) error {
	s.mu.Lock()
	if s.isClosed {
		s.mu.Unlock()
		return errors.New("server is closed")
	}
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

	// gomutants:disable-next-line REMOVE_LOGICAL_NOT,BRANCH_IF reason="fallback timeout applies only when active connections stall during shutdown"
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="shutdown grace period duration is a timeout policy"
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
	return s.GetHTTPURL()
}

// GetHTTPURL returns the streamable HTTP endpoint URL.
func (s *Server) GetHTTPURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.actualAddr == "" {
		return ""
	}
	return (&url.URL{Scheme: "http", Host: s.actualAddr, Path: "/mcp"}).String()
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

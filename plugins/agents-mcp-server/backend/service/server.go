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
	// newConns holds the accepted connections that have not sent a request yet. http.Server.Shutdown
	// waits five seconds for such a connection, so its shutdown hook closes them.
	newConns map[net.Conn]struct{}
	// isShuttingDown is set by the shutdown hook of the HTTP server, after which trackConn closes a
	// new connection at once.
	isShuttingDown bool
}

// NewServer creates an MCP server with the provided configuration and host tool client.
func NewServer(cfg Config, client ToolClient) (*Server, error) {
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
		config:         cfg,
		mcpServer:      mcpServer,
		httpServer:     nil,
		handler:        nil,
		aggregator:     aggregator,
		actualAddr:     "",
		isClosed:       false,
		mu:             sync.Mutex{},
		serveErr:       nil,
		newConns:       make(map[net.Conn]struct{}),
		isShuttingDown: false,
	}
	srv.handler = srv.createHTTPHandler(mcpServer)
	srv.httpServer = srv.newHTTPServer()

	return srv, nil
}

// newSyncMiddleware syncs the host tools before a listing of the tools and before the initialize
// result, and writes the instructions of the latest sync into that result. The SDK returns the
// fixed instructions of the server options for initialize, and for server/discover of the
// stateless protocol.
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

// newHTTPServer creates an HTTP server that serves the handler of s and closes the connections
// without a request on shutdown.
func (s *Server) newHTTPServer() *http.Server {
	httpSrv := new(http.Server)
	httpSrv.Handler = s.handler
	httpSrv.ReadHeaderTimeout = 10 * time.Second
	httpSrv.ConnState = s.trackConn
	httpSrv.RegisterOnShutdown(s.closeNewConns)
	return httpSrv
}

// trackConn records whether conn has sent a request yet and closes a connection accepted after the
// HTTP server has started its shutdown.
func (s *Server) trackConn(conn net.Conn, state http.ConnState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state != http.StateNew {
		delete(s.newConns, conn)
		return
	}
	if s.isShuttingDown {
		closeConn(conn)
		return
	}
	s.newConns[conn] = struct{}{}
}

// closeNewConns closes the connections that have not sent a request. http.Server.Shutdown runs it
// after it has marked itself as shutting down, and from then on net/http drops a request it reads
// on such a connection.
func (s *Server) closeNewConns() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isShuttingDown = true
	for conn := range s.newConns {
		closeConn(conn)
		delete(s.newConns, conn)
	}
}

// closeConn closes a connection that carries no request and writes a failure to the standard error
// stream of the plugin process.
func closeConn(conn net.Conn) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="stderr diagnostic output does not affect program state"
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		fmt.Fprintf(os.Stderr, "close a connection without a request: %v\n", err)
	}
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

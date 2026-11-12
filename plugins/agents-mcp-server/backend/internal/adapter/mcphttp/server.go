// Package mcphttp serves an MCP server over streamable HTTP behind a bearer token.
package mcphttp

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

	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/access"
)

// Server is the HTTP transport of an MCP server.
type Server struct {
	token          string
	httpServer     *http.Server
	handler        http.Handler
	actualAddr     string
	isClosed       bool
	mu             sync.Mutex
	serveErr       error
	newConns       map[net.Conn]struct{}
	isShuttingDown bool
}

// New creates the HTTP transport of the MCP server. Requests need the token.
func New(mcpServer *sdk.Server, token string) *Server {
	srv := &Server{
		token:          token,
		httpServer:     nil,
		handler:        nil,
		actualAddr:     "",
		isClosed:       false,
		mu:             sync.Mutex{},
		serveErr:       nil,
		newConns:       make(map[net.Conn]struct{}),
		isShuttingDown: false,
	}
	srv.handler = srv.createHTTPHandler(mcpServer)
	srv.httpServer = srv.newHTTPServer()
	return srv
}

func (s *Server) newHTTPServer() *http.Server {
	httpSrv := new(http.Server)
	httpSrv.Handler = s.handler
	httpSrv.ReadHeaderTimeout = 10 * time.Second
	httpSrv.ConnState = s.trackConn
	httpSrv.RegisterOnShutdown(s.closeNewConns)
	return httpSrv
}

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

func (s *Server) closeNewConns() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isShuttingDown = true
	for conn := range s.newConns {
		closeConn(conn)
		delete(s.newConns, conn)
	}
}

func closeConn(conn net.Conn) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="stderr diagnostic output does not affect program state"
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		fmt.Fprintf(os.Stderr, "close a connection without a request: %v\n", err)
	}
}

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
		access.RequireToken(s.GetToken(), mux).ServeHTTP(w, r)
	})
}

func newStreamableOptions() *sdk.StreamableHTTPOptions {
	opts := new(sdk.StreamableHTTPOptions)
	opts.SessionTimeout = 30 * time.Minute
	return opts
}

// SetToken replaces the token a client authenticates with.
func (s *Server) SetToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
}

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
	if errors.Is(shutdownErr, net.ErrClosed) {
		shutdownErr = nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return errors.Join(shutdownErr, s.serveErr)
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

// GetHTTPURL returns the streamable HTTP endpoint URL, or an empty string before Serve.
func (s *Server) GetHTTPURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.actualAddr == "" {
		return ""
	}
	return (&url.URL{Scheme: "http", Host: s.actualAddr, Path: "/mcp"}).String()
}

func (s *Server) GetToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

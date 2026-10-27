package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
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
		},
		&sdk.ServerOptions{
			Instructions: "Numén notes and plugins tools server.",
		},
	)

	aggregator := NewToolAggregator(client, mcpServer)

	mcpServer.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if method == "tools/list" {
				_ = aggregator.SyncTools(ctx)
			}
			return next(ctx, method, req)
		}
	})

	_ = aggregator.SyncTools(context.Background())

	authHandler := createHTTPHandler(mcpServer, cfg.Token)

	srv := &Server{
		config:     cfg,
		mcpServer:  mcpServer,
		handler:    authHandler,
		aggregator: aggregator,
		httpServer: &http.Server{
			Handler:           authHandler,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	return srv, nil
}

// createHTTPHandler sets up HTTP routes and token authentication.
func createHTTPHandler(mcpServer *sdk.Server, token string) http.Handler {
	mux := http.NewServeMux()

	sseHandler := sdk.NewSSEHandler(
		func(*http.Request) *sdk.Server { return mcpServer },
		&sdk.SSEOptions{
			DisableLocalhostProtection: true,
		},
	)

	streamHandler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return mcpServer },
		&sdk.StreamableHTTPOptions{
			SessionTimeout: 30 * time.Minute,
		},
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

	return requireToken(token, mux)
}

// Start opens the network listener and begins serving requests.
func (s *Server) Start() error {
	addr := s.config.Addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}

	ln, err := net.Listen("tcp", addr)
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

	go func() {
		_ = s.httpServer.Serve(ln)
	}()

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

	return s.httpServer.Shutdown(ctx)
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
	return s.mcpServer.Run(ctx, &sdk.StdioTransport{})
}

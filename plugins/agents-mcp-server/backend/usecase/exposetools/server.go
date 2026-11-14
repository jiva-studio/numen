package exposetools

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/instructions"
)

// Server is the MCP server with the host tools registered on it.
type Server struct {
	MCP        *sdk.Server
	Aggregator *Aggregator
	report     func(error)
}

// NewServer creates an MCP server for the host tools and loads them once. A failed sync goes to report.
func NewServer(client Host, report func(error)) *Server {
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
	s := &Server{MCP: mcpServer, Aggregator: NewAggregator(client, mcpServer), report: report}
	mcpServer.AddReceivingMiddleware(s.newSyncMiddleware())
	s.Sync(context.Background())
	return s
}

// Sync loads the host tools and reports a failure.
func (s *Server) Sync(ctx context.Context) {
	if err := s.Aggregator.SyncTools(ctx); err != nil {
		s.report(err)
	}
}

func (s *Server) newSyncMiddleware() sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			switch method {
			case "tools/list":
				s.Sync(ctx)
			case "initialize", "server/discover":
				s.Sync(ctx)
				result, err := next(ctx, method, req)
				if err == nil {
					writeInstructions(result, instructions.Join(s.Aggregator.GetInstructions()))
				}
				return result, err
			}
			return next(ctx, method, req)
		}
	}
}

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

func newServerOptions() *sdk.ServerOptions {
	opts := new(sdk.ServerOptions)
	opts.Instructions = instructions.Server
	return opts
}

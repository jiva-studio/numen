package service

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/instructions"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/internal/adapter/mcphttp"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/usecase/exposetools"
)

// Server is the MCP server with its HTTP transport and the host tools registered on it.
type Server struct {
	*mcphttp.Server
	mcpServer  *sdk.Server
	aggregator *exposetools.Aggregator
}

// NewServer creates an MCP server with the provided configuration and host tool client.
func NewServer(cfg Config, client exposetools.Host) (*Server, error) {
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

	aggregator := exposetools.NewAggregator(client, mcpServer)

	mcpServer.AddReceivingMiddleware(newSyncMiddleware(aggregator))

	reportSyncError(aggregator.SyncTools(context.Background()))

	return &Server{
		Server:     mcphttp.New(mcpServer, cfg.Token),
		mcpServer:  mcpServer,
		aggregator: aggregator,
	}, nil
}

func newSyncMiddleware(aggregator *exposetools.Aggregator) sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			switch method {
			case "tools/list":
				reportSyncError(aggregator.SyncTools(ctx))
			case "initialize", "server/discover":
				reportSyncError(aggregator.SyncTools(ctx))
				result, err := next(ctx, method, req)
				if err == nil {
					writeInstructions(result, instructions.Join(aggregator.GetInstructions()))
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

func reportSyncError(err error) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="stderr diagnostic output does not affect program state"
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync the host tools: %v\n", err)
	}
}

// GetAggregator returns the tool aggregator.
func (s *Server) GetAggregator() *exposetools.Aggregator {
	return s.aggregator
}

// GetMCPServer returns the underlying MCP SDK server.
func (s *Server) GetMCPServer() *sdk.Server {
	return s.mcpServer
}

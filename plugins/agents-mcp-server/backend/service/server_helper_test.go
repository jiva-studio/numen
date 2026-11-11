package service

import (
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/hostserver"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/usecase/exposetools"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config = hostserver.Settings

type testServer struct{ *endpoint }

func NewServer(cfg Config, client exposetools.Host) (*testServer, error) {
	return &testServer{endpoint: newEndpoint(cfg.Token, client)}, nil
}

func (s *testServer) GetAggregator() *exposetools.Aggregator { return s.host.Aggregator }

func (s *testServer) GetMCPServer() *sdk.Server { return s.host.MCP }

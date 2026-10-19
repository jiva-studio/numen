package service

import (
	"context"
	"sync"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	numenv1connect.UnimplementedCompletionServiceHandler
	// acpAgents returns the agents completions are answered by. Nil is those the settings name.
	acpAgents func() []acpAgent
	acpModels acpModelCache

	mu sync.RWMutex
	// commands are the command lines the settings gave at the last program listing, by program.
	commands map[string][]string
}

// GetAgentPrograms lists no program and keeps the command lines of the settings, which name the
// agents completions are answered by.
func (s *agentServer) GetAgentPrograms(
	_ context.Context,
	req *connect.Request[numenv1.GetAgentProgramsRequest],
) (*connect.Response[numenv1.GetAgentProgramsResponse], error) {
	commands := make(map[string][]string, len(req.Msg.GetCommands()))
	for _, one := range req.Msg.GetCommands() {
		commands[one.GetProgram()] = one.GetCommand()
	}
	s.mu.Lock()
	s.commands = commands
	s.mu.Unlock()
	return connect.NewResponse(&numenv1.GetAgentProgramsResponse{}), nil
}

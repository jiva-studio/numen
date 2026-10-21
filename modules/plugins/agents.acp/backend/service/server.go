package service

import (
	"sync"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	// acpAgents returns the agents the programs are listed for. Nil is those the settings name.
	acpAgents func() []acpAgent
	acpModels acpModelCache

	mu sync.RWMutex
	// commands are the command lines the settings gave at the last program listing, by program.
	commands map[string][]string
}

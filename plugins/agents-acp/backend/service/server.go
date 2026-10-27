package service

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"strings"
	"sync"
)

type agentServer struct {
	agentrun.UnimplementedAgentServiceHandler
	// acpAgents returns the agents the programs are listed for. Nil is those the settings name.
	acpAgents func() []acpAgent
	acpModels acpModelCache

	mu             sync.RWMutex
	executablePath string
	// commands are the command lines the settings gave at the last program listing, by program.
	commands map[string][]string
}

func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executable_path"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
}

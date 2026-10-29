package service

import (
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
)

type agentServer struct {
	agentrun.UnimplementedAgentServiceHandler
	// acpAgents returns the agents the programs are listed for. Nil is those the settings name.
	acpAgents func() []acpAgent
	// discover returns the ACP agents discovered on the machine.
	discover  func() []acpAgent
	acpModels *cache.Cache[acpListing]

	mu              sync.RWMutex
	executablePaths []string
	// commands are the command lines the settings gave at the last program listing, by program.
	commands map[string][]string
}

func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executable_path"]; ok {
		var paths []string
		for one := range strings.SplitSeq(execPath, "\n") {
			entry := strings.TrimSpace(one)
			if entry != "" {
				paths = append(paths, entry)
			}
		}
		s.executablePaths = paths
	}
}

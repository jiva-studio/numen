package service

import (
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
)

type agentServer struct {
	agentrun.Unimplemented
	acpAgents func() []acpAgent
	lookPath  func(string) (string, error)
	found     []acpAgent
	acpModels *cache.Cache[acpListing]

	mu              sync.RWMutex
	executablePaths []string
	commands        map[string][]string
}

func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executablePath"]; ok {
		var paths []string
		for one := range strings.SplitSeq(execPath, "\n") {
			entry := strings.TrimSpace(one)
			if entry != "" {
				paths = append(paths, entry)
			}
		}
		s.executablePaths = paths
	}
	s.found = findACPAgents(s.lookPath)
}

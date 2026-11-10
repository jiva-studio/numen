package service

import (
	"context"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/catalog"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/antigravity"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/runs"
)

type agentServer struct {
	agentrun.Service
	mu             sync.RWMutex
	executablePath string
	catalog        *catalog.Catalog
	runner         *runs.Runner
}

func newAgentServer(isAsync bool) *agentServer {
	supervisor := process.Supervisor{}
	cli := antigravity.New(supervisor)
	s := new(agentServer)
	s.catalog = catalog.New(cli, cli, isAsync)
	s.runner = runs.New(&s.Registry, cli, supervisor, s.catalog)
	return s
}

func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executablePath"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
}

func (s *agentServer) getExecutablePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executablePath
}

func (s *agentServer) Ask(ctx context.Context, req *agentrun.Request, send func(*agentrun.Response) error) error {
	return s.runner.Ask(ctx, req, s.getExecutablePath(), send)
}

func (s *agentServer) GetState(_ context.Context) (*agentrun.State, error) {
	return s.catalog.GetState(s.getExecutablePath()), nil
}

func (s *agentServer) GetPrograms(ctx context.Context, commands []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	return s.catalog.GetPrograms(ctx, s.getExecutablePath(), commands), nil
}

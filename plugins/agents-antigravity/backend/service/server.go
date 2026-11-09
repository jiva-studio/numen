package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
)

// defaultModelName is the model name that selects the program's own default model.
const defaultModelName = "default"

type agentServer struct {
	agentrun.Service
	mu             sync.RWMutex
	executablePath string
	requestTimeout time.Duration
	models         *cache.Cache[[]antigravityModel]
}

func newModelCache(isAsync bool) *cache.Cache[[]antigravityModel] {
	return cache.New[[]antigravityModel](
		"models",
		cache.WithTimeout(modelListLimit),
		cache.WithFailureBackoff(failureBackoff),
		cache.WithAsync(isAsync),
	)
}

func (s *agentServer) getModelCache() *cache.Cache[[]antigravityModel] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.models == nil {
		s.models = newModelCache(false)
	}
	return s.models
}

func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executablePath"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
	if timeout, ok := values[settingRequestTimeout]; ok {
		s.requestTimeout = parseRequestTimeout(timeout)
	}
}

func (s *agentServer) getExecutablePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executablePath
}

func (s *agentServer) Ask(ctx context.Context, req *agentrun.Request, send func(*agentrun.Response) error) error {
	if err := send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return err
	}

	s.resolveModel(ctx, req)
	work, err := start(ctx, &s.Registry, newSpec(req, &s.Registry, s.getExecutablePath()))
	if err != nil {
		return send(&agentrun.Response{
			Step: &agentrun.StoppedStep{Stopped: err.Error()},
		})
	}
	return agentrun.ForwardResponses(ctx, work, send)
}

func (s *agentServer) GetState(_ context.Context) (*agentrun.State, error) {
	if !IsAntigravityInstalled(s.getExecutablePath()) {
		return &agentrun.State{Unreachable: "Antigravity was not found on this machine"}, nil
	}
	return &agentrun.State{Unreachable: ""}, nil
}

// GetPrograms lists Antigravity, whether it is installed, and the models it lists.
func (s *agentServer) GetPrograms(ctx context.Context, commands []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	command := programCommand(commands, "antigravity")
	customPath := s.getExecutablePath()
	if len(command) == 0 && customPath != "" {
		command = []string{customPath}
	}
	antigravity := &agentrun.Program{
		Name: "antigravity", Title: "Antigravity", IsInstalled: IsAntigravityInstalled(customPath) || len(command) > 0,
		IsDefault: false, Models: nil, LimitsTools: false,
	}
	if antigravity.IsInstalled {
		for _, one := range s.getFamilies(ctx, command) {
			antigravity.Models = append(antigravity.Models, one.GetModel())
		}
	}
	return []*agentrun.Program{antigravity}, nil
}

func (s *agentServer) getFamilies(ctx context.Context, command []string) []reasoning.Family {
	list := func(ctx context.Context) ([]antigravityModel, error) { return listAntigravityModels(ctx, command) }
	items, _, err := s.getModelCache().Get(ctx, strings.Join(command, " "), list)
	// gomutants:disable-next-line BRANCH_IF reason="A failed load returns the cached items folded identically to a succeeded load."
	if err != nil {
		// A failed load leaves the last list of the cache, or none, and the program is offered with it.
		return foldModels(items)
	}
	return foldModels(items)
}

func foldModels(items []antigravityModel) []reasoning.Family {
	models := make([]reasoning.Listed, 0, len(items))
	for _, one := range items {
		models = append(models, reasoning.Listed{Name: one.Name, Title: one.Title})
	}
	return reasoning.Fold(models)
}

func (s *agentServer) resolveModel(ctx context.Context, req *agentrun.Request) {
	if req.Model == "" || req.Model == defaultModelName {
		return
	}
	req.Model = reasoning.ResolveModel(s.getFamilies(ctx, req.Command), req.Model, req.Reasoning)
}

package service

import (
	"context"
	"strings"
	"sync"

	"connectrpc.com/connect"

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
	models         *cache.Cache[[]antigravityModel]
}

func newModelCache(isAsync bool) *cache.Cache[[]antigravityModel] {
	return cache.New[[]antigravityModel](
		"models",
		cache.WithPluginID("agents-antigravity"),
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
	if execPath, ok := values["executable_path"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
}

func (s *agentServer) getExecutablePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executablePath
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[agentrun.AskAgentRequest],
	stream *connect.ServerStream[agentrun.AskAgentResponse],
) error {
	if err := stream.Send(&agentrun.AskAgentResponse{
		Step: &agentrun.AskAgentResponse_ModelStarted{
			ModelStarted: &agentrun.ModelStarted{},
		},
	}); err != nil {
		return err
	}

	s.resolveModel(ctx, req.Msg)
	work, err := start(ctx, &s.Registry, newSpec(req.Msg, &s.Registry, s.getExecutablePath()))
	if err != nil {
		return stream.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponse_Stopped{Stopped: err.Error()},
		})
	}
	return agentrun.ForwardResponses(ctx, work, stream)
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[agentrun.GetAgentStateRequest],
) (*connect.Response[agentrun.GetAgentStateResponse], error) {
	if !IsAntigravityInstalled(s.getExecutablePath()) {
		return connect.NewResponse(&agentrun.GetAgentStateResponse{
			Unreachable: "Antigravity was not found on this machine",
		}), nil
	}
	return connect.NewResponse(&agentrun.GetAgentStateResponse{Unreachable: ""}), nil
}

// GetAgentPrograms lists Antigravity, whether it is installed, and the models it lists.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[agentrun.GetAgentProgramsRequest],
) (*connect.Response[agentrun.GetAgentProgramsResponse], error) {
	command := programCommand(req.Msg.GetCommands(), "antigravity")
	customPath := s.getExecutablePath()
	if len(command) == 0 && customPath != "" {
		command = []string{customPath}
	}
	antigravity := &agentrun.AgentProgram{
		Name: "antigravity", Title: "Antigravity", IsInstalled: IsAntigravityInstalled(customPath) || len(command) > 0,
		IsDefault: false, Models: nil, LimitsTools: false,
	}
	if antigravity.GetIsInstalled() {
		for _, one := range s.getFamilies(ctx, command) {
			antigravity.Models = append(antigravity.Models, one.ToProto())
		}
	}
	return connect.NewResponse(&agentrun.GetAgentProgramsResponse{
		Programs: []*agentrun.AgentProgram{antigravity},
	}), nil
}

// getFamilies are the models the program lists, the ones that differ by level folded into one.
func (s *agentServer) getFamilies(ctx context.Context, command []string) []reasoning.Family {
	list := func(ctx context.Context) ([]antigravityModel, error) { return listAntigravityModels(ctx, command) }
	items, _, err := s.getModelCache().Get(ctx, strings.Join(command, " "), list)
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

// resolveModel sets the model of the request to the one the program lists for the model and the level
// the request names.
func (s *agentServer) resolveModel(ctx context.Context, req *agentrun.AskAgentRequest) {
	if req.GetModel() == "" || req.GetModel() == defaultModelName {
		return
	}
	req.Model = reasoning.ResolveModel(s.getFamilies(ctx, req.GetCommand()), req.GetModel(), req.GetReasoning())
}

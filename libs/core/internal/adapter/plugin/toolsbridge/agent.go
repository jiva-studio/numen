package toolsbridge

import (
	"context"
	"errors"
	"maps"
	"time"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	agentport "github.com/jiva-studio/numen/libs/core/usecase/completion/agentcontract"
)

// listLimit is how long the plugins have to list their tools when a run starts.
const listLimit = 10 * time.Second

// Agent starts each run of the agent it wraps with an MCP endpoint of its own, which serves the
// tools the window allows and lives until the run is stopped.
type Agent struct {
	agent   agentport.Agent
	plugins plugins
	server  Server
}

// NewAgent wraps the agent so that its runs reach the tools the plugins of the host offer on the
// server.
func NewAgent(agent agentport.Agent, host Host, server Server) *Agent {
	return &Agent{agent: agent, plugins: plugins{host: host}, server: server}
}

// StartRun serves the tools the window allows and starts the run with their endpoint, their names
// and their declarations. A run with no tool to call is started with no endpoint.
func (a *Agent) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	listing, cancel := context.WithTimeout(ctx, listLimit)
	available := a.plugins.listTools(listing, a.server, "")
	cancel()
	if len(available) == 0 {
		task.Tools, task.ToolDeclarations, task.Endpoint = nil, nil, agentport.ToolEndpoint{}
		return a.agent.StartRun(ctx, task)
	}

	served, err := serveEndpoint(available, callScope{context: maps.Clone(task.Context), vault: task.Vault}, a.plugins)
	if err != nil {
		return nil, err
	}
	task.Endpoint = agentport.ToolEndpoint{URL: served.url, Token: served.token}
	task.Tools = make([]string, 0, len(available))
	task.ToolDeclarations = make([]agentport.ToolDeclaration, 0, len(available))
	for _, one := range available {
		task.Tools = append(task.Tools, one.spec.GetName())
		task.ToolDeclarations = append(task.ToolDeclarations, agentport.ToolDeclaration{
			Name: one.spec.GetName(), Title: one.spec.GetTitle(), Kind: getStepKind(one.spec),
		})
	}

	run, err := a.agent.StartRun(ctx, task)
	if err != nil {
		return nil, errors.Join(err, served.close(context.WithoutCancel(ctx)))
	}
	return &endpointRun{Run: run, endpoint: served}, nil
}

// EndConversation ends the conversation on the agent it wraps.
func (a *Agent) EndConversation(ctx context.Context, conversation string) error {
	return a.agent.EndConversation(ctx, conversation)
}

// endpointRun is a run and the endpoint serving its tools, which is closed when the run stops.
type endpointRun struct {
	agentport.Run
	endpoint *endpoint
}

// Stop stops the run, then the endpoint.
func (r *endpointRun) Stop() error {
	return errors.Join(r.Run.Stop(), r.endpoint.close(context.Background()))
}

// getStepKind is how a call of the tool is shown to a person.
func getStepKind(spec *v1.PluginTool) domain.StepKind {
	switch spec.GetStep() {
	case v1.ToolStep_TOOL_STEP_SEARCH:
		return domain.StepSearch
	case v1.ToolStep_TOOL_STEP_REMOVE:
		return domain.StepRemove
	case v1.ToolStep_TOOL_STEP_MOVE:
		return domain.StepMove
	case v1.ToolStep_TOOL_STEP_UNSPECIFIED:
	}
	if spec.GetEffect() == v1.ToolEffect_TOOL_EFFECT_READ_WRITE {
		return domain.StepEdit
	}
	return domain.StepRead
}

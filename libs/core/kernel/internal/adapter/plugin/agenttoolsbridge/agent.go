package agenttoolsbridge

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

// listLimit is how long the plugins have to list their tools when a run starts.
const listLimit = 10 * time.Second

// Agent starts each run of the agent it wraps with an MCP endpoint of its own, which serves the
// tools the window allows and lives until the run is stopped.
type Agent struct {
	agent    agentport.Agent
	plugins  plugins
	server   Server
	callerID string
}

// NewAgent wraps the agent of the plugin given so its runs can call the tools that the host's
// plugins provide on the server and whose required permissions that plugin has.
func NewAgent(agent agentport.Agent, host Host, server Server, callerID string) *Agent {
	return &Agent{agent: agent, plugins: plugins{host: host}, server: server, callerID: callerID}
}

// StartRun serves the tools the window allows and starts the run with their endpoint, their names
// and their declarations, and appends the sentences of those tools to its instructions. A run with
// no tool to call is started with no endpoint.
func (a *Agent) StartRun(ctx context.Context, task agentdomain.Task) (agentport.Run, error) {
	listing, cancel := context.WithTimeout(ctx, listLimit)
	available := a.plugins.listToolsFor(listing, a.server, a.callerID)
	cancel()
	if len(available) == 0 {
		task.Tools, task.ToolDeclarations, task.Endpoint = nil, nil, agentdomain.ToolEndpoint{}
		return a.agent.StartRun(ctx, task)
	}

	served, err := serveEndpoint(ctx, available, callScope{context: maps.Clone(task.Context), vault: task.Vault}, a.plugins)
	if err != nil {
		return nil, err
	}
	task.Endpoint = agentdomain.ToolEndpoint{URL: served.url, Token: served.token}
	task.Instructions = appendToolSentences(task.Instructions, available)
	task.Tools = make([]string, 0, len(available))
	task.ToolDeclarations = make([]agentdomain.ToolDeclaration, 0, len(available))
	for _, one := range available {
		task.Tools = append(task.Tools, one.spec.GetName())
		task.ToolDeclarations = append(task.ToolDeclarations, agentdomain.ToolDeclaration{
			Name: one.spec.GetName(), Title: one.spec.GetTitle(), Kind: getStepKind(one.spec),
		})
	}

	run, err := a.agent.StartRun(ctx, task)
	if err != nil {
		return nil, errors.Join(err, served.close(context.WithoutCancel(ctx)))
	}
	return &endpointRun{Run: run, endpoint: served}, nil
}

func (a *Agent) EndConversation(ctx context.Context, conversation string) error {
	return a.agent.EndConversation(ctx, conversation)
}

// endpointRun is a run and the endpoint serving its tools, which is closed when the run stops.
type endpointRun struct {
	agentport.Run
	endpoint *endpoint
}

func (r *endpointRun) Stop() error {
	return errors.Join(r.Run.Stop(), r.endpoint.close(context.Background()))
}

func appendToolSentences(instructions string, available []tool) string {
	var sentences []string
	for _, one := range available {
		for _, sentence := range one.spec.GetInstructions() {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	switch {
	case len(sentences) == 0:
		return instructions
	case instructions == "":
		return strings.Join(sentences, " ")
	default:
		return instructions + "\n\n" + strings.Join(sentences, " ")
	}
}

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

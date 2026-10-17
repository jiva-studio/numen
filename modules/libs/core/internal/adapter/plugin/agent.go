package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	// ErrNoAgentPlugin is returned when no plugin provides agent capability.
	ErrNoAgentPlugin = errors.New("no active agent plugin found")
)

// AgentPluginHost provides plugin resolution and activation for AI agents.
type AgentPluginHost interface {
	port.PluginResolver
	port.PluginActivator
}

// AgentBridge connects port.Agent calls to an out-of-process agent plugin via Connect-RPC.
type AgentBridge struct {
	host     port.PluginActivator
	pluginID string
	client   numenv1connect.AgentServiceClient
}

// OpenAgentBridge activates or connects to the agent plugin and returns an AgentBridge.
func OpenAgentBridge(ctx context.Context, host AgentPluginHost) (*AgentBridge, error) {
	if p, ok := host.GetPlugin("md.numen.agent-acp"); ok {
		return OpenAgentBridgeForPlugin(ctx, host, p.Manifest.ID)
	}
	if p, ok := host.GetPlugin("agent-acp"); ok {
		return OpenAgentBridgeForPlugin(ctx, host, p.Manifest.ID)
	}
	p, ok := host.FindPluginByCapability("agent")
	if !ok {
		p, ok = host.FindPluginByCapability("ai.agent.provider")
		if !ok {
			return nil, ErrNoAgentPlugin
		}
	}
	return OpenAgentBridgeForPlugin(ctx, host, p.Manifest.ID)
}

// OpenAgentBridgeForPlugin activates or connects to a specific agent plugin and returns an AgentBridge.
func OpenAgentBridgeForPlugin(ctx context.Context, host port.PluginActivator, pluginID string) (*AgentBridge, error) {
	endpoint, err := host.EnsureActive(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("ensure active %s: %w", pluginID, err)
	}

	httpClient := EndpointClient(endpoint)
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := numenv1connect.NewAgentServiceClient(httpClient, endpoint.BaseURL(), interceptor)

	return &AgentBridge{
		host:     host,
		pluginID: pluginID,
		client:   client,
	}, nil
}

// StartRun starts execution of the given task through the agent plugin.
func (b *AgentBridge) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	req := connect.NewRequest(&v1.AskAgentRequest{
		Prompt:       task.Question,
		Focus:        task.Focus,
		Conversation: task.Conversation,
		Agent:        task.Agent,
		Model:        task.Model,
		Instructions: task.Instructions,
		Tools:        task.Tools,
		MaxSteps:     int32(task.MaxSteps),
	})

	runCtx, cancel := context.WithCancel(ctx)
	stream, err := b.client.AskAgent(runCtx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("rpc ask agent: %w", err)
	}

	run := &pluginRun{
		ctx:    runCtx,
		stream: stream,
		cancel: cancel,
		steps:  make(chan port.Step, 16),
		onClose: func() {
			_ = b.Close()
		},
	}
	go run.readSteps()
	return run, nil
}

// EndConversation notifies the agent that the conversation has ended.
func (b *AgentBridge) EndConversation(ctx context.Context, conversation string) error {
	req := connect.NewRequest(&v1.FinishConversationRequest{
		Conversation: conversation,
	})
	_, err := b.client.FinishConversation(ctx, req)
	if err != nil {
		return fmt.Errorf("rpc finish conversation: %w", err)
	}
	return nil
}

// RegisterTool registers a custom agent tool with the running agent plugin.
func (b *AgentBridge) RegisterTool(ctx context.Context, name, description, schema string) error {
	req := connect.NewRequest(&v1.RegisterToolRequest{
		Tool: &v1.AgentToolDefinition{
			Name:                 name,
			Description:          description,
			ParametersJsonSchema: schema,
		},
	})
	_, err := b.client.RegisterTool(ctx, req)
	if err != nil {
		return fmt.Errorf("rpc register tool: %w", err)
	}
	return nil
}

// ExtendSystemPrompt extends the system prompt instructions of the agent plugin.
func (b *AgentBridge) ExtendSystemPrompt(ctx context.Context, extension string) (string, error) {
	req := connect.NewRequest(&v1.ExtendSystemPromptRequest{
		Extension: extension,
	})
	resp, err := b.client.ExtendSystemPrompt(ctx, req)
	if err != nil {
		return "", fmt.Errorf("rpc extend prompt: %w", err)
	}
	return resp.Msg.GetFullPrompt(), nil
}

// GetAgentTools retrieves the current tools and prompt extension from the agent plugin.
func (b *AgentBridge) GetAgentTools(ctx context.Context) ([]*v1.AgentToolDefinition, string, error) {
	req := connect.NewRequest(&v1.GetAgentToolsRequest{})
	resp, err := b.client.GetAgentTools(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("rpc get agent tools: %w", err)
	}
	return resp.Msg.GetTools(), resp.Msg.GetSystemPromptExtension(), nil
}

// Close releases the active reference count for the agent plugin in the host supervisor.
func (b *AgentBridge) Close() error {
	b.host.Release(b.pluginID)
	return nil
}

type pluginRun struct {
	ctx       context.Context
	stream    *connect.ServerStreamForClient[v1.AskAgentResponse]
	cancel    context.CancelFunc
	steps     chan port.Step
	closeOnce sync.Once
	onClose   func()
}

func (r *pluginRun) GetSteps() <-chan port.Step {
	return r.steps
}

func (r *pluginRun) Stop() error {
	r.closeOnce.Do(func() {
		r.cancel()
		if r.stream != nil {
			_ = r.stream.Close()
		}
		if r.onClose != nil {
			r.onClose()
		}
	})
	return nil
}

func (r *pluginRun) readSteps() {
	defer close(r.steps)
	defer func() {
		_ = r.Stop()
	}()

	for r.stream.Receive() {
		msg := r.stream.Msg()
		step := convertResponseStep(msg)
		select {
		case <-r.ctx.Done():
			return
		case r.steps <- step:
		}
		if step.Kind == domain.StepStop {
			return
		}
	}
}

// getStepKind is what a call does to the vault, by the word the plugin reports it in. A word the
// core does not know is a call of unknown effect.
func getStepKind(kind string) domain.StepKind {
	switch kind {
	case "read":
		return domain.StepRead
	case "edit":
		return domain.StepEdit
	case "remove":
		return domain.StepRemove
	case "move":
		return domain.StepMove
	case "search":
		return domain.StepSearch
	default:
		return domain.StepToolCall
	}
}

func convertResponseStep(msg *v1.AskAgentResponse) port.Step {
	switch s := msg.GetStep().(type) {
	case *v1.AskAgentResponse_Said:
		return port.Step{
			Kind: domain.StepSay,
			Text: s.Said,
		}
	case *v1.AskAgentResponse_ToolCall:
		tc := s.ToolCall
		var spans []domain.ByteSpan
		if tc.GetSpan() != nil {
			spans = []domain.ByteSpan{{
				From: int(tc.GetSpan().GetFrom()),
				To:   int(tc.GetSpan().GetTo()),
			}}
		}
		return port.Step{
			Kind:  getStepKind(tc.GetKind()),
			Call:  tc.GetCall(),
			Tool:  tc.GetTool(),
			About: tc.GetAbout(),
			Count: int(tc.GetWritten()),
			Place: domain.Place{
				Path:  tc.GetPath(),
				Spans: spans,
			},
		}
	case *v1.AskAgentResponse_ToolFinished:
		return port.Step{
			Kind: domain.StepToolDone,
		}
	case *v1.AskAgentResponse_ModelStarted:
		return port.Step{
			Kind: domain.StepModelStart,
		}
	case *v1.AskAgentResponse_Stopped:
		return port.Step{
			Kind:   domain.StepStop,
			Detail: s.Stopped,
		}
	default:
		return port.Step{
			Kind: domain.StepSay,
		}
	}
}

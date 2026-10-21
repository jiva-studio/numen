package plugin

import (
	"context"
	"fmt"
	"math"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// AgentPluginHost provides plugin resolution and activation for AI agents.
type AgentPluginHost interface {
	port.PluginResolver
	port.PluginActivator
}

// AgentBridge connects port.Agent calls to an out-of-process agent plugin via Connect-RPC.
type AgentBridge struct {
	pluginID string
	service  pluginService[numenv1connect.AgentServiceClient]
	lease    *pluginLease
}

// OpenAgentBridgeForPlugin activates or connects to a specific agent plugin and returns an AgentBridge.
func OpenAgentBridgeForPlugin(ctx context.Context, host AgentPluginHost, pluginID string) (*AgentBridge, error) {
	_, lease, err := openPlugin(ctx, host, pluginID, numenv1connect.NewAgentServiceClient)
	if err != nil {
		return nil, err
	}

	return &AgentBridge{
		pluginID: pluginID,
		service:  pluginService[numenv1connect.AgentServiceClient]{host: host, pluginID: pluginID, newClient: numenv1connect.NewAgentServiceClient},
		lease:    lease,
	}, nil
}

// newAskRequest is the task in the schema's words.
func newAskRequest(task port.Task) *v1.AskAgentRequest {
	declarations := make([]*v1.ToolDeclaration, 0, len(task.ToolDeclarations))
	for _, one := range task.ToolDeclarations {
		declarations = append(declarations, &v1.ToolDeclaration{
			Name:    one.Name,
			Title:   one.Title,
			Kind:    getKindWord(one.Kind),
			About:   one.About,
			Element: one.Element,
			Match:   one.Match,
			Text:    one.Text,
		})
	}
	return &v1.AskAgentRequest{
		Prompt:           task.Question,
		Focus:            task.Focus,
		Conversation:     task.Conversation,
		Agent:            task.Agent,
		Model:            task.Model,
		Reasoning:        task.Reasoning,
		Instructions:     task.Instructions,
		Tools:            task.Tools,
		MaxSteps:         int32(min(max(task.MaxSteps, 0), math.MaxInt32)),
		ToolDeclarations: declarations,
		Root:             task.Root,
		Command:          task.Command,
		Options:          task.Options,
		ToolsUrl:         task.Endpoint.URL,
		ToolsToken:       task.Endpoint.Token,
	}
}

// StartRun starts execution of the given task through the agent plugin, at the endpoint it is at
// now. The run holds its own reference on the plugin until it stops.
func (b *AgentBridge) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	req := connect.NewRequest(newAskRequest(task))

	client, lease, err := b.service.open(ctx)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	stream, err := client.AskAgent(runCtx, req)
	if err != nil {
		cancel()
		_ = lease.Close()
		return nil, fmt.Errorf("rpc ask agent: %w", err)
	}

	run := &pluginRun{
		ctx:    runCtx,
		stream: stream,
		cancel: cancel,
		steps:  make(chan port.Step, 16),
		onClose: func() {
			_ = lease.Close()
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
	client, call, err := b.service.open(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = call.Close() }()

	_, err = client.FinishConversation(ctx, req)
	if err != nil {
		return fmt.Errorf("rpc finish conversation: %w", err)
	}
	return nil
}

// GetPrograms lists the programs the plugin runs, with the models each offers. Commands are the
// command lines the installation set, by program.
func (b *AgentBridge) GetPrograms(ctx context.Context, commands map[string][]string) ([]port.AgentProgram, error) {
	request := make([]*v1.AgentProgramCommand, 0, len(commands))
	for program, command := range commands {
		request = append(request, &v1.AgentProgramCommand{Program: program, Command: command})
	}
	client, call, err := b.service.open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.GetAgentPrograms(ctx, connect.NewRequest(&v1.GetAgentProgramsRequest{Commands: request}))
	if err != nil {
		return nil, fmt.Errorf("rpc get agent programs: %w", err)
	}

	programs := make([]port.AgentProgram, 0, len(resp.Msg.GetPrograms()))
	for _, one := range resp.Msg.GetPrograms() {
		models := make([]port.AgentModel, 0, len(one.GetModels()))
		for _, model := range one.GetModels() {
			models = append(models, port.AgentModel{
				Name: model.GetName(), Title: model.GetTitle(), IsDefault: model.GetIsDefault(),
				ReasoningLevels: newReasoningLevels(model.GetReasoningLevels()), DefaultReasoning: model.GetDefaultReasoning(),
			})
		}
		programs = append(programs, port.AgentProgram{
			Provider:    b.pluginID,
			Name:        one.GetName(),
			Title:       one.GetTitle(),
			IsInstalled: one.GetIsInstalled(),
			IsDefault:   one.GetIsDefault(),
			LimitsTools: one.GetLimitsTools(),
			Models:      models,
		})
	}
	return programs, nil
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
	client, call, err := b.service.open(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = call.Close() }()

	_, err = client.RegisterTool(ctx, req)
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
	client, call, err := b.service.open(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.ExtendSystemPrompt(ctx, req)
	if err != nil {
		return "", fmt.Errorf("rpc extend prompt: %w", err)
	}
	return resp.Msg.GetFullPrompt(), nil
}

// GetAgentTools retrieves the current tools and prompt extension from the agent plugin.
func (b *AgentBridge) GetAgentTools(ctx context.Context) ([]*v1.AgentToolDefinition, string, error) {
	req := connect.NewRequest(&v1.GetAgentToolsRequest{})
	client, call, err := b.service.open(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.GetAgentTools(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("rpc get agent tools: %w", err)
	}
	return resp.Msg.GetTools(), resp.Msg.GetSystemPromptExtension(), nil
}

// Close releases the reference the bridge holds on the agent plugin, once. A run started through the bridge holds its own.
func (b *AgentBridge) Close() error {
	return b.lease.Close()
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
	if err := r.stream.Err(); err != nil && r.ctx.Err() == nil {
		select {
		case <-r.ctx.Done():
		case r.steps <- port.Step{Kind: domain.StepStop, Detail: err.Error()}:
		}
	}
}

func convertResponseStep(msg *v1.AskAgentResponse) port.Step {
	switch s := msg.GetStep().(type) {
	case *v1.AskAgentResponse_Text:
		return port.Step{
			Kind: domain.StepSay,
			Text: s.Text,
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
			Count: int(tc.GetLength()),
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
	case *v1.AskAgentResponse_EditProgress:
		progress := s.EditProgress
		return port.Step{
			Kind:  domain.StepDraft,
			Call:  progress.GetCall(),
			Place: domain.Place{Path: progress.GetPath()},
			Match: progress.GetMatch(),
			Text:  progress.GetText(),
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

// stepKindWords are what a call does to the vault, in the words the plugins report it in.
var stepKindWords = map[domain.StepKind]string{
	domain.StepRead:   "read",
	domain.StepEdit:   "edit",
	domain.StepRemove: "remove",
	domain.StepMove:   "move",
	domain.StepSearch: "search",
}

// getKindWord is the word a plugin is told what a call does by. A kind that says nothing about the
// vault is told as no word.
func getKindWord(kind domain.StepKind) string { return stepKindWords[kind] }

// getStepKind is what a call does to the vault, by the word the plugin reports it in. A word the
// core does not know is a call of unknown effect.
func getStepKind(kind string) domain.StepKind {
	for effect, word := range stepKindWords {
		if word == kind {
			return effect
		}
	}
	return domain.StepToolCall
}

// newReasoningLevels is the levels of reasoning a plugin reports, as the port holds them.
func newReasoningLevels(reported []*v1.ReasoningLevel) []port.ReasoningLevel {
	if len(reported) == 0 {
		return nil
	}
	levels := make([]port.ReasoningLevel, 0, len(reported))
	for _, one := range reported {
		levels = append(levels, port.ReasoningLevel{Name: one.GetName(), Title: one.GetTitle()})
	}
	return levels
}

package agentbridge

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	agentport "github.com/jiva-studio/numen/libs/core/usecase/completion/agentcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// Host provides plugin resolution and activation for AI agents.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
}

// Agent connects port.Agent calls to an out-of-process agent plugin via Connect-RPC.
type Agent struct {
	pluginID string
	service  bridge.Service[numenv1connect.AgentServiceClient]
	lease    *bridge.Lease
}

// OpenForPlugin activates or connects to a specific agent plugin and returns an AgentBridge.
func OpenForPlugin(ctx context.Context, host Host, pluginID string) (*Agent, error) {
	_, lease, err := bridge.Open(ctx, host, pluginID, numenv1connect.NewAgentServiceClient)
	if err != nil {
		return nil, err
	}

	return &Agent{
		pluginID: pluginID,
		service:  bridge.Service[numenv1connect.AgentServiceClient]{Host: host, PluginID: pluginID, NewClient: numenv1connect.NewAgentServiceClient},
		lease:    lease,
	}, nil
}

// newAskRequest is the task in the schema's words.
func newAskRequest(task agentport.Task) *v1.AskAgentRequest {
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
		Context:          task.Context,
		VaultId:          string(task.Vault),
	}
}

// StartRun starts execution of the given task through the agent plugin, at the endpoint it is at
// now. The run holds its own reference on the plugin until it stops.
func (b *Agent) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	req := connect.NewRequest(newAskRequest(task))

	client, lease, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	stream, err := client.AskAgent(runCtx, req)
	if err != nil {
		cancel()
		return nil, errors.Join(fmt.Errorf("rpc ask agent: %w", err), lease.Close())
	}

	run := &pluginRun{
		stream:  stream,
		cancel:  cancel,
		steps:   make(chan agentport.Step, 16),
		onClose: lease.Close,
	}
	go run.readSteps(runCtx)
	return run, nil
}

// EndConversation notifies the agent that the conversation has ended.
func (b *Agent) EndConversation(ctx context.Context, conversation string) (err error) {
	req := connect.NewRequest(&v1.FinishConversationRequest{
		Conversation: conversation,
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	_, err = client.FinishConversation(ctx, req)
	if err != nil {
		return fmt.Errorf("rpc finish conversation: %w", err)
	}
	return nil
}

// GetPrograms lists the programs the plugin runs, with the models each offers. Commands are the
// command lines the installation set, by program.
func (b *Agent) GetPrograms(ctx context.Context, commands map[string][]string) (programs []agentport.AgentProgram, err error) {
	request := make([]*v1.AgentProgramCommand, 0, len(commands))
	for program, command := range commands {
		request = append(request, &v1.AgentProgramCommand{Program: program, Command: command})
	}
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.GetAgentPrograms(ctx, connect.NewRequest(&v1.GetAgentProgramsRequest{Commands: request}))
	if err != nil {
		return nil, fmt.Errorf("rpc get agent programs: %w", err)
	}

	programs = make([]agentport.AgentProgram, 0, len(resp.Msg.GetPrograms()))
	for _, one := range resp.Msg.GetPrograms() {
		models := make([]agentport.AgentModel, 0, len(one.GetModels()))
		for _, model := range one.GetModels() {
			models = append(models, agentport.AgentModel{
				Name: model.GetName(), Title: model.GetTitle(), IsDefault: model.GetIsDefault(),
				ReasoningLevels: newReasoningLevels(model.GetReasoningLevels()), DefaultReasoning: model.GetDefaultReasoning(),
			})
		}
		programs = append(programs, agentport.AgentProgram{
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
func (b *Agent) RegisterTool(ctx context.Context, name, description, schema string) (err error) {
	req := connect.NewRequest(&v1.RegisterToolRequest{
		Tool: &v1.AgentToolDefinition{
			Name:                 name,
			Description:          description,
			ParametersJsonSchema: schema,
		},
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	_, err = client.RegisterTool(ctx, req)
	if err != nil {
		return fmt.Errorf("rpc register tool: %w", err)
	}
	return nil
}

// ExtendSystemPrompt extends the system prompt instructions of the agent plugin.
func (b *Agent) ExtendSystemPrompt(ctx context.Context, extension string) (prompt string, err error) {
	req := connect.NewRequest(&v1.ExtendSystemPromptRequest{
		Extension: extension,
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.ExtendSystemPrompt(ctx, req)
	if err != nil {
		return "", fmt.Errorf("rpc extend prompt: %w", err)
	}
	return resp.Msg.GetFullPrompt(), nil
}

// GetAgentTools retrieves the current tools and prompt extension from the agent plugin.
func (b *Agent) GetAgentTools(ctx context.Context) (tools []*v1.AgentToolDefinition, extension string, err error) {
	req := connect.NewRequest(&v1.GetAgentToolsRequest{})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { err = errors.Join(err, call.Close()) }()

	resp, err := client.GetAgentTools(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("rpc get agent tools: %w", err)
	}
	return resp.Msg.GetTools(), resp.Msg.GetSystemPromptExtension(), nil
}

// Close releases the reference the bridge holds on the agent plugin, once. A run started through the bridge holds its own.
func (b *Agent) Close() error {
	return b.lease.Close()
}

type pluginRun struct {
	stream    *connect.ServerStreamForClient[v1.AskAgentResponse]
	cancel    context.CancelFunc
	steps     chan agentport.Step
	closeOnce sync.Once
	onClose   func() error
}

func (r *pluginRun) GetSteps() <-chan agentport.Step {
	return r.steps
}

// Stop cancels the run and releases its stream and plugin reference. Only the first call reports
// the release error.
func (r *pluginRun) Stop() error {
	var err error
	r.closeOnce.Do(func() {
		r.cancel()
		if r.stream != nil {
			err = errors.Join(err, r.stream.Close())
		}
		if r.onClose != nil {
			err = errors.Join(err, r.onClose())
		}
	})
	return err
}

func (r *pluginRun) readSteps(ctx context.Context) {
	defer close(r.steps)
	defer func() {
		if err := r.Stop(); err != nil {
			select {
			case r.steps <- agentport.Step{Kind: domain.StepStop, Detail: err.Error()}:
			default:
			}
		}
	}()

	for r.stream.Receive() {
		msg := r.stream.Msg()
		step := convertResponseStep(msg)
		select {
		case <-ctx.Done():
			return
		case r.steps <- step:
		}
		if step.Kind == domain.StepStop {
			return
		}
	}
	if err := r.stream.Err(); err != nil && ctx.Err() == nil {
		select {
		case <-ctx.Done():
		case r.steps <- agentport.Step{Kind: domain.StepStop, Detail: err.Error()}:
		}
	}
}

func convertResponseStep(msg *v1.AskAgentResponse) agentport.Step {
	switch s := msg.GetStep().(type) {
	case *v1.AskAgentResponse_Text:
		return agentport.Step{
			Kind: domain.StepSay,
			Text: s.Text,
		}
	case *v1.AskAgentResponse_ToolCall:
		tc := s.ToolCall
		var spans []domain.ByteRange
		if tc.GetRange() != nil {
			spans = []domain.ByteRange{{
				Start: int(tc.GetRange().GetStart()),
				End:   int(tc.GetRange().GetEnd()),
			}}
		}
		return agentport.Step{
			Kind:  getStepKind(tc.GetKind()),
			Call:  tc.GetCall(),
			Tool:  tc.GetTool(),
			About: tc.GetAbout(),
			Count: int(tc.GetLength()),
			Place: domain.Place{
				Path:   tc.GetPath(),
				Ranges: spans,
			},
		}
	case *v1.AskAgentResponse_ToolFinished:
		return agentport.Step{
			Kind: domain.StepToolDone,
		}
	case *v1.AskAgentResponse_ModelStarted:
		return agentport.Step{
			Kind: domain.StepModelStart,
		}
	case *v1.AskAgentResponse_EditProgress:
		progress := s.EditProgress
		return agentport.Step{
			Kind:  domain.StepDraft,
			Call:  progress.GetCall(),
			Place: domain.Place{Path: progress.GetPath()},
			Match: progress.GetMatch(),
			Text:  progress.GetText(),
		}
	case *v1.AskAgentResponse_Stopped:
		return agentport.Step{
			Kind:   domain.StepStop,
			Detail: s.Stopped,
		}
	default:
		return agentport.Step{
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

// getKindWord returns the word that identifies to a plugin what a call does. A
// kind that does not act on the vault maps to the empty word.
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
func newReasoningLevels(reported []*v1.ReasoningLevel) []agentport.ReasoningLevel {
	if len(reported) == 0 {
		return nil
	}
	levels := make([]agentport.ReasoningLevel, 0, len(reported))
	for _, one := range reported {
		levels = append(levels, agentport.ReasoningLevel{Name: one.GetName(), Title: one.GetTitle()})
	}
	return levels
}

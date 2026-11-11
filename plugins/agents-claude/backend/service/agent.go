package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/toolset"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/executable"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/toolconfig"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/programs"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/run"
)

// agent answers the calls of the agent service with the use cases of the plugin.
type agent struct {
	settings *settings.Settings
	toolset  *toolset.Toolset
	programs *programs.Programs
	run      *run.Run
}

// parts are what differs between the plugin process, a host that links the plugin in, and a test.
type parts struct {
	environment ports.Environment
	clock       run.Clock
	isAsync     bool
}

func newAgent(p parts) (*agent, error) {
	agentSettings := new(settings.Settings)
	tools := new(toolset.Toolset)
	lister := process.Lister{}
	models, err := catalog.New(lister, p.isAsync)
	if err != nil {
		return nil, err
	}
	models.SetReport(lister.ReportError)
	known, err := programs.New(programs.Dependencies{
		Settings: agentSettings, Finder: executable.Finder{}, Models: models, Environment: p.environment,
	})
	if err != nil {
		return nil, err
	}
	runner, err := run.New(run.Dependencies{
		Settings: agentSettings, Prompt: tools, Commands: known, Reasoning: models,
		Launcher: process.Launcher{}, ToolConfig: toolconfig.Files{}, Environment: p.environment, Clock: p.clock,
	})
	if err != nil {
		return nil, err
	}
	return &agent{settings: agentSettings, toolset: tools, programs: known, run: runner}, nil
}

func (a *agent) Ask(ctx context.Context, msg *agentrun.Request, send func(*agentrun.Response) error) error {
	return a.run.Ask(ctx, msg, send)
}

func (a *agent) FinishConversation(ctx context.Context, conversation string) error {
	return a.run.FinishConversation(ctx, conversation)
}

func (a *agent) GetState(_ context.Context) (*agentrun.State, error) {
	return a.programs.GetState(), nil
}

func (a *agent) GetPrograms(ctx context.Context, commands []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	return a.programs.List(ctx, commands), nil
}

func (a *agent) RegisterTool(_ context.Context, tool *agentrun.ToolDefinition) error {
	a.toolset.Register(tool)
	return nil
}

func (a *agent) ExtendSystemPrompt(_ context.Context, extension string) (string, error) {
	return a.toolset.ExtendPrompt(extension), nil
}

func (a *agent) GetTools(_ context.Context) (*agentrun.ToolSet, error) {
	return a.toolset.Get(), nil
}

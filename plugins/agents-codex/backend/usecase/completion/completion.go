// Package completion answers the completion requests of a connection with one run of OpenAI Codex
// and lists the models and reasoning levels the program offers a connection.
package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
)

const (
	roleSystem          = "system"
	roleUser            = "user"
	roleAssistant       = "assistant"
	stoppedDone         = "done"
	defaultRequestLimit = 120 * time.Second
)

var errNoQuestion = errors.New("the request holds no message to answer")

// Asker runs a task of the agent service.
type Asker interface {
	Ask(ctx context.Context, msg *agentrun.Request, send func(*agentrun.Response) error) error
}

// Programs reports on the program the plugin runs.
type Programs interface {
	GetState(executablePath string) *agentrun.State
	GetPrograms(ctx context.Context, executablePath string, commands []agentrun.ProgramCommand) []*agentrun.Program
}

// Dependencies are what a Completion needs.
type Dependencies struct {
	Settings *settings.Settings
	Asker    Asker
	Programs Programs
}

// Completion serves the provider roles of the plugin.
type Completion struct {
	deps Dependencies
}

// New creates a Completion.
func New(deps Dependencies) *Completion {
	return &Completion{deps: deps}
}

// Complete answers the request with one run of OpenAI Codex that is given no tool and cannot write to
// disk. A program that is not installed, and a run that outlasts the request timeout, are failures to
// try the next connection after. The text of a failed run does not leave the plugin.
func (c *Completion) Complete(
	ctx context.Context, call connection.Call, request connection.ProviderCompleteRequest,
) (connection.CompleteResult, error) {
	task, err := newTask(call, request)
	if err != nil {
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	if !c.isInstalled() {
		return connection.CompleteResult{}, connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	limited, cancel := context.WithTimeout(ctx, c.getRequestLimit())
	defer cancel()

	var answer strings.Builder
	stopReason := ""
	runErr := c.deps.Asker.Ask(limited, task, func(response *agentrun.Response) error {
		answer.WriteString(response.GetText())
		if reason := response.GetStopped(); reason != "" {
			stopReason = reason
		}
		return nil
	})
	if err := ctx.Err(); err != nil {
		return connection.CompleteResult{}, err
	}
	switch {
	case limited.Err() != nil:
		return connection.CompleteResult{}, connection.NewTryNextError(connection.ReasonTimeout, 0)
	case runErr != nil, stopReason != "" && stopReason != stoppedDone, strings.TrimSpace(answer.String()) == "":
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonUnknown)
	}
	return connection.CompleteResult{Text: answer.String()}, nil
}

// ListModels lists the models OpenAI Codex offers.
func (c *Completion) ListModels(ctx context.Context, _ connection.Call, _ string) ([]connection.Model, error) {
	programModels, err := c.getProgramModels(ctx)
	if err != nil {
		return nil, err
	}
	models := make([]connection.Model, 0, len(programModels))
	for _, one := range programModels {
		title := one.Title
		if title == "" {
			title = one.Name
		}
		models = append(models, connection.Model{ID: one.Name, Title: title, Operations: nil})
	}
	return models, nil
}

// ListReasoningLevels lists the levels of reasoning the model of the call takes, none for a model that
// offers no choice.
func (c *Completion) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.ReasoningLevel, error) {
	programModels, err := c.getProgramModels(ctx)
	if err != nil {
		return nil, err
	}
	var levels []connection.ReasoningLevel
	for _, one := range programModels {
		if one.Name != call.Model {
			continue
		}
		for _, level := range one.ReasoningLevels {
			levels = append(levels, connection.ReasoningLevel{Value: level.Name, Title: level.Title})
		}
	}
	return levels, nil
}

func (c *Completion) getRequestLimit() time.Duration {
	if limit := c.deps.Settings.RequestTimeout(); limit > 0 {
		return limit
	}
	return defaultRequestLimit
}

func (c *Completion) isInstalled() bool {
	return c.deps.Programs.GetState(c.deps.Settings.ExecutablePath()).Unreachable == ""
}

func (c *Completion) getProgramModels(ctx context.Context) ([]*reasoning.Model, error) {
	if !c.isInstalled() {
		return nil, connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	var models []*reasoning.Model
	for _, one := range c.deps.Programs.GetPrograms(ctx, c.deps.Settings.ExecutablePath(), nil) {
		models = append(models, one.Models...)
	}
	return models, nil
}

func newTask(call connection.Call, request connection.ProviderCompleteRequest) (*agentrun.Request, error) {
	var system []string
	var turns []connection.Message
	for _, message := range request.Messages {
		switch message.Role {
		case roleSystem:
			system = append(system, message.Text)
		case "", roleUser, roleAssistant:
			turns = append(turns, message)
		default:
			return nil, fmt.Errorf("unknown role %q", message.Role)
		}
	}
	if len(turns) == 0 {
		return nil, errNoQuestion
	}
	return &agentrun.Request{
		Prompt: writePrompt(turns), Focus: "", Conversation: "", Context: nil, Agent: "", Model: call.Model,
		Instructions: strings.Join(system, "\n\n"), Tools: nil, MaxSteps: 0, ToolDeclarations: nil, Root: "",
		Command: nil, ToolsURL: "", ToolsToken: "", Reasoning: call.Reasoning,
	}, nil
}

func writePrompt(turns []connection.Message) string {
	if len(turns) == 1 {
		return turns[0].Text
	}
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		role := "User"
		if turn.Role == roleAssistant {
			role = "Assistant"
		}
		parts = append(parts, role+":\n"+turn.Text)
	}
	return strings.Join(parts, "\n\n")
}

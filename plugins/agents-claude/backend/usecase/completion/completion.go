// Package completion answers the completion requests of a connection with one run of Claude Code
// and lists the models and reasoning levels the program offers a connection.
package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
)

const (
	roleSystem          = "system"
	roleUser            = "user"
	roleAssistant       = "assistant"
	defaultRequestLimit = 120 * time.Second
)

var errNoQuestion = errors.New("the request holds no message to answer")

// Answerer runs the program once on a prompt, with no tool.
type Answerer interface {
	FindProgram() string
	GetEnvironment() []string
	Answer(ctx context.Context, call connection.Call, instructions, prompt string) (string, error)
}

// Models lists the models of the program on a command line.
type Models interface {
	List(ctx context.Context, command string, extra, env []string) []model.Model
}

// Dependencies are what a Completion needs.
type Dependencies struct {
	Settings *settings.Settings
	Answerer Answerer
	Models   Models
}

// Completion serves the provider roles of the plugin.
type Completion struct {
	deps Dependencies
}

// New creates a Completion, and refuses dependencies that are missing.
func New(deps Dependencies) (*Completion, error) {
	if deps.Settings == nil || deps.Answerer == nil || deps.Models == nil {
		return nil, errors.New("completion: every dependency is required")
	}
	return &Completion{deps: deps}, nil
}

// Complete answers the request with one run of Claude Code that is given no tool and cannot write to
// disk. A program that is not installed, and a run that outlasts the request timeout, are failures to
// try the next connection after. The text of a failed run does not leave the plugin.
func (c *Completion) Complete(
	ctx context.Context, call connection.Call, request *llm.CompleteRequest,
) (*llm.CompleteResult, error) {
	instructions, prompt, err := splitMessages(request.GetMessages())
	if err != nil {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	if c.deps.Answerer.FindProgram() == "" {
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	limited, cancel := context.WithTimeout(ctx, c.getRequestLimit())
	defer cancel()

	answer, runErr := c.deps.Answerer.Answer(limited, call, instructions, prompt)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch {
	case limited.Err() != nil:
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	case runErr != nil:
		if _, isTryNext := connection.IsTryNext(runErr); isTryNext {
			return nil, runErr
		}
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	return &llm.CompleteResult{Text: answer}, nil
}

// ListModels lists the models Claude Code offers.
func (c *Completion) ListModels(ctx context.Context, _ connection.Call, _ connection.OptionsQuery) (connection.ListedOptions, error) {
	programModels, err := c.getProgramModels(ctx)
	if err != nil {
		return connection.ListedOptions{}, err
	}
	models := make([]connection.Option, 0, len(programModels))
	for _, one := range programModels {
		title := one.DisplayName
		if title == "" {
			title = one.Value
		}
		models = append(models, connection.Option{Value: one.Value, Title: title})
	}
	return connection.ListedOptions{Options: models}, nil
}

// ListReasoningLevels lists the levels of reasoning the model of the call takes, none for a model that
// offers no choice.
func (c *Completion) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.Option, error) {
	programModels, err := c.getProgramModels(ctx)
	if err != nil {
		return nil, err
	}
	var levels []connection.Option
	for _, one := range programModels {
		if one.Value != call.Options[llm.OptionModel] {
			continue
		}
		for _, level := range one.GetLevels() {
			levels = append(levels, connection.Option{Value: level.Name, Title: level.Title})
		}
	}
	return levels, nil
}

func (c *Completion) getRequestLimit() time.Duration {
	if limit := c.deps.Settings.Get().RequestTimeout; limit > 0 {
		return limit
	}
	return defaultRequestLimit
}

func (c *Completion) getProgramModels(ctx context.Context) ([]model.Model, error) {
	program := c.deps.Answerer.FindProgram()
	if program == "" {
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	return c.deps.Models.List(ctx, program, nil, c.deps.Answerer.GetEnvironment()), nil
}

func splitMessages(messages []*llm.Message) (instructions, prompt string, err error) {
	var system []string
	var turns []*llm.Message
	for _, message := range messages {
		switch message.GetRole() {
		case roleSystem:
			system = append(system, message.GetText())
		case "", roleUser, roleAssistant:
			turns = append(turns, message)
		default:
			return "", "", fmt.Errorf("unknown role %q", message.GetRole())
		}
	}
	if len(turns) == 0 {
		return "", "", errNoQuestion
	}
	return strings.Join(system, "\n\n"), writePrompt(turns), nil
}

func writePrompt(turns []*llm.Message) string {
	if len(turns) == 1 {
		return turns[0].GetText()
	}
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		role := "User"
		if turn.GetRole() == roleAssistant {
			role = "Assistant"
		}
		parts = append(parts, role+":\n"+turn.GetText())
	}
	return strings.Join(parts, "\n\n")
}

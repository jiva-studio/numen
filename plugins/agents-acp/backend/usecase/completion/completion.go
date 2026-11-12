// Package completion answers the completion requests of a connection in one session of the ACP
// agent the connection names, and lists the models and reasoning levels the agent reports.
package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/settings"
)

const (
	fieldProgram        = "program"
	roleSystem          = "system"
	roleUser            = "user"
	roleAssistant       = "assistant"
	defaultRequestLimit = 120 * time.Second
)

var errNoQuestion = errors.New("the request holds no message to answer")

// Answerer runs one session of an agent and returns its text.
type Answerer interface {
	Answer(ctx context.Context, agent agents.Agent, model, reasoning, system, prompt string) (string, error)
}

// Listings reads the models an agent reports.
type Listings interface {
	GetListing(ctx context.Context, agent agents.Agent) acp.Listing
}

// LookPath finds a program on the machine. A nil LookPath finds none.
type LookPath func(program string) (string, error)

// Dependencies are what a Completion needs.
type Dependencies struct {
	Settings *settings.Settings
	Answerer Answerer
	Listings Listings
	LookPath LookPath
}

// Completion serves the provider roles of the plugin.
type Completion struct {
	deps Dependencies
}

// New creates a Completion, and refuses dependencies that are missing.
func New(deps Dependencies) (*Completion, error) {
	if deps.Settings == nil || deps.Answerer == nil || deps.Listings == nil {
		return nil, errors.New("completion: every dependency is required")
	}
	return &Completion{deps: deps}, nil
}

// Complete answers the request in one session of the program the call names. The session runs in an
// empty temporary folder and is offered no file system, no terminal and no tool, and every permission
// request is refused. A program that cannot be found, and a session that outlasts the request timeout,
// are failures to try the next connection after. The text of a failed session does not leave the plugin.
func (c *Completion) Complete(
	ctx context.Context, call connection.Call, request connection.ProviderCompleteRequest,
) (connection.CompleteResult, error) {
	system, prompt, err := splitMessages(request.Messages)
	if err != nil {
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	agent, err := c.getCallAgent(call)
	if err != nil {
		return connection.CompleteResult{}, err
	}
	limited, cancel := context.WithTimeout(ctx, c.getRequestLimit())
	defer cancel()

	text, runErr := c.deps.Answerer.Answer(limited, agent, call.Model, call.Reasoning, system, prompt)
	if err := ctx.Err(); err != nil {
		return connection.CompleteResult{}, err
	}
	switch {
	case limited.Err() != nil:
		return connection.CompleteResult{}, connection.NewTryNextError(connection.ReasonTimeout, 0)
	case runErr != nil, strings.TrimSpace(text) == "":
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonUnknown)
	}
	return connection.CompleteResult{Text: text}, nil
}

// ListModels lists the models the program the call names reports.
func (c *Completion) ListModels(ctx context.Context, call connection.Call, _ string) ([]connection.Model, error) {
	listing, err := c.readListing(ctx, call)
	if err != nil {
		return nil, err
	}
	models := make([]connection.Model, 0, len(listing.Models))
	for _, one := range listing.Models {
		title := one.Name
		if title == "" {
			title = one.ID
		}
		models = append(models, connection.Model{ID: one.ID, Title: title, Operations: nil})
	}
	return models, nil
}

// ListReasoningLevels lists the levels of reasoning the program reports for the model it answers with.
// A model of the call other than that one is listed with none.
func (c *Completion) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.ReasoningLevel, error) {
	listing, err := c.readListing(ctx, call)
	if err != nil {
		return nil, err
	}
	if call.Model != "" && call.Model != listing.Current {
		return nil, nil
	}
	levels := make([]connection.ReasoningLevel, 0, len(listing.Levels))
	for _, one := range listing.Levels {
		levels = append(levels, connection.ReasoningLevel{Value: one.Name, Title: one.Title})
	}
	return levels, nil
}

func (c *Completion) readListing(ctx context.Context, call connection.Call) (acp.Listing, error) {
	agent, err := c.getCallAgent(call)
	if err != nil {
		return acp.Listing{}, err
	}
	return c.deps.Listings.GetListing(ctx, agent), nil
}

func (c *Completion) getCallAgent(call connection.Call) (agents.Agent, error) {
	program := strings.TrimSpace(call.Fields[fieldProgram])
	if program == "" {
		return agents.Agent{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	if c.deps.LookPath == nil {
		return agents.Agent{}, connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	if _, err := c.deps.LookPath(program); err != nil {
		return agents.Agent{}, connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	return agents.NewFromProgram(program), nil
}

func (c *Completion) getRequestLimit() time.Duration {
	if limit := c.deps.Settings.RequestTimeout(); limit > 0 {
		return limit
	}
	return defaultRequestLimit
}

func splitMessages(messages []connection.Message) (system, prompt string, err error) {
	var instructions []string
	var turns []connection.Message
	for _, message := range messages {
		switch message.Role {
		case roleSystem:
			instructions = append(instructions, message.Text)
		case "", roleUser, roleAssistant:
			turns = append(turns, message)
		default:
			return "", "", fmt.Errorf("unknown role %q", message.Role)
		}
	}
	if len(turns) == 0 {
		return "", "", errNoQuestion
	}
	return strings.Join(instructions, "\n\n"), writePrompt(turns), nil
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

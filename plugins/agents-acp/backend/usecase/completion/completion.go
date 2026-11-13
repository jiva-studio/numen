// Package completion answers the completion requests of a connection in one session of the ACP
// program the connection names, and lists the models and reasoning levels the program reports.
package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
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
	// maxAnswerBytes is the most text an answer may hold.
	maxAnswerBytes = 4 << 20
)

var (
	errNoQuestion     = errors.New("the request holds no message to answer")
	errAnswerTooLarge = errors.New("the answer of the program is too large")
)

// Listings reads the models an agent reports.
type Listings interface {
	GetListing(ctx context.Context, agent agents.Agent) acp.Listing
}

// LookPath finds a program on the machine. A nil LookPath finds none.
type LookPath func(program string) (string, error)

// Dependencies are what a Completion needs.
type Dependencies struct {
	Settings *settings.Settings
	Launcher acp.Launcher
	Listings Listings
	LookPath LookPath
}

// Completion serves the provider roles of the plugin.
type Completion struct {
	deps Dependencies
}

// New creates a Completion, and refuses dependencies that are missing.
func New(deps Dependencies) (*Completion, error) {
	if deps.Settings == nil || deps.Launcher == nil || deps.Listings == nil {
		return nil, errors.New("completion: every dependency is required")
	}
	return &Completion{deps: deps}, nil
}

// Complete answers the request in one session of the program the call names. The session runs in an
// empty temporary folder and is offered no file system, no terminal and no tool, and every permission
// request is refused. A program that cannot be found, and a session that outlasts the request timeout,
// are failures to try the next connection after, and so is a turn the program does not end. The text of a
// failed session does not leave the plugin.
func (c *Completion) Complete(
	ctx context.Context, call connection.Call, request *llm.CompleteRequest,
) (*llm.CompleteResult, error) {
	system, prompt, err := splitMessages(request.GetMessages())
	if err != nil {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	agent, err := c.getCallAgent(call)
	if err != nil {
		return nil, err
	}
	limited, cancel := context.WithTimeout(ctx, c.getRequestLimit())
	defer cancel()

	text, runErr := c.answer(limited, agent, call, []string{system, prompt})
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch {
	case limited.Err() != nil:
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	case runErr != nil, strings.TrimSpace(text) == "":
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	return &llm.CompleteResult{Text: text}, nil
}

func (c *Completion) answer(ctx context.Context, agent agents.Agent, call connection.Call, texts []string) (text string, err error) {
	session, err := acp.Open(ctx, c.deps.Launcher, acp.Start{Command: agent.Command, Servers: nil})
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, session.Close(ctx)) }()
	if err := session.SetModel(ctx, call.Options[llm.OptionModel]); err != nil {
		return "", err
	}
	if err := session.SetReasoning(ctx, call.Options[llm.OptionReasoning]); err != nil {
		return "", err
	}
	answer := &collector{text: strings.Builder{}}
	reason, err := session.Prompt(ctx, texts, answer)
	if err != nil {
		return "", err
	}
	if reason != acp.StopEndTurn && reason != acp.StopMaxTokens && reason != acp.StopMaxTurnRequests {
		return "", fmt.Errorf("the program ended the turn with %q", reason)
	}
	return strings.TrimSpace(answer.text.String()), nil
}

// collector keeps the text of a turn and refuses every tool.
type collector struct {
	text strings.Builder
}

func (c *collector) HandleUpdate(update acp.Update) error {
	if update.Chunk == nil {
		return nil
	}
	if c.text.Len()+len(update.Chunk.Text) > maxAnswerBytes {
		return errAnswerTooLarge
	}
	c.text.WriteString(update.Chunk.Text)
	return nil
}

func (*collector) IsAllowed(string) bool { return false }

// ListModels lists the models the program the call names reports.
func (c *Completion) ListModels(ctx context.Context, call connection.Call, _ []string) ([]connection.Option, error) {
	listing, err := c.readListing(ctx, call)
	if err != nil {
		return nil, err
	}
	models := make([]connection.Option, 0, len(listing.Models))
	for _, one := range listing.Models {
		title := one.Name
		if title == "" {
			title = one.ID
		}
		models = append(models, connection.Option{Value: one.ID, Title: getModelTitle(title)})
	}
	return models, nil
}

// ListReasoningLevels lists the levels of reasoning the program reports for the model it answers with.
// A model of the call other than that one is listed with none.
func (c *Completion) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.Option, error) {
	listing, err := c.readListing(ctx, call)
	if err != nil {
		return nil, err
	}
	if call.Options[llm.OptionModel] != "" && call.Options[llm.OptionModel] != listing.Current {
		return nil, nil
	}
	levels := make([]connection.Option, 0, len(listing.Levels))
	for _, one := range listing.Levels {
		levels = append(levels, connection.Option{Value: one.Name, Title: one.Title})
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
	agent, err := agents.Find(call.Type, call.Fields[fieldProgram], c.deps.LookPath)
	if errors.Is(err, agents.ErrNoProgram) {
		return agent, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	if err != nil {
		return agent, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	return agent, nil
}

func (c *Completion) getRequestLimit() time.Duration {
	if limit := c.deps.Settings.RequestTimeout(); limit > 0 {
		return limit
	}
	return defaultRequestLimit
}

func splitMessages(messages []*llm.Message) (system, prompt string, err error) {
	var instructions []string
	var turns []*llm.Message
	for _, message := range messages {
		switch message.GetRole() {
		case roleSystem:
			instructions = append(instructions, message.GetText())
		case "", roleUser, roleAssistant:
			turns = append(turns, message)
		default:
			return "", "", fmt.Errorf("unknown role %q", message.GetRole())
		}
	}
	if len(turns) == 0 {
		return "", "", errNoQuestion
	}
	return strings.Join(instructions, "\n\n"), writePrompt(turns), nil
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

// getModelTitle writes the provider and the model of a title such as "OpenRouter/DeepSeek R1" apart
// with a middle dot, as the other agents list them.
func getModelTitle(title string) string {
	provider, model, isFound := strings.Cut(title, "/")
	if !isFound || provider == "" || model == "" {
		return title
	}
	return provider + " · " + model
}

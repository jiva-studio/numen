package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

const (
	settingRequestTimeout = "requestTimeout"
	fieldProgram          = "program"
	defaultRequestLimit   = 120 * time.Second
)

var errNoQuestion = errors.New("the request holds no message to answer")

// Provider returns the roles the plugin fills for its connection type: complete and the lists of
// models and reasoning levels.
func (s *agentServer) Provider() connection.Provider {
	return connection.Provider{Completer: s, Embedder: nil, Models: s, Profiles: nil}
}

func (s *agentServer) getRequestLimit() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.requestTimeout > 0 {
		return s.requestTimeout
	}
	return defaultRequestLimit
}

func parseRequestTimeout(value string) time.Duration {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || seconds <= 0 || math.IsInf(seconds, 0) || seconds >= float64(math.MaxInt64/int64(time.Second)) {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func getCallAgent(call connection.Call) (acpAgent, error) {
	program := strings.TrimSpace(call.Fields[fieldProgram])
	if program == "" {
		return acpAgent{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	if _, err := exec.LookPath(program); err != nil {
		return acpAgent{}, connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	return buildCustomACPAgent(program, false), nil
}

// Complete answers the request in one session of the program the call names. The session runs in an empty
// temporary folder and is offered no file system, no terminal and no tool, and every permission
// request is refused. A program that cannot be found, and a session that outlasts the request timeout,
// are failures to try the next connection after. The text of a failed session does not leave the plugin.
func (s *agentServer) Complete(
	ctx context.Context, call connection.Call, request connection.ProviderCompleteRequest,
) (connection.CompleteResult, error) {
	system, prompt, err := splitMessages(request.Messages)
	if err != nil {
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	agent, err := getCallAgent(call)
	if err != nil {
		return connection.CompleteResult{}, err
	}
	limited, cancel := context.WithTimeout(ctx, s.getRequestLimit())
	defer cancel()

	text, runErr := s.answer(limited, agent, question{
		model: call.Model, reasoning: call.Reasoning, system: system, prompt: prompt,
		tools: toolEndpoint{url: "", token: "", tools: nil}, words: nil, report: nil,
	})
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

func splitMessages(messages []connection.Message) (system, prompt string, err error) {
	var instructions []string
	var turns []connection.Message
	for _, message := range messages {
		switch message.Role {
		case "system":
			instructions = append(instructions, message.Text)
		case "", "user", "assistant":
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
		if turn.Role == "assistant" {
			role = "Assistant"
		}
		parts = append(parts, role+":\n"+turn.Text)
	}
	return strings.Join(parts, "\n\n")
}

// ListModels lists the models the program the call names reports.
func (s *agentServer) ListModels(ctx context.Context, call connection.Call, _ string) ([]connection.Model, error) {
	listing, err := s.readCallListing(ctx, call)
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
func (s *agentServer) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.ReasoningLevel, error) {
	listing, err := s.readCallListing(ctx, call)
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

func (s *agentServer) readCallListing(ctx context.Context, call connection.Call) (acpListing, error) {
	agent, err := getCallAgent(call)
	if err != nil {
		return acpListing{}, err
	}
	return s.getACPListing(ctx, agent), nil
}

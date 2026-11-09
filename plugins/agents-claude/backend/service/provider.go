package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

const (
	settingRequestTimeout = "requestTimeout"
	defaultRequestLimit   = 120 * time.Second
	stoppedDone           = "done"
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

// Complete answers the request with one run of Claude Code that is given no tool and cannot write to disk.
// A program that is not installed, and a run that outlasts the request timeout, are failures to try
// the next connection after. The text of a failed run does not leave the plugin.
func (s *agentServer) Complete(
	ctx context.Context, call connection.Call, request connection.ProviderCompleteRequest,
) (connection.CompleteResult, error) {
	task, err := newCompletionTask(call, request)
	if err != nil {
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	if s.findClaude() == "" {
		return connection.CompleteResult{}, connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	limited, cancel := context.WithTimeout(ctx, s.getRequestLimit())
	defer cancel()

	var answer strings.Builder
	stopReason := ""
	runErr := s.Ask(limited, task, func(response *agentrun.Response) error {
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

func newCompletionTask(call connection.Call, request connection.ProviderCompleteRequest) (*agentrun.Request, error) {
	var system []string
	var turns []connection.Message
	for _, message := range request.Messages {
		switch message.Role {
		case "system":
			system = append(system, message.Text)
		case "", "user", "assistant":
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
		if turn.Role == "assistant" {
			role = "Assistant"
		}
		parts = append(parts, role+":\n"+turn.Text)
	}
	return strings.Join(parts, "\n\n")
}

// ListModels lists the models Claude Code offers.
func (s *agentServer) ListModels(ctx context.Context, _ connection.Call, _ string) ([]connection.Model, error) {
	programModels, err := s.getProgramModels(ctx)
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
func (s *agentServer) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.ReasoningLevel, error) {
	programModels, err := s.getProgramModels(ctx)
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

func (s *agentServer) getProgramModels(ctx context.Context) ([]*reasoning.Model, error) {
	if s.findClaude() == "" {
		return nil, connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	programs, err := s.GetPrograms(ctx, nil)
	if err != nil {
		return nil, connection.NewStopError(connection.ReasonUnknown)
	}
	var models []*reasoning.Model
	for _, one := range programs {
		models = append(models, one.Models...)
	}
	return models, nil
}

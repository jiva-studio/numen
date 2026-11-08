package service

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type reasoningBody struct {
	Effort string `json:"effort"`
}

type chatBody struct {
	Model               string         `json:"model"`
	Messages            []chatMessage  `json:"messages"`
	Temperature         *float64       `json:"temperature,omitempty"`
	MaxTokens           int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string         `json:"reasoning_effort,omitempty"`
	Reasoning           *reasoningBody `json:"reasoning,omitempty"`
}

type chatAnswer struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete asks the chat completions endpoint of the service for an answer.
func (s *Service) Complete(ctx context.Context, call connection.Call, request connection.ProviderCompleteRequest) (connection.CompleteResult, error) {
	if call.Reasoning != "" && !slices.ContainsFunc(getReasoningLevels(call.Type, call.Model), func(level connection.ReasoningLevel) bool {
		return level.Value == call.Reasoning
	}) {
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	messages := make([]chatMessage, len(request.Messages))
	for i, message := range request.Messages {
		messages[i] = chatMessage{Role: message.Role, Content: message.Text}
	}
	body := chatBody{Model: call.Model, Messages: messages, Temperature: request.Temperature, MaxTokens: 0, MaxCompletionTokens: 0, ReasoningEffort: "", Reasoning: nil}
	if call.Type == typeOpenAI {
		body.MaxCompletionTokens = request.MaxTokens
	} else {
		body.MaxTokens = request.MaxTokens
	}
	if call.Type == typeOpenRouter && call.Reasoning != "" {
		body.Reasoning = &reasoningBody{Effort: call.Reasoning}
	} else {
		body.ReasoningEffort = call.Reasoning
	}

	var answer chatAnswer
	if err := s.send(ctx, call, endpoint{operationComplete, http.MethodPost, "/chat/completions"}, body, &answer); err != nil {
		return connection.CompleteResult{}, err
	}
	if len(answer.Choices) == 0 {
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonUnknown)
	}
	s.markServed(call)
	return connection.CompleteResult{Text: answer.Choices[0].Message.Content}, nil
}

func getReasoningLevels(typeID, model string) []connection.ReasoningLevel {
	switch typeID {
	case typeOpenRouter:
		return listReasoningLevels()
	case typeOpenAI:
		name := strings.ToLower(strings.TrimSpace(model))
		for _, prefix := range []string{"o1", "o3", "o4", "gpt-5"} {
			if strings.HasPrefix(name, prefix) {
				return listReasoningLevels()
			}
		}
	}
	return nil
}

func listReasoningLevels() []connection.ReasoningLevel {
	return []connection.ReasoningLevel{{Value: "low", Title: "Low"}, {Value: "medium", Title: "Medium"}, {Value: "high", Title: "High"}}
}

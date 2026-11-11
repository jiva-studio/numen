package chatapi

import (
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
)

// reasoningRequest is the reasoning object of a request.
type reasoningRequest struct {
	Effort string `json:"effort"`
}

// chatRequest is the request body for the next message of a conversation.
type chatRequest struct {
	Model           string                      `json:"model"`
	Messages        []conversation.Message      `json:"messages"`
	Tools           []conversation.FunctionTool `json:"tools,omitempty"`
	Stream          bool                        `json:"stream,omitempty"`
	Temperature     *float64                    `json:"temperature,omitempty"`
	MaxTokens       *int                        `json:"max_tokens,omitempty"`
	ReasoningEffort string                      `json:"reasoning_effort,omitempty"`
	Reasoning       *reasoningRequest           `json:"reasoning,omitempty"`
}

func newChatRequest(cfg settings.Settings, turn conversation.Turn, isStreamed bool) chatRequest {
	request := chatRequest{
		Model: turn.Model, Messages: turn.Messages, Tools: turn.Tools, Stream: isStreamed,
		Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens, ReasoningEffort: "", Reasoning: nil,
	}
	switch turn.Effort.Field {
	case conversation.FieldReasoningEffort:
		request.ReasoningEffort = turn.Effort.Level
	case conversation.FieldReasoning:
		request.Reasoning = &reasoningRequest{Effort: turn.Effort.Level}
	}
	return request
}

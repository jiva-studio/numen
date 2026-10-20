package harness

import (
	"slices"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/reasoning"
)

// The fields through which a chat API takes the depth of reasoning of a model: the one of OpenAI, and
// the one of OpenRouter, which holds the depth beside the other settings of the reasoning.
const (
	fieldReasoningEffort = "reasoning_effort"
	fieldReasoning       = "reasoning"
)

// effortLevels are the depths of reasoning a chat API takes for a model that lists the field.
var effortLevels = reasoning.NewLevels("low", "medium", "high")

// reasoningRequest is the reasoning object of a request.
type reasoningRequest struct {
	Effort string `json:"effort"`
}

// effort is the depth of reasoning asked of the model, and the field it goes in. The zero value asks
// for none.
type effort struct {
	field string
	level string
}

// getReasoningField is the field the parameters a model lists accept the depth of reasoning in, empty
// where they accept none.
func getReasoningField(supported []string) string {
	switch {
	case slices.Contains(supported, fieldReasoningEffort):
		return fieldReasoningEffort
	case slices.Contains(supported, fieldReasoning):
		return fieldReasoning
	}
	return ""
}

// newEffort is the depth of reasoning asked of the model: the level if the model lists a field to take
// it and the level is one it offers, else none.
func newEffort(listed model, level string) effort {
	if listed.ReasoningField == "" || reasoning.Pick(effortLevels, level) == "" {
		return effort{}
	}
	return effort{field: listed.ReasoningField, level: level}
}

// newChatRequest is the request for the next message of a conversation.
func newChatRequest(model string, asked effort, messages []message, tools []functionTool, isStreamed bool) chatRequest {
	request := chatRequest{Model: model, Messages: messages, Tools: tools, Stream: isStreamed}
	switch asked.field {
	case fieldReasoningEffort:
		request.ReasoningEffort = asked.level
	case fieldReasoning:
		request.Reasoning = &reasoningRequest{Effort: asked.level}
	}
	return request
}

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

// effortNone is the entry of a list of depths that stands for reasoning switched off.
const effortNone = "none"

// commonLevels are the depths offered for a model whose listing names the field that takes the depth and
// no depths. OpenAI-compatible servers list the field and nothing more, so these are the depths the
// field is documented to take.
var commonLevels = reasoning.NewLevels("low", "medium", "high")

// reasoningListing is the reasoning object a chat API lists for a model: OpenRouter names the depths the
// model takes and the one it works at unasked.
type reasoningListing struct {
	SupportedEfforts []string `json:"supported_efforts"`
	DefaultEffort    string   `json:"default_effort"`
}

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

// newModel is the model a chat API lists, with the depths of reasoning it names for the model.
func newModel(name, title string, supported []string, listing *reasoningListing) model {
	one := model{Name: name, Title: title, ReasoningField: getReasoningField(supported)}
	if listed != nil {
		names := slices.DeleteFunc(slices.Clone(listed.SupportedEfforts), func(level string) bool { return level == effortNone })
		if len(names) > 0 {
			one.ReasoningLevels = reasoning.NewLevels(names...)
			reasoning.Sort(one.ReasoningLevels)
			if reasoning.Contains(one.ReasoningLevels, listed.DefaultEffort) {
				one.DefaultReasoning = listed.DefaultEffort
			}
			if one.ReasoningField == "" {
				one.ReasoningField = fieldReasoning
			}
			return one
		}
	}
	if one.ReasoningField != "" {
		one.ReasoningLevels = commonLevels
	}
	return one
}

// newEffort is the depth of reasoning asked of the model: the level if the model lists a field to take
// it and the level is one it offers, else none.
func newEffort(m model, level string) effort {
	if listed.ReasoningField == "" || reasoning.Pick(listed.ReasoningLevels, level) == "" {
		return effort{}
	}
	return effort{field: listed.ReasoningField, level: level}
}

// newChatRequest is the request for the next message of a conversation.
func newChatRequest(model string, reqEffort effort, messages []message, tools []functionTool, isStreamed bool) chatRequest {
	request := chatRequest{Model: model, Messages: messages, Tools: tools, Stream: isStreamed}
	switch asked.field {
	case fieldReasoningEffort:
		request.ReasoningEffort = asked.level
	case fieldReasoning:
		request.Reasoning = &reasoningRequest{Effort: asked.level}
	}
	return request
}

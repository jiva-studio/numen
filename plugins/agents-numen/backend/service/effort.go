package service

import (
	"slices"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
)

// The fields through which a chat API takes the depth of reasoning of a model: the one of OpenAI, and
// the one of OpenRouter, which holds the depth beside the other settings of the reasoning.
const (
	fieldReasoningEffort = "reasoning_effort"
	fieldReasoning       = "reasoning"
)

// effortNone is the entry in a list of depths that means reasoning is disabled.
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

// effort is the reasoning depth requested from the model, and the field it is sent in. The zero value
// requests none.
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

func parseListing(listing *reasoningListing, currentField string) ([]reasoning.Level, string, string, bool) {
	if listing == nil {
		return nil, "", "", false
	}
	names := slices.DeleteFunc(slices.Clone(listing.SupportedEfforts), func(level string) bool { return level == effortNone })
	if len(names) == 0 {
		return nil, "", "", false
	}
	levels := reasoning.NewLevels(names...)
	reasoning.Sort(levels)
	defaultEffort := ""
	if reasoning.Contains(levels, listing.DefaultEffort) {
		defaultEffort = listing.DefaultEffort
	}
	field := currentField
	if field == "" {
		field = fieldReasoning
	}
	return levels, defaultEffort, field, true
}

// newModel is the model a chat API lists, with the depths of reasoning it names for the model.
func newModel(name, title string, supported []string, listing *reasoningListing) model {
	one := model{Name: name, Title: title, ReasoningField: getReasoningField(supported)}
	if levels, def, field, ok := parseListing(listing, one.ReasoningField); ok {
		one.ReasoningLevels = levels
		one.DefaultReasoning = def
		one.ReasoningField = field
		return one
	}
	if one.ReasoningField != "" {
		one.ReasoningLevels = commonLevels
	}
	return one
}

// newEffort returns the reasoning depth requested from the model: the level if the model lists a field to
// receive it and offers the level, else none.
func newEffort(m model, level string) effort {
	if m.ReasoningField == "" || reasoning.Pick(m.ReasoningLevels, level) == "" {
		return effort{}
	}
	return effort{field: m.ReasoningField, level: level}
}

// newChatRequest is the request for the next message of a conversation.
func newChatRequest(model string, reqEffort effort, messages []message, tools []functionTool, isStreamed bool) chatRequest {
	request := chatRequest{Model: model, Messages: messages, Tools: tools, Stream: isStreamed}
	switch reqEffort.field {
	case fieldReasoningEffort:
		request.ReasoningEffort = reqEffort.level
	case fieldReasoning:
		request.Reasoning = &reasoningRequest{Effort: reqEffort.level}
	}
	return request
}

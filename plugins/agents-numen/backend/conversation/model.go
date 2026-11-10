package conversation

import (
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
)

// The fields through which a chat API takes the depth of reasoning of a model: the one of OpenAI, and
// the one of OpenRouter, which holds the depth beside the other settings of the reasoning.
const (
	FieldReasoningEffort = "reasoning_effort"
	FieldReasoning       = "reasoning"
)

// effortNone is the entry in a list of depths that means reasoning is disabled.
const effortNone = "none"

// Model is one model the chat API lists.
type Model struct {
	Name             string
	Title            string
	ReasoningField   string
	ReasoningLevels  []reasoning.Level
	DefaultReasoning string
}

// ReasoningListing is the reasoning object a chat API lists for a model: OpenRouter names the depths the
// model takes and the one it works at unasked.
type ReasoningListing struct {
	SupportedEfforts []string
	DefaultEffort    string
}

// UnmarshalJSON reads the reasoning object in the keys the chat API writes it with.
func (l *ReasoningListing) UnmarshalJSON(data []byte) error {
	raw, err := readFields(data)
	if err != nil {
		return err
	}
	read := ReasoningListing{SupportedEfforts: nil, DefaultEffort: ""}
	err = errors.Join(
		readField(raw, "supported_efforts", &read.SupportedEfforts),
		readField(raw, "default_effort", &read.DefaultEffort),
	)
	if err != nil {
		return err
	}
	*l = read
	return nil
}

// GetCommonLevels returns the depths offered for a model that takes a reasoning field and names none.
func GetCommonLevels() []reasoning.Level {
	return reasoning.NewLevels("low", "medium", "high")
}

// NewModel returns the model with the depths its listing names; a model that lists a reasoning
// field and no depths takes the common ones.
func NewModel(id, label string, supported []string, listing *ReasoningListing) Model {
	one := Model{
		Name: id, Title: label, ReasoningField: getReasoningField(supported),
		ReasoningLevels: nil, DefaultReasoning: "",
	}
	if levels, def, field, ok := parseListing(listing, one.ReasoningField); ok {
		one.ReasoningLevels = levels
		one.DefaultReasoning = def
		one.ReasoningField = field
		return one
	}
	if one.ReasoningField != "" {
		one.ReasoningLevels = GetCommonLevels()
	}
	return one
}

func getReasoningField(supported []string) string {
	switch {
	case slices.Contains(supported, FieldReasoningEffort):
		return FieldReasoningEffort
	case slices.Contains(supported, FieldReasoning):
		return FieldReasoning
	}
	return ""
}

func parseListing(listing *ReasoningListing, currentField string) ([]reasoning.Level, string, string, bool) {
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
		field = FieldReasoning
	}
	return levels, defaultEffort, field, true
}

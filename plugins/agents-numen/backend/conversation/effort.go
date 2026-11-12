package conversation

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"

// Effort is the reasoning depth requested from the model, and the field it is sent in. The zero value
// requests none.
type Effort struct {
	Field string
	Level string
}

// NewEffort returns the request for the depth on the model, or none where the model takes no
// reasoning field or does not name the depth.
func NewEffort(m Model, depth string) Effort {
	if m.ReasoningField == "" || reasoning.Pick(m.ReasoningLevels, depth) == "" {
		return Effort{Field: "", Level: ""}
	}
	return Effort{Field: m.ReasoningField, Level: depth}
}

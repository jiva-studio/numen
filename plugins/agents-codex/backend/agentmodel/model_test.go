package agentmodel

import (
	"slices"
	"testing"
)

func TestAModelIgnoresEmptyAndRepeatedEffortNames(t *testing.T) {
	model := New("m", "M", true, []string{"", "high", "low", "high"}, "high")

	var names []string
	for _, one := range model.Reasoning {
		names = append(names, one.Name)
	}
	if !slices.Equal(names, []string{"low", "high"}) {
		t.Errorf("the levels are %v, want [low high]", names)
	}
	if model.DefaultReasoning != "high" || !model.IsDefault || model.Name != "m" || model.Title != "M" {
		t.Errorf("the model is %+v", model)
	}
}

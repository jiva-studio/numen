package service

import (
	"strings"
	"testing"
)

func TestTheInstructionsTellTheAgentHowToShowAPlace(t *testing.T) {
	for _, want := range []string{"window_tabs_get", "source_focus"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("the instructions do not name %s: %q", want, instructions)
		}
	}
	if strings.Contains(instructions, "\n\n") {
		t.Errorf("the instructions are more than one paragraph: %q", instructions)
	}
}

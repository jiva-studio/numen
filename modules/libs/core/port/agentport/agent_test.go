package agentport_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port/agentport"
)

// The default program is an installed one where there is one: the installed one marked so, else the
// first installed, and only then what is not installed.
func TestTheDefaultProgramIsAnInstalledOneWhereThereIsOne(t *testing.T) {
	for name, one := range map[string]struct {
		programs []agentport.AgentProgram
		want     string
	}{
		"installed and marked":    {[]agentport.AgentProgram{{Name: "a", IsInstalled: true}, {Name: "b", IsInstalled: true, IsDefault: true}}, "b"},
		"marked but not there":    {[]agentport.AgentProgram{{Name: "a", IsDefault: true}, {Name: "b", IsInstalled: true}}, "b"},
		"nothing installed":       {[]agentport.AgentProgram{{Name: "a"}, {Name: "b", IsDefault: true}}, "b"},
		"nothing marked or there": {[]agentport.AgentProgram{{Name: "a"}, {Name: "b"}}, "a"},
	} {
		got, found := agentport.GetDefaultProgram(one.programs)
		if !found || got.Name != one.want {
			t.Errorf("%s: got %q, %v, want %q", name, got.Name, found, one.want)
		}
	}
	if _, found := agentport.GetDefaultProgram(nil); found {
		t.Error("a program was found where none is listed")
	}
}

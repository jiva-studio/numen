package container_test

import (
	"strings"
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Each plugin that answers completions is listed with the models it says
// Complete answers with, and not with the models of the programs it runs.
func TestAnAgentIsListedWithTheModelsItCompletesWith(t *testing.T) {
	one := &providerServer{
		programs:         []*v1.AgentProgram{{Name: "other", Models: []*v1.AgentModel{{Name: "program-model"}}}},
		completionModels: []*v1.CompletionModel{{Name: "fast", Title: "Fast", IsDefault: true}, {Name: "deep", Title: "Deep"}},
	}
	host := hostWith(t,
		map[string][]string{
			providerOne:      {"ai.completion.provider", "ai.agent.provider"},
			providerTwo:      {"ai.completion.provider"},
			"md.test.chatty": {"agent"},
		},
		map[string]*providerServer{providerOne: one})
	var reported []error
	cfg := container.Config{PluginHost: host, ErrorHandler: func(err error) { reported = append(reported, err) }}

	byID := map[string]port.PluginInfo{}
	for _, info := range cfg.Plugins()() {
		byID[info.ID] = info
	}

	want := []port.AgentModel{{Name: "fast", Title: "Fast", IsDefault: true}, {Name: "deep", Title: "Deep"}}
	got := byID[providerOne].Models
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the agent is listed with %+v, want %+v", got, want)
	}
	if len(byID[providerTwo].Models) != 0 || len(byID["md.test.chatty"].Models) != 0 {
		t.Errorf("a plugin that cannot answer, or answers no completions, lists %+v and %+v",
			byID[providerTwo].Models, byID["md.test.chatty"].Models)
	}
	if len(reported) != 1 || !strings.Contains(reported[0].Error(), providerTwo) {
		t.Errorf("reported %v, want the agent that cannot start", reported)
	}
}

package plugin_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type fakeConnectionHost struct {
	plugins []manifest.Discovered
	enabled map[string]bool
}

func (h fakeConnectionHost) GetPlugins() []manifest.Discovered { return h.plugins }

func (h fakeConnectionHost) GetPlugin(id string) (manifest.Discovered, bool) {
	for _, one := range h.plugins {
		if one.Manifest.ID == id && h.enabled[id] {
			return one, true
		}
	}
	return manifest.Discovered{}, false
}

func connectionProvider(id, name string, types ...string) manifest.Discovered {
	var declared []manifest.ConnectionType
	for _, one := range types {
		declared = append(declared, manifest.ConnectionType{ID: one, Title: one, Operations: []string{"llm.complete"}})
	}
	return manifest.Discovered{Manifest: manifest.Manifest{
		ID: id, Name: name, Contributes: manifest.Contributes{ConnectionTypes: declared},
	}}
}

func TestTheRegistryListsTheTypesOfEveryEnabledPluginWithComposedIDs(t *testing.T) {
	registry := plugin.NewConnectionTypes(fakeConnectionHost{
		plugins: []manifest.Discovered{
			connectionProvider("md.numen.agents.acp", "Claude Code", "claude-code"),
			connectionProvider("md.numen.connections.llm", "LLM connections", "openrouter", "openai"),
			connectionProvider("md.numen.tools.none", "No types"),
		},
		enabled: map[string]bool{"md.numen.agents.acp": true, "md.numen.connections.llm": true, "md.numen.tools.none": true},
	})

	types := registry.ListTypes()

	var ids []string
	for _, one := range types {
		ids = append(ids, one.GetID())
	}
	want := []string{"md.numen.agents.acp/claude-code", "md.numen.connections.llm/openrouter", "md.numen.connections.llm/openai"}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	for index := range want {
		if ids[index] != want[index] {
			t.Errorf("ids = %v, want %v", ids, want)
		}
	}
	if types[1].PluginName != "LLM connections" || !types[1].IsAvailable {
		t.Errorf("type = %+v", types[1])
	}
}

func TestTheRegistryFindsATypeByItsComposedID(t *testing.T) {
	registry := plugin.NewConnectionTypes(fakeConnectionHost{
		plugins: []manifest.Discovered{connectionProvider("md.numen.connections.llm", "LLM", "openrouter")},
		enabled: map[string]bool{"md.numen.connections.llm": true},
	})

	found, ok := registry.GetType("md.numen.connections.llm/openrouter")
	if !ok || found.Declaration.ID != "openrouter" || found.PluginID != "md.numen.connections.llm" {
		t.Errorf("found = %+v, %v", found, ok)
	}
	for _, missing := range []string{"md.numen.connections.llm/other", "other.plugin/openrouter", "openrouter", ""} {
		if _, ok := registry.GetType(missing); ok {
			t.Errorf("%q was found", missing)
		}
	}
}

func TestTheRegistryDoesNotListTheTypesOfADisabledPlugin(t *testing.T) {
	registry := plugin.NewConnectionTypes(fakeConnectionHost{
		plugins: []manifest.Discovered{connectionProvider("md.numen.connections.llm", "LLM", "openrouter")},
		enabled: map[string]bool{},
	})

	if types := registry.ListTypes(); len(types) != 0 {
		t.Errorf("types = %+v", types)
	}
	if _, ok := registry.GetType("md.numen.connections.llm/openrouter"); ok {
		t.Error("the type of a disabled plugin was found")
	}
}

func titling(id string, operations ...manifest.Operation) manifest.Discovered {
	return manifest.Discovered{Manifest: manifest.Manifest{ID: id, Name: id, Contributes: manifest.Contributes{Operations: operations}}}
}

// The first title of an operation in the order of the plugin ids is the one listed, and a disabled
// plugin titles nothing.
func TestTheRegistryListsTheFirstTitleOfEachOperation(t *testing.T) {
	registry := plugin.NewConnectionTypes(fakeConnectionHost{
		plugins: []manifest.Discovered{
			titling("md.numen.agents.acp", manifest.Operation{Name: "acp.task", Title: "run agent tasks"}),
			titling("md.numen.agents.numen", manifest.Operation{Name: "acp.task", Title: "run tasks"}),
			titling("md.numen.connections.llm",
				manifest.Operation{Name: "llm.complete", Title: "complete text"}, manifest.Operation{Name: "llm.chat", Title: "chat"}),
			titling("md.numen.tools.off", manifest.Operation{Name: "tools.off", Title: "switch off"}),
		},
		enabled: map[string]bool{"md.numen.agents.acp": true, "md.numen.agents.numen": true, "md.numen.connections.llm": true},
	})

	got := registry.ListOperationTitles()

	want := []manifest.Operation{
		{Name: "acp.task", Title: "run agent tasks"}, {Name: "llm.complete", Title: "complete text"}, {Name: "llm.chat", Title: "chat"},
	}
	if len(got) != len(want) {
		t.Fatalf("titles = %+v, want %+v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("titles = %+v, want %+v", got, want)
		}
	}
}

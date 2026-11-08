package connectiontypes_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/connectiontypes"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

type fakeHost struct {
	plugins []plugincontract.DiscoveredPlugin
	enabled map[string]bool
}

func (h fakeHost) GetPluginInfos() []plugincontract.PluginInfo {
	var infos []plugincontract.PluginInfo
	for _, one := range h.plugins {
		infos = append(infos, plugincontract.PluginInfo{ID: one.Manifest.ID, Name: one.Manifest.Name})
	}
	return infos
}

func (h fakeHost) GetPlugin(id string) (plugincontract.DiscoveredPlugin, bool) {
	for _, one := range h.plugins {
		if one.Manifest.ID == id && h.enabled[id] {
			return one, true
		}
	}
	return plugincontract.DiscoveredPlugin{}, false
}

func provider(id, name string, types ...string) plugincontract.DiscoveredPlugin {
	var declared []manifest.ConnectionType
	for _, one := range types {
		declared = append(declared, manifest.ConnectionType{ID: one, Title: one, Operations: []string{"complete"}})
	}
	return plugincontract.DiscoveredPlugin{Manifest: manifest.Manifest{
		ID: id, Name: name, Contributes: manifest.Contributes{ConnectionTypes: declared},
	}}
}

func TestTheRegistryListsTheTypesOfEveryEnabledPluginWithComposedIDs(t *testing.T) {
	registry := connectiontypes.NewRegistry(fakeHost{
		plugins: []plugincontract.DiscoveredPlugin{
			provider("md.numen.agents.claude", "Claude Code", "claude-code"),
			provider("md.numen.connections.llm", "LLM connections", "openrouter", "openai"),
			provider("md.numen.tools.none", "No types"),
		},
		enabled: map[string]bool{"md.numen.agents.claude": true, "md.numen.connections.llm": true, "md.numen.tools.none": true},
	})

	types := registry.ListTypes()

	var ids []string
	for _, one := range types {
		ids = append(ids, one.GetID())
	}
	want := []string{"md.numen.agents.claude/claude-code", "md.numen.connections.llm/openrouter", "md.numen.connections.llm/openai"}
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
	registry := connectiontypes.NewRegistry(fakeHost{
		plugins: []plugincontract.DiscoveredPlugin{provider("md.numen.connections.llm", "LLM", "openrouter")},
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
	registry := connectiontypes.NewRegistry(fakeHost{
		plugins: []plugincontract.DiscoveredPlugin{provider("md.numen.connections.llm", "LLM", "openrouter")},
		enabled: map[string]bool{},
	})

	if types := registry.ListTypes(); len(types) != 0 {
		t.Errorf("types = %+v", types)
	}
	if _, ok := registry.GetType("md.numen.connections.llm/openrouter"); ok {
		t.Error("the type of a disabled plugin was found")
	}
}

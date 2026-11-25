package pluginmanifest

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/searchfunction"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func shelfFunctions(names ...string) searchfunction.Set {
	declared := searchfunction.Set{Version: 1}
	for _, name := range names {
		declared.Functions = append(declared.Functions, searchfunction.Function{
			Name: name, Level: searchfunction.LevelDocument, Description: name, Example: name + "()",
		})
	}
	return declared
}

func TestListSearchFunctions_ReadsTheManifestsOfEnabledUncontestedPlugins(t *testing.T) {
	host := &manifestHost{
		plugins: []manifest.Discovered{
			{Manifest: manifest.Manifest{ID: "org.example.shelf", Contributes: manifest.Contributes{SearchFunctions: shelfFunctions("shelved")}}},
			{Manifest: manifest.Manifest{ID: "org.example.off", Contributes: manifest.Contributes{SearchFunctions: shelfFunctions("hidden")}}},
			{Manifest: manifest.Manifest{ID: "org.example.copy", Contributes: manifest.Contributes{SearchFunctions: shelfFunctions("boxed", "shelved")}}},
		},
		disabled: map[string]bool{"org.example.off": true},
	}

	functions := New(host).ListSearchFunctions()

	if len(functions) != 1 || functions[0].Name != "shelved" || functions[0].PluginID != "org.example.shelf" {
		t.Errorf("got %+v, want shelved of the shelf plugin only", functions)
	}
}

package manifest_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestGetIdentityTellsRequirementsOfOneIDApart(t *testing.T) {
	base := manifest.DependencyRequirement{ID: "m", Kind: manifest.DependencyKindModel, Version: "1", SourceURL: "https://a/m.bin", PluginDir: "/p"}
	others := map[string]manifest.DependencyRequirement{
		"version":   {ID: "m", Kind: manifest.DependencyKindModel, Version: "2", SourceURL: "https://a/m.bin", PluginDir: "/p"},
		"source":    {ID: "m", Kind: manifest.DependencyKindModel, Version: "1", SourceURL: "https://b/m.bin", PluginDir: "/p"},
		"directory": {ID: "m", Kind: manifest.DependencyKindModel, Version: "1", SourceURL: "https://a/m.bin", PluginDir: "/q"},
		"kind":      {ID: "m", Kind: manifest.DependencyKindDataset, Version: "1", SourceURL: "https://a/m.bin", PluginDir: "/p"},
	}
	for name, other := range others {
		if other.GetIdentity() == base.GetIdentity() {
			t.Errorf("a requirement that differs in %s has the identity of the base", name)
		}
	}
}

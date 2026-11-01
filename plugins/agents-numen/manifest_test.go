package agent

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

// The model setting suggests models, and the model it defaults to is one of them.
func TestManifest_ModelSettingSuggestsItsDefault(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)
	for _, one := range manifest.Settings {
		if one.Key != "model" {
			continue
		}
		var values []string
		for _, option := range one.Options {
			values = append(values, option.Value)
		}
		if defaultModel, isText := one.Default.(string); !isText || !slices.Contains(values, defaultModel) {
			t.Errorf("the model setting suggests %v, which lack its default %q", values, one.Default)
		}
		return
	}
	t.Error("the plugin declares no model setting")
}

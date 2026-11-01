package agent

import (
	"encoding/json"
	"slices"
	"testing"
)

// The model setting suggests models, and the model it defaults to is one of them.
func TestManifest_ModelSettingSuggestsItsDefault(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key     string   `json:"key"`
			Default any      `json:"default"`
			Options []string `json:"options"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Settings {
		if one.Key == "model" {
			if defaultModel, isText := one.Default.(string); !isText || !slices.Contains(one.Options, defaultModel) {
				t.Errorf("the model setting suggests %v, which lack its default %q", one.Options, one.Default)
			}
			return
		}
	}
	t.Error("the plugin declares no model setting")
}

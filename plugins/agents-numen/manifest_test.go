package agent

import (
	"encoding/json"
	"slices"
	"testing"
)

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

// The step limit is a setting of the plugin, 25 requests by default.
func TestManifest_DeclaresTheStepLimit(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key     string `json:"key"`
			Type    string `json:"type"`
			Default any    `json:"default"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Settings {
		if one.Key == "maxSteps" {
			if one.Type != "number" || one.Default != float64(25) {
				t.Errorf("maxSteps is %s with default %v", one.Type, one.Default)
			}
			return
		}
	}
	t.Error("the plugin declares no maxSteps setting")
}

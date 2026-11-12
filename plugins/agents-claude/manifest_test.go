package agentclaude

import (
	"encoding/json"
	"testing"
)

// The option that lets the program read the hooks and skills of the person is a setting of the plugin,
// off by default.
func TestManifestDeclaresTheHooksAndSkillsSetting(t *testing.T) {
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
		if one.Key != "readsHooksAndSkills" {
			continue
		}
		if one.Type != "boolean" || one.Default != false {
			t.Errorf("the setting is a %s defaulting to %v, want a boolean defaulting to false", one.Type, one.Default)
		}
		return
	}
	t.Error("the plugin declares no readsHooksAndSkills setting")
}

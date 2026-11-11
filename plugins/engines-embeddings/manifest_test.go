package embeddings

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

// The plugin declares the ONNX Runtime as a runtime dependency.
func TestManifestDeclaresONNXRuntimeDependency(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID        string `json:"id"`
			Kind      string `json:"kind"`
			SourceURL string `json:"sourceUrl"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Dependencies) != 1 || manifest.Dependencies[0].ID != "onnxruntime" || manifest.Dependencies[0].Kind != "runtime" || manifest.Dependencies[0].SourceURL != "" {
		t.Errorf("dependencies = %+v, want one onnxruntime runtime", manifest.Dependencies)
	}
}

// The core finds the provider by its capability; no window calls the service.
func TestManifestDeclaresTheEmbeddingCapabilityAndNoWindowService(t *testing.T) {
	var manifest struct {
		Capabilities     []string `json:"capabilities"`
		ActivationEvents []string `json:"activationEvents"`
		Services         []string `json:"services"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(manifest.Capabilities, "embedding") {
		t.Errorf("capabilities = %v, want embedding", manifest.Capabilities)
	}
	if len(manifest.ActivationEvents) != 0 {
		t.Errorf("activationEvents = %v, want none", manifest.ActivationEvents)
	}
	if len(manifest.Services) != 0 {
		t.Errorf("serves %v to the windows, want nothing", manifest.Services)
	}
}

// The plugin reports the model and a starting floor for it and draws nothing: the search floor is
// measured and set by another plugin.
func TestManifestDeclaresNoFrontendAndNoSettingsView(t *testing.T) {
	var manifest struct {
		Frontend      map[string]any   `json:"frontend"`
		SettingsViews []map[string]any `json:"settingsViews"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Frontend) != 0 || len(manifest.SettingsViews) != 0 {
		t.Errorf("frontend %v and settingsViews %v, want neither", manifest.Frontend, manifest.SettingsViews)
	}
}

// The smallest piece of text to index is a whole number from 0 to 1000 that starts at 40, and the
// settings page draws it as it draws any number.
func TestManifestDeclaresTheSmallestPieceToIndex(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key         string   `json:"key"`
			Type        string   `json:"type"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Default     any      `json:"default"`
			Min         *float64 `json:"min"`
			Max         *float64 `json:"max"`
			Step        *float64 `json:"step"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Settings {
		if one.Key != "minChunkLength" {
			continue
		}
		if one.Type != "number" || one.Default != float64(40) || one.Min == nil || *one.Min != 0 ||
			one.Max == nil || *one.Max != 1000 || one.Step == nil || *one.Step != 1 {
			t.Errorf("setting = %+v, want a number from 0 to 1000 by 1 that starts at 40", one)
		}
		if one.Title == "" || one.Description == "" {
			t.Errorf("setting has no title or description: %+v", one)
		}
		return
	}
	t.Error("the manifest declares no minChunkLength setting")
}

type manifestOption struct {
	Value  string `json:"value"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// The local models of the settings are the curated profiles of the plugin, in the order of the table,
// and the model setting defaults to the default model. Edit manifest.json to match backend/domain/model/profiles.json.
func TestManifestListsTheCuratedProfilesAndDefaultsToTheDefaultModel(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key       string `json:"key"`
			Default   any    `json:"default"`
			OptionsBy struct {
				Options map[string][]manifestOption `json:"options"`
			} `json:"optionsBy"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	profiles, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	var want []manifestOption
	for _, p := range profiles.GetCurated() {
		want = append(want, manifestOption{Value: p.ID, Title: p.GetOptionTitle(), Detail: p.GetOptionDetail()})
	}
	for _, setting := range manifest.Settings {
		if setting.Key != "model" {
			continue
		}
		if setting.Default != any(model.DefaultModel) {
			t.Errorf("model defaults to %q, want %q", setting.Default, model.DefaultModel)
		}
		if got := setting.OptionsBy.Options["local"]; !slices.Equal(got, want) {
			t.Errorf("local models = %+v, want the curated profiles %+v", got, want)
		}
		return
	}
	t.Fatal("manifest has no model setting")
}

// The longest chunk of a note is a whole number from 16 to 8192 that starts at 256, and the settings
// page draws it as it draws any number.
func TestManifestDeclaresTheLongestChunkOfANote(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key         string   `json:"key"`
			Type        string   `json:"type"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Default     any      `json:"default"`
			Min         *float64 `json:"min"`
			Max         *float64 `json:"max"`
			Step        *float64 `json:"step"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Settings {
		if one.Key != "chunkTokens" {
			continue
		}
		if one.Type != "number" || one.Default != float64(256) || one.Min == nil || *one.Min != 16 ||
			one.Max == nil || *one.Max != 8192 || one.Step == nil || *one.Step != 1 {
			t.Errorf("setting = %+v, want a number from 16 to 8192 by 1 that starts at 256", one)
		}
		if one.Title == "" || one.Description == "" {
			t.Errorf("setting has no title or description: %+v", one)
		}
		return
	}
	t.Error("the manifest declares no chunkTokens setting")
}

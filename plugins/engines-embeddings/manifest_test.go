package embeddings

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

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

// The core finds the search source by its capability; no window calls the service.
func TestManifestDeclaresTheSearchSourceCapabilityAndNoWindowService(t *testing.T) {
	var manifest struct {
		Capabilities     []string `json:"capabilities"`
		ActivationEvents []string `json:"activationEvents"`
		Services         []string `json:"services"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(manifest.Capabilities, "search.source") {
		t.Errorf("capabilities = %v, want search.source", manifest.Capabilities)
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

type manifestCondition struct {
	Setting string   `json:"setting"`
	Is      []string `json:"is"`
}

type manifestSetting struct {
	Key         string             `json:"key"`
	Type        string             `json:"type"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Default     any                `json:"default"`
	Options     []manifestOption   `json:"options"`
	Operations  []string           `json:"operations"`
	Max         *float64           `json:"max"`
	When        *manifestCondition `json:"when"`
}

func readSettings(t *testing.T) map[string]manifestSetting {
	t.Helper()
	var manifest struct {
		Settings []manifestSetting `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	out := map[string]manifestSetting{}
	for _, one := range manifest.Settings {
		out[one.Key] = one
	}
	return out
}

// The local models of the settings are the curated profiles of the plugin, in the order of the table,
// and the model setting defaults to the default model. Edit manifest.json to match backend/domain/model/profiles.json.
func TestManifestListsTheCuratedProfilesAndDefaultsToTheDefaultModel(t *testing.T) {
	profiles, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	var want []manifestOption
	for _, p := range profiles.GetCurated() {
		want = append(want, manifestOption{Value: p.ID, Title: p.GetOptionTitle(), Detail: p.GetOptionDetail()})
	}
	localModel, ok := readSettings(t)["model"]
	if !ok {
		t.Fatal("manifest has no model setting")
	}
	if localModel.Default != any(model.DefaultModel) {
		t.Errorf("model defaults to %q, want %q", localModel.Default, model.DefaultModel)
	}
	if !slices.Equal(localModel.Options, want) {
		t.Errorf("local models = %+v, want the curated profiles %+v", localModel.Options, want)
	}
}

// The source is the local model unless a person chooses a connection, and the model of the local
// source is asked for only while that source is chosen.
func TestManifestOffersTheLocalModelOrAConnectionAsTheSource(t *testing.T) {
	settings := readSettings(t)
	source, ok := settings["source"]
	if !ok {
		t.Fatal("manifest has no source setting")
	}
	var values []string
	var titles []string
	for _, one := range source.Options {
		values, titles = append(values, one.Value), append(titles, one.Title)
	}
	if source.Type != "enum" || source.Title != "Source" || source.Default != any("local") ||
		!slices.Equal(values, []string{"local", "connection"}) || !slices.Equal(titles, []string{"Local model", "Connection"}) {
		t.Errorf("source = %+v, want the enum Source of Local model and Connection that starts at local", source)
	}
	if when := settings["model"].When; when == nil || when.Setting != "source" || !slices.Equal(when.Is, []string{"local"}) {
		t.Errorf("model is shown when %+v, want the source is local", when)
	}
}

// The connection of the vectors is one row for the operations llm.embed and llm.profile, shown while the
// source is a connection.
func TestManifestDeclaresOneConnectionRowForEmbed(t *testing.T) {
	rows, ok := readSettings(t)["connections"]
	if !ok {
		t.Fatal("manifest has no connections setting")
	}
	if rows.Type != "connections" || !slices.Equal(rows.Operations, []string{"llm.embed", "llm.profile"}) || rows.Max == nil || *rows.Max != 1 ||
		rows.When == nil || rows.When.Setting != "source" || !slices.Equal(rows.When.Is, []string{"connection"}) {
		t.Errorf("connections = %+v, want one row for llm.embed and llm.profile shown while the source is a connection", rows)
	}
}

// The plugin keeps no key, no address and no service of its own.
func TestManifestDeclaresNoKeyNoAddressAndNoProvider(t *testing.T) {
	var manifest struct {
		Secrets []json.RawMessage `json:"secrets"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Secrets) != 0 {
		t.Errorf("%d secrets declared, want none", len(manifest.Secrets))
	}
	settings := readSettings(t)
	for _, key := range []string{"provider", "baseUrl", "apiKey"} {
		if _, ok := settings[key]; ok {
			t.Errorf("manifest still declares the setting %s", key)
		}
	}
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

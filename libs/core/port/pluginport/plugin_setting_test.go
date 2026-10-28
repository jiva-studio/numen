package pluginport_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	_ "github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	. "github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// readTestManifest is a plugin with two models, the languages of each, a provider and the settings
// of its remote service.
func readTestManifest(t *testing.T) PluginManifest {
	t.Helper()
	m, err := ReadManifest(strings.NewReader(`{
		"id": "org.example.sample", "name": "Sample", "version": "1.0.0",
		"backend": {"type": "process", "entrypoint": "bin/sample"},
		"dependencies": [
			{"id": "shared", "kind": "model"},
			{"id": "small-weights", "kind": "model"},
			{"id": "large-weights", "kind": "model"}
		],
		"secrets": [{"name": "apiKey", "title": "API key", "when": {"setting": "provider", "is": ["remote"]}}],
		"settings": [
			{"key": "model", "type": "enum", "title": "Model", "default": "large", "options": [
				{"value": "large", "title": "Large", "dependencies": ["large-weights"]},
				{"value": "small", "title": "Small", "dependencies": ["small-weights"]}
			]},
			{"key": "language", "type": "enum", "title": "Language", "optionsBy": {"setting": "model", "options": {
				"large": [{"value": "auto", "title": "Detected by the model"}, {"value": "en", "title": "English"}],
				"small": [{"value": "en", "title": "English"}]
			}}},
			{"key": "provider", "type": "enum", "title": "Provider", "default": "local", "options": ["local", "remote"]},
			{"key": "url", "type": "string", "title": "URL", "default": "https://example.org", "when": {"setting": "provider", "is": ["remote"]}},
			{"key": "dictionary", "type": "file", "title": "Dictionary", "accept": [".txt"]}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// Options read from plain values and from objects alike.
func TestOptionsAreReadFromValuesAndFromObjects(t *testing.T) {
	m := readTestManifest(t)
	provider, _ := m.GetSetting("provider")
	if got := provider.Options[1]; got.Value != "remote" || got.GetTitle() != "remote" {
		t.Errorf("provider option = %+v, want the value remote as its own title", got)
	}
	model, _ := m.GetSetting("model")
	if got := model.Options[1]; got.Value != "small" || got.GetTitle() != "Small" || !slices.Equal(got.Dependencies, []string{"small-weights"}) {
		t.Errorf("model option = %+v", got)
	}
}

// A setting whose options follow another holds its stored value while that value is one of them,
// and the first of them otherwise.
func TestASettingWhoseOptionsFollowAnotherFallsBackToTheirFirst(t *testing.T) {
	m := readTestManifest(t)
	for name, tc := range map[string]struct {
		stored map[string]any
		want   string
	}{
		"nothing stored":            {map[string]any{}, "auto"},
		"a value of the model":      {map[string]any{"language": "en"}, "en"},
		"the default of the other":  {map[string]any{"model": "small"}, "en"},
		"a value the model lacks":   {map[string]any{"model": "small", "language": "auto"}, "en"},
		"a value no model has":      {map[string]any{"language": "xx"}, "auto"},
		"a value of the wrong type": {map[string]any{"language": 3.0}, "auto"},
	} {
		if got := m.ResolveSettings(tc.stored)["language"]; got != tc.want {
			t.Errorf("%s: language = %v, want %s", name, got, tc.want)
		}
	}
}

func TestAStringSettingWithOptionsByFollowsProviderWhenStoredValueBelongsToAnotherProvider(t *testing.T) {
	m, err := ReadManifest(strings.NewReader(`{
		"id": "md.numen.engines.embeddings", "name": "Embeddings", "version": "1.0.0",
		"settings": [
			{"key": "provider", "type": "enum", "title": "Provider", "default": "local", "options": ["local", "openrouter"]},
			{"key": "model", "type": "string", "title": "Model", "default": "Xenova/bge-m3", "optionsBy": {
				"setting": "provider",
				"options": {
					"local": ["Xenova/bge-m3", "Xenova/multilingual-e5-small"],
					"openrouter": ["openai/text-embedding-3-small", "baai/bge-m3"]
				}
			}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		stored map[string]any
		want   string
	}{
		"default local":                {map[string]any{}, "Xenova/bge-m3"},
		"default openrouter":           {map[string]any{"provider": "openrouter"}, "openai/text-embedding-3-small"},
		"remote model under local":     {map[string]any{"provider": "local", "model": "openai/text-embedding-3-small"}, "Xenova/bge-m3"},
		"local model under openrouter": {map[string]any{"provider": "openrouter", "model": "Xenova/multilingual-e5-small"}, "openai/text-embedding-3-small"},
		"custom model under local":     {map[string]any{"provider": "local", "model": "org/custom-onnx"}, "org/custom-onnx"},
	} {
		if got := m.ResolveSettings(tc.stored)["model"]; got != tc.want {
			t.Errorf("%s: model = %v, want %s", name, got, tc.want)
		}
	}
}

// A setting whose condition fails is not handed over, and a stored value it holds stays stored.
func TestASettingWhoseConditionFailsIsLeftOut(t *testing.T) {
	m := readTestManifest(t)
	stored := map[string]any{"url": "https://mine.example.org", "enabled": true}
	values := m.ResolveSettings(stored)
	if _, ok := values["url"]; ok {
		t.Errorf("url is handed over under the local provider: %v", values)
	}
	if values["enabled"] != true {
		t.Errorf("a key the manifest does not declare is lost: %v", values)
	}
	if stored["url"] != "https://mine.example.org" {
		t.Errorf("the stored value changed: %v", stored)
	}
	stored["provider"] = "remote"
	if got := m.ResolveSettings(stored)["url"]; got != "https://mine.example.org" {
		t.Errorf("url under the remote provider = %v", got)
	}
	secret := m.Secrets[0]
	if m.IsSecretActive(secret, values) || !m.IsSecretActive(secret, m.ResolveSettings(stored)) {
		t.Errorf("the key follows the provider wrongly")
	}
}

// The host provisions the dependencies of the option in force, and those of no other option.
func TestTheDependenciesOfTheChosenOptionAreRequired(t *testing.T) {
	m := readTestManifest(t)
	for model, want := range map[string][]string{
		"":      {"shared", "large-weights"},
		"small": {"shared", "small-weights"},
	} {
		stored := map[string]any{}
		if model != "" {
			stored["model"] = model
		}
		var got []string
		for _, req := range m.GetDependencyRequirements("/p", m.ResolveSettings(stored)) {
			got = append(got, req.ID)
		}
		if !slices.Equal(got, want) {
			t.Errorf("model %q: requirements = %v, want %v", model, got, want)
		}
	}
}

// The validator names each link between settings that leads nowhere.
func TestTheValidatorRejectsLinksThatLeadNowhere(t *testing.T) {
	for name, settings := range map[string]string{
		"options follow nothing":     `{"key":"a","type":"enum","title":"A","optionsBy":{"setting":"b","options":{}}}`,
		"options follow a string":    `{"key":"b","type":"string","title":"B"},{"key":"a","type":"enum","title":"A","optionsBy":{"setting":"b","options":{}}}`,
		"a value with no options":    `{"key":"b","type":"enum","title":"B","options":["x","y"]},{"key":"a","type":"enum","title":"A","optionsBy":{"setting":"b","options":{"x":["1"]}}}`,
		"options follow a later":     `{"key":"a","type":"enum","title":"A","optionsBy":{"setting":"b","options":{"x":["1"]}}},{"key":"b","type":"enum","title":"B","options":["x"]}`,
		"a condition on nothing":     `{"key":"a","type":"string","title":"A","when":{"setting":"b","is":["x"]}}`,
		"a condition on no option":   `{"key":"b","type":"enum","title":"B","options":["x"]},{"key":"a","type":"string","title":"A","when":{"setting":"b","is":["y"]}}`,
		"a condition on no values":   `{"key":"b","type":"enum","title":"B","options":["x"]},{"key":"a","type":"string","title":"A","when":{"setting":"b","is":[]}}`,
		"an option of no dependency": `{"key":"a","type":"enum","title":"A","options":[{"value":"x","dependencies":["none"]}]}`,
		"extensions of a string":     `{"key":"a","type":"string","title":"A","accept":[".txt"]}`,
		"an extension with no dot":   `{"key":"a","type":"file","title":"A","accept":["txt"]}`,
	} {
		_, err := ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","settings":[` + settings + `]}`))
		if !errors.Is(err, ErrInvalidSetting) {
			t.Errorf("%s: err = %v, want ErrInvalidSetting", name, err)
		}
	}
}

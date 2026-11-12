package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	. "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readTestManifest(t *testing.T) Manifest {
	t.Helper()
	m, err := manifestjson.Read(strings.NewReader(`{
		"id": "org.example.sample", "name": "Sample", "version": "1.0.0",
		"backend": {"entrypoint": "bin/sample"},
		"dependencies": [
			{"id": "shared", "kind": "model"},
			{"id": "small-weights", "kind": "model"},
			{"id": "large-weights", "kind": "model"}
		],
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

// A setting whose options follow another keeps its stored value while that
// value is one of them, and falls back to the first of them when it is not.
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

func TestResolveSettings_OptionsByFollowsProvider(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{
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
		"empty model under local":      {map[string]any{"provider": "local", "model": ""}, "Xenova/bge-m3"},
	} {
		if got := m.ResolveSettings(tc.stored)["model"]; got != tc.want {
			t.Errorf("%s: model = %v, want %s", name, got, tc.want)
		}
	}
}

// A setting whose condition fails is not passed, and a stored value it holds stays stored.
func TestASettingWhoseConditionFailsIsLeftOut(t *testing.T) {
	m := readTestManifest(t)
	stored := map[string]any{"url": "https://mine.example.org", "enabled": true}
	values := m.ResolveSettings(stored)
	if _, ok := values["url"]; ok {
		t.Errorf("url is passed under the local provider: %v", values)
	}
	if _, ok := values["dictionary"]; ok {
		t.Errorf("dictionary with no default should not be in values: %v", values)
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
			if req.PluginDir != "/p" {
				t.Errorf("PluginDir = %q, want /p", req.PluginDir)
			}
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
		"a condition on bad bool":    `{"key":"b","type":"boolean","title":"B"},{"key":"a","type":"string","title":"A","when":{"setting":"b","is":["yes"]}}`,
		"an option of no dependency": `{"key":"a","type":"enum","title":"A","options":[{"value":"x","dependencies":["none"]}]}`,
		"optionsBy on number":        `{"key":"b","type":"enum","title":"B","options":["x"]},{"key":"a","type":"number","title":"A","optionsBy":{"setting":"b","options":{"x":["1"]}}}`,
		"optionsBy with own options": `{"key":"b","type":"enum","title":"B","options":["x"]},{"key":"a","type":"enum","title":"A","options":["1"],"optionsBy":{"setting":"b","options":{"x":["1"]}}}`,
		"optionsBy on optionsBy": `{"key":"c","type":"enum","title":"C","options":["x"]},` +
			`{"key":"b","type":"enum","title":"B","optionsBy":{"setting":"c","options":{"x":["y"]}}},` +
			`{"key":"a","type":"enum","title":"A","optionsBy":{"setting":"b","options":{"y":["1"]}}}`,
		"extensions of a string":    `{"key":"a","type":"string","title":"A","accept":[".txt"]}`,
		"an extension with no dot":  `{"key":"a","type":"file","title":"A","accept":["txt"]}`,
		"a bare dot extension":      `{"key":"a","type":"file","title":"A","accept":["."]}`,
		"several paths of a string": `{"key":"a","type":"string","title":"A","multiple":true}`,
	} {
		_, err := manifestjson.Read(
			strings.NewReader(
				`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"dependencies":[{"id":"dep1","kind":"model"}],"settings":[` + settings + `]}`,
			),
		)
		if !errors.Is(err, ErrInvalidSetting) {
			t.Errorf("%s: err = %v, want ErrInvalidSetting", name, err)
		}
	}
}

func TestSettingAndManifestValidationEdgeCases(t *testing.T) {
	if got := FormatSettingValue(nil); got != "" {
		t.Errorf("FormatSettingValue(nil) = %q, want empty", got)
	}
	if got := FormatSettingValue("abc"); got != "abc" {
		t.Errorf("FormatSettingValue(abc) = %q, want abc", got)
	}
	if got := FormatSettingValue(true); got != "true" {
		t.Errorf("FormatSettingValue(true) = %q, want true", got)
	}
	if got := FormatSettingValue(1000000.5); got != "1000000.5" {
		t.Errorf("FormatSettingValue(1000000.5) = %q, want 1000000.5", got)
	}
	if got := FormatSettingValue(42); got != "42" {
		t.Errorf("FormatSettingValue(42) = %q, want 42", got)
	}

	m := readTestManifest(t)
	if got := m.ResolveSettings(nil); got["provider"] != "local" {
		t.Errorf("ResolveSettings(nil) = %v", got)
	}

	// Setting with OptionsBy that resolves to empty options and no default deletes stale value.
	emptyOptsManifest := Manifest{
		Settings: []Setting{
			{Key: "provider", Type: SettingTypeEnum, Default: "custom", Options: SettingOptions{{Value: "local"}, {Value: "custom"}}},
			{Key: "model", Type: SettingTypeString, OptionsBy: &OptionsBy{
				Setting: "provider",
				Options: map[string]SettingOptions{"local": {{Value: "m1", Dependencies: []string{"dep1"}}}},
			}},
		},
		Dependencies: []DependencyRequirement{{ID: "dep1", Kind: DependencyKindModel}},
	}
	resolved := emptyOptsManifest.ResolveSettings(map[string]any{"model": "m1"})
	if _, ok := resolved["model"]; ok {
		t.Errorf("expected stale model from another group to be deleted when active group has no options and no default: %v", resolved)
	}
	if reqs := emptyOptsManifest.GetDependencyRequirements("/p", map[string]any{"provider": "local", "model": "m1"}); len(reqs) != 1 || reqs[0].ID != "dep1" {
		t.Errorf("expected dep1 when m1 is active: %v", reqs)
	}
	if reqs := emptyOptsManifest.GetDependencyRequirements("/p", map[string]any{"provider": "custom"}); len(reqs) != 0 {
		t.Errorf("expected no deps when custom is active: %v", reqs)
	}

	// Cyclic When condition and missing parent setting in When condition.
	cyclic := Manifest{
		Settings: []Setting{
			{Key: "a", Type: SettingTypeString, Default: "x", When: &Condition{Setting: "a", Is: []string{"x"}}},
			{Key: "b", Type: SettingTypeString, Default: "y", When: &Condition{Setting: "missing", Is: []string{"x"}}},
		},
	}
	cyclicResolved := cyclic.ResolveSettings(map[string]any{"a": "x", "b": "y", "missing": "x"})
	if _, ok := cyclicResolved["a"]; ok {
		t.Errorf("cyclic setting should be inactive: %v", cyclicResolved)
	}
	if _, ok := cyclicResolved["b"]; ok {
		t.Errorf("setting with missing parent should be inactive: %v", cyclicResolved)
	}

	// Valid settings: boolean When with true/false, string When, string OptionsBy with partial keys, file with 2-char extension, range bounds.
	validJSON := `{
		"id": "a.b", "name": "A", "version": "1.0.0",
		"backend": {"entrypoint": "bin/a"},
		"dependencies": [{"id": "d1", "kind": "model"}],
		"settings": [
			{"key": "flag", "type": "boolean", "title": "Flag"},
			{"key": "str", "type": "string", "title": "Str", "when": {"setting": "flag", "is": ["true", "false"]}},
			{"key": "dependent", "type": "string", "title": "Dep", "when": {"setting": "str", "is": ["custom"]}},
			{"key": "prov", "type": "enum", "title": "Prov", "options": ["a", "b"]},
			{"key": "optStr", "type": "string", "title": "OptStr", "optionsBy": {"setting": "prov", "options": {"a": [{"value": "v1", "dependencies": ["d1"]}]}}},
			{"key": "r1", "type": "range", "title": "R1", "min": 10, "max": 10},
			{"key": "r2", "type": "range", "title": "R2", "min": 10},
			{"key": "r3", "type": "range", "title": "R3", "max": 10},
			{"key": "n1", "type": "number", "title": "N1", "min": 20, "max": 10},
			{"key": "f1", "type": "file", "title": "F1", "accept": [".a"], "multiple": true}
		]
	}`
	if _, err := manifestjson.Read(strings.NewReader(validJSON)); err != nil {
		t.Fatalf("valid manifest failed: %v", err)
	}

	// Manifest-level validation rules in package manifest.
	sum := strings.Repeat("a", 64)
	for name, tc := range map[string]struct {
		json string
		want error
	}{
		"empty id":              {`{"name":"A","version":"1.0.0"}`, ErrEmptyID},
		"invalid id":            {`{"id":"bad","name":"A","version":"1.0.0"}`, ErrInvalidID},
		"empty name":            {`{"id":"a.b","version":"1.0.0"}`, ErrEmptyName},
		"empty version":         {`{"id":"a.b","name":"A"}`, ErrEmptyVersion},
		"invalid version":       {`{"id":"a.b","name":"A","version":"1.0"}`, ErrInvalidVersion},
		"protocol negative":     {`{"id":"a.b","name":"A","version":"1.0.0","protocol":-1}`, ErrUnsupportedProtocol},
		"protocol too high":     {`{"id":"a.b","name":"A","version":"1.0.0","protocol":2}`, ErrUnsupportedProtocol},
		"invalid capability":    {`{"id":"a.b","name":"A","version":"1.0.0","capabilities":["none"]}`, ErrInvalidCapability},
		"invalid activation":    {`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"activationEvents":["bad"]}`, ErrInvalidActivationEvent},
		"wildcard activation":   {`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"activationEvents":["*"]}`, ErrInvalidActivationEvent},
		"blank entrypoint":      {`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":" "}}`, ErrInvalidBackend},
		"activation no backend": {`{"id":"a.b","name":"A","version":"1.0.0","activationEvents":["onStartup"]}`, ErrNoBackend},
		"services no backend":   {`{"id":"a.b","name":"A","version":"1.0.0","services":["a.b.S"]}`, ErrNoBackend},
		"runtime no backend":    {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"r","kind":"runtime"}]}`, ErrNoBackend},
		"invalid permission":    {`{"id":"a.b","name":"A","version":"1.0.0","permissions":["bad"]}`, ErrInvalidPermission},
		"dep empty id":          {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"kind":"model"}]}`, ErrInvalidDependency},
		"dep duplicate id":      {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"d","kind":"model"},{"id":"d","kind":"model"}]}`, ErrInvalidDependency},
		"dep unknown kind":      {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"d","kind":"bad"}]}`, ErrInvalidDependency},
		"dep checksum no url":   {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"d","kind":"model","checksumSha256":"` + sum + `"}]}`, ErrInvalidDependency},
		"dep http url": {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"d","kind":"model",` +
			`"sourceUrl":"http://example.com/a","checksumSha256":"` + sum + `"}]}`, ErrInvalidDependency},
		"dep bad checksum":        {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"d","kind":"model","sourceUrl":"https://example.com/a","checksumSha256":"bad"}]}`, ErrInvalidDependency},
		"service bad name":        {`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"services":["Bad"]}`, ErrInvalidService},
		"service duplicate":       {`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"services":["a.b.S","a.b.S"]}`, ErrInvalidService},
		"setting empty key":       {`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"","type":"string","title":"T"}]}`, ErrEmptySettingKey},
		"setting named enabled":   {`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"enabled","type":"boolean","title":"T"}]}`, ErrReservedSettingKey},
		"setting duplicate key":   {`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"string","title":"T"},{"key":"k","type":"string","title":"T"}]}`, ErrDuplicateSettingKey},
		"setting empty title":     {`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"string","title":" "}]}`, ErrEmptySettingTitle},
		"setting bad type":        {`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"bad","title":"T"}]}`, ErrInvalidSettingType},
		"setting enum no options": {`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"enum","title":"T"}]}`, ErrEnumMissingOptions},
		"setting range min > max": {`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"range","title":"T","min":20,"max":10}]}`, ErrRangeInvalidBounds},
		"setting badoption dep": {`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"d1","kind":"model"}],` +
			`"settings":[{"key":"k","type":"enum","title":"T","options":[{"value":"v","dependencies":["d2"]}]}]}`, ErrInvalidSetting},
		"setting view no frontend": {`{"id":"a.b","name":"A","version":"1.0.0","settingsViews":[{"id":"v","title":"T"}]}`, ErrInvalidSettingsView},
	} {
		_, err := manifestjson.Read(strings.NewReader(tc.json))
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}

func readOptionWhenManifest(option string) error {
	_, err := manifestjson.Read(
		strings.NewReader(
			`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"settings":[{"key":"a","type":"enum","title":"A","options":["x",` + option + `]}]}`,
		),
	)
	return err
}

func TestAnOptionMayCarryACondition(t *testing.T) {
	if err := readOptionWhenManifest(`{"value":"y","when":"app.platform != 'darwin'"}`); err != nil {
		t.Fatal(err)
	}
	m, err := manifestjson.Read(
		strings.NewReader(
			`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"settings":[{"key":"a","type":"enum","title":"A","options":[{"value":"y","when":"app.platform == 'linux'"}]}]}`,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	setting, _ := m.GetSetting("a")
	if got := setting.Options[0].When; got != "app.platform == 'linux'" {
		t.Errorf("When = %q", got)
	}
}

func TestAnOptionConditionThatIsNotValidIsRejected(t *testing.T) {
	for _, text := range []string{"app.os == 'linux'", "app.platform =="} {
		err := readOptionWhenManifest(`{"value":"y","when":"` + text + `"}`)
		if !errors.Is(err, ErrInvalidSetting) || !strings.Contains(err.Error(), "when") {
			t.Errorf("%q: err = %v, want an invalid setting naming when", text, err)
		}
	}
}

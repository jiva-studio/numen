package manifestjson_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestReadManifest(t *testing.T) {
	raw := `{
		"id": "md.numen.engines.ocr",
		"name": "Document OCR",
		"version": "1.0.0",
		"description": "Optical character recognition",
		"capabilities": ["tools"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onStartup", "onVectorsReady", "onFile:pdf"],
		"permissions": ["vault:read"],
		"backend": {
			"entrypoint": "bin/engines.ocr"
		},
		"services": ["md.numen.engines.ocr"],
		"frontend": {},
		"settings": [
			{
				"key": "dpi",
				"type": "range",
				"title": "Scan DPI",
				"description": "Resolution",
				"default": 300,
				"palette": false,
				"options": [],
				"min": 150,
				"max": 600,
				"step": 50
			}
		]
	}`

	m, err := manifestjson.Read(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.ID != "md.numen.engines.ocr" {
		t.Errorf("got id %q, want md.numen.engines.ocr", m.ID)
	}
	if !m.CanTarget("desktop.editor") {
		t.Errorf("expected target desktop.editor to be accepted")
	}
	if m.CanTarget("desktop.samples") {
		t.Errorf("expected target desktop.samples to be rejected")
	}
	if !m.HasCapability("tools") {
		t.Errorf("expected capability tools")
	}
	if !m.IsActivatedBy("onStartup") || !m.IsActivatedBy("onVectorsReady:vault-1") || !m.IsActivatedBy("onFile:doc.pdf") || m.IsActivatedBy("onFile:doc.md") {
		t.Errorf("unexpected activation matching for %v", m.ActivationEvents)
	}
	if !m.HasService("md.numen.engines.ocr") || !m.HasFrontend() {
		t.Errorf("expected the service and the frontend")
	}
}

func TestReadManifestValidation(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{name: "missing id", json: `{"name":"A","version":"1.0.0"}`},
		{name: "missing name", json: `{"id":"a.b","version":"1.0.0"}`},
		{name: "missing version", json: `{"id":"a.b","name":"A"}`},
		{name: "invalid json", json: `{invalid}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(tc.json))
			if err == nil {
				t.Fatalf("expected error for case %s, got nil", tc.name)
			}
		})
	}
}

func TestReadManifestSettingsValidation(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{
			name: "empty setting key",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"","type":"boolean","title":"T"}]}`,
		},
		{
			name: "duplicate setting key",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"boolean","title":"T"},{"key":"k","type":"string","title":"T2"}]}`,
		},
		{
			name: "invalid setting type",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"unsupported","title":"T"}]}`,
		},
		{
			name: "enum without options",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"enum","title":"T","options":[]}]}`,
		},
		{
			name: "enum without options",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"enum","title":"T","options":[]}]}`,
		},
		{
			name: "enum missing options",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"enum","title":"T"}]}`,
		},
		{
			name: "range min greater than max",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"range","title":"T","min":100,"max":50}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(tc.json))
			if err == nil {
				t.Fatalf("expected error for case %s, got nil", tc.name)
			}
		})
	}
}

func TestReadManifestValidSettings(t *testing.T) {
	raw := `{
		"id": "md.numen.sample",
		"name": "Sample",
		"version": "1.0.0",
		"settings": [
			{"key": "verbose", "type": "boolean", "title": "Verbose", "default": true, "palette": true},
			{"key": "model", "type": "enum", "title": "Model", "default": "fast", "options": ["fast", "accurate"]},
			{"key": "retries", "type": "number", "title": "Retries", "default": 3},
			{"key": "zoom", "type": "range", "title": "Zoom", "default": 1.0, "min": 0.5, "max": 2.0, "step": 0.1},
			{"key": "prefix", "type": "string", "title": "Prefix", "default": "test"},
			{"key": "programs", "type": "file", "title": "Programs", "default": "", "multiple": true}
		]
	}`

	m, err := manifestjson.Read(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error reading manifest with settings: %v", err)
	}

	if len(m.Settings) != 6 {
		t.Fatalf("got %d settings, want 6", len(m.Settings))
	}
	if !m.Settings[0].Palette {
		t.Errorf("expected setting enabled to have palette: true")
	}
	if len(m.Settings[1].Options) != 2 {
		t.Errorf("expected enum setting to have 2 options")
	}
	if m.Settings[4].Multiple || !m.Settings[5].Multiple {
		t.Errorf("multiple = %v, %v, want false for the string and true for the file setting", m.Settings[4].Multiple, m.Settings[5].Multiple)
	}
}

func TestManifestRejectsASettingTypeTheFormDoesNotDraw(t *testing.T) {
	for _, kind := range []string{"select", "folder"} {
		raw := `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"mode","type":"` + kind + `","title":"Mode"}]}`
		if _, err := manifestjson.Read(strings.NewReader(raw)); !errors.Is(err, manifest.ErrInvalidSettingType) {
			t.Errorf("type %q: error = %v, want ErrInvalidSettingType", kind, err)
		}
	}
}

func TestManifestRulesReportTheirSentinel(t *testing.T) {
	cases := []struct {
		name string
		json string
		want error
	}{
		{"id with a space", `{"id":"bad id","name":"A","version":"1.0.0"}`, manifest.ErrInvalidID},
		{"version without patch", `{"id":"a.b","name":"A","version":"1.0"}`, manifest.ErrInvalidVersion},
		{"version with leading zero", `{"id":"a.b","name":"A","version":"01.0.0"}`, manifest.ErrInvalidVersion},
		{"prerelease with leading zero", `{"id":"a.b","name":"A","version":"1.0.0-01"}`, manifest.ErrInvalidVersion},
		{"id of one segment", `{"id":"foo","name":"A","version":"1.0.0"}`, manifest.ErrInvalidID},
		{"protocol newer than the host", `{"id":"a.b","name":"A","version":"1.0.0","protocol":2}`, manifest.ErrUnsupportedProtocol},
		{"negative protocol", `{"id":"a.b","name":"A","version":"1.0.0","protocol":-1}`, manifest.ErrUnsupportedProtocol},
		{"backend with a blank entrypoint", `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":" "}}`, manifest.ErrInvalidBackend},
		{
			"the wildcard activation event",
			`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"activationEvents":["*"]}`,
			manifest.ErrInvalidActivationEvent,
		},
		{
			"an activation event with an empty extension",
			`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"activationEvents":["onFile:"]}`,
			manifest.ErrInvalidActivationEvent,
		},
		{
			"an activation event with a wildcard extension",
			`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"activationEvents":["onFile:*"]}`,
			manifest.ErrInvalidActivationEvent,
		},
		{
			"an activation event that is not declared in the guide",
			`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"activationEvents":["onCommand:a"]}`,
			manifest.ErrInvalidActivationEvent,
		},
		{"a capability no host resolves", `{"id":"a.b","name":"A","version":"1.0.0","capabilities":["agent"]}`, manifest.ErrInvalidCapability},
		{
			"activation events without a backend",
			`{"id":"a.b","name":"A","version":"1.0.0","activationEvents":["onStartup"],"frontend":{}}`,
			manifest.ErrNoBackend,
		},
		{"services without a backend", `{"id":"a.b","name":"A","version":"1.0.0","services":["numen.v1.SketchService"]}`, manifest.ErrNoBackend},
		{"secrets without a backend", `{"id":"a.b","name":"A","version":"1.0.0","secrets":[{"name":"apiKey","title":"T"}]}`, manifest.ErrNoBackend},
		{
			"runtime dependencies without a backend",
			`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"sample-runtime","kind":"runtime"}]}`,
			manifest.ErrNoBackend,
		},
		{"setting without title", `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"boolean"}]}`, manifest.ErrEmptySettingTitle},
		{"unknown permission", `{"id":"a.b","name":"A","version":"1.0.0","permissions":["shell"]}`, manifest.ErrInvalidPermission},
		{
			"settings view without a frontend",
			`{"id":"a.b","name":"A","version":"1.0.0","settingsViews":[{"id":"v","title":"T"}]}`,
			manifest.ErrInvalidSettingsView,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(tc.json))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestManifestAcceptsCurrentProtocolAndAbsentProtocol(t *testing.T) {
	for _, raw := range []string{
		`{"id":"a.b","name":"A","version":"1.0.0"}`,
		`{"id":"a.b","name":"A","version":"1.0.0","protocol":1}`,
	} {
		m, err := manifestjson.Read(strings.NewReader(raw))
		if err != nil {
			t.Errorf("%s: unexpected error: %v", raw, err)
		}
		if got := m.GetProtocol(); got != 1 {
			t.Errorf("%s: GetProtocol() = %d, want 1", raw, got)
		}
	}
}

func TestManifestListIssuesReportsEveryViolation(t *testing.T) {
	m := manifest.Manifest{ID: "bad id", Version: "x", Permissions: []string{"shell", "clipboard"}}

	issues := m.ListIssues()

	for _, want := range []error{manifest.ErrInvalidID, manifest.ErrEmptyName, manifest.ErrInvalidVersion, manifest.ErrInvalidPermission} {
		if !slices.ContainsFunc(issues, func(e error) bool { return errors.Is(e, want) }) {
			t.Errorf("issues %v lack %v", issues, want)
		}
	}
	if len(issues) != 5 {
		t.Errorf("got %d issues, want 5: %v", len(issues), issues)
	}
	if err := m.Validate(); !errors.Is(err, manifest.ErrInvalidID) || !errors.Is(err, manifest.ErrEmptyName) {
		t.Errorf("Validate must join every issue, got %v", err)
	}
}

func TestManifestReadsSettingsViews(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","frontend":{},
		"settings":[{"key":"model","type":"string","title":"Model"}],
		"settingsViews":[{"id":"floor","title":"Search relevance"},{"id":"top","title":"Top"}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []manifest.SettingsView{{ID: "floor", Title: "Search relevance"}, {ID: "top", Title: "Top"}}
	if !slices.Equal(m.SettingsViews, want) {
		t.Errorf("SettingsViews = %+v, want %+v", m.SettingsViews, want)
	}
}

func TestManifestListIssuesReportsASettingsViewOfNoFrontend(t *testing.T) {
	m := manifest.Manifest{ID: "a.b", Name: "A", Version: "1.0.0", SettingsViews: []manifest.SettingsView{{ID: "v", Title: "T"}}}
	if issues := m.ListIssues(); len(issues) != 1 || !errors.Is(issues[0], manifest.ErrInvalidSettingsView) {
		t.Errorf("ListIssues = %v, want one ErrInvalidSettingsView", issues)
	}
}

func TestManifestNetworkAlias(t *testing.T) {
	if got := manifest.NormalizePermission("network"); got != manifest.PermissionLinkOpen {
		t.Errorf("got %q, want %q", got, manifest.PermissionLinkOpen)
	}
	if got := manifest.NormalizePermission(manifest.PermissionVaultRead); got != manifest.PermissionVaultRead {
		t.Errorf("got %q, want %q", got, manifest.PermissionVaultRead)
	}

	m := manifest.Manifest{Permissions: []string{"network"}}
	if !m.HasPermission(manifest.PermissionLinkOpen) || !m.HasPermission("network") {
		t.Errorf("alias must hold both spellings")
	}
	if m.HasPermission(manifest.PermissionVaultRead) {
		t.Errorf("alias must not grant vault:read")
	}
	if err := m.ValidatePermissions(); err != nil {
		t.Errorf("alias must validate: %v", err)
	}
}

func TestManifestAcceptsAICompletePermission(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","permissions":["ai:complete"]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.HasPermission(manifest.PermissionAIComplete) {
		t.Errorf("manifest must hold %s", manifest.PermissionAIComplete)
	}
}

// A secret is a plain name, declared once, and not also a setting.
func TestAManifestDeclaresItsSecretsByPlainDistinctNames(t *testing.T) {
	for _, one := range []struct {
		name    string
		secrets string
		wrong   bool
	}{
		{"none", ``, false},
		{"one", `"secrets":[{"name":"apiKey","title":"API key"}]`, false},
		{"two", `"secrets":[{"name":"apiKey","title":"API key"},{"name":"org_id-2","title":"Organization"}]`, false},
		{"empty name", `"secrets":[{"name":"","title":"T"}]`, true},
		{"slash", `"secrets":[{"name":"a/b","title":"T"}]`, true},
		{"space", `"secrets":[{"name":"api key","title":"T"}]`, true},
		{"twice", `"secrets":[{"name":"apiKey","title":"T"},{"name":"apiKey","title":"T"}]`, true},
		{"also a setting", `"secrets":[{"name":"model","title":"T"}],"settings":[{"key":"model","type":"string","title":"Model"}]`, true},
	} {
		t.Run(one.name, func(t *testing.T) {
			text := `{"id":"a.b","name":"A","version":"1.0.0"`
			if one.secrets != "" {
				text += "," + one.secrets
			}
			_, err := manifestjson.Read(strings.NewReader(text + "}"))
			if one.wrong != errors.Is(err, manifest.ErrInvalidSecret) {
				t.Errorf("error = %v, want invalid secret: %v", err, one.wrong)
			}
		})
	}
}

func TestAnAgentSettingTypeIsNotInTheSchema(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"model","type":"agent","title":"Model"}]}`))
	if !errors.Is(err, manifest.ErrInvalidSettingType) {
		t.Errorf("error = %v, want ErrInvalidSettingType", err)
	}
}

// A service a plugin serves to the windows is a full name of dot-separated words, listed once, that
// starts with the id of the plugin or is a service the protocol declares.
func TestTheServicesAPluginServesAreFullNamesListedOnce(t *testing.T) {
	cases := []struct {
		name     string
		services string
		wrong    bool
	}{
		{"a full name", `["a.b.SampleService"]`, false},
		{"two names", `["a.b.SampleService","a.b.PresetsService"]`, false},
		{"the id itself", `["a.b"]`, false},
		{"a service of the protocol", `["numen.v1.AgentService"]`, false},
		{"a bare name", `["SampleService"]`, true},
		{"a path", `["/a.b.SampleService"]`, true},
		{"a name listed twice", `["a.b.SampleService","a.b.SampleService"]`, true},
		{"a name of another plugin", `["c.d.SampleService"]`, true},
		{"a name that only starts like the id", `["a.bc.SampleService"]`, true},
		{"a name of the protocol the protocol does not declare", `["numen.v1.SketchService"]`, true},
		{"a short name outside the id", `["numen.epub"]`, true},
		{"an empty name", `[""]`, true},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			text := `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"services":` + one.services + `}`

			parsed, err := manifestjson.Read(strings.NewReader(text))

			if one.wrong != errors.Is(err, manifest.ErrInvalidService) {
				t.Errorf("error = %v, want invalid service: %v", err, one.wrong)
			}
			if !one.wrong && len(parsed.Services) == 0 {
				t.Errorf("the services came out as %v", parsed.Services)
			}
		})
	}
}

func TestAManifestDeclaresDependencies(t *testing.T) {
	sum := strings.Repeat("a", 64)
	m, err := manifestjson.Read(
		strings.NewReader(
			`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"dependencies":[{"id":"sample-runtime","kind":"runtime"},` +
				`{"id":"whisper","kind":"model","version":"1","sourceUrl":"https://example.com/w.bin","checksumSha256":"` + sum + `","metadata":{"fileName":"w.bin"}}]}`,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	got := m.GetDependencyRequirements("/p", nil)
	if len(got) != 2 || got[0].Kind != manifest.DependencyKindRuntime || got[0].ID != "sample-runtime" || got[1].ID != "whisper" ||
		got[1].Metadata["fileName"] != "w.bin" ||
		got[1].PluginDir != "/p" {
		t.Errorf("requirements = %+v", got)
	}
}

func TestAManifestRejectsAnInvalidDependency(t *testing.T) {
	for name, dependency := range map[string]string{
		"no id":              `{"kind":"model"}`,
		"unknown kind":       `{"id":"x","kind":"plugin"}`,
		"http address":       `{"id":"x","kind":"model","sourceUrl":"http://example.com/x","checksumSha256":"` + strings.Repeat("a", 64) + `"}`,
		"no checksum":        `{"id":"x","kind":"model","sourceUrl":"https://example.com/x"}`,
		"checksum no source": `{"id":"x","kind":"model","checksumSha256":"` + strings.Repeat("a", 64) + `"}`,
	} {
		_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[` + dependency + `]}`))
		if !errors.Is(err, manifest.ErrInvalidDependency) {
			t.Errorf("%s: err = %v, want ErrInvalidDependency", name, err)
		}
	}
}

func TestSettingBoundsAreAbsentUntilTheManifestSetsThem(t *testing.T) {
	raw := `{"id":"a.b","name":"A","version":"1.0.0","settings":[
		{"key":"zero","type":"range","title":"Zero","min":0,"max":0,"step":0},
		{"key":"none","type":"number","title":"None"}]}`
	m, err := manifestjson.Read(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	zero, none := m.Settings[0], m.Settings[1]
	if zero.Min == nil || zero.Max == nil || zero.Step == nil || *zero.Min != 0 || *zero.Max != 0 || *zero.Step != 0 {
		t.Errorf("bounds set to 0 are absent: %+v", zero)
	}
	if none.Min != nil || none.Max != nil || none.Step != nil {
		t.Errorf("bounds the manifest does not set are present: %+v", none)
	}
}

// The field enabled of the object of a plugin in the settings file is the host's, so a setting of that
// name would meet it.
func TestAManifestDoesNotDeclareASettingNamedEnabled(t *testing.T) {
	_, err := manifestjson.Read(
		strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"enabled","type":"boolean","title":"Enabled"}]}`),
	)
	if !errors.Is(err, manifest.ErrReservedSettingKey) {
		t.Errorf("error = %v, want ErrReservedSettingKey", err)
	}
}

func TestASettingMayPreviewWhileFocusedAndListItsOptionsFromARegistry(t *testing.T) {
	raw := `{
		"id": "md.numen.sample",
		"name": "Sample",
		"version": "1.0.0",
		"settings": [
			{"key": "theme", "type": "enum", "title": "Theme", "optionsFrom": "themes", "preview": true},
			{"key": "retries", "type": "number", "title": "Retries"}
		]
	}`

	m, err := manifestjson.Read(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("a registry enum without options of its own was rejected: %v", err)
	}
	if got := m.Settings[0]; !got.Preview || got.OptionsFrom != "themes" {
		t.Errorf("the theme reads as preview=%v from %q", got.Preview, got.OptionsFrom)
	}
	if got := m.Settings[1]; got.Preview || got.OptionsFrom != "" {
		t.Errorf("a plain setting reads as preview=%v from %q", got.Preview, got.OptionsFrom)
	}
}

// The first level of the settings file is the version v, the owner app and the ids of plugins, so a
// plugin cannot take the first two as its id.
func TestAManifestRejectsAnIdThatTheSettingsFileReserves(t *testing.T) {
	for _, id := range []string{"app", "v"} {
		_, err := manifestjson.Read(strings.NewReader(`{"id":"` + id + `","name":"A","version":"1.0.0"}`))
		if err == nil {
			t.Errorf("the id %q was accepted", id)
		}
	}
}

// Not yet decided by the person: a key of a plugin setting is one segment, since a dot joins the levels of a
// path. This test is dropped if the person decides otherwise.
func TestAManifestDoesNotDeclareASettingKeyWithADot(t *testing.T) {
	_, err := manifestjson.Read(
		strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"max.tokens","type":"number","title":"Tokens"}]}`),
	)
	if !errors.Is(err, manifest.ErrInvalidSettingKey) {
		t.Errorf("error = %v, want ErrInvalidSettingKey", err)
	}
}

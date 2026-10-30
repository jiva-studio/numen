package manifestjson_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

func TestReadManifest(t *testing.T) {
	raw := `{
		"id": "md.numen.engines.ocr",
		"name": "Document OCR",
		"version": "1.0.0",
		"description": "Optical character recognition",
		"capabilities": ["recognition"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onCommand:ocr.recognize"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 180,
			"keepAliveWhileVisible": true
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/engines.ocr",
			"serviceName": "numen.v1.TextProviderService"
		},
		"frontend": {
			"entrypoint": "dist/index.js",
			"styles": "dist/styles.css"
		},
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
	if !m.HasCapability("recognition") {
		t.Errorf("expected capability recogniser")
	}
	if m.CanActivateOn("onFile:document.pdf") {
		t.Errorf("unexpected activation on onFile:document.pdf: a file event is matched by content type")
	}
	if !m.CanActivateOn("onCommand:ocr.recognize") {
		t.Errorf("expected activation on onCommand:ocr.recognize")
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
			name: "select without options",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"select","title":"T","options":[]}]}`,
		},
		{
			name: "select missing options",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"select","title":"T"}]}`,
		},
		{
			name: "models of a program on a setting that is not a string",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"number","title":"T","modelsOf":"p"}]}`,
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
			{"key": "enabled", "type": "boolean", "title": "Enabled", "default": true, "palette": true},
			{"key": "model", "type": "enum", "title": "Model", "default": "fast", "options": ["fast", "accurate"]},
			{"key": "retries", "type": "number", "title": "Retries", "default": 3},
			{"key": "zoom", "type": "range", "title": "Zoom", "default": 1.0, "min": 0.5, "max": 2.0, "step": 0.1},
			{"key": "prefix", "type": "string", "title": "Prefix", "default": "test"}
		]
	}`

	m, err := manifestjson.Read(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error reading manifest with settings: %v", err)
	}

	if len(m.Settings) != 5 {
		t.Fatalf("got %d settings, want 5", len(m.Settings))
	}
	if !m.Settings[0].Palette {
		t.Errorf("expected setting enabled to have palette: true")
	}
	if len(m.Settings[1].Options) != 2 {
		t.Errorf("expected enum setting to have 2 options")
	}
}

func TestManifestSelectSetting(t *testing.T) {
	t.Run("passes when options are present", func(t *testing.T) {
		raw := `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"mode","type":"select","title":"Mode","options":["fast","accurate"]}]}`
		m, err := manifestjson.Read(strings.NewReader(raw))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.Settings) != 1 {
			t.Fatalf("got %d settings, want 1", len(m.Settings))
		}
		if m.Settings[0].Type != "select" {
			t.Errorf("got type %q, want select", m.Settings[0].Type)
		}
		if len(m.Settings[0].Options) != 2 {
			t.Errorf("expected select setting to have 2 options")
		}
	})

	t.Run("fails when options are missing", func(t *testing.T) {
		raw := `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"mode","type":"select","title":"Mode"}]}`
		_, err := manifestjson.Read(strings.NewReader(raw))
		if err == nil {
			t.Fatal("expected error for select without options, got nil")
		}
	})
}

func TestManifestLifecycleMemoryLimit(t *testing.T) {
	t.Run("decodes memoryLimitBytes", func(t *testing.T) {
		raw := `{"id":"a.b","name":"A","version":"1.0.0","lifecycle":{"mode":"on-demand","memoryLimitBytes":1073741824}}`
		m, err := manifestjson.Read(strings.NewReader(raw))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Lifecycle.MemoryLimitBytes != 1073741824 {
			t.Errorf("got memoryLimitBytes %d, want 1073741824", m.Lifecycle.MemoryLimitBytes)
		}
	})

	t.Run("decodes memoryLimitMB", func(t *testing.T) {
		raw := `{"id":"a.b","name":"A","version":"1.0.0","lifecycle":{"mode":"on-demand","memoryLimitMB":512}}`
		m, err := manifestjson.Read(strings.NewReader(raw))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Lifecycle.MemoryLimitBytes != 512*1024*1024 {
			t.Errorf("got memoryLimitBytes %d, want %d", m.Lifecycle.MemoryLimitBytes, 512*1024*1024)
		}
	})
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
		{"negative idle timeout", `{"id":"a.b","name":"A","version":"1.0.0","lifecycle":{"idleTimeoutSeconds":-1}}`, manifest.ErrInvalidLifecycle},
		{"wasm backend", `{"id":"a.b","name":"A","version":"1.0.0","backend":{"type":"wasm","entrypoint":"x.wasm"}}`, manifest.ErrInvalidBackend},
		{"backend without entrypoint", `{"id":"a.b","name":"A","version":"1.0.0","backend":{"type":"process"}}`, manifest.ErrInvalidBackend},
		{"activation events without a backend", `{"id":"a.b","name":"A","version":"1.0.0","activationEvents":["onCommand:a"],"frontend":{"entrypoint":"dist/index.js"}}`, manifest.ErrNoBackend},
		{"commands without a backend", `{"id":"a.b","name":"A","version":"1.0.0","commands":[{"id":"run","title":"Run"}]}`, manifest.ErrNoBackend},
		{"services without a backend", `{"id":"a.b","name":"A","version":"1.0.0","services":["numen.v1.SketchService"]}`, manifest.ErrNoBackend},
		{"secrets without a backend", `{"id":"a.b","name":"A","version":"1.0.0","secrets":[{"name":"apiKey","title":"T"}]}`, manifest.ErrNoBackend},
		{"runtimes without a backend", `{"id":"a.b","name":"A","version":"1.0.0","runtimes":["onnxruntime"]}`, manifest.ErrNoBackend},
		{"frontend without entrypoint", `{"id":"a.b","name":"A","version":"1.0.0","frontend":{"styles":"s.css"}}`, manifest.ErrInvalidFrontend},
		{"setting without title", `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"boolean"}]}`, manifest.ErrEmptySettingTitle},
		{"unknown permission", `{"id":"a.b","name":"A","version":"1.0.0","permissions":["shell"]}`, manifest.ErrInvalidPermission},
		{"unknown runtime", `{"id":"a.b","name":"A","version":"1.0.0","runtimes":["tensorrt"]}`, manifest.ErrInvalidRuntime},
		{"runtime declared twice", `{"id":"a.b","name":"A","version":"1.0.0","runtimes":["onnxruntime","onnxruntime"]}`, manifest.ErrInvalidRuntime},
		{"settings view without a frontend", `{"id":"a.b","name":"A","version":"1.0.0","settingsViews":[{"id":"v","title":"T"}]}`, manifest.ErrInvalidSettingsView},
		{"settings view after an unknown setting", `{"id":"a.b","name":"A","version":"1.0.0","frontend":{"entrypoint":"dist/index.js"},"settingsViews":[{"id":"v","title":"T","after":"nope"}]}`, manifest.ErrInvalidSettingsView},
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
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","frontend":{"entrypoint":"dist/index.js"},
		"settings":[{"key":"model","type":"string","title":"Model"}],
		"settingsViews":[{"id":"floor","title":"Search relevance","after":"model"},{"id":"top","title":"Top"}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []manifest.SettingsView{{ID: "floor", Title: "Search relevance", After: "model"}, {ID: "top", Title: "Top"}}
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
	if got := manifest.NormalizePermission("network"); got != manifest.PermissionNetworkOutbound {
		t.Errorf("got %q, want %q", got, manifest.PermissionNetworkOutbound)
	}
	if got := manifest.NormalizePermission(manifest.PermissionVaultRead); got != manifest.PermissionVaultRead {
		t.Errorf("got %q, want %q", got, manifest.PermissionVaultRead)
	}

	m := manifest.Manifest{Permissions: []string{"network"}}
	if !m.HasPermission(manifest.PermissionNetworkOutbound) || !m.HasPermission("network") {
		t.Errorf("alias must hold both spellings")
	}
	if m.HasPermission(manifest.PermissionVaultRead) {
		t.Errorf("alias must not grant vault:read")
	}
	if err := m.ValidatePermissions(); err != nil {
		t.Errorf("alias must validate: %v", err)
	}
}

func TestManifestAcceptsTheONNXRuntime(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","runtimes":["onnxruntime"],"backend":{"entrypoint":"bin/a"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(m.Runtimes, []string{manifest.RuntimeONNX}) {
		t.Errorf("runtimes = %v, want [%s]", m.Runtimes, manifest.RuntimeONNX)
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
		{"two", `"secrets":[{"name":"apiKey","title":"API key"},{"name":"org_id-2","title":"Organisation"}]`, false},
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

// The names under which the host records the stored secrets and the consent are
// neither a setting nor a secret of a manifest.
func TestAManifestDoesNotTakeTheNameOfTheStoredSecrets(t *testing.T) {
	for _, one := range []struct {
		name  string
		field string
	}{
		{"a setting", `"settings":[{"key":"storedSecrets","type":"string","title":"Stored"}]`},
		{"a secret", `"secrets":[{"name":"storedSecrets","title":"T"}]`},
		{"the consent as a setting", `"settings":[{"key":"aiConsent","type":"boolean","title":"Allowed","default":true}]`},
		{"the consent as a secret", `"secrets":[{"name":"aiConsent","title":"T"}]`},
	} {
		t.Run(one.name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",` + one.field + `}`))
			if !errors.Is(err, manifest.ErrReservedSettingKey) {
				t.Errorf("error = %v, want port.ErrReservedSettingKey", err)
			}
		})
	}
}

// A setting of type agent is at most one, declared by a plugin that may query a
// model, and leaves the choice to the user.
func TestAManifestDeclaresTheAgentSettingOnceWithoutADefault(t *testing.T) {
	const agent = `{"key":"model","type":"agent","title":"Model"}`
	for _, one := range []struct {
		name, fields string
		wrong        bool
	}{
		{"one", `"permissions":["ai:complete"],"settings":[` + agent + `]`, false},
		{"none", `"permissions":["ai:complete"]`, false},
		{"without the permission", `"settings":[` + agent + `]`, true},
		{"two", `"permissions":["ai:complete"],"settings":[` + agent + `,{"key":"other","type":"agent","title":"Other"}]`, true},
		{"a default", `"permissions":["ai:complete"],"settings":[{"key":"model","type":"agent","title":"Model","default":{"agent":"md.numen.agents.numen"}}]`, true},
	} {
		t.Run(one.name, func(t *testing.T) {
			text := `{"id":"a.b","name":"A","version":"1.0.0",` + one.fields + `}`
			m, err := manifestjson.Read(strings.NewReader(text))
			if one.wrong != errors.Is(err, manifest.ErrInvalidAgentSetting) {
				t.Fatalf("error = %v, want invalid agent setting: %v", err, one.wrong)
			}
			if one.wrong {
				return
			}
			key, found := m.GetAgentSettingKey()
			if found != strings.Contains(one.fields, `"type":"agent"`) || (found && key != "model") {
				t.Errorf("the agent setting is %q, %v", key, found)
			}
		})
	}
}

// A service a plugin serves to the windows is a full name of dot-separated words, listed once.
func TestTheServicesAPluginServesAreFullNamesListedOnce(t *testing.T) {
	cases := []struct {
		name     string
		services string
		wrong    bool
	}{
		{"a full name", `["numen.v1.SampleService"]`, false},
		{"two names", `["numen.v1.SampleService","numen.v1.PresetsService"]`, false},
		{"a bare name", `["SampleService"]`, true},
		{"a path", `["/numen.v1.SampleService"]`, true},
		{"a name listed twice", `["numen.v1.SampleService","numen.v1.SampleService"]`, true},
		{"an empty name", `[""]`, true},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			text := `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"services":` + one.services + `}`

			parsed, err := manifestjson.Read(strings.NewReader(text))

			if one.wrong != errors.Is(err, manifest.ErrInvalidService) {
				t.Errorf("error = %v, want invalid service: %v", err, one.wrong)
			}
			if !one.wrong && !parsed.HasService("numen.v1.SampleService") {
				t.Errorf("the services came out as %v", parsed.Services)
			}
		})
	}
}

func TestReadManifest_KeepsModelsOfProgram(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(
		`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"model","type":"string","title":"Model","modelsOf":"prog"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Settings[0].ModelsOf; got != "prog" {
		t.Errorf("ModelsOf = %q, want prog", got)
	}
}

func TestAManifestDeclaresDependencies(t *testing.T) {
	sum := strings.Repeat("a", 64)
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"runtimes":["onnxruntime"],"dependencies":[{"id":"whisper","kind":"model","version":"1","sourceUrl":"https://example.com/w.bin","checksumSha256":"` + sum + `","metadata":{"fileName":"w.bin"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	got := m.GetDependencyRequirements("/p", nil)
	if len(got) != 2 || got[0].Kind != manifest.DependencyKindRuntime || got[0].ID != manifest.RuntimeONNX || got[1].ID != "whisper" || got[1].Metadata["fileName"] != "w.bin" || got[1].PluginDir != "/p" {
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

package plugin

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestReadManifest(t *testing.T) {
	raw := `{
		"id": "md.numen.ocr",
		"name": "Document OCR",
		"version": "1.0.0",
		"description": "Optical character recognition",
		"capabilities": ["recognition"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onFile:*.pdf", "onCommand:ocr.recognize"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 180,
			"keepAliveWhileVisible": true
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/ocr",
			"serviceName": "numen.v1.OcrService"
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

	m, err := ReadManifest(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.ID != "md.numen.ocr" {
		t.Errorf("got id %q, want md.numen.ocr", m.ID)
	}
	if !m.CanTarget("desktop.editor") {
		t.Errorf("expected target desktop.editor to be accepted")
	}
	if m.CanTarget("desktop.flashcards") {
		t.Errorf("expected target desktop.flashcards to be rejected")
	}
	if !m.HasCapability("recognition") {
		t.Errorf("expected capability recogniser")
	}
	if !m.CanActivateOn("onFile:document.pdf") {
		t.Errorf("expected activation on onFile:document.pdf")
	}
	if !m.CanActivateOn("onFile:pdf") {
		t.Errorf("expected activation on onFile:pdf")
	}
	if !m.CanActivateOn("onFile:.pdf") {
		t.Errorf("expected activation on onFile:.pdf")
	}
	if m.CanActivateOn("onFile:book.epub") {
		t.Errorf("unexpected activation on onFile:book.epub")
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
			_, err := ReadManifest(strings.NewReader(tc.json))
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
			name: "range min greater than max",
			json: `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"range","title":"T","min":100,"max":50}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadManifest(strings.NewReader(tc.json))
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

	m, err := ReadManifest(strings.NewReader(raw))
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
		m, err := ReadManifest(strings.NewReader(raw))
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
		_, err := ReadManifest(strings.NewReader(raw))
		if err == nil {
			t.Fatal("expected error for select without options, got nil")
		}
	})
}

func TestManifestLifecycleMemoryLimit(t *testing.T) {
	t.Run("decodes memoryLimitBytes", func(t *testing.T) {
		raw := `{"id":"a.b","name":"A","version":"1.0.0","lifecycle":{"mode":"on-demand","memoryLimitBytes":1073741824}}`
		m, err := ReadManifest(strings.NewReader(raw))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Lifecycle.MemoryLimitBytes != 1073741824 {
			t.Errorf("got memoryLimitBytes %d, want 1073741824", m.Lifecycle.MemoryLimitBytes)
		}
	})

	t.Run("decodes memoryLimitMB", func(t *testing.T) {
		raw := `{"id":"a.b","name":"A","version":"1.0.0","lifecycle":{"mode":"on-demand","memoryLimitMB":512}}`
		m, err := ReadManifest(strings.NewReader(raw))
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
		{"id with a space", `{"id":"bad id","name":"A","version":"1.0.0"}`, port.ErrInvalidID},
		{"version without patch", `{"id":"a.b","name":"A","version":"1.0"}`, port.ErrInvalidVersion},
		{"version with leading zero", `{"id":"a.b","name":"A","version":"01.0.0"}`, port.ErrInvalidVersion},
		{"prerelease with leading zero", `{"id":"a.b","name":"A","version":"1.0.0-01"}`, port.ErrInvalidVersion},
		{"id of one segment", `{"id":"foo","name":"A","version":"1.0.0"}`, port.ErrInvalidID},
		{"protocol newer than the host", `{"id":"a.b","name":"A","version":"1.0.0","protocol":2}`, port.ErrUnsupportedProtocol},
		{"negative protocol", `{"id":"a.b","name":"A","version":"1.0.0","protocol":-1}`, port.ErrUnsupportedProtocol},
		{"negative idle timeout", `{"id":"a.b","name":"A","version":"1.0.0","lifecycle":{"idleTimeoutSeconds":-1}}`, port.ErrInvalidLifecycle},
		{"wasm backend", `{"id":"a.b","name":"A","version":"1.0.0","backend":{"type":"wasm","entrypoint":"x.wasm"}}`, port.ErrInvalidBackend},
		{"backend without entrypoint", `{"id":"a.b","name":"A","version":"1.0.0","backend":{"type":"process"}}`, port.ErrInvalidBackend},
		{"frontend without entrypoint", `{"id":"a.b","name":"A","version":"1.0.0","frontend":{"styles":"s.css"}}`, port.ErrInvalidFrontend},
		{"setting without title", `{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"k","type":"boolean"}]}`, port.ErrEmptySettingTitle},
		{"unknown permission", `{"id":"a.b","name":"A","version":"1.0.0","permissions":["shell"]}`, port.ErrInvalidPermission},
		{"unknown runtime", `{"id":"a.b","name":"A","version":"1.0.0","runtimes":["tensorrt"]}`, port.ErrInvalidRuntime},
		{"runtime declared twice", `{"id":"a.b","name":"A","version":"1.0.0","runtimes":["onnxruntime","onnxruntime"]}`, port.ErrInvalidRuntime},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadManifest(strings.NewReader(tc.json))
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
		m, err := ReadManifest(strings.NewReader(raw))
		if err != nil {
			t.Errorf("%s: unexpected error: %v", raw, err)
		}
		if got := m.GetProtocol(); got != 1 {
			t.Errorf("%s: GetProtocol() = %d, want 1", raw, got)
		}
	}
}

func TestManifestListIssuesReportsEveryViolation(t *testing.T) {
	m := Manifest{ID: "bad id", Version: "x", Permissions: []string{"shell", "clipboard"}}

	issues := m.ListIssues()

	for _, want := range []error{port.ErrInvalidID, port.ErrEmptyName, port.ErrInvalidVersion, port.ErrInvalidPermission} {
		if !slices.ContainsFunc(issues, func(e error) bool { return errors.Is(e, want) }) {
			t.Errorf("issues %v lack %v", issues, want)
		}
	}
	if len(issues) != 5 {
		t.Errorf("got %d issues, want 5: %v", len(issues), issues)
	}
	if err := m.Validate(); !errors.Is(err, port.ErrInvalidID) || !errors.Is(err, port.ErrEmptyName) {
		t.Errorf("Validate must join every issue, got %v", err)
	}
}

func TestManifestNetworkAlias(t *testing.T) {
	if got := port.NormalizePermission("network"); got != PermissionNetworkOutbound {
		t.Errorf("got %q, want %q", got, PermissionNetworkOutbound)
	}
	if got := port.NormalizePermission(PermissionVaultRead); got != PermissionVaultRead {
		t.Errorf("got %q, want %q", got, PermissionVaultRead)
	}

	m := Manifest{Permissions: []string{"network"}}
	if !m.HasPermission(PermissionNetworkOutbound) || !m.HasPermission("network") {
		t.Errorf("alias must hold both spellings")
	}
	if m.HasPermission(PermissionVaultRead) {
		t.Errorf("alias must not grant vault:read")
	}
	if err := m.ValidatePermissions(); err != nil {
		t.Errorf("alias must validate: %v", err)
	}
}

func TestManifestAcceptsTheONNXRuntime(t *testing.T) {
	m, err := ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","runtimes":["onnxruntime"]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(m.Runtimes, []string{port.RuntimeONNX}) {
		t.Errorf("runtimes = %v, want [%s]", m.Runtimes, port.RuntimeONNX)
	}
}

func TestManifestAcceptsAICompletePermission(t *testing.T) {
	m, err := ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","permissions":["ai:complete"]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.HasPermission(PermissionAIComplete) {
		t.Errorf("manifest must hold %s", PermissionAIComplete)
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
		{"one", `"secrets":["apiKey"]`, false},
		{"two", `"secrets":["apiKey","org_id-2"]`, false},
		{"empty name", `"secrets":[""]`, true},
		{"slash", `"secrets":["a/b"]`, true},
		{"space", `"secrets":["api key"]`, true},
		{"twice", `"secrets":["apiKey","apiKey"]`, true},
		{"also a setting", `"secrets":["model"],"settings":[{"key":"model","type":"string","title":"Model"}]`, true},
	} {
		t.Run(one.name, func(t *testing.T) {
			text := `{"id":"a.b","name":"A","version":"1.0.0"`
			if one.secrets != "" {
				text += "," + one.secrets
			}
			_, err := ReadManifest(strings.NewReader(text + "}"))
			if one.wrong != errors.Is(err, port.ErrInvalidSecret) {
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
		{"a secret", `"secrets":["storedSecrets"]`},
		{"the consent as a setting", `"settings":[{"key":"aiConsent","type":"boolean","title":"Allowed","default":true}]`},
		{"the consent as a secret", `"secrets":["aiConsent"]`},
	} {
		t.Run(one.name, func(t *testing.T) {
			_, err := ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",` + one.field + `}`))
			if !errors.Is(err, port.ErrReservedSettingKey) {
				t.Errorf("error = %v, want port.ErrReservedSettingKey", err)
			}
		})
	}
}

// A setting of type agent is one at most, declared by a plugin that may ask a
// model, and leaves the choice to the person.
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
		{"a default", `"permissions":["ai:complete"],"settings":[{"key":"model","type":"agent","title":"Model","default":{"agent":"md.numen.agent"}}]`, true},
	} {
		t.Run(one.name, func(t *testing.T) {
			text := `{"id":"a.b","name":"A","version":"1.0.0",` + one.fields + `}`
			m, err := ReadManifest(strings.NewReader(text))
			if one.wrong != errors.Is(err, port.ErrInvalidAgentSetting) {
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

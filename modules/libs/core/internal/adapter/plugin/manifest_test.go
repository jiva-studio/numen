package plugin

import (
	"strings"
	"testing"
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
		{name: "missing id", json: `{"name":"A","version":"1"}`},
		{name: "missing name", json: `{"id":"a","version":"1"}`},
		{name: "missing version", json: `{"id":"a","name":"A"}`},
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
			json: `{"id":"a","name":"A","version":"1","settings":[{"key":"","type":"boolean","title":"T"}]}`,
		},
		{
			name: "duplicate setting key",
			json: `{"id":"a","name":"A","version":"1","settings":[{"key":"k","type":"boolean","title":"T"},{"key":"k","type":"string","title":"T2"}]}`,
		},
		{
			name: "invalid setting type",
			json: `{"id":"a","name":"A","version":"1","settings":[{"key":"k","type":"unsupported","title":"T"}]}`,
		},
		{
			name: "enum without options",
			json: `{"id":"a","name":"A","version":"1","settings":[{"key":"k","type":"enum","title":"T","options":[]}]}`,
		},
		{
			name: "select without options",
			json: `{"id":"a","name":"A","version":"1","settings":[{"key":"k","type":"select","title":"T","options":[]}]}`,
		},
		{
			name: "select missing options",
			json: `{"id":"a","name":"A","version":"1","settings":[{"key":"k","type":"select","title":"T"}]}`,
		},
		{
			name: "range min greater than max",
			json: `{"id":"a","name":"A","version":"1","settings":[{"key":"k","type":"range","title":"T","min":100,"max":50}]}`,
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
		raw := `{"id":"a","name":"A","version":"1","settings":[{"key":"mode","type":"select","title":"Mode","options":["fast","accurate"]}]}`
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
		raw := `{"id":"a","name":"A","version":"1","settings":[{"key":"mode","type":"select","title":"Mode"}]}`
		_, err := ReadManifest(strings.NewReader(raw))
		if err == nil {
			t.Fatal("expected error for select without options, got nil")
		}
	})
}


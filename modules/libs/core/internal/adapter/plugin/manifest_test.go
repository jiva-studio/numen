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
		"capabilities": ["recogniser"],
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
	if !m.HasCapability("recogniser") {
		t.Errorf("expected capability recogniser")
	}
	if !m.CanActivateOn("onFile:document.pdf") {
		t.Errorf("expected activation on onFile:document.pdf")
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

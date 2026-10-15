package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverPlugins(t *testing.T) {
	tempDir := t.TempDir()

	plugin1Dir := filepath.Join(tempDir, "plugin1")
	if err := os.Mkdir(plugin1Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest1 := `{
		"id": "md.numen.ocr",
		"name": "OCR",
		"version": "1.0.0",
		"description": "OCR plugin",
		"capabilities": ["recogniser"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onFile:*.pdf"],
		"permissions": [],
		"lifecycle": {"mode": "on-demand", "idleTimeoutSeconds": 100, "keepAliveWhileVisible": false},
		"backend": {"type": "process", "entrypoint": "bin/ocr", "serviceName": "OcrService"},
		"frontend": {"entrypoint": "", "styles": ""},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(plugin1Dir, "manifest.json"), []byte(manifest1), 0o644); err != nil {
		t.Fatal(err)
	}

	plugin2Dir := filepath.Join(tempDir, "plugin2")
	if err := os.Mkdir(plugin2Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest2 := `{
		"id": "md.numen.cards",
		"name": "Cards",
		"version": "1.0.0",
		"description": "Cards plugin",
		"capabilities": ["cards"],
		"targets": ["desktop.flashcards"],
		"activationEvents": ["*"],
		"permissions": [],
		"lifecycle": {"mode": "persistent", "idleTimeoutSeconds": 0, "keepAliveWhileVisible": true},
		"backend": {"type": "process", "entrypoint": "bin/cards", "serviceName": "CardsService"},
		"frontend": {"entrypoint": "", "styles": ""},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(plugin2Dir, "manifest.json"), []byte(manifest2), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := DiscoverPlugins([]string{tempDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("got %d plugins, want 2", len(found))
	}

	editorPlugins := FilterPluginsByTarget(found, "desktop.editor")
	if len(editorPlugins) != 1 || editorPlugins[0].Manifest.ID != "md.numen.ocr" {
		t.Errorf("unexpected editor plugins: %v", editorPlugins)
	}

	flashcardsPlugins := FilterPluginsByTarget(found, "desktop.flashcards")
	if len(flashcardsPlugins) != 1 || flashcardsPlugins[0].Manifest.ID != "md.numen.cards" {
		t.Errorf("unexpected flashcards plugins: %v", flashcardsPlugins)
	}

	ocrPlugin, ok := FindPluginByCapability(found, "recogniser")
	if !ok || ocrPlugin.Manifest.ID != "md.numen.ocr" {
		t.Errorf("expected to find ocr plugin by capability")
	}

	_, ok = FindPluginByCapability(found, "transcriber")
	if ok {
		t.Errorf("did not expect to find transcriber plugin")
	}
}

func TestDiscoverPluginsDuplicateID(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	manifest := `{
		"id": "duplicate.id",
		"name": "Dup",
		"version": "1.0.0",
		"description": "Dup",
		"capabilities": [],
		"targets": [],
		"activationEvents": [],
		"permissions": [],
		"lifecycle": {"mode": "", "idleTimeoutSeconds": 0, "keepAliveWhileVisible": false},
		"backend": {"type": "", "entrypoint": "", "serviceName": ""},
		"frontend": {"entrypoint": "", "styles": ""},
		"settings": []
	}`

	p1 := filepath.Join(tempDir1, "p1")
	_ = os.Mkdir(p1, 0o755)
	_ = os.WriteFile(filepath.Join(p1, "manifest.json"), []byte(manifest), 0o644)

	p2 := filepath.Join(tempDir2, "p2")
	_ = os.Mkdir(p2, 0o755)
	_ = os.WriteFile(filepath.Join(p2, "manifest.json"), []byte(manifest), 0o644)

	_, err := DiscoverPlugins([]string{tempDir1, tempDir2})
	if err == nil {
		t.Fatalf("expected error on duplicate plugin id, got nil")
	}
}

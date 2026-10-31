package manifestjson_test

import (
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	tempDir := t.TempDir()

	plugin1Dir := filepath.Join(tempDir, "plugin1")
	if err := os.Mkdir(plugin1Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest1 := `{
		"id": "md.numen.engines.ocr",
		"name": "OCR",
		"version": "1.0.0",
		"description": "OCR plugin",
		"capabilities": ["recognition"],
		"targets": ["desktop.editor"],
		"activationEvents": [],
		"permissions": [],
		"lifecycle": {"mode": "on-demand", "idleTimeoutSeconds": 100, "keepAliveWhileVisible": false},
		"backend": {"type": "process", "entrypoint": "bin/engines.ocr", "serviceName": "TextProviderService"},
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
		"id": "org.example.sample",
		"name": "Sample",
		"version": "1.0.0",
		"description": "Sample plugin",
		"capabilities": ["samples"],
		"targets": ["desktop.samples"],
		"activationEvents": ["*"],
		"permissions": [],
		"lifecycle": {"mode": "persistent", "idleTimeoutSeconds": 0, "keepAliveWhileVisible": true},
		"backend": {"type": "process", "entrypoint": "bin/sample", "serviceName": "SampleService"},
		"frontend": {"entrypoint": "", "styles": ""},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(plugin2Dir, "manifest.json"), []byte(manifest2), 0o644); err != nil {
		t.Fatal(err)
	}

	found, problems := manifestjson.Discover([]string{tempDir})
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if len(found) != 2 {
		t.Fatalf("got %d plugins, want 2", len(found))
	}

	editorPlugins := manifest.FilterByTarget(found, "desktop.editor")
	if len(editorPlugins) != 1 || editorPlugins[0].Manifest.ID != "md.numen.engines.ocr" {
		t.Errorf("unexpected editor plugins: %v", editorPlugins)
	}

	samplePlugins := manifest.FilterByTarget(found, "desktop.samples")
	if len(samplePlugins) != 1 || samplePlugins[0].Manifest.ID != "org.example.sample" {
		t.Errorf("unexpected sample plugins: %v", samplePlugins)
	}
}

func TestDiscoverPluginsDuplicateID(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	manifestText := `{
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
	if err := os.Mkdir(p1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p1, "manifest.json"), []byte(manifestText), 0o644); err != nil {
		t.Fatal(err)
	}

	p2 := filepath.Join(tempDir2, "p2")
	if err := os.Mkdir(p2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p2, "manifest.json"), []byte(manifestText), 0o644); err != nil {
		t.Fatal(err)
	}

	found, problems := manifestjson.Discover([]string{tempDir1, tempDir2})
	if len(found) != 1 {
		t.Fatalf("expected the first plugin with the id to be kept, got %d", len(found))
	}
	if len(problems) != 1 || problems[0].Dir != p2 {
		t.Fatalf("expected the duplicate to be reported, got %v", problems)
	}
}

func TestDiscoverPluginsSameDirectoryRepeated(t *testing.T) {
	tempDir := t.TempDir()
	p := filepath.Join(tempDir, "p1")
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestText := `{
		"id": "single.id",
		"name": "Single",
		"version": "1.0.0",
		"description": "Single",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"activationEvents": [],
		"permissions": [],
		"lifecycle": {"mode": "", "idleTimeoutSeconds": 0, "keepAliveWhileVisible": false},
		"backend": {"type": "", "entrypoint": "", "serviceName": ""},
		"frontend": {"entrypoint": "", "styles": ""},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(p, "manifest.json"), []byte(manifestText), 0o644); err != nil {
		t.Fatal(err)
	}

	// Passing same directory twice (e.g. from NUMEN_PLUGINS_PATH and executable relative path)
	found, problems := manifestjson.Discover([]string{tempDir, tempDir})
	if len(problems) != 0 || len(found) != 1 {
		t.Fatalf("expected one plugin and no problem when a directory is repeated, got %d and %v", len(found), problems)
	}
}

func TestABrokenManifestDoesNotHideTheOtherPlugins(t *testing.T) {
	tempDir := t.TempDir()
	writeSamplePlugin(t, tempDir)
	broken := filepath.Join(tempDir, "broken")
	if err := os.Mkdir(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "manifest.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	found, problems := manifestjson.Discover([]string{tempDir})

	if len(found) != 1 || found[0].Manifest.ID != "org.example.sample" {
		t.Fatalf("expected the valid plugin to be discovered, got %v", found)
	}
	if len(problems) != 1 || problems[0].Dir != broken || problems[0].Err == nil {
		t.Fatalf("expected the broken manifest to be reported, got %v", problems)
	}
}

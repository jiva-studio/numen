package manifestjson_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
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
		"capabilities": ["tools"],
		"targets": ["desktop.editor"],
				"permissions": [],
		"backend": {"entrypoint": "bin/engines.ocr"},
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
		"capabilities": ["search.source"],
		"targets": ["desktop.samples"],
		"activationEvents": ["onSearchSourceReady"],
		"permissions": [],
		"backend": {"entrypoint": "bin/sample"},
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
				"permissions": [],
		"backend": {},
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
		"capabilities": ["search.source"],
		"targets": ["desktop.editor"],
				"permissions": [],
		"backend": {},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(p, "manifest.json"), []byte(manifestText), 0o644); err != nil {
		t.Fatal(err)
	}

	// Passing same directory twice (e.g. from NUMEN_PLUGINS_PATH and executable relative path)
	otherDir := t.TempDir()
	writeSamplePlugin(t, otherDir)
	found, problems := manifestjson.Discover([]string{"", tempDir, tempDir, otherDir})
	if len(problems) != 0 || len(found) != 2 {
		t.Fatalf("expected two plugins and no problem when a directory is repeated, got %d and %v", len(found), problems)
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
	if err := os.Mkdir(filepath.Join(tempDir, "empty-dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	notADir := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(notADir, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}

	found, problems := manifestjson.Discover([]string{tempDir})

	if len(found) != 1 || found[0].Manifest.ID != "org.example.sample" {
		t.Fatalf("expected the valid plugin to be discovered, got %v", found)
	}
	var syntaxErr *json.SyntaxError
	if len(problems) != 1 || problems[0].Dir != broken || !errors.As(problems[0].Err, &syntaxErr) {
		t.Fatalf("expected the broken manifest syntax error to be reported, got %v", problems)
	}

	if _, err := manifestjson.ReadFile(filepath.Join(tempDir, "nonexistent.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadFile nonexistent = %v", err)
	}
	if f, p := manifestjson.Discover([]string{filepath.Join(tempDir, "nonexistent")}); len(f) != 0 || len(p) != 0 {
		t.Errorf("Discover nonexistent = %v, %v", f, p)
	}
	if f, p := manifestjson.Discover([]string{notADir}); len(f) != 0 || len(p) != 1 {
		t.Errorf("Discover regular file = %v, %v", f, p)
	}
	t.Chdir(tempDir)
	relFound, relProblems := manifestjson.Discover([]string{"."})
	if len(relProblems) != 1 || len(relFound) != 1 || !filepath.IsAbs(relFound[0].Dir) {
		t.Errorf("Discover relative = %v, %v", relFound, relProblems)
	}
}

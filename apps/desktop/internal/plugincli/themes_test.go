package plugincli

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// An archive holds the stylesheets the manifest declares, and validation fails for one that is missing.
func TestPluginPackage_ShipsDeclaredThemeFiles(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "themes-sample")
	files := map[string]string{
		"manifest.json": `{"id": "md.numen.themes-sample", "name": "Themes", "version": "1.0.0",
			"contributes": {"themes": [{"id": "night", "name": "Night", "file": "themes/night.css"}]}}`,
		"themes/night.css":  ":root {}",
		"themes/unused.css": ":root {}",
	}
	for name, content := range files {
		path := filepath.Join(pluginDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	archivePath, err := Package(pluginDir, "", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected package error: %v", err)
	}
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := zipReader.Close(); err != nil {
			t.Error(err)
		}
	})
	var zipped []string
	for _, f := range zipReader.File {
		zipped = append(zipped, f.Name)
	}
	slices.Sort(zipped)
	if want := []string{"manifest.json", "themes/night.css"}; !slices.Equal(zipped, want) {
		t.Errorf("archive holds %v, want %v", zipped, want)
	}

	if err := os.Remove(filepath.Join(pluginDir, "themes", "night.css")); err != nil {
		t.Fatal(err)
	}
	issues, err := Validate(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "night") {
		t.Errorf("issues = %v, want one naming the missing theme", issues)
	}
}

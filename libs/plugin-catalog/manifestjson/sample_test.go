package manifestjson_test

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSamplePlugin writes the folder of a plugin with a valid manifest under dir.
func writeSamplePlugin(t *testing.T, dir string) {
	t.Helper()
	pluginDir := filepath.Join(dir, "sample")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id": "org.example.sample", "name": "Sample Plugin", "version": "1.0.0",
		"targets": ["desktop.editor"], "backend": {"type": "process", "entrypoint": "bin/sample"}}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

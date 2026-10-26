package container

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestThePagesLoadOnlyThePluginsThatDeclareAFrontend(t *testing.T) {
	root := t.TempDir()
	manifests := map[string]string{
		"server": `{"id": "md.numen.server", "name": "Server", "version": "1.0.0", "targets": ["desktop.editor"],
			"backend": {"entrypoint": "bin/server"}}`,
		"view": `{"id": "md.numen.view", "name": "View", "version": "1.0.0", "targets": ["desktop.editor"],
			"frontend": {"entrypoint": "dist/index.js"}}`,
	}
	for dir, manifest := range manifests {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := Config{PluginHost: NewPluginHost("desktop.editor", root)}

	if got := cfg.GetFrontendPluginIDs(); !slices.Equal(got, []string{"md.numen.view"}) {
		t.Fatalf("GetFrontendPluginIDs() = %v, want [md.numen.view]", got)
	}
}

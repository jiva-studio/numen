package container

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

const sizedManifest = `{
	"id": "md.numen.sized",
	"name": "Sized",
	"version": "1.0.0",
	"capabilities": [],
	"targets": ["desktop.editor"],
	"permissions": [],
	"settings": [{"key": "size", "type": "number", "title": "Size", "default": 40, "min": 0, "max": 1000}]
}`

// The applied callback runs after the plugin host holds the new plugin setting.
func TestAppliedRunsAfterThePluginHostHasTheSetting(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sized")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(sizedManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewPluginHost("desktop.editor", root)
	cfg := NewConfig(Config{PluginHost: host, SettingsPath: filepath.Join(t.TempDir(), "numen.json")})

	values := make(chan any, 1)
	err := cfg.GetAppliedSettingsWriter()(
		[]settingsport.Setting{{Path: []string{"plugins", "md.numen.sized", "size"}, JSON: `0`}},
		func() { values <- host.GetPluginSettings("md.numen.sized")["size"] },
	)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case value := <-values:
		if value != float64(0) {
			t.Errorf("the plugin host had %v in applied, want 0", value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("applied did not run")
	}
}

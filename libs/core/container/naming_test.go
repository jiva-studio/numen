package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
)

// writeSettings is a settings file beside the vault list, which is where an
// installation pointed somewhere of its own keeps one.
func writeSettings(t *testing.T, body string) container.Config {
	t.Helper()
	target := t.TempDir()
	if body != "" {
		path := filepath.Join(target, "numen.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return container.Config{RegistryPath: filepath.Join(target, "vaults.json")}
}

// The setting is read at each rename, so changing it in the palette applies
// to the next rename and not only after a restart.
func TestTheSettingIsReadAsEachRenameIsMade(t *testing.T) {
	cfg := writeSettings(t, `{"naming":{"sync_title_and_filename":true}}`)
	setting := cfg.SyncSetting()
	if !setting.GetSetting() {
		t.Fatal("a title and a filename are distinguished")
	}

	path := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(path, []byte(`{"naming":{"sync_title_and_filename":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if setting.GetSetting() {
		t.Error("the setting turned under it and the rename read the old one")
	}
}

// A file that is empty, and one that cannot be read at all, keep the two
// one name.
func TestWhatIsReadWhereTheFileReturnsNothing(t *testing.T) {
	for name, body := range map[string]string{
		"a file nobody wrote":        "",
		"a file naming no section":   `{"appearance":{"text_scale":1.5}}`,
		"a section naming no field":  `{"naming":{}}`,
		"a file that does not parse": `{"naming":`,
	} {
		t.Run(name, func(t *testing.T) {
			if !writeSettings(t, body).SyncSetting().GetSetting() {
				t.Error("a title and a filename are distinguished")
			}
		})
	}
}

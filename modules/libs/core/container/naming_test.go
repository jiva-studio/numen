package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// writeSettings is a settings file beside the vault list, which is where an
// installation pointed somewhere of its own keeps one.
func writeSettings(t *testing.T, body string) container.Config {
	t.Helper()
	held := t.TempDir()
	if body != "" {
		path := filepath.Join(held, "numen.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return container.Config{RegistryPath: filepath.Join(held, "vaults.json")}
}

// The setting is read as each rename is made, so a person who turns it in the
// palette is answered by the next rename and not by the next launch.
func TestTheSettingIsReadAsEachRenameIsMade(t *testing.T) {
	cfg := writeSettings(t, `{"naming":{"sync_title_and_filename":true}}`)
	asking := cfg.SyncSetting()
	if !asking.Kept() {
		t.Fatal("a title and a filename are told apart")
	}

	path := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(path, []byte(`{"naming":{"sync_title_and_filename":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if asking.Kept() {
		t.Error("the setting turned under it and the rename read the old one")
	}
}

// A file that names nothing, and one that cannot be read at all, keep the two
// one name.
func TestWhatIsReadWhereTheFileSaysNothing(t *testing.T) {
	for name, body := range map[string]string{
		"a file nobody wrote":        "",
		"a file naming no section":   `{"appearance":{"text_scale":1.5}}`,
		"a section naming no field":  `{"naming":{}}`,
		"a file that does not parse": `{"naming":`,
	} {
		t.Run(name, func(t *testing.T) {
			if !writeSettings(t, body).SyncSetting().Kept() {
				t.Error("a title and a filename are told apart")
			}
		})
	}
}

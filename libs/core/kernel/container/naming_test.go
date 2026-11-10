package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func writeSettings(t *testing.T, body string) container.Config {
	t.Helper()
	target := t.TempDir()
	if body != "" {
		path := filepath.Join(target, "numen.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return container.NewConfig(container.Config{
		Paths: container.Paths{RegistryPath: filepath.Join(target, "vaults.json")},
	})
}

// The setting is read at each rename, so changing it in the palette applies
// to the next rename and not only after a restart.
func TestTheSettingIsReadAsEachRenameIsMade(t *testing.T) {
	cfg := writeSettings(t, `{"app":{"naming":{"syncTitleAndFilename":true}}}`)
	setting := container.SyncSetting(cfg)
	if !setting.GetSetting() {
		t.Fatal("a title and a filename are distinguished")
	}

	path := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(path, []byte(`{"app":{"naming":{"syncTitleAndFilename":false}}}`), 0o644); err != nil {
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
		"a file naming no section":   `{"app":{"appearance":{"textScale":1.5}}}`,
		"a section naming no field":  `{"app":{"naming":{}}}`,
		"a file that does not parse": `{"app":{"naming":`,
	} {
		t.Run(name, func(t *testing.T) {
			if !container.SyncSetting(writeSettings(t, body)).GetSetting() {
				t.Error("a title and a filename are distinguished")
			}
		})
	}
}

package settings_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	domaintheme "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// FuzzAt checks that a settings file yields either settings this build can run
// with or an error describing the file.
// Sizes and count are numbers their settings accept, the version is supported,
// and the mode is a word the window displays.
// The values are arbitrary user input, so none is assumed valid.
func FuzzAt(f *testing.F) {
	// The file the application writes, a size the setting does not go to, a word no half of a
	// color pair is called, a theme named nothing at all, and a version the file was never
	// written at.
	f.Add(1, "system", "preset:numen", 1.0, 1.0)
	f.Add(1, "dark", "mine:dracula", 2.0, 1.75)
	f.Add(1, "light", "preset:numen", 9.0, 1.0)
	f.Add(1, "purple", "preset:numen", 1.0, 1.0)
	f.Add(1, "", "", 1.0, 1.0)
	f.Add(0, "system", "preset:numen", 1.0, 1.0)
	f.Add(1, "system", "preset:numen", 0.8, 0.8)

	path := filepath.Join(f.TempDir(), "numen.json")
	f.Fuzz(func(t *testing.T, version int, mode, theme string,
		interfaceScale, textScale float64) {
		raw, err := json.Marshal(file{
			Version: version,
			App: app{Appearance: appearance{
				InterfaceScale: interfaceScale,
				TextScale:      textScale,
				Mode:           mode,
				Theme:          theme,
			}},
		})
		if err != nil {
			// A size that is not a number is not something a file can hold.
			return
		}

		target, err := openAt(writeFile(t, path, string(raw)))
		if err != nil {
			// The file is well formed and every value in it is of the kind its
			// setting takes, so the one thing left to refuse it for is a number
			// the setting does not go to — and the rejection names which.
			if !errors.As(err, new(*settingsdomain.OutsideBoundsError)) {
				t.Fatalf("%s failed as %v, which names no setting", raw, err)
			}
			return
		}

		if outside := target.Appearance.Check(); outside != nil {
			t.Fatalf("%s was read as settings the window is not drawn at: %v", raw, outside)
		}
		if target.Version < 1 {
			t.Fatalf("%s was read at version %d", raw, target.Version)
		}
		switch target.Appearance.Mode {
		case domaintheme.SchemeSystem, domaintheme.SchemeLight, domaintheme.SchemeDark:
		default:
			t.Fatalf("%s was read as %q, which is no half of a color pair",
				raw, target.Appearance.Mode)
		}
	})
}

// file mirrors the layout of the settings file, so a fuzzed value is stored
// under the key the reader looks up.
type file struct {
	Version int `json:"v"`
	App     app `json:"app"`
}

type app struct {
	Appearance appearance `json:"appearance"`
}

type appearance struct {
	InterfaceScale float64 `json:"interfaceScale"`
	TextScale      float64 `json:"textScale"`
	Mode           string  `json:"mode"`
	Theme          string  `json:"theme"`
}

func writeFile(tb testing.TB, path, raw string) string {
	tb.Helper()
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		tb.Fatal(err)
	}
	return path
}

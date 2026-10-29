package settings_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
)

// A settings file yields either settings this build can run with or an error
// describing what is wrong with the file. Both sizes and the count are numbers
// that their settings accept, the version is one the reader supports, and the
// mode is a word the window displays.
//
// When the file names a setting this build cannot use and the defaults apply,
// the reader reports it.
//
// The values are arbitrary user input between the braces, so none of them is
// assumed to be valid.
func FuzzAt(f *testing.F) {
	// The file the application writes, a size the setting does not go to, a
	// word no half of a colour pair is called, a theme named nothing at all, an
	// hour that is no hour and one with spaces around it, and a version the file
	// was never written at.
	f.Add(1, "system", "preset:numen", "04:00", 1.0, 1.0)
	f.Add(1, "dark", "mine:dracula", "23:59", 2.0, 1.75)
	f.Add(1, "light", "preset:numen", "04:00", 9.0, 1.0)
	f.Add(1, "purple", "preset:numen", "04:00", 1.0, 1.0)
	f.Add(1, "", "", "", 1.0, 1.0)
	f.Add(1, "system", "preset:numen", "99:99", 1.0, 1.0)
	f.Add(1, "system", "preset:numen", "  07:30  ", 1.0, 1.0)
	f.Add(0, "system", "preset:numen", "04:00", 1.0, 1.0)
	f.Add(1, "system", "preset:numen", "04:00", 0.8, 0.8)

	path := filepath.Join(f.TempDir(), "numen.json")
	f.Fuzz(func(t *testing.T, version int, mode, theme, dayStartTime string,
		interfaceScale, textScale float64) {
		raw, err := json.Marshal(file{
			Version: version,
			Appearance: appearance{
				InterfaceScale: interfaceScale,
				TextScale:      textScale,
				Mode:           mode,
				Theme:          theme,
			},
			Review: review{DayStartTime: dayStartTime},
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
			var outside *settings.OutsideBounds
			if !errors.As(err, &outside) {
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
		case settings.ModeSystem, settings.ModeLight, settings.ModeDark:
		default:
			t.Fatalf("%s was read as wearing %q, which is no half of a colour pair",
				raw, target.Appearance.Mode)
		}
	})
}

// file mirrors the layout of the settings file, so a fuzzed value is stored
// under the key the reader looks up.
type file struct {
	Version    int        `json:"v"`
	Appearance appearance `json:"appearance"`
	Review     review     `json:"review"`
}

type appearance struct {
	InterfaceScale float64 `json:"interface_scale"`
	TextScale      float64 `json:"text_scale"`
	Mode           string  `json:"mode"`
	Theme          string  `json:"theme"`
}

type review struct {
	DayStartTime string `json:"day_start_time"`
}

// is. One file is written over and over: a folder for each of a million inputs
// is a folder a run never gets to the end of.
func writeFile(tb testing.TB, path, raw string) string {
	tb.Helper()
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		tb.Fatal(err)
	}
	return path
}

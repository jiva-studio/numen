package settings_test

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// A file naming settings is answered for either with settings this build can be
// run on or with what is wrong with the file. Both sizes and the count are
// numbers their settings take, the version is one the file is read at, the mode
// is a word the window draws, and how large a recording may be is a size at
// least as large as the megabytes the person named.
//
// Where the file names a setting this build cannot use and the defaults stand,
// the reader says so.
//
// The values are what a person types between the braces, so each is that
// person's and none of them is this application's.
func FuzzAt(f *testing.F) {
	// The file the application writes, a size the setting does not go to, a
	// word no half of a colour pair is called, a theme named nothing at all, an
	// hour that is no hour and one with spaces around it, a version the file
	// was never written at, more megabytes than a size in bytes reaches, and no
	// limit at all.
	f.Add(1, "system", "preset:numen", "04:00", 1.0, 1.0, 6, 300)
	f.Add(1, "dark", "mine:dracula", "23:59", 2.0, 1.75, 12, 1)
	f.Add(1, "light", "preset:numen", "04:00", 9.0, 1.0, 6, 300)
	f.Add(1, "purple", "preset:numen", "04:00", 1.0, 1.0, 6, 300)
	f.Add(1, "", "", "", 1.0, 1.0, 6, 300)
	f.Add(1, "system", "preset:numen", "99:99", 1.0, 1.0, 6, 300)
	f.Add(1, "system", "preset:numen", "  07:30  ", 1.0, 1.0, 6, 300)
	f.Add(0, "system", "preset:numen", "04:00", 1.0, 1.0, 6, math.MaxInt64)
	f.Add(1, "system", "preset:numen", "04:00", 0.8, 0.8, 1, -1)

	path := filepath.Join(f.TempDir(), "numen.json")
	f.Fuzz(func(t *testing.T, version int, mode, theme, dayStartTime string,
		interfaceScale, textScale float64, headings, mb int) {
		raw, err := json.Marshal(file{
			Version: version,
			Appearance: appearance{
				InterfaceScale:     interfaceScale,
				TextScale:          textScale,
				Mode:               mode,
				Theme:              theme,
				NestedHeadingLimit: headings,
			},
			Indexing: indexing{TranscribeUnderMB: mb},
			Review:   review{DayStartTime: dayStartTime},
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
		if mb > 0 && target.Indexing.GetTranscribeLimit() < int64(mb) {
			t.Fatalf("%d megabytes was read as %d bytes", mb, target.Indexing.GetTranscribeLimit())
		}
	})
}

// file is the settings as the file lays them out, so that a fuzzed value stands
// in the file under the name the reader looks for it by.
type file struct {
	Version    int        `json:"v"`
	Appearance appearance `json:"appearance"`
	Indexing   indexing   `json:"indexing"`
	Review     review     `json:"review"`
}

type appearance struct {
	InterfaceScale     float64 `json:"interface_scale"`
	TextScale          float64 `json:"text_scale"`
	Mode               string  `json:"mode"`
	Theme              string  `json:"theme"`
	NestedHeadingLimit int     `json:"nested_heading_limit"`
}

type indexing struct {
	TranscribeUnderMB int `json:"transcribe_under_mb"`
}

type review struct {
	DayStartTime string `json:"day_start_time"`
}

// byteSeeds are the shapes a settings file arrives in: what the application
// writes, one setting indented another way, a name this build does not know, a
// name written twice, a value of the wrong kind, a file somebody was still
// typing, a bare null, the file as something other than an object, an empty
// object, an empty file, and bytes that are no settings at all.
var byteSeeds = []string{
	`{"v":1,"appearance":{"interface_scale":1,"text_scale":1,"mode":"dark",` +
		`"theme":"preset:numen","nest_headings_under_notes":true,"nested_heading_limit":6},` +
		`"review":{"day_start_time":"04:00"}}`,
	"{\n  \"appearance\": {\n    \"theme\": \"mine:dracula\"\n  }\n}\n",
	`{"appearance":{"zoom":1.25}}`,
	`{"appearance":{"interface_scale":1,"interface_scale":9}}`,
	`{"appearance":{"interface_scale":"1.5"}}`,
	`{"v":1,"appearance":{"text_sc`,
	"null",
	"[]",
	"{}\n",
	"",
	"\x00\xff\xfe",
}

// Bytes the settings are saved from land in the file as they were typed, and
// what lands is read back as settings: a file the application itself took is
// name, and the file is left holding what it held — a person's own settings are
// worth more than the one being written over them.
//
// The bytes come from an editor a person types into, so they are that person's
// and not this application's.
func FuzzWrite(f *testing.F) {
	for _, seed := range byteSeeds {
		f.Add(seed)
	}
	path := filepath.Join(f.TempDir(), "numen.json")
	f.Fuzz(func(t *testing.T, raw string) {
		const status = `{"appearance":{"theme":"mine:dracula"}}`
		writeFile(t, path, status)

		switch err := settings.Write(path, []byte(raw), nil, newDocument()); {
		case err == nil:
			if got := reading(t, path); got != raw {
				t.Fatalf("saved %q, found %q in file", raw, got)
			}
			if _, err := openAt(path); err != nil {
				t.Fatalf("%q was saved and reads back as %v", raw, err)
			}
		default:
			if !errors.Is(err, settingsport.ErrNotASetting) {
				t.Fatalf("%q failed as %v, which does not say what is wrong with it", raw, err)
			}
			if got := reading(t, path); got != status {
				t.Fatalf("%q failed and left %q in the file", raw, got)
			}
		}
	})
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

func reading(tb testing.TB, path string) string {
	tb.Helper()
	raw, err := settings.Read(path)
	if err != nil {
		tb.Fatal(err)
	}
	return string(raw)
}

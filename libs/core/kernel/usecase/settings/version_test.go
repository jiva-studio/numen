package settings_test

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// fileSettings is the settings use cases over a real file in a temporary folder.
type fileSettings struct {
	get   *settings.Get
	write *settings.WriteSettings
}

func openFileSettings(t *testing.T) fileSettings {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	store := settingsadapter.NewStore(settingsadapter.NewFiles(), func() (string, error) { return path, nil }, nil, nil)
	check := settings.NewCheckSettings(jsonCodec{}, declare([]string{"app", "appearance", "mode"}), nil, none.PluginSettings{})
	return fileSettings{
		get:   settings.NewGet(store.ReadJSON, nil, nil),
		write: settings.NewWriteSettings(check, store, noRefresh{}),
	}
}

func setMode(mode string) []settingsdomain.Setting {
	return []settingsdomain.Setting{{Path: []string{"app", "appearance", "mode"}, JSON: `"` + mode + `"`}}
}

func readMode(t *testing.T, written string) string {
	t.Helper()
	var whole struct {
		App struct {
			Appearance struct {
				Mode string `json:"mode"`
			} `json:"appearance"`
		} `json:"app"`
	}
	if err := json.Unmarshal([]byte(written), &whole); err != nil {
		t.Fatal(err)
	}
	return whole.App.Appearance.Mode
}

// A write raises the version, and reads with no write between them answer the same one.
func TestWriteRaisesTheVersionTheReadsAnswer(t *testing.T) {
	t.Parallel()
	use := openFileSettings(t)
	// The first read of a folder with no file writes the defaults, which is a write too.
	if _, err := use.get.Execute(); err != nil {
		t.Fatal(err)
	}
	before, err := use.get.Execute()
	if err != nil {
		t.Fatal(err)
	}
	again, err := use.get.Execute()
	if err != nil {
		t.Fatal(err)
	}

	if err := use.write.Execute(setMode("dark")); err != nil {
		t.Fatal(err)
	}
	after, err := use.get.Execute()
	if err != nil {
		t.Fatal(err)
	}

	if again.Version != before.Version {
		t.Errorf("two reads with no write between them answer versions %d and %d", before.Version, again.Version)
	}
	if after.Version <= before.Version {
		t.Errorf("the version after a write is %d, not above %d", after.Version, before.Version)
	}
}

// A read answers the version of the state it read: the value written and the version that goes with it.
func TestGetAnswersTheVersionOfTheStateItRead(t *testing.T) {
	t.Parallel()
	use := openFileSettings(t)
	first, second := writeThenRead(t, use, "dark"), writeThenRead(t, use, "light")

	if first.mode != "dark" || second.mode != "light" {
		t.Errorf("reads answer the modes %q and %q, want dark and light", first.mode, second.mode)
	}
	if second.version <= first.version {
		t.Errorf("the versions are %d and %d, the second is not above the first", first.version, second.version)
	}
}

type observed struct {
	mode    string
	version uint64
}

func writeThenRead(t *testing.T, use fileSettings, mode string) observed {
	t.Helper()
	if err := use.write.Execute(setMode(mode)); err != nil {
		t.Fatal(err)
	}
	got, err := use.get.Execute()
	if err != nil {
		t.Fatal(err)
	}
	return observed{mode: readMode(t, got.Written), version: got.Version}
}

// A write answers the new value and the version, the same that a read after it answers.
func TestWriteAnswersTheNewValueAndVersion(t *testing.T) {
	t.Parallel()
	use := openFileSettings(t)
	before, err := use.get.Execute()
	if err != nil {
		t.Fatal(err)
	}

	applied, err := use.write.Write(setMode("dark"))
	if err != nil {
		t.Fatal(err)
	}
	read, err := use.get.Execute()
	if err != nil {
		t.Fatal(err)
	}

	if readMode(t, applied.Written) != "dark" {
		t.Errorf("the write answered %s, want the mode dark", applied.Written)
	}
	if applied.Version <= before.Version {
		t.Errorf("the write answered version %d, not above %d", applied.Version, before.Version)
	}
	if applied.Version != read.Version || applied.Written != read.Written {
		t.Errorf("the write answered %d %s, the read after it %d %s", applied.Version, applied.Written, read.Version, read.Written)
	}
}

// Writes that race each answer the value and the version of their own write, never of the other.
func TestRacingWritesAnswerTheirOwnState(t *testing.T) {
	t.Parallel()
	use := openFileSettings(t)
	const rounds = 40
	type answer struct {
		mode    string
		version uint64
	}
	modes := []string{"dark", "light"}
	answers := make([][]answer, len(modes))
	var wg sync.WaitGroup
	for i, mode := range modes {
		wg.Go(func() {
			for range rounds {
				applied, err := use.write.Write(setMode(mode))
				if err != nil {
					t.Error(err)
					return
				}
				answers[i] = append(answers[i], answer{readMode(t, applied.Written), applied.Version})
			}
		})
	}
	wg.Wait()

	seen := map[uint64]bool{}
	for i, mode := range modes {
		for _, one := range answers[i] {
			if one.mode != mode {
				t.Fatalf("the write of %q answered the state of %q", mode, one.mode)
			}
			if seen[one.version] {
				t.Fatalf("version %d was answered to two writes", one.version)
			}
			seen[one.version] = true
		}
	}
}

// noRefresh is a refresher that keeps nothing, so writes on several goroutines share no state through it.
type noRefresh struct{}

func (noRefresh) Refresh() {}

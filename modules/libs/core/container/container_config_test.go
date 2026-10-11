package container

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// isolate points everything the platform keeps per user at folders of the
// test's own.
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	return home
}

// isolateWithNoHome leaves the platform no folder to keep anything per user in.
func isolateWithNoHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
}

func TestTheClockIsTheOneTheInstallationNamed(t *testing.T) {
	named := time.Date(2020, 2, 3, 4, 5, 6, 0, time.UTC)
	if got := (Config{Now: func() time.Time { return named }}).Clock()(); !got.Equal(named) {
		t.Errorf("the clock reads %v, and the installation named %v", got, named)
	}
}

func TestAnInstallationNamingNoClockReadsTheMachinesOwn(t *testing.T) {
	before := time.Now()
	got := Config{}.Clock()()
	if got.Before(before) || got.After(time.Now()) {
		t.Errorf("the clock reads %v, and it is %v", got, before)
	}
}

func TestSettingsThatCannotBeReadAreSaidWithTheFileTheyStandIn(t *testing.T) {
	cfg, path := writeSettings(t, `{not json`)

	written, said, err := cfg.ReadSettings()()
	if err == nil {
		t.Fatalf("settings that are not JSON were read out as %q", written)
	}
	if said != path {
		t.Errorf("the settings stand at %q, not %q", said, path)
	}
}

func TestWithNoFolderForSettingsNothingIsReadOrWritten(t *testing.T) {
	isolateWithNoHome(t)

	if _, said, err := (Config{}).ReadSettings()(); err == nil || said != "" {
		t.Errorf("settings were read from %q: %v", said, err)
	}
	if _, said, err := (Config{}).ReadSettingsFile()(); err == nil || said != "" {
		t.Errorf("the settings file was read from %q: %v", said, err)
	}
	err := Config{}.TurnsSetting()([]port.Setting{
		{Path: []string{"agent", "claude", "model"}, JSON: `"opus"`},
	})
	if err == nil {
		t.Error("a setting was written with nowhere to write it")
	}
}

func TestTheSettingsStandInTheUsersConfigFolderUnlessPointedElsewhere(t *testing.T) {
	isolate(t)
	want, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(want, "numen", "numen.json")

	got, err := Config{}.settingsFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("the settings stand at %q, not %q", got, want)
	}
}

func TestTheSettingsStandBesideTheRegistryWhenItIsPointedElsewhere(t *testing.T) {
	isolate(t)
	registry := filepath.Join(t.TempDir(), "registry.json")

	got, err := Config{RegistryPath: registry}.settingsFile()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(registry), "numen.json"); got != want {
		t.Errorf("the settings stand at %q, not %q", got, want)
	}
}

func TestVaultsAreOpenedForWritingWithTheOptionsOfTheInstallation(t *testing.T) {
	cfg := Config{ServiceDir: ".mine", BookExtensions: []string{".pdf", ".epub"}}

	opened, ok := cfg.VaultWriters().(filesystem.VaultWriters)
	if !ok {
		t.Fatalf("vaults are opened for writing by %T", cfg.VaultWriters())
	}
	if opened.Options.ServiceDir != ".mine" || !slices.Equal(opened.Options.BookExtensions, cfg.BookExtensions) {
		t.Errorf("vaults are opened with %+v", opened.Options)
	}
	read, ok := cfg.VaultReaders().(filesystem.VaultReaders)
	if !ok {
		t.Fatalf("vaults are opened for reading by %T", cfg.VaultReaders())
	}
	if read.Options.ServiceDir != ".mine" || !slices.Equal(read.Options.BookExtensions, cfg.BookExtensions) {
		t.Errorf("vaults are opened with %+v", read.Options)
	}
}

func TestTheIndexIsWhereTheInstallationNamedItAndItsFolderIsMade(t *testing.T) {
	isolate(t)
	named := filepath.Join(t.TempDir(), "nested", "deeper", "index.db")

	got, err := Config{IndexPath: named}.IndexPathOrDefault()
	if err != nil {
		t.Fatal(err)
	}
	if got != named {
		t.Errorf("the index is at %q, not %q", got, named)
	}
	if info, err := os.Stat(filepath.Dir(named)); err != nil || !info.IsDir() {
		t.Errorf("the folder of the index was not made: %v", err)
	}
}

func TestAnIndexWhoseFolderCannotBeMadeIsAReason(t *testing.T) {
	isolate(t)
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Config{IndexPath: filepath.Join(file, "index.db")}.IndexPathOrDefault()
	if err == nil || got != "" {
		t.Errorf("the index is at %q: %v", got, err)
	}
	if _, err := (Config{IndexPath: filepath.Join(file, "index.db")}).OpenIndex(t.Context()); err == nil {
		t.Error("an index was opened where its folder cannot be made")
	}
}

func TestAnIndexNobodyNamedIsInTheUsersCacheFolder(t *testing.T) {
	isolate(t)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}

	got, err := Config{}.IndexPathOrDefault()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cache, "numen", "index.db"); got != want {
		t.Errorf("the index is at %q, not %q", got, want)
	}
	if info, err := os.Stat(filepath.Dir(got)); err != nil || !info.IsDir() {
		t.Errorf("the folder of the index was not made: %v", err)
	}
}

func TestAnIndexIsNotMadeWhereTheCacheFolderCannotBeMade(t *testing.T) {
	isolate(t)
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", file)
	t.Setenv("HOME", file)

	if got, err := (Config{}).IndexPathOrDefault(); err == nil || got != "" {
		t.Errorf("the index is at %q: %v", got, err)
	}
	isolateWithNoHome(t)
	if got, err := (Config{}).IndexPathOrDefault(); err == nil || got != "" {
		t.Errorf("the index is at %q: %v", got, err)
	}
}

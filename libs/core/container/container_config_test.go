package container

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
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

	content, response, err := cfg.ReadSettings()()
	if err == nil {
		t.Fatalf("settings that are not JSON were read out as %q", content)
	}
	if response != path {
		t.Errorf("the settings stand at %q, not %q", response, path)
	}
}

func TestWithNoFolderForSettingsNothingIsReadOrSaved(t *testing.T) {
	isolateWithNoHome(t)

	if _, response, err := (Config{}).ReadSettings()(); err == nil || response != "" {
		t.Errorf("settings were read from %q: %v", response, err)
	}
	err := Config{}.GetSettingsWriter()([]settingsport.Setting{
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

func TestTheSettingsStandAdjacentTheRegistryWhenItIsPointedElsewhere(t *testing.T) {
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
	cfg := Config{ServiceDir: ".mine"}

	opened, ok := cfg.VaultWriters().(vaultwriter.VaultWriters)
	if !ok {
		t.Fatalf("vaults are opened for writing by %T", cfg.VaultWriters())
	}
	if opened.Options.ServiceDir != ".mine" {
		t.Errorf("vaults are opened with %+v", opened.Options)
	}
	read, ok := cfg.VaultReaders().(vaultreader.Readers)
	if !ok {
		t.Fatalf("vaults are opened for reading by %T", cfg.VaultReaders())
	}
	if read.Options.ServiceDir != ".mine" {
		t.Errorf("vaults are opened with %+v", read.Options)
	}
}

func TestTheIndexIsWhereTheInstallationNamedItAndItsFolderIsMade(t *testing.T) {
	isolate(t)
	named := filepath.Join(t.TempDir(), "nested", "deeper", "index.db")

	got, err := Config{IndexPath: named}.indexPath()
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

	got, err := Config{IndexPath: filepath.Join(file, "index.db")}.indexPath()
	if err == nil || got != "" {
		t.Errorf("the index is at %q: %v", got, err)
	}
	if _, err := (Config{IndexPath: filepath.Join(file, "index.db")}).OpenIndex(t.Context()); err == nil {
		t.Error("an index was opened where its folder cannot be made")
	}
}

func TestWhatWentWrongIsReportedToWhoeverAskedToBeReported(t *testing.T) {
	var notified []error
	problem := errors.New("a vacuum that could not run")

	Config{ErrorHandler: func(err error) { notified = append(notified, err) }}.handleError(problem)
	if len(notified) != 1 || !errors.Is(notified[0], problem) {
		t.Errorf("told %v", notified)
	}
	Config{}.handleError(problem)
}

func TestWithNoFolderForSettingsTheFileIsNotSaved(t *testing.T) {
	isolateWithNoHome(t)

	err := Config{}.GetSettingsWriter()([]settingsport.Setting{{Path: []string{"agent", "claude", "model"}, JSON: `"opus"`}})
	if err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Errorf("got %v", err)
	}
}

func TestASettingThatIsNotJSONIsNotASetting(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)

	err := cfg.GetSettingsWriter()([]settingsport.Setting{{Path: []string{"agent", "claude", "model"}, JSON: `opus`}})
	if !errors.Is(err, settingsport.ErrNotASetting) {
		t.Errorf("got %v", err)
	}
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Errorf("what was wrong with it is not said: %v", err)
	}
	got, gotErr := os.ReadFile(path)
	if gotErr != nil {
		t.Fatal(gotErr)
	}
	if string(got) != `{}` {
		t.Errorf("the file holds %q", got)
	}
}

func TestTheRegistryIsWhereTheInstallationPointedItAndOtherwiseInTheConfigFolder(t *testing.T) {
	isolate(t)
	named := filepath.Join(t.TempDir(), "vaults.json")
	type placed interface{ Path() string }

	got, err := Config{RegistryPath: named}.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if pos, ok := got.(placed); !ok || pos.Path() != named {
		t.Errorf("the registry is %#v, and it was pointed at %q", got, named)
	}

	got, err = Config{}.Registry()
	if err != nil {
		t.Fatal(err)
	}
	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if offset, ok := got.(placed); !ok || offset.Path() != filepath.Join(config, "numen", "vaults.json") {
		t.Errorf("the registry is %#v", got)
	}
}

func TestEveryFolderOfTheMachineIsGivenAnIdentityAndATrash(t *testing.T) {
	isolate(t)
	if (Config{}).VaultIdentity() == nil {
		t.Error("folders are given no identity")
	}
	if (Config{}).Trash() == nil {
		t.Error("what is deleted has nowhere to go")
	}
}

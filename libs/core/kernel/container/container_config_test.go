package container

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	return home
}

func isolateWithNoHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
}

func TestTheClockIsTheOneTheInstallationNamed(t *testing.T) {
	named := time.Date(2020, 2, 3, 4, 5, 6, 0, time.UTC)
	resources := NewResources()
	resources.Now = testsupport.NewClockAt(named)
	if got := GetClock(resources).Now(); !got.Equal(named) {
		t.Errorf("the clock reads %v, and the installation named %v", got, named)
	}
}

func TestAnInstallationNamingNoClockReadsTheMachinesOwn(t *testing.T) {
	before := testsupport.GetMachine().Now()
	got := GetClock(NewResources()).Now()
	if got.Before(before) || got.After(testsupport.GetMachine().Now()) {
		t.Errorf("the clock reads %v, and it is %v", got, before)
	}
}

func TestReadSettings_InvalidJSONReturnsPath(t *testing.T) {
	cfg, path := writeSettings(t, `{not json`)

	snapshot, err := createSettingsStore(cfg.buildSettingsWiring()).ReadJSON()
	content, response := snapshot.Content, snapshot.Path
	if err == nil {
		t.Fatalf("settings that are not JSON were read out as %q", content)
	}
	if response != path {
		t.Errorf("the settings stand at %q, not %q", response, path)
	}
}

func TestWithNoFolderForSettingsNothingIsReadOrSaved(t *testing.T) {
	isolateWithNoHome(t)

	if snapshot, err := createSettingsStore(createSettingsWiring(Config{}, nil, nil, NewResources())).ReadJSON(); err == nil || snapshot.Path != "" {
		t.Errorf("settings were read from %q: %v", snapshot.Path, err)
	}
	err := getSettingsWriter(createSettingsWiring(Config{}, nil, nil, NewResources()))([]settings.Setting{
		{Path: []string{"app", "naming", "syncTitleAndFilename"}, JSON: `false`},
	})
	if err == nil {
		t.Error("a setting was written with nowhere to write it")
	}
}

func TestSettingsFile_DefaultsToUserConfigDir(t *testing.T) {
	isolate(t)
	want, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(want, "numen", "numen.json")

	got, err := getSettingsFile(createSettingsWiring(Config{}, nil, nil, NewResources()))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("the settings stand at %q, not %q", got, want)
	}
}

func TestSettingsFile_FollowsRegistryDirectory(t *testing.T) {
	isolate(t)
	registry := filepath.Join(t.TempDir(), "registry.json")

	got, err := getSettingsFile(createSettingsWiring(Config{
		Paths: Paths{RegistryPath: registry},
	}, nil, nil, NewResources()))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(registry), "numen.json"); got != want {
		t.Errorf("the settings stand at %q, not %q", got, want)
	}
}

func TestVaultsAreOpenedForWritingWithTheOptionsOfTheInstallation(t *testing.T) {
	cfg := NewTestInstallation(Config{Paths: Paths{ServiceDir: ".mine"}}, nil)

	opened := cfg.buildVaultWiring().Writers
	if opened.Options.ServiceDir != ".mine" {
		t.Errorf("vaults are opened with %+v", opened.Options)
	}
	read, ok := cfg.buildVaultWiring().Readers.(vaultreader.Readers)
	if !ok {
		t.Fatalf("vaults are opened for reading by %T", cfg.buildVaultWiring().Readers)
	}
	if read.Options.ServiceDir != ".mine" {
		t.Errorf("vaults are opened with %+v", read.Options)
	}
}

func TestTheIndexIsWhereTheInstallationNamedItAndItsFolderIsMade(t *testing.T) {
	isolate(t)
	named := filepath.Join(t.TempDir(), "nested", "deeper", "index.db")

	got, err := prepareIndexPath(NewTestInstallation(Config{Paths: Paths{IndexPath: named}}, nil).buildVaultWiring())
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

	got, err := prepareIndexPath(NewTestInstallation(Config{
		Paths: Paths{IndexPath: filepath.Join(file, "index.db")},
	}, nil).buildVaultWiring())
	if err == nil || got != "" {
		t.Errorf("the index is at %q: %v", got, err)
	}
	if _, err := openIndex(NewTestInstallation(Config{
		Paths: Paths{IndexPath: filepath.Join(file, "index.db")},
	}, nil).buildVaultWiring(), t.Context()); err == nil {
		t.Error("an index was opened where its folder cannot be made")
	}
}

func TestHandleError_CallsConfiguredHandler(t *testing.T) {
	var notified []error
	problem := errors.New("a vacuum that could not run")

	resources := NewResources()
	resources.ErrorHandler = func(err error) { notified = append(notified, err) }
	getErrorHandler("", resources)(problem)
	if len(notified) != 1 || !errors.Is(notified[0], problem) {
		t.Errorf("told %v", notified)
	}
	getErrorHandler("", NewResources())(problem)
}

func TestWithNoFolderForSettingsTheFileIsNotSaved(t *testing.T) {
	isolateWithNoHome(t)

	err := getSettingsWriter(createSettingsWiring(Config{}, nil, nil, NewResources()))([]settings.Setting{{Path: []string{"app", "naming", "syncTitleAndFilename"}, JSON: `false`}})
	if err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Errorf("got %v", err)
	}
}

func TestASettingThatIsNotJSONIsNotASetting(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)

	err := getSettingsWriter(cfg.buildSettingsWiring())([]settings.Setting{{Path: []string{"app", "naming", "syncTitleAndFilename"}, JSON: `nope`}})
	if !errors.Is(err, settings.ErrNotASetting) {
		t.Errorf("got %v", err)
	}
	if !errors.As(err, new(*json.SyntaxError)) {
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

func TestRegistry_UsesConfiguredPathElseConfigDir(t *testing.T) {
	isolate(t)
	named := filepath.Join(t.TempDir(), "vaults.json")
	type placed interface{ Path() string }

	got, err := openVaultRegistry(named)
	if err != nil {
		t.Fatal(err)
	}
	if pos, ok := got.Vaults.(placed); !ok || pos.Path() != named {
		t.Errorf("the registry is %#v, and it was pointed at %q", got, named)
	}

	got, err = openVaultRegistry("")
	if err != nil {
		t.Fatal(err)
	}
	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if offset, ok := got.Vaults.(placed); !ok || offset.Path() != filepath.Join(config, "numen", "vaults.json") {
		t.Errorf("the registry is %#v", got)
	}
}

func TestEveryFolderOfTheMachineIsGivenAnIdentityAndATrash(t *testing.T) {
	isolate(t)
	if createVaultIdentity(NewTestInstallation(Config{}, nil).buildVaultWiring().Options) == nil {
		t.Error("folders are given no identity")
	}
	if createTrash(NewTestInstallation(Config{}, nil).buildVaultWiring().Clock) == nil {
		t.Error("what is deleted has nowhere to go")
	}
}

func TestTheVaultWaitsOnTheClockTheInstallationNamed(t *testing.T) {
	named := testsupport.NewClock()
	resources := NewResources()
	resources.Now = named
	quiet := openVaultAccess(Paths{}, GetClock(resources), resources.vault.locks, resources.vault.watches, nil).Clock.StartTimer(indexing.QuietPeriod)
	named.Advance(indexing.QuietPeriod)
	select {
	case <-quiet:
	default:
		t.Fatal("the quiet period did not end when the named clock passed it")
	}
}

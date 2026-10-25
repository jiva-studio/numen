package container

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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
	if _, responseVal, err := (Config{}).ReadSettingsFile()(); err == nil || responseVal != "" {
		t.Errorf("the settings file was read from %q: %v", responseVal, err)
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
	cfg := Config{ServiceDir: ".mine", BookExtensions: []string{".pdf", ".epub"}}

	opened, ok := cfg.VaultWriters().(vaultwriter.VaultWriters)
	if !ok {
		t.Fatalf("vaults are opened for writing by %T", cfg.VaultWriters())
	}
	if opened.Options.ServiceDir != ".mine" || !slices.Equal(opened.Options.BookExtensions, cfg.BookExtensions) {
		t.Errorf("vaults are opened with %+v", opened.Options)
	}
	read, ok := cfg.VaultReaders().(vaultreader.Readers)
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

func TestWhatWentWrongIsReportedToWhoeverAskedToBeReported(t *testing.T) {
	var notified []error
	problem := errors.New("a vacuum that could not run")

	Config{ErrorHandler: func(err error) { notified = append(notified, err) }}.handleError(problem)
	if len(notified) != 1 || !errors.Is(notified[0], problem) {
		t.Errorf("told %v", notified)
	}
	Config{}.handleError(problem)
}

func TestTheSettingsFileIsReadOutAsItsPersonWroteIt(t *testing.T) {
	const typed = `{"appearance": {"theme": "mine:sea"}}`
	cfg, path := writeSettings(t, typed)

	raw, response, err := cfg.ReadSettingsFile()()
	if err != nil {
		t.Fatal(err)
	}
	if raw != typed || response != path {
		t.Errorf("read %q at %q", raw, response)
	}
}

func TestASettingsFileThatCannotBeReadIsSaidWithWhereItExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}

	raw, response, err := Config{SettingsPath: path}.ReadSettingsFile()()
	if err == nil || raw != "" || response != path {
		t.Errorf("read %q at %q: %v", raw, response, err)
	}
}

func TestTheSettingsFileIsReplacedWholeByWhatWasTyped(t *testing.T) {
	cfg, path := writeSettings(t, `{"appearance": {"theme": "mine:sea"}}`)
	const typed = `{"appearance": {"theme": "mine:sand"}}`

	if err := cfg.WriteConfiguredFile()(typed, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != typed {
		t.Errorf("the file holds %q: %v", got, err)
	}
}

func TestWriteConfiguredFileUpdatesPluginSettings(t *testing.T) {
	cfg, _ := writeSettings(t, `{"appearance": {"theme": "mine:sea"}}`)
	tmp := t.TempDir()
	host := NewPluginHost("desktop.editor", tmp)
	cfg.PluginHost = host

	const typed = `{"appearance": {"theme": "mine:sand"}, "plugins": {"md.numen.pdf": {"engine": "mupdf", "max_pages": 50}}}`
	if err := cfg.WriteConfiguredFile()(typed, nil); err != nil {
		t.Fatal(err)
	}

	pluginSettings := host.GetPluginSettings("md.numen.pdf")
	if got := pluginSettings["engine"]; got != "mupdf" {
		t.Errorf("expected engine mupdf, got %v", got)
	}
	if got := pluginSettings["max_pages"]; got != float64(50) && got != 50 {
		t.Errorf("expected max_pages 50, got %v", got)
	}
}

func TestTheSettingsFileIsLeftAloneWhenItMovedSinceItWasRead(t *testing.T) {
	const standing = `{"appearance": {"theme": "mine:sea"}}`
	cfg, path := writeSettings(t, standing)
	baseline := `{"appearance": {"theme": "mine:forest"}}`

	err := cfg.WriteConfiguredFile()(`{"appearance": {"theme": "mine:sand"}}`, &baseline)
	if !errors.Is(err, vaultport.ErrStale) {
		t.Errorf("got %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != standing {
		t.Errorf("the file holds %q", got)
	}
}

func TestSettingsTheFileCannotBeReadOutOfAreRefusedAndTheFileIsCached(t *testing.T) {
	const standing = `{"appearance": {"theme": "mine:sea"}}`
	cfg, path := writeSettings(t, standing)

	if err := cfg.WriteConfiguredFile()(`{not json`, nil); err == nil {
		t.Error("bytes that are not JSON were written")
	}
	if got, _ := os.ReadFile(path); string(got) != standing {
		t.Errorf("the file holds %q", got)
	}
}

func TestWithNoFolderForSettingsTheFileIsNotSaved(t *testing.T) {
	isolateWithNoHome(t)

	err := Config{}.WriteConfiguredFile()(`{}`, nil)
	if err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Errorf("got %v", err)
	}
	err = Config{}.GetSettingsWriter()([]settingsport.Setting{{Path: []string{"agent", "claude", "model"}, JSON: `"opus"`}})
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
	if got, _ := os.ReadFile(path); string(got) != `{}` {
		t.Errorf("the file holds %q", got)
	}
}

func TestTheBoundsOnNestedHeadingLimitAreTheSettingsOwn(t *testing.T) {
	got := Config{}.NestedHeadingLimitBounds()
	if got != settings.NestedHeadingLimitBounds || got == (settings.Bounds{}) {
		t.Errorf("got %+v", got)
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

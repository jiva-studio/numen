package container

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/theme"

	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func writeSettings(t *testing.T, output string) (TestInstallation, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		t.Fatal(err)
	}
	return NewTestInstallation(Config{Paths: Paths{SettingsPath: path}}, nil), path
}

// What the window is given is every setting, so a setting the file leaves out
// is read out as what this installation is doing about it.
func TestTheSettingsReadOutHoldWhatTheFilePreservesOut(t *testing.T) {
	cfg, path := writeSettings(t, `{"app": {"appearance": {"theme": "mine:sea"}}}`)

	snapshot, err := settingsStore(cfg.buildSettingsWiring()).ReadJSON()
	content, response := snapshot.Content, snapshot.Path
	if err != nil {
		t.Fatal(err)
	}
	if response != path {
		t.Errorf("the settings stand at %q, not %q", response, path)
	}

	var target settings.File
	if err := json.Unmarshal([]byte(content), &target); err != nil {
		t.Fatal(err)
	}
	if target.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", target.Appearance.Theme)
	}
	if target.Appearance.Mode != theme.SchemeSystem {
		t.Errorf("the mode reads %q", target.Appearance.Mode)
	}
}

// A written setting is stored at its current position, and every key the user
// added stays in place.
func TestASettingSavedPreservesTheRestOfTheFileAlone(t *testing.T) {
	cfg, path := writeSettings(t, `{
  "app": {"appearance": {"theme": "mine:sea"}},
  "something_this_build_knows_nothing_about": 7
}`)

	err := getSettingsWriter(cfg.buildSettingsWiring())([]settingsdomain.Setting{
		{Path: []string{"app", "naming", "syncTitleAndFilename"}, JSON: `false`},
	})
	if err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "something_this_build_knows_nothing_about") {
		t.Errorf("the file came back as %s", after)
	}

	target, err := settingsGetSettingsAt(cfg.buildSettingsWiring(), path)
	if err != nil {
		t.Fatal(err)
	}
	if target.Sync() {
		t.Error("the titles and filenames are still kept as one name")
	}
	if target.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", target.Appearance.Theme)
	}
}

// Several settings are written together.
func TestSettingsSavedTogetherAllArrive(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)

	err := getSettingsWriter(cfg.buildSettingsWiring())([]settingsdomain.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
		{Path: []string{"app", "naming", "syncTitleAndFilename"}, JSON: `false`},
	})
	if err != nil {
		t.Fatal(err)
	}

	target, err := settingsGetSettingsAt(cfg.buildSettingsWiring(), path)
	if err != nil {
		t.Fatal(err)
	}
	if target.Appearance.Mode != "dark" || target.Sync() {
		t.Errorf("the settings read %+v and %+v", target.Appearance, target.Titles)
	}
}

// A machine with no location for settings returns an error and no settings.
func TestSettingsWithNowhereToLiveAreRefused(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	target, why := NewTestInstallation(Config{}, nil).Settings()
	if why == nil {
		t.Fatal("no reason")
	}
	if !reflect.DeepEqual(target, settings.File{}) {
		t.Errorf("got settings: %+v", target)
	}
}

func TestASettingsFileThatIsNotReadableIsRefused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg, _ := writeSettings(t, `{"app": {"appearance": `)

	target, why := cfg.Settings()
	if why == nil {
		t.Fatal("no reason")
	}
	if !reflect.DeepEqual(target, settings.File{}) {
		t.Errorf("got settings: %+v", target)
	}
}

func TestSettingsThatCannotBeSavedAreRefused(t *testing.T) {
	target := DefaultSettings()
	target.Appearance.InterfaceScale = math.NaN()

	content, why := target.WriteJSON()
	if why == nil {
		t.Fatal("no reason")
	}
	if content != "" {
		t.Errorf("wrote %q", content)
	}
}

// Every section of the settings arrives in the configuration, and the compute
// plan is resolved from the performance section.
func TestAConfigurationCarriesEverySectionOfTheSettings(t *testing.T) {
	response := DefaultSettings()
	response.Performance = settings.Performance{Profile: machinedomain.ProfileEco}

	got := NewTestInstallation(Config{}, nil).SetSettings(response)

	if got.GetMachine().ResolvedCompute.Profile != machinedomain.ProfileEco || got.GetMachine().ResolvedCompute.Workers != 1 {
		t.Errorf("the compute plan is %+v", got.GetMachine().ResolvedCompute)
	}
}

// A notice is stored for the settings reader to display to the user.
func TestSettings_AddNoticeAccumulatesNotices(t *testing.T) {
	var target settings.File
	target.AddNotice("first")
	target.AddNotice("second")
	if !slices.Equal(target.Notices, []string{"first", "second"}) {
		t.Errorf("said %v", target.Notices)
	}
}

// profileHost records the profiles the container gives the plugin host.
type profileHost struct {
	settingsusecase.PluginConfig
	profiles []string
}

func (p *profileHost) SetPerformance(_ context.Context, performance plugindomain.Performance) {
	p.profiles = append(p.profiles, performance.Profile)
}

func TestWritingAnotherSettingSendsNoProfile(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)
	host := &profileHost{}
	cfg.TestPlugins = &TestPlugins{Preferences: &PluginPreferences{Configuration: pluginConfiguration{PluginConfig: host}}}

	err := getSettingsWriter(cfg.buildSettingsWiring())([]settingsdomain.Setting{{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`}})
	if err != nil {
		t.Fatal(err)
	}
	if len(host.profiles) != 0 {
		t.Errorf("profiles sent to the host = %v, want none", host.profiles)
	}
}

// A configuration copied before the settings were applied keeps the plan it had.
func TestApplySettingsKeepsTheOriginalConfiguration(t *testing.T) {
	response := DefaultSettings()
	response.Performance = settings.Performance{Profile: machinedomain.ProfileEco}
	before := NewTestInstallation(Config{}, nil)

	after := before.SetSettings(response)

	if after.GetMachine().ResolvedCompute.Workers != 1 {
		t.Fatalf("the applied configuration plans %+v", after.GetMachine().ResolvedCompute)
	}
	if got := before.GetMachine().ResolvedCompute; got.Workers != 0 || got.Profile != "" {
		t.Errorf("the configuration the call was made on changed to %+v", got)
	}
}

func TestThePluginsReceiveTheBudgetAndTheNicenessTheHostResolved(t *testing.T) {
	hw := machinedomain.Specs{LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8}
	got := getPluginPerformance(machinedomain.ResolveCompute(machinedomain.ProfileHigh, hw))

	if want := (plugindomain.Performance{Profile: "high", Threads: 12, Nice: 5}); got != want {
		t.Errorf("the plugins receive %+v, want %+v", got, want)
	}
}

func TestEcoReachesThePluginsAsOneThreadAtTheLowestPriority(t *testing.T) {
	if got := resolveProfile(machinedomain.ProfileEco); got != (plugindomain.Performance{Profile: "eco", Threads: 1, Nice: 19}) {
		t.Errorf("eco resolves to %+v", got)
	}
}

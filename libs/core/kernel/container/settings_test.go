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
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/theme"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

func writeSettings(t *testing.T, output string) (Config, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		t.Fatal(err)
	}
	return NewConfig(Config{
		Paths: Paths{SettingsPath: path},
	}), path
}

// What the window is given is every setting, so a setting the file leaves out
// is read out as what this installation is doing about it.
func TestTheSettingsReadOutHoldWhatTheFilePreservesOut(t *testing.T) {
	cfg, path := writeSettings(t, `{"app": {"appearance": {"theme": "mine:sea"}}}`)

	content, response, err := readSettings(settingsOf(cfg))()
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

	err := getSettingsWriter(settingsOf(cfg))([]settingsdomain.Setting{
		{Path: []string{"app", "agent", "claude", "model"}, JSON: `"opus"`},
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

	target, err := settingsGetSettingsAt(settingsOf(cfg), path)
	if err != nil {
		t.Fatal(err)
	}
	if target.Agent.GetProgram("claude").Model != "opus" {
		t.Errorf("the model reads %q", target.Agent.GetProgram("claude").Model)
	}
	if target.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", target.Appearance.Theme)
	}
}

// Several settings are written together, as when choosing one model that
// determines more than its own name.
func TestSettingsSavedTogetherAllArrive(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)

	err := getSettingsWriter(settingsOf(cfg))([]settingsdomain.Setting{
		{Path: []string{"app", "agent", "use"}, JSON: `"claude"`},
		{Path: []string{"app", "agent", "claude", "model"}, JSON: `"opus"`},
	})
	if err != nil {
		t.Fatal(err)
	}

	target, err := settingsGetSettingsAt(settingsOf(cfg), path)
	if err != nil {
		t.Fatal(err)
	}
	if target.Agent.GetUse() != "claude" || target.Agent.GetProgram("claude").Model != "opus" {
		t.Errorf("the agent reads %+v", target.Agent)
	}
}

// A machine with no location for settings returns an error and no settings.
func TestSettingsWithNowhereToLiveAreRefused(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	target, why := NewConfig(Config{}).Settings()
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
	response.Performance = hardware.Performance{Profile: settingsdomain.ProfileEco}

	got := NewConfig(Config{}).SetSettings(response)

	if got.GetMachine().Hardware.LogicalCPUs < 1 {
		t.Errorf("the hardware was not detected: %+v", got.GetMachine().Hardware)
	}
	if got.GetMachine().ResolvedCompute.Profile != settingsdomain.ProfileEco || got.GetMachine().ResolvedCompute.Workers != 1 {
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

// Settings that cannot be read are answered with the defaults, and ones that
// can are answered as they stand.
func TestGetSettingsOrDefaults_FallsBackToDefaultsWhenUnreadable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	cfg, _ := writeSettings(t, `{"app": {"agent": {"use": "somebody-else"}}}`)
	if got := settingsGetSettingsOrDefaults(settingsOf(cfg)).Agent.GetUse(); got != "somebody-else" {
		t.Errorf("the agent is %q", got)
	}

	cfg, _ = writeSettings(t, `{"app": {"agent": `)
	if got := settingsGetSettingsOrDefaults(settingsOf(cfg)); !reflect.DeepEqual(got, DefaultSettings()) {
		t.Errorf("an unreadable file gave %+v", got)
	}

	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := settingsGetSettingsOrDefaults(settingsOf((NewConfig(Config{})))); !reflect.DeepEqual(got, DefaultSettings()) {
		t.Errorf("a machine with nowhere for settings gave %+v", got)
	}
}

// The models a setting can take are listed as the window requests them, against
// the active settings.
func TestTheModelsAreListedAgainstTheSettingsInForce(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg, _ := writeSettings(t, `{}`)

	models := models(cfg)()
	if len(models) == 0 {
		t.Fatal("no models are listed")
	}
	for _, one := range models {
		if len(one.Path) == 0 {
			t.Errorf("a model stands nowhere: %+v", one)
		}
	}
}

// profileHost records the profiles the container gives the plugin host.
type profileHost struct {
	pluginport.SettingsPush
	profiles []string
}

func (p *profileHost) SetPerformanceProfile(_ context.Context, profile string) {
	p.profiles = append(p.profiles, profile)
}

func TestWritingAnotherSettingSendsNoProfile(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)
	host := &profileHost{}
	cfg.PluginHost = &PluginHost{Configuration: PluginConfiguration{SettingsPush: host}}

	err := getSettingsWriter(settingsOf(cfg))([]settingsdomain.Setting{{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`}})
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
	response.Performance = hardware.Performance{Profile: settingsdomain.ProfileEco}
	before := NewConfig(Config{})

	after := before.SetSettings(response)

	if after.GetMachine().ResolvedCompute.Workers != 1 {
		t.Fatalf("the applied configuration plans %+v", after.GetMachine().ResolvedCompute)
	}
	if got := before.GetMachine().ResolvedCompute; got.Workers != 0 || got.Profile != "" {
		t.Errorf("the configuration the call was made on changed to %+v", got)
	}
}

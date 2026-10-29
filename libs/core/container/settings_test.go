package container

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// writeSettings is a settings file holding that text, and the container reading
// it.
func writeSettings(t *testing.T, output string) (Config, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		t.Fatal(err)
	}
	return Config{SettingsPath: path}, path
}

// What the window is given is every setting, so a setting the file leaves out
// is read out as what this installation is doing about it.
func TestTheSettingsReadOutHoldWhatTheFilePreservesOut(t *testing.T) {
	cfg, path := writeSettings(t, `{"appearance": {"theme": "mine:sea"}}`)

	content, response, err := cfg.ReadSettings()()
	if err != nil {
		t.Fatal(err)
	}
	if response != path {
		t.Errorf("the settings stand at %q, not %q", response, path)
	}

	var target Settings
	if err := json.Unmarshal([]byte(content), &target); err != nil {
		t.Fatal(err)
	}
	if target.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", target.Appearance.Theme)
	}
	if target.Appearance.Mode != settings.ModeSystem {
		t.Errorf("the mode reads %q", target.Appearance.Mode)
	}
}

// A setting written is written at its current position, and every key a person typed
// stays where it was.
func TestASettingSavedPreservesTheRestOfTheFileAlone(t *testing.T) {
	cfg, path := writeSettings(t, `{
  "appearance": {"theme": "mine:sea"},
  "something_this_build_knows_nothing_about": 7
}`)

	err := cfg.GetSettingsWriter()([]settingsport.Setting{
		{Path: []string{"agent", "claude", "model"}, JSON: `"opus"`},
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

	target, err := cfg.getSettingsAt(path)
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

// Several settings are written together, which is what choosing one model that
// decides more than its own name asks for.
func TestSettingsSavedTogetherAllArrive(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)

	err := cfg.GetSettingsWriter()([]settingsport.Setting{
		{Path: []string{"agent", "use"}, JSON: `"claude"`},
		{Path: []string{"agent", "claude", "max_steps"}, JSON: `12`},
	})
	if err != nil {
		t.Fatal(err)
	}

	target, err := cfg.getSettingsAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if target.Agent.GetUse() != "claude" || target.Agent.GetProgram("claude").MaxSteps != 12 {
		t.Errorf("the agent reads %+v", target.Agent)
	}
}

// A machine that has nowhere to keep settings is told so, and is handed none.
func TestSettingsWithNowhereToLiveAreRefused(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	target, why := Config{}.Settings()
	if why == nil {
		t.Fatal("no reason")
	}
	if !reflect.DeepEqual(target, Settings{}) {
		t.Errorf("got settings: %+v", target)
	}
}

func TestASettingsFileThatIsNotReadableIsRefused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg, _ := writeSettings(t, `{"appearance": `)

	target, why := cfg.Settings()
	if why == nil {
		t.Fatal("no reason")
	}
	if !reflect.DeepEqual(target, Settings{}) {
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
	response.Agent.Use = settings.UseProgram("claude")
	response.Performance = hardware.Performance{Profile: hardware.ProfileEco}

	got := Config{}.SetSettings(response)

	if got.Agent.GetUse() != "claude" {
		t.Errorf("the agent is %q", got.Agent.GetUse())
	}
	if got.Performance.Profile != hardware.ProfileEco {
		t.Errorf("the performance profile is %q", got.Performance.Profile)
	}
	if got.Hardware.LogicalCPUs < 1 {
		t.Errorf("the hardware was not detected: %+v", got.Hardware)
	}
	if got.ResolvedCompute.Profile != hardware.ProfileEco || got.ResolvedCompute.Workers != 1 {
		t.Errorf("the compute plan is %+v", got.ResolvedCompute)
	}
}

// A line said is kept for whoever read the settings to put in front of a person.
func TestSettings_AddNoticeAccumulatesNotices(t *testing.T) {
	var target Settings
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

	cfg, _ := writeSettings(t, `{"agent": {"use": "somebody-else"}}`)
	if got := cfg.getSettingsOrDefaults().Agent.GetUse(); got != "somebody-else" {
		t.Errorf("the agent is %q", got)
	}

	cfg, _ = writeSettings(t, `{"agent": `)
	if got := cfg.getSettingsOrDefaults(); !reflect.DeepEqual(got, DefaultSettings()) {
		t.Errorf("an unreadable file gave %+v", got)
	}

	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := (Config{}).getSettingsOrDefaults(); !reflect.DeepEqual(got, DefaultSettings()) {
		t.Errorf("a machine with nowhere for settings gave %+v", got)
	}
}

// The models a setting can be set to are asked for as the window asks, against
// the settings in force.
func TestTheModelsAreListedAgainstTheSettingsInForce(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg, _ := writeSettings(t, `{}`)

	models := cfg.Models()()
	if len(models) == 0 {
		t.Fatal("no models are listed")
	}
	for _, one := range models {
		if len(one.Path) == 0 {
			t.Errorf("a model stands nowhere: %+v", one)
		}
	}
}

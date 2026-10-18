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

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// writeSettings is a settings file holding that text, and the container reading
// it.
func writeSettings(t *testing.T, written string) (Config, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}
	return Config{SettingsPath: path}, path
}

// What the window is given is every setting, so a setting the file leaves out
// is read out as what this installation is doing about it.
func TestTheSettingsReadOutHoldWhatTheFileLeavesOut(t *testing.T) {
	cfg, path := writeSettings(t, `{"appearance": {"theme": "mine:sea"}}`)

	written, said, err := cfg.ReadSettings()()
	if err != nil {
		t.Fatal(err)
	}
	if said != path {
		t.Errorf("the settings stand at %q, not %q", said, path)
	}

	var held Settings
	if err := json.Unmarshal([]byte(written), &held); err != nil {
		t.Fatal(err)
	}
	if held.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", held.Appearance.Theme)
	}
	if held.Appearance.NestedHeadingLimit != settings.DefaultNestedHeadingLimit {
		t.Errorf("the count reads %d", held.Appearance.NestedHeadingLimit)
	}
}

// A setting written is written where it stands, and every key a person typed
// stays where it was.
func TestASettingWrittenLeavesTheRestOfTheFileAlone(t *testing.T) {
	cfg, path := writeSettings(t, `{
  "appearance": {"theme": "mine:sea"},
  "something_this_build_knows_nothing_about": 7
}`)

	err := cfg.GetSettingsWriter()([]port.Setting{
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

	held, err := cfg.getSettingsAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if held.Agent.GetProgram("claude").Model != "opus" {
		t.Errorf("the model reads %q", held.Agent.GetProgram("claude").Model)
	}
	if held.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", held.Appearance.Theme)
	}
}

// Several settings are written together, which is what choosing one model that
// decides more than its own name asks for.
func TestSettingsWrittenTogetherAllArrive(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)

	err := cfg.GetSettingsWriter()([]port.Setting{
		{Path: []string{"agent", "use"}, JSON: `"claude"`},
		{Path: []string{"agent", "claude", "max_steps"}, JSON: `12`},
	})
	if err != nil {
		t.Fatal(err)
	}

	held, err := cfg.getSettingsAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if held.Agent.GetUse() != "claude" || held.Agent.GetProgram("claude").MaxSteps != 12 {
		t.Errorf("the agent reads %+v", held.Agent)
	}
}

// A machine that has nowhere to keep settings is told so, and is handed none.
func TestSettingsWithNowhereToLiveAreRefused(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	held, why := Config{}.Settings()
	if why == nil {
		t.Fatal("no reason")
	}
	if !reflect.DeepEqual(held, Settings{}) {
		t.Errorf("got settings: %+v", held)
	}
}

// A file that cannot be read is refused, and what is handed back is nothing.
func TestASettingsFileThatIsNotReadableIsRefused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg, _ := writeSettings(t, `{"appearance": `)

	held, why := cfg.Settings()
	if why == nil {
		t.Fatal("no reason")
	}
	if !reflect.DeepEqual(held, Settings{}) {
		t.Errorf("got settings: %+v", held)
	}
}

// Settings that cannot be written as JSON are refused with no text.
func TestSettingsThatCannotBeWrittenAreRefused(t *testing.T) {
	held := DefaultSettings()
	held.Appearance.InterfaceScale = math.NaN()

	written, why := held.WriteJSON()
	if why == nil {
		t.Fatal("no reason")
	}
	if written != "" {
		t.Errorf("wrote %q", written)
	}
}

// Every section of the settings arrives in the configuration, and the compute
// plan is resolved from the performance section.
func TestAConfigurationCarriesEverySectionOfTheSettings(t *testing.T) {
	said := DefaultSettings()
	said.Indexing.Recognition.Proofread = Proofread{Profile: "for scans", IsAutomatic: true}
	said.Indexing.Transcription.Proofread = Proofread{Profile: "for speech"}
	said.Agent.Use = settings.UseProgram("claude")
	said.Importing.Captions = []string{"sa", "en"}
	said.Performance = settings.Performance{Profile: settings.ProfileEco}

	got := Config{}.SetSettings(said)

	if got.ScanProofreading != said.Indexing.Recognition.Proofread {
		t.Errorf("scans are put right with %+v", got.ScanProofreading)
	}
	if got.TranscriptProofreading != said.Indexing.Transcription.Proofread {
		t.Errorf("speech is put right with %+v", got.TranscriptProofreading)
	}
	if got.Agent.GetUse() != "claude" {
		t.Errorf("the agent is %q", got.Agent.GetUse())
	}
	if !slices.Equal(got.Importing.Captions, []string{"sa", "en"}) {
		t.Errorf("the captions are %v", got.Importing.Captions)
	}
	if got.Performance.Profile != settings.ProfileEco {
		t.Errorf("the performance profile is %q", got.Performance.Profile)
	}
	if got.Hardware.LogicalCPUs < 1 {
		t.Errorf("the hardware was not detected: %+v", got.Hardware)
	}
	if got.ResolvedCompute.Profile != settings.ProfileEco || got.ResolvedCompute.OcrThreads != 1 {
		t.Errorf("the compute plan is %+v", got.ResolvedCompute)
	}
}

// A line said is kept for whoever read the settings to put in front of a person.
func TestWhatIsSaidWhileReadingIsKept(t *testing.T) {
	var held Settings
	held.AddNotice("first")
	held.AddNotice("second")
	if !slices.Equal(held.Notices, []string{"first", "second"}) {
		t.Errorf("said %v", held.Notices)
	}
}

func TestNestedHeadingLimitIsWhatTheAppearanceSays(t *testing.T) {
	held := DefaultSettings()
	if !held.HasNestedHeadings() {
		t.Error("untouched settings hang no headings under a node")
	}
	off := false
	held.Appearance.NestHeadingsUnderNotes = &off
	if held.HasNestedHeadings() {
		t.Error("settings turning it off still hang headings under a node")
	}
	held.Appearance.NestedHeadingLimit = 7
	if held.GetNestedHeadingLimit() != 7 {
		t.Errorf("%d headings stand under a node", held.GetNestedHeadingLimit())
	}
}

// Settings that cannot be read are answered with the defaults, and ones that
// can are answered as they stand.
func TestSettingsOrDefaultsAnswerWhatIsReadableAndTheDefaultsOtherwise(t *testing.T) {
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

	models, _ := cfg.GetCatalogue()()
	if len(models) == 0 {
		t.Fatal("no models are listed")
	}
	for _, one := range models {
		if len(one.Path) == 0 {
			t.Errorf("a model stands nowhere: %+v", one)
		}
	}
}

package container

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	var held settings.Config
	if err := json.Unmarshal([]byte(written), &held); err != nil {
		t.Fatal(err)
	}
	if held.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", held.Appearance.Theme)
	}
	if held.Appearance.PartsUnderANode != settings.DefaultParts {
		t.Errorf("the count reads %d", held.Appearance.PartsUnderANode)
	}
}

// A setting written is written where it stands, and every key a person typed
// stays where it was.
func TestASettingWrittenLeavesTheRestOfTheFileAlone(t *testing.T) {
	cfg, path := writeSettings(t, `{
  "appearance": {"theme": "mine:sea"},
  "something_this_build_knows_nothing_about": 7
}`)

	err := cfg.TurnsSetting()([]port.Setting{
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

	held, err := settings.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if held.Agent.Claude.Model != "opus" {
		t.Errorf("the model reads %q", held.Agent.Claude.Model)
	}
	if held.Appearance.Theme != "mine:sea" {
		t.Errorf("the theme reads %q", held.Appearance.Theme)
	}
}

// Several settings are written together, which is what choosing one model that
// decides more than its own name asks for.
func TestSettingsWrittenTogetherAllArrive(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)

	err := cfg.TurnsSetting()([]port.Setting{
		{Path: []string{"agent", "use"}, JSON: `"claude"`},
		{Path: []string{"agent", "claude", "max_steps"}, JSON: `12`},
	})
	if err != nil {
		t.Fatal(err)
	}

	held, err := settings.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if held.Agent.Use != "claude" || held.Agent.Claude.MaxSteps != 12 {
		t.Errorf("the agent reads %+v", held.Agent)
	}
}

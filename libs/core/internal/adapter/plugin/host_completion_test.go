package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const (
	askingPluginID = "md.numen.asking"
	plainPluginID  = "md.numen.plain"
)

// newAskingHost discovers a plugin that may ask a model and declares the agent
// setting, and one that does neither.
func newAskingHost(t *testing.T) *Host {
	t.Helper()
	root := t.TempDir()
	for name, manifest := range map[string]string{
		"asking": `{"id":"` + askingPluginID + `","name":"Asking","version":"1.0.0","targets":["desktop.editor"],
			"permissions":["ai:complete"],
			"settings":[{"key":"model","type":"agent","title":"Model"}]}`,
		"plain": `{"id":"` + plainPluginID + `","name":"Plain","version":"1.0.0","targets":["desktop.editor"]}`,
	} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	host := NewHost("desktop.editor", root)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	return host
}

// The choice is read from the setting of type agent, as the person stored it.
func TestTheChoiceIsTheStoredValueOfTheAgentSetting(t *testing.T) {
	host := newAskingHost(t)
	if got := host.GetCompletionChoice(askingPluginID); got != (completionport.CompletionChoice{}) {
		t.Errorf("a plugin never set chose %+v", got)
	}
	host.SetSettings(map[string]map[string]any{
		askingPluginID: {"model": map[string]any{"agent": "md.numen.agent-x", "model": "m"}},
		plainPluginID:  {"model": map[string]any{"agent": "md.numen.agent-x"}},
	})
	if want := (completionport.CompletionChoice{Agent: "md.numen.agent-x", Model: "m"}); host.GetCompletionChoice(askingPluginID) != want {
		t.Errorf("chose %+v, want %+v", host.GetCompletionChoice(askingPluginID), want)
	}
	if got := host.GetCompletionChoice(plainPluginID); got != (completionport.CompletionChoice{}) {
		t.Errorf("a plugin declaring no agent setting chose %+v", got)
	}
	if got := host.GetCompletionChoice("md.numen.unknown"); got != (completionport.CompletionChoice{}) {
		t.Errorf("an unknown plugin chose %+v", got)
	}
}

// A value of another shape is the empty choice.
func TestAChoiceOfAnotherShapeIsEmpty(t *testing.T) {
	host := newAskingHost(t)
	for _, value := range []any{"md.numen.agent-x", 3, nil, map[string]any{"agent": 7}} {
		host.SetSettings(map[string]map[string]any{askingPluginID: {"model": value}})
		if got := host.GetCompletionChoice(askingPluginID); got.Agent != "" {
			t.Errorf("%v chose %+v", value, got)
		}
	}
}

// A plugin is answered only once the person allowed it. One not answered for is
// recorded as asking; a rejected one is not asking.
func TestTheConsentIsThePersonsAnswer(t *testing.T) {
	host := newAskingHost(t)
	info := func() pluginport.CompletionConsent {
		for _, one := range host.GetPluginInfos() {
			if one.ID == askingPluginID {
				return one.Consent
			}
		}
		t.Fatal("the plugin is not listed")
		return pluginport.ConsentNone
	}
	if info() != pluginport.ConsentUnanswered {
		t.Errorf("before asking: %v, want unanswered", info())
	}
	if err := host.CheckCompletionConsent(askingPluginID); !errors.Is(err, completionport.ErrNoConsent) {
		t.Fatalf("unanswered: error = %v, want port.ErrNoConsent", err)
	}
	if info() != pluginport.ConsentAsked {
		t.Errorf("after asking: %v, want asked", info())
	}

	if err := host.UpdatePluginSetting(t.Context(), askingPluginID, pluginport.SettingAIConsent, true); err != nil {
		t.Fatal(err)
	}
	if err := host.CheckCompletionConsent(askingPluginID); err != nil {
		t.Errorf("allowed: error = %v", err)
	}
	if info() != pluginport.ConsentAllowed {
		t.Errorf("allowed: %v", info())
	}

	if err := host.UpdatePluginSetting(t.Context(), askingPluginID, pluginport.SettingAIConsent, false); err != nil {
		t.Fatal(err)
	}
	if err := host.CheckCompletionConsent(askingPluginID); !errors.Is(err, completionport.ErrNoConsent) {
		t.Errorf("rejected: error = %v, want port.ErrNoConsent", err)
	}
	if info() != pluginport.ConsentRefused {
		t.Errorf("rejected: %v", info())
	}
}

// A plugin without the permission is listed with no consent at all.
func TestAPluginThatDoesNotAskHasNoConsent(t *testing.T) {
	host := newAskingHost(t)
	for _, one := range host.GetPluginInfos() {
		if one.ID == plainPluginID && one.Consent != pluginport.ConsentNone {
			t.Errorf("listed with %v", one.Consent)
		}
	}
}

// The consent is the host's record, and the plugin is not handed it.
func TestTheConsentIsNotInThePluginsSettings(t *testing.T) {
	host := newAskingHost(t)
	host.SetSettings(map[string]map[string]any{askingPluginID: {pluginport.SettingAIConsent: true}})
	if _, found := host.GetPluginSettings(askingPluginID)[pluginport.SettingAIConsent]; found {
		t.Error("the plugin's settings hold the consent")
	}
}

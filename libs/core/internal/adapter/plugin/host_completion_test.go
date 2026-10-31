package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const (
	askingPluginID = "md.numen.asking"
	plainPluginID  = "md.numen.plain"
)

// newAskingHost discovers a plugin that may ask a model, and one that may not.
func newAskingHost(t *testing.T) *Host {
	t.Helper()
	root := t.TempDir()
	for name, manifest := range map[string]string{
		"asking": `{"id":"` + askingPluginID + `","name":"Asking","version":"1.0.0","targets":["desktop.editor"],
			"permissions":["ai:complete"]}`,
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
	host := NewHost(testsupport.Machine, "desktop.editor", root)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	return host
}

// A plugin is served only after the user has allowed it. A plugin that is not
// served is recorded as asking; a rejected plugin is not recorded as asking.
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
	if info() != pluginport.ConsentPending {
		t.Errorf("before asking: %v, want unanswered", info())
	}
	if err := host.CheckCompletionConsent(askingPluginID); !errors.Is(err, completionport.ErrNoConsent) {
		t.Fatalf("unanswered: error = %v, want port.ErrNoConsent", err)
	}
	if info() != pluginport.ConsentRequested {
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
	if info() != pluginport.ConsentDenied {
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

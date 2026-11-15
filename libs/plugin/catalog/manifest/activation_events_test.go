package manifest_test

import (
	"slices"
	"testing"

	. "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestIndexChangedAndFilesChangedAreTheActivationEventsAPluginDeclares(t *testing.T) {
	if got := PluginActivationEvents(); !slices.Equal(got, []string{"onIndexChanged", "onFilesChanged"}) {
		t.Errorf("activation events = %v, want [onIndexChanged onFilesChanged]", got)
	}
}

func TestAPluginOnIndexChangedIsActivatedByTheIndexChangeOfAnyVault(t *testing.T) {
	m := Manifest{ID: "a.b", ActivationEvents: []string{ActivationEventIndexChanged}}
	for _, event := range []string{"onIndexChanged", "onIndexChanged:01VAULT"} {
		if !m.IsActivatedBy(event) {
			t.Errorf("%q does not activate the plugin", event)
		}
	}
	for _, event := range []string{"", "*", "onIndexChanged:", "onIndex", "onIndex:01VAULT"} {
		if m.IsActivatedBy(event) {
			t.Errorf("%q activates the plugin", event)
		}
	}
}
func TestAPluginIsActivatedByTheVaultEventsItDeclaresOnly(t *testing.T) {
	m := Manifest{ID: "a.b", ActivationEvents: []string{ActivationEventFilesChanged}}
	if !m.IsActivatedBy("onFilesChanged:vault-1") || !m.IsActivatedBy("onFilesChanged") {
		t.Error("the declared event does not activate the plugin")
	}
	for _, event := range []string{"onFilesChanged:", "onIndexChanged:vault-1", "onFilesChangedX:vault-1"} {
		if m.IsActivatedBy(event) {
			t.Errorf("%q activates the plugin", event)
		}
	}
	if (Manifest{ID: "a.b", ActivationEvents: []string{"onCommand"}}).IsActivatedBy("onCommand:run") {
		t.Error("an event no host emits activates the plugin")
	}
}

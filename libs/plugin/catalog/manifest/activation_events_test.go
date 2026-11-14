package manifest_test

import (
	"slices"
	"testing"

	. "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestIndexChangedIsTheOnlyActivationEventAPluginDeclares(t *testing.T) {
	if got := PluginActivationEvents(); !slices.Equal(got, []string{"onIndexChanged"}) {
		t.Errorf("activation events = %v, want [onIndexChanged]", got)
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

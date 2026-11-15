package manifest

import "testing"

func TestAPluginIsActivatedByTheVaultEventsItDeclaresOnly(t *testing.T) {
	m := Manifest{ActivationEvents: []string{ActivationEventFilesChanged}}
	if !m.IsActivatedBy("onFilesChanged:vault-1") || !m.IsActivatedBy("onFilesChanged") {
		t.Error("the declared event does not activate the plugin")
	}
	for _, event := range []string{"onFilesChanged:", "onIndexChanged:vault-1", "onFilesChangedX:vault-1"} {
		if m.IsActivatedBy(event) {
			t.Errorf("%q activates the plugin", event)
		}
	}
	if (Manifest{ActivationEvents: []string{"onCommand"}}).IsActivatedBy("onCommand:run") {
		t.Error("an event no host emits activates the plugin")
	}
}

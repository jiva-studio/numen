package manifest_test

import (
	"errors"
	"slices"
	"testing"

	. "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
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

func TestFilesChangedMaskSelectsMatchingTypes(t *testing.T) {
	m := Manifest{ID: "a.b", ActivationEvents: []string{"onFilesChanged:image/*", "onFilesChanged:application/pdf"}}
	if !m.IsActivatedBy("onFilesChanged:vault-1") {
		t.Error("the masked event does not activate the plugin")
	}
	if !m.IsTakingFileTypes([]domain.ContentType{"text/markdown", "image/png"}) || !m.IsTakingFileTypes([]domain.ContentType{"application/pdf"}) {
		t.Error("the plugin ignores a type its masks match")
	}
	if m.IsTakingFileTypes([]domain.ContentType{"text/markdown", "audio/mpeg"}) {
		t.Error("the plugin takes types no mask matches")
	}
	if !(Manifest{ID: "a.b", ActivationEvents: []string{"onFilesChanged"}}).IsTakingFileTypes([]domain.ContentType{"text/markdown"}) {
		t.Error("a bare event does not take every type")
	}
	if !(Manifest{ID: "a.b", ActivationEvents: []string{"onIndexChanged"}}).IsTakingFileTypes([]domain.ContentType{"text/markdown"}) {
		t.Error("a plugin with no files event filters its feed")
	}
	for _, declared := range []string{"onFilesChanged:", "onFilesChanged:image", "onIndexChanged:image/*"} {
		issues := (Manifest{ID: "a.b", ActivationEvents: []string{declared}}).ListIssues()
		if !slices.ContainsFunc(issues, func(issue error) bool { return errors.Is(issue, ErrInvalidActivationEvent) }) {
			t.Errorf("%q is a valid activation event", declared)
		}
	}
}

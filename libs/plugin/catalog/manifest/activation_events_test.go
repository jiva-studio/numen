package manifest_test

import (
	"errors"
	"slices"
	"testing"

	. "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestIndexChangedAndFilesChangedAreTheActivationEventsAPluginDeclares(t *testing.T) {
	if got := ListPluginActivationEvents(); !slices.Equal(got, []string{"onIndexChanged", "onFilesChanged"}) {
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
	if !m.IsTakingFileTypes([]domain.FileType{{Content: "text/markdown"}, {Content: "image/png"}}) || !m.IsTakingFileTypes([]domain.FileType{{Content: "application/pdf"}}) {
		t.Error("the plugin ignores a type its masks match")
	}
	if m.IsTakingFileTypes([]domain.FileType{{Content: "text/markdown"}, {Content: "audio/mpeg"}}) {
		t.Error("the plugin takes types no mask matches")
	}
	if !(Manifest{ID: "a.b", ActivationEvents: []string{"onFilesChanged"}}).IsTakingFileTypes([]domain.FileType{{Content: "text/markdown"}}) {
		t.Error("a bare event does not take every type")
	}
	if !(Manifest{ID: "a.b", ActivationEvents: []string{"onIndexChanged"}}).IsTakingFileTypes([]domain.FileType{{Content: "text/markdown"}}) {
		t.Error("a plugin with no files event filters its feed")
	}
	for _, declared := range []string{"onFilesChanged:", "onFilesChanged:image", "onFilesChanged:note:", "onFilesChanged:note:two words", "onIndexChanged:image/*"} {
		issues := (Manifest{ID: "a.b", ActivationEvents: []string{declared}}).ListIssues()
		if !slices.ContainsFunc(issues, func(issue error) bool { return errors.Is(issue, ErrInvalidActivationEvent) }) {
			t.Errorf("%q is a valid activation event", declared)
		}
	}
}

func TestFilesChangedMasksSelectNoteTypesAndContentTypes(t *testing.T) {
	m := Manifest{ID: "a.b", ActivationEvents: []string{"onFilesChanged:note:link", "onFilesChanged:image/png"}}
	if !m.IsActivatedBy("onFilesChanged:vault-1") {
		t.Fatal("masked event did not match the vault event")
	}
	for file, want := range map[domain.FileType]bool{
		{Content: "text/markdown", Note: "link"}:          true,
		{Content: "text/markdown", Note: domain.TypeNote}: false,
		{Content: "image/png"}:                            true,
		{Content: "image/jpeg"}:                           false,
	} {
		if got := m.IsTakingFileTypes([]domain.FileType{file}); got != want {
			t.Errorf("takes %+v = %v, want %v", file, got, want)
		}
	}
}

package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

func TestACommandScopeIsEmptyOrNote(t *testing.T) {
	for _, scope := range []string{"", "note"} {
		text := `{"id":"a.b","name":"A","version":"1.0.0","frontend":{},"contributes":{"commands":[{"id":"tree.open","title":"T","scope":"` + scope + `"}]}}`

		if _, err := manifestjson.Read(strings.NewReader(text)); err != nil {
			t.Errorf("the scope %q was refused: %v", scope, err)
		}
	}
}

func TestACommandScopeOutsideTheKnownOnesIsInvalid(t *testing.T) {
	text := `{"id":"a.b","name":"A","version":"1.0.0","frontend":{},"contributes":{"commands":[{"id":"tree.open","title":"T","scope":"window"}]}}`

	_, err := manifestjson.Read(strings.NewReader(text))

	if !errors.Is(err, manifest.ErrInvalidCommand) || !strings.Contains(err.Error(), `scope "window"`) {
		t.Errorf("error = %v, want an invalid command naming the scope", err)
	}
}

func TestAFrontendCommandKeepsItsScopeAndShortcut(t *testing.T) {
	text := `{"id":"a.b","name":"A","version":"1.0.0","frontend":{},"backend":{"entrypoint":"bin/a"},"contributes":{"commands":[{"id":"tree.open","title":"T","scope":"note","shortcut":"mod+shift+e"}]}}`

	if _, err := manifestjson.Read(strings.NewReader(text)); err != nil {
		t.Errorf("a plugin with both parts was refused: %v", err)
	}
}

func TestACommandOfAPluginWithOnlyABackendDeclaresNeitherScopeNorShortcut(t *testing.T) {
	for name, fields := range map[string]string{
		"a scope":    `"scope":"note"`,
		"a shortcut": `"shortcut":"mod+shift+e"`,
	} {
		text := `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"contributes":{"commands":[{"id":"deck.create","title":"T",` + fields + `}]}}`

		_, err := manifestjson.Read(strings.NewReader(text))

		if !errors.Is(err, manifest.ErrInvalidCommand) {
			t.Errorf("%s: error = %v, want an invalid command", name, err)
		}
	}
}

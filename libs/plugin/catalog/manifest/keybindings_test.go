package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const keybindingManifestHead = `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"contributes":{"commands":[{"id":"deck.create","title":"Create","arguments":[{"name":"deck-title","title":"Title","type":"string","required":false}]}],"keybindings":`

func TestReadManifest_Keybindings(t *testing.T) {
	cases := map[string]struct {
		keybindings string
		want        error
	}{
		"a chord of the plugin command": {`[{"key":"mod+shift+p","command":"deck.create","when":"window.isVaultOpen"}]}}`, nil},
		"a single letter":               {`[{"key":"n","command":"deck.create"}]}}`, nil},
		"values of a declared argument": {`[{"key":"n","command":"deck.create","values":{"deck-title":"Inbox"}}]}}`, nil},
		"an app command with a value":   {`[{"key":"mod+alt+e","command":"app:file.open","values":{"placement":"split"}}]}}`, nil},
		"the same key under two conditions": {`[
			{"key":"n","command":"deck.create","when":"window.isVaultOpen"},
			{"key":"n","command":"deck.create","when":"!window.isVaultOpen"}]}}`, nil},
		"a key with a modifier only":       {`[{"key":"mod+shift","command":"deck.create"}]}}`, manifest.ErrInvalidKeybinding},
		"a key in capitals":                {`[{"key":"Mod+P","command":"deck.create"}]}}`, manifest.ErrInvalidKeybinding},
		"a key with two letters":           {`[{"key":"mod+pq","command":"deck.create"}]}}`, manifest.ErrInvalidKeybinding},
		"a command that is not declared":   {`[{"key":"n","command":"deck.remove"}]}}`, manifest.ErrInvalidKeybinding},
		"a condition that does not parse":  {`[{"key":"n","command":"deck.create","when":"window.isVaultOpen &&"}]}}`, manifest.ErrInvalidKeybinding},
		"an argument the command lacks":    {`[{"key":"n","command":"deck.create","values":{"size":"3"}}]}}`, manifest.ErrInvalidSlotEntry},
		"an argument an app command lacks": {`[{"key":"n","command":"app:file.open","values":{"size":"3"}}]}}`, manifest.ErrInvalidSlotEntry},
		"a value outside the options":      {`[{"key":"n","command":"app:file.open","values":{"placement":"floating"}}]}}`, manifest.ErrInvalidSlotEntry},
		"the same key and condition twice": {`[
			{"key":"n","command":"deck.create","when":"window.isVaultOpen"},
			{"key":"n","command":"app:file.open","when":"window.isVaultOpen"}]}}`, manifest.ErrInvalidKeybinding},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(keybindingManifestHead + one.keybindings))

			if !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

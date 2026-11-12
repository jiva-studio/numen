package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestACommandIdIsAnObjectAndAnActionJoinedByADot(t *testing.T) {
	for _, id := range []string{"deck.create", "cardModel.create", "link.downloadText", "recognition.delete", "tree.revealNote", "note.goTo", "appearance.chooseInterfaceSize", "deck2.create", "a.b"} {
		text := commandManifestHead + `[{"id":"` + id + `","title":"C"}]}}`

		if _, err := manifestjson.Read(strings.NewReader(text)); err != nil {
			t.Errorf("the id %q was refused: %v", id, err)
		}
	}
}

func TestACommandIdOutsideTheObjectAndActionFormIsInvalid(t *testing.T) {
	for _, id := range []string{
		"create-deck",
		"card-model.create",
		"link.download-text",
		"note.open-beside",
		"deck.create_now",
		"deck_set.create",
		"convert",
		"Deck.create",
		"deck.Create",
		"deck.",
		".create",
		"deck..create",
		"deck.create.now",
		"deck:create",
		"app:deck.create",
		"deck create.now",
		"1deck.create",
	} {
		text := commandManifestHead + `[{"id":"` + id + `","title":"C"}]}}`

		_, err := manifestjson.Read(strings.NewReader(text))

		if !errors.Is(err, manifest.ErrInvalidCommand) {
			t.Errorf("the id %q gave %v, want an invalid command", id, err)
		}
	}
}

func TestACommandArgumentNameStaysLowercaseWordsJoinedByHyphens(t *testing.T) {
	ok := commandManifestHead + `[{"id":"deck.create","title":"C","arguments":[{"name":"deck-title","title":"T","type":"string"}]}]}}`
	bad := commandManifestHead + `[{"id":"deck.create","title":"C","arguments":[{"name":"deck.title","title":"T","type":"string"}]}]}}`

	if _, err := manifestjson.Read(strings.NewReader(ok)); err != nil {
		t.Errorf("a hyphenated argument name was refused: %v", err)
	}
	if _, err := manifestjson.Read(strings.NewReader(bad)); !errors.Is(err, manifest.ErrInvalidCommandArgument) {
		t.Errorf("a dotted argument name gave %v, want an invalid command argument", err)
	}
}

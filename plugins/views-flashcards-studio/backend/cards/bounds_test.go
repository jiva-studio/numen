package cards_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
)

func TestRenamingAFieldIsRefusedWhereTheVaultContainsMoreDecksThanTheBound(t *testing.T) {
	vs := newVaults(t)
	before := read(t, vs.first, "Animal.md")

	_, err := newBoundedRenameField(t, vs, 1).Execute(t.Context(), vs.first, cards.Rename{
		CardModel: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err == nil || !strings.Contains(err.Error(), "there are 2 of them") || !strings.Contains(err.Error(), "more than the 1") {
		t.Fatalf("a vault past the bound was renamed in: %v", err)
	}
	if read(t, vs.first, "Animal.md") != before {
		t.Error("the card model was written")
	}

	if _, err := newBoundedRenameField(t, vs, 2).Execute(t.Context(), vs.first, cards.Rename{
		CardModel: "Animal.md", From: "Height", To: "Shoulder height",
	}); err != nil {
		t.Errorf("a vault at the bound failed: %v", err)
	}
}

func newBoundedRenameField(t *testing.T, vs vaulted, maxDecks int) cards.RenameField {
	t.Helper()
	return cards.NewRenameField(
		vs.ports.Readers, vs.ports.Writers,
		vs.queries, vs.queries, vs.index(t), time.Now, maxDecks,
	)
}

func TestACardModelIsMadeWithItsTemplatesLaidDown(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, time.Now)

	made, err := u.CardModel(t.Context(), v, cards.New{
		Title: "Bird", Fields: []string{"Name", "Call"},
		Templates: []format.CardTemplate{{Name: "Recognise", Front: "{{Name}}", Back: "{{Call}}"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := read(t, v, made.Path)
	if !strings.Contains(body, "## Recognise") || !strings.Contains(body, "{{Call}}") {
		t.Errorf("the templates are not in the card model:\n%s", body)
	}
}

func TestACardModelWithTemplatesLargerThanANoteIsRefused(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, time.Now)

	_, err := u.CardModel(t.Context(), v, cards.New{
		Title: "Bird", Fields: []string{"Name"},
		Templates: []format.CardTemplate{{Name: "Big", Front: strings.Repeat("x", 1<<20), Back: "{{Name}}"}},
	})
	if err == nil || !strings.Contains(err.Error(), "more than this writes at once") {
		t.Errorf("an oversized card model was made: %v", err)
	}
}

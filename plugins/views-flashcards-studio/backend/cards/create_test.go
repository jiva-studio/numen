package cards_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// A title no file can be named after fails, and nothing is written.
func TestATitleNoFileCanBeNamedAfterIsRefused(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock())

	for name, title := range map[string]string{
		"empty":     "  ",
		"two lines": "Birds\nof prey",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := u.Deck(t.Context(), v, cards.New{Title: title}); !errors.Is(err, domain.ErrInvalidName) {
				t.Errorf("make a deck titled %q = %v, want it rejected", title, err)
			}
		})
	}
	entries, err := filepath.Glob(filepath.Join(v.Path, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("a rejected title left %v behind", entries)
	}
}

// The title key is written where the filename cannot carry the whole name, and
// only there.
func TestATitleIsSavedOnlyWhereTheFilenameCannotCarryIt(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock())

	plain, err := u.Deck(t.Context(), v, cards.New{Title: "Birds"})
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, v, plain.Path); strings.Contains(got, "title:") {
		t.Errorf("a title the filename carries was written as a key: %q", got)
	}

	assigned, err := u.Deck(t.Context(), v, cards.New{Title: "Birds: a guide"})
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Path != "Birds- a guide.md" {
		t.Errorf("path = %q", assigned.Path)
	}
	if got := read(t, v, assigned.Path); !strings.Contains(got, "title:") || !strings.Contains(got, "Birds: a guide") {
		t.Errorf("the title the filename cannot carry was not written: %q", got)
	}
}

// A card model of one field declares it.
func TestACardModelOfOneFieldDeclaresIt(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock())

	made, err := u.CardModel(t.Context(), v, cards.New{Title: "Word", Fields: []string{"Meaning"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, v, made.Path); !strings.Contains(got, "fields:\n  - Meaning\n") {
		t.Errorf("the field was not declared: %q", got)
	}
}

// A note created with no index to update is still created, and reports its path.
func TestANoteIsMadeWithNoIndexToBringLevel(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock())

	made, err := u.Deck(t.Context(), v, cards.New{Title: "Birds", Body: "About birds."})
	if err != nil {
		t.Fatal(err)
	}
	if made.Title != "Birds" || made.ID == "" || made.Path != "Birds.md" {
		t.Errorf("made %+v", made)
	}
	if got := read(t, v, made.Path); !strings.Contains(got, "About birds.") {
		t.Errorf("the body was not written: %q", got)
	}
}

// A path something already occupies fails, and what is there is left.
func TestANoteIsNotMadeOverOneThatIsThere(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock())
	if err := os.WriteFile(filepath.Join(v.Path, "Birds.md"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := u.Deck(t.Context(), v, cards.New{Title: "Birds"}); err == nil {
		t.Fatal("a deck was made over a note")
	}
	if got := read(t, v, "Birds.md"); got != "mine" {
		t.Errorf("the note there is now %q", got)
	}
}

type unopenable struct{ ports.VaultWriters }

var errUnopenable = errors.New("the vault cannot be opened for writing")

func (unopenable) Open(domain.Vault) (ports.VaultWriter, error) { return nil, errUnopenable }

// A vault that cannot be opened for writing is the answer, and nothing is made.
func TestANoteIsNotMadeInAVaultThatCannotBeSaved(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(unopenable{}, nil, vaulttest.NewClock())

	if _, err := u.Deck(t.Context(), v, cards.New{Title: "Birds"}); !errors.Is(err, errUnopenable) {
		t.Errorf("make a deck = %v, want %v", err, errUnopenable)
	}
}

// The index a creation is given brings the new file level, with its path.
func TestANewNoteIsBroughtLevelInTheIndexItWasGiven(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	var levelled []string
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock()).WithIndex(
		func(_ context.Context, _ domain.Vault, paths []string) error {
			levelled = append(levelled, paths...)
			return nil
		})

	made, err := u.Deck(t.Context(), v, cards.New{Title: "Birds"})
	if err != nil {
		t.Fatal(err)
	}
	if len(levelled) != 1 || levelled[0] != made.Path {
		t.Errorf("the index was brought level with %v, want %s", levelled, made.Path)
	}
}

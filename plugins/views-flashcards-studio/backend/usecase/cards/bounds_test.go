package cards_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

type releaseFailWriters struct {
	ports.VaultWriters
	releaseErr error
}

func (w releaseFailWriters) Lock(ctx context.Context, v domain.Vault) (func() error, error) {
	release, err := w.VaultWriters.Lock(ctx, v)
	if err != nil {
		return nil, err
	}
	return func() error {
		return errors.Join(release(), w.releaseErr)
	}, nil
}

func TestRenameField_ErrorsWhenDeckCountExceedsBound(t *testing.T) {
	vs := newVaults(t)
	before := read(t, vs.first, "Animal.md")

	if cards.MaxDecksPerRename != 200 {
		t.Fatalf("MaxDecksPerRename = %d, want 200", cards.MaxDecksPerRename)
	}

	_, err := newRenameFieldOver(t, vs, vs.ports.Writers, vs.queries, 1).Execute(t.Context(), vs.first, cards.Rename{
		CardModel: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err == nil || !strings.Contains(err.Error(), "there are 2 of them") || !strings.Contains(err.Error(), "more than the 1") {
		t.Fatalf("a vault past the bound was renamed in: %v", err)
	}
	if read(t, vs.first, "Animal.md") != before {
		t.Error("the card model was written")
	}

	// checkDeckCount error propagation when maxDecks > 0.
	failBound := newRenameFieldOver(t, vs, vs.ports.Writers, decksOf{err: errOutOfReach}, 10)
	if _, err := failBound.Execute(t.Context(), vs.first, cards.Rename{
		CardModel: "Animal.md", From: "Height", To: "Shoulder height",
	}); !errors.Is(err, errOutOfReach) {
		t.Fatalf("got %v, want %v", err, errOutOfReach)
	}

	// maxDecks == 0 and maxDecks < 0 skip the deck count query.
	for _, limit := range []int{0, -1} {
		noopBound := newRenameFieldOver(t, vs, vs.ports.Writers, decksOf{err: errOutOfReach}, limit)
		got, err := noopBound.Execute(t.Context(), vs.first, cards.Rename{
			CardModel: "Animal.md", From: "Height", To: "Height",
		})
		if err != nil || got.Cards != 0 {
			t.Fatalf("limit %d: got (%+v, %v), want (Cards: 0, nil)", limit, got, err)
		}
	}

	// Release error from Lock is joined into the returned error.
	wantReleaseErr := errors.New("lock release failed")
	relRename := newRenameFieldOver(t, vs, releaseFailWriters{
		VaultWriters: vs.ports.Writers,
		releaseErr:   wantReleaseErr,
	}, vs.queries, 2)
	if _, err := relRename.Execute(t.Context(), vs.first, cards.Rename{
		CardModel: "Animal.md", From: "Height", To: "Shoulder height",
	}); !errors.Is(err, wantReleaseErr) {
		t.Fatalf("got %v, want %v", err, wantReleaseErr)
	}
}

func TestACardModelIsMadeWithItsTemplatesLaidDown(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock())

	made, err := u.CardModel(t.Context(), v, cards.New{
		Title: "Bird", Fields: []string{"Name", "Call"},
		Templates: []format.CardTemplate{{Name: "Recognize", Front: "{{Name}}", Back: "{{Call}}"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := read(t, v, made.Path)
	if !strings.Contains(body, "## Recognize") || !strings.Contains(body, "{{Call}}") {
		t.Errorf("the templates are not in the card model:\n%s", body)
	}
}

func TestACardModelWithTemplatesLargerThanANoteIsRefused(t *testing.T) {
	env := vaulttest.New(t, nil)
	v := env.Vault
	u := cards.NewCreate(env.Ports.Writers, nil, vaulttest.NewClock())

	_, err := u.CardModel(t.Context(), v, cards.New{
		Title: "Bird", Fields: []string{"Name"},
		Templates: []format.CardTemplate{{Name: "Big", Front: strings.Repeat("x", 1<<20), Back: "{{Name}}"}},
	})
	if err == nil || !strings.Contains(err.Error(), "more than this writes at once") {
		t.Errorf("an oversized card model was made: %v", err)
	}

	// CardModel body of exact note.MaxBytes succeeds.
	prefix := "\n## Exact\n\n### Front\n\n"
	suffix := "\n\n### Back\n\nx\n"
	exactFront := strings.Repeat("a", note.MaxBytes-len(prefix)-len(suffix))
	if _, err := u.CardModel(t.Context(), v, cards.New{
		Title: "ExactModel", Fields: []string{"Name"},
		Templates: []format.CardTemplate{{Name: "Exact", Front: exactFront, Back: "x"}},
	}); err != nil {
		t.Fatalf("expected exact note.MaxBytes card model to succeed, got %v", err)
	}
}

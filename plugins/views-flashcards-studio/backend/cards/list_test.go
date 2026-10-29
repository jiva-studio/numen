package cards_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// cardModelsOf is an index that names the card models of a vault from a list, or fails.
type cardModelsOf struct {
	err error
}

func (s cardModelsOf) GetCardModels(context.Context, domain.VaultID) ([]ports.CardModel, error) {
	return nil, s.err
}

// An index that cannot name the card models is an error of the listing, and no list
// is handed over beside it.
func TestAListingWhoseIndexCannotNameTheCardModelsIsAnError(t *testing.T) {
	vs := newVaults(t)

	got, count, err := cards.NewList(vs.ports.Readers, cardModelsOf{err: errOutOfReach}).
		Execute(t.Context(), vs.first, 0)
	if !errors.Is(err, errOutOfReach) {
		t.Fatalf("list = %v, want the index's error", err)
	}
	if got != nil || count != 0 {
		t.Errorf("list = %+v and %d, want nothing beside the error", got, count)
	}
}

// A card model the vault cannot look at is an error of the listing, and no list is
// handed over beside it.
func TestAListingWhoseCardModelCannotBeLookedAtIsAnError(t *testing.T) {
	vs := newVaults(t)
	readers := faulty{inner: vs.ports.Readers, stat: map[string]error{"Animal.md": errOutOfReach}}

	got, count, err := cards.NewList(readers, vs.queries).Execute(t.Context(), vs.first, 0)
	if !errors.Is(err, errOutOfReach) {
		t.Fatalf("list = %v, want the vault's error", err)
	}
	if got != nil || count != 0 {
		t.Errorf("list = %+v and %d, want nothing beside the error", got, count)
	}
}

// A limit above the number of card models lists all of them, and a limit below it
// lists that many while the count still says how many the vault holds.
func TestList_ReturnsTotalCountForAnyLimit(t *testing.T) {
	vs := newVaults(t)
	u := cards.NewList(vs.ports.Readers, vs.queries)

	for _, c := range []struct {
		limit int
		want  int
	}{{limit: 0, want: 2}, {limit: 1, want: 1}, {limit: 5, want: 2}} {
		got, count, err := u.Execute(t.Context(), vs.first, c.limit)
		if err != nil {
			t.Fatalf("limit %d: %v", c.limit, err)
		}
		if len(got) != c.want || count != 2 {
			t.Errorf("limit %d: listed %d of %d, want %d of 2", c.limit, len(got), count, c.want)
		}
	}
}

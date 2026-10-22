package cards_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
)

// stencilsOf is an index that names the stencils of a vault from a list, or fails.
type stencilsOf struct {
	err error
}

func (s stencilsOf) GetStencils(context.Context, domain.VaultID) ([]ports.Stencil, error) {
	return nil, s.err
}

// An index that cannot name the stencils is an error of the listing, and no list
// is handed over beside it.
func TestAListingWhoseIndexCannotNameTheStencilsIsAnError(t *testing.T) {
	vs := newVaults(t)

	got, count, err := cards.NewList(vs.ports.Readers, stencilsOf{err: errOutOfReach}).
		Execute(t.Context(), vs.first, 0)
	if !errors.Is(err, errOutOfReach) {
		t.Fatalf("list = %v, want the index's error", err)
	}
	if got != nil || count != 0 {
		t.Errorf("list = %+v and %d, want nothing beside the error", got, count)
	}
}

// A stencil the vault cannot look at is an error of the listing, and no list is
// handed over beside it.
func TestAListingWhoseStencilCannotBeLookedAtIsAnError(t *testing.T) {
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

// A limit above the number of stencils lists all of them, and a limit below it
// lists that many while the count still says how many the vault holds.
func TestAListingLimitedAboveOrBelowTheStencilsReturnsHowManyThereAre(t *testing.T) {
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

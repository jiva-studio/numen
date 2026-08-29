package review

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
)

// Owing is what one vault's cards come to today: what it holds, what is owed,
// and what has never been asked.
type Owing struct {
	// Seats is every card the vault holds, counted once for each face it is
	// shown through.
	Seats int
	// Due is the seats answered before and owed in the day holding now. New is
	// the seats nobody has answered.
	Due   int
	New   int
	Decks []DeckOwing
}

// DeckOwing is one deck's share of it, by the path of its file.
type DeckOwing struct {
	Deck  string
	Seats int
	Due   int
	New   int
}

// Owed is what a vault owes, which is what its front door shows.
type Owed struct {
	Seats     Seats
	Schedules Schedules
	Day       history.Day
	Now       func() time.Time
}

// Execute counts one vault.
func (u Owed) Execute(ctx context.Context, v domain.Vault) (Owing, error) {
	standing, err := u.Seats.Execute(ctx, v)
	if err != nil {
		return Owing{}, err
	}
	schedules, err := u.Schedules.Execute(ctx, v)
	if err != nil {
		return Owing{}, err
	}

	now := u.now()
	out := Owing{Seats: len(standing)}
	decks := make(map[string]*DeckOwing)
	for _, seat := range standing {
		deck, held := decks[seat.Deck]
		if !held {
			deck = &DeckOwing{Deck: seat.Deck}
			decks[seat.Deck] = deck
		}
		deck.Seats++

		s, answered := schedules[seat.Seat]
		switch {
		case !answered:
			out.New++
			deck.New++
		case u.Day.Owed(s, now):
			out.Due++
			deck.Due++
		}
	}

	for _, deck := range decks {
		out.Decks = append(out.Decks, *deck)
	}
	slices.SortFunc(out.Decks, func(a, b DeckOwing) int {
		return strings.Compare(a.Deck, b.Deck)
	})
	return out, nil
}

func (u Owed) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}

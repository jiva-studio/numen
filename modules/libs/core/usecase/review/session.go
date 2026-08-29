package review

import (
	"context"
	"slices"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
)

// Asked is one card face as it is put to a person: where it stands, how it is laid
// out, and where the answers so far have left it.
type Asked struct {
	Standing
	Schedule history.Schedule
	// Ahead is how long each of the four answers would leave this card, from
	// the moment it is asked. A person choosing between them is choosing
	// between these, so they are worked out with the card and not after it.
	Ahead map[history.Rating]time.Duration
}

// Sitting is what a person sits down to: the cards to ask, and what could not
// be acted on in getting them.
type Sitting struct {
	Asked []Asked
	// Unwritten are the decks holding a card with no mark that could not be
	// given one. Their cards are not in Asked and are asked for at the next
	// sitting.
	Unwritten []string
	// Skipped is how many lines of the vault's answers could not be read: a run
	// that stopped partway, or a line of a version this build does not know.
	Skipped int
}

// Session is what a person is asked, in the order they are asked it.
//
// A card owed and answered before comes first, the one waiting longest at the
// front, because a card left late is the one closest to being forgotten. Cards
// nobody has answered come after them, in the order they stand in their decks:
// a person wrote them in an order, and it is as good an order as any.
type Session struct {
	// Marking gives a mark to the cards of this vault that carry none, so that
	// what is asked can be answered. It is the one write review makes, and it
	// is made when a person sits down to a vault.
	Marking   Marking
	Standings Standings
	Schedules Schedules
	Day       history.Day
	Now       func() time.Time
}

// Execute is what to ask, in order.
//
// Deck is the path of one deck, or empty for every deck the vault holds. The
// whole vault is the ordinary way to sit down to this: a person owes what they
// owe, and which file a card is written in is not something they think about.
func (u Session) Execute(ctx context.Context, v domain.Vault, deck string) (Sitting, error) {
	marked, err := u.Marking.Execute(ctx, v)
	if err != nil {
		return Sitting{}, err
	}
	standing, err := u.Standings.Execute(ctx, v)
	if err != nil {
		return Sitting{}, err
	}

	// The log is read once here and the schedules worked out from it, so that
	// what a person is told about lines that could not be read is the reading
	// their own cards were laid out from.
	held, err := Log{Stores: u.Schedules.Logs}.Read(ctx, v)
	if err != nil {
		return Sitting{}, err
	}
	schedules := u.Schedules.From(ctx, v, held)

	out := Sitting{Unwritten: marked.Unwritten, Skipped: held.Skipped}
	now := u.now()
	var seen, fresh []Asked
	for _, one := range standing {
		if deck != "" && one.Deck != deck {
			continue
		}
		s, answered := schedules[one.CardFace]
		switch {
		case !answered:
			fresh = append(fresh, Asked{Standing: one, Ahead: u.ahead(history.Schedule{}, now)})
		case u.Day.Owed(s, now):
			seen = append(seen, Asked{Standing: one, Schedule: s, Ahead: u.ahead(s, now)})
		}
	}

	slices.SortStableFunc(seen, func(a, b Asked) int {
		return a.Schedule.Due.Compare(b.Schedule.Due)
	})
	out.Asked = append(seen, fresh...)
	return out, nil
}

// ahead is how long each of the four would leave a card standing where this
// schedule leaves it. It is what the scheduler answers and nothing else: the
// four are asked of it, and the card is left where it was.
func (u Session) ahead(s history.Schedule, now time.Time) map[history.Rating]time.Duration {
	by := u.Schedules.By
	if by == nil {
		return nil
	}
	out := make(map[history.Rating]time.Duration, 4)
	for _, r := range []history.Rating{history.Again, history.Hard, history.Good, history.Easy} {
		out[r] = by.Next(s, now, r).Due.Sub(now)
	}
	return out
}

func (u Session) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}

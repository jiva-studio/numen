package flashcards

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// CardsDue is what one vault's cards come to today: what it holds, what is owed,
// and what has never been asked.
type CardsDue struct {
	// Faces is every card the vault holds, counted once for each face it is
	// shown through.
	Faces int
	// Due is the card faces answered before and owed in the day holding now.
	// New is the ones nobody has answered. Both are held to what the budgets of
	// the day leave, so they are what a session will ask.
	Due   int
	New   int
	Decks []DeckCardsDue
	// Presets is what the day comes to under every preset the vault holds,
	// whether a deck points at it or not.
	Presets []PresetCardsDue
}

// DeckCardsDue is one deck's share of it, by the path of its file.
type DeckCardsDue struct {
	Deck  string
	Faces int
	Due   int
	New   int
	// Unbegun is how many of the deck's card faces nobody has answered at all.
	// A deck every face of which is one of these has nothing that can come
	// round until something begins them.
	Unbegun int
	// Answered is how many of the deck's cards were answered in the day holding
	// now, counted the way its preset counts.
	Answered int
	// Learned is how many of the deck's card faces stand learned at this
	// instant, under the rule the preset scheduling the deck counts by. Two
	// decks on one preset are counted under the one rule, and a deck naming no
	// preset under the defaults.
	Learned int
}

// PresetCardsDue is one preset's day, by the path of the note it stands in. A
// preset standing in no note schedules the decks naming none.
type PresetCardsDue struct {
	Preset string
	// Decks is how many decks name it, whatever they hold, and Cards is how
	// many card faces stand in those decks. A deck holding no cards points at
	// its preset all the same.
	Decks int
	Cards int
	// Due and New are what the day leaves under it: the card faces owed and the
	// ones nobody has answered, held to its budget. They are what a session over
	// this preset asks, because a preset is the whole scope of its own budget.
	Due int
	New int
	// Answered is how many of its cards were answered in the day holding now,
	// and Took is how long those answers took. AnsweredNew and AnsweredReviews
	// divide that count the way a budget does, so each is weighed against the
	// budget of its own kind.
	Answered        int
	AnsweredNew     int
	AnsweredReviews int
	Took            time.Duration
	// Budget is what the preset keeps for this day of the week. A budget its
	// goal does not name stands here as the person left it and closes nothing,
	// so Due and New are held to Limits and not to all three of these.
	Budget review.Budget
	// Limits is which of the three closes the day, and what each is called when
	// it does.
	Limits review.Limits
	// Stops is why the preset schedules nothing on this day, and empty where it
	// schedules something.
	Stops review.StopReason
}

// CountCardsDue is what a vault owes, which is what its front door shows.
type CountCardsDue struct {
	CardFaces ListCardFaces
	Schedules Schedules
	// Presets says which preset each deck is scheduled by. A build holding no
	// links schedules every deck by the defaults.
	Presets Presets
	Day     review.Day
	Now     port.Clock
}

// NewCountCardsDue is what a vault's front door is counted through: what stands
// in the vault, where the answers have left each card face, which preset each
// deck is scheduled by, where one day of review gives way to the next, and what
// time it is.
func NewCountCardsDue(
	faces ListCardFaces, schedules Schedules, presets Presets,
	day review.Day, now port.Clock,
) CountCardsDue {
	return CountCardsDue{
		CardFaces: faces, Schedules: schedules, Presets: presets, Day: day, Now: now,
	}
}

// Execute counts one vault.
//
// The log is read once here and the schedules worked out from it, so the count
// and the session it stands for are the one reading. Nothing is written into
// the vault: the person is shown every vault they hold, and none of them is
// written for that. What a replay came to is kept, because this is the path
// every launch waits on.
//
// A count nobody is waiting for is dropped at the next phase: reading the
// decks, reading the log and replaying it each run to their end.
func (u CountCardsDue) Execute(ctx context.Context, v domain.Vault) (CardsDue, error) {
	if err := ctx.Err(); err != nil {
		return CardsDue{}, err
	}
	faces, err := u.CardFaces.Execute(ctx, v)
	if err != nil {
		return CardsDue{}, err
	}
	if err := ctx.Err(); err != nil {
		return CardsDue{}, err
	}
	log, err := Log{Stores: u.Schedules.Logs}.Read(ctx, v)
	if err != nil {
		return CardsDue{}, err
	}
	// One reading of this vault's presets answers the schedulers, the budgets
	// and how many decks name each preset.
	reading := u.Presets.Reading()
	asks, err := u.Schedules.under(ctx, v, reading, faces)
	if err != nil {
		return CardsDue{}, err
	}
	schedules := u.Schedules.getSchedulesCached(ctx, v, log, asks)
	if err := ctx.Err(); err != nil {
		return CardsDue{}, err
	}

	now := u.Now()
	day, err := getBudgets(
		ctx, v, reading, u.Day, faces, schedules, log,
		u.Schedules.By, u.Schedules.at, now,
	)
	if err != nil {
		return CardsDue{}, err
	}
	holds := day.asks(faces, schedules, u.Day, now, Scope{})

	out := CardsDue{Faces: len(faces)}
	decks := make(map[string]*DeckCardsDue)
	at := func(deck string) *DeckCardsDue {
		one, held := decks[deck]
		if !held {
			one = &DeckCardsDue{Deck: deck}
			decks[deck] = one
		}
		return one
	}
	// The schedules and the presets are both in hand, so what stands learned is
	// counted off the reading that is already here.
	for _, one := range faces {
		row := at(one.Deck)
		row.Faces++
		if !schedules[one.ID].Seen() {
			row.Unbegun++
		}
		if asks.under(one.ID).Preset.Learned(schedules[one.ID], now) {
			row.Learned++
		}
	}
	for deck, one := range day.spentUnder {
		at(deck).Answered = one.Answered
	}
	// What each preset leaves is counted from the same pass the deck rows are,
	// so the tile over a preset and the session it opens are one number.
	due, fresh := make(map[string]int), make(map[string]int)
	for _, one := range holds.seen {
		out.Due++
		at(one.Deck).Due++
		due[day.under[one.ID]]++
	}
	for _, one := range holds.fresh {
		out.New++
		at(one.Deck).New++
		fresh[day.under[one.ID]]++
	}

	for _, deck := range decks {
		out.Decks = append(out.Decks, *deck)
	}
	slices.SortFunc(out.Decks, func(a, b DeckCardsDue) int {
		return strings.Compare(a.Deck, b.Deck)
	})
	out.Presets, err = u.presets(ctx, v, reading, day, due, fresh)
	if err != nil {
		return CardsDue{}, err
	}
	return out, nil
}

// presets is every preset the vault holds: the ones its decks point at, and
// then the ones nothing points at.
//
// How many decks name a preset is counted over every deck the vault holds, so a
// deck of no cards points at its preset like any other. A preset no deck names
// stands at nothing.
func (u CountCardsDue) presets(
	ctx context.Context, v domain.Vault, reading *PresetReads, day *budgets,
	due, fresh map[string]int,
) ([]PresetCardsDue, error) {
	out := day.getCardsDue(due, fresh)
	if u.CardFaces.Notes == nil {
		return out, nil
	}

	decks, err := u.CardFaces.Notes.OfType(ctx, v.ID, domain.TypeDeck)
	if err != nil {
		return nil, err
	}
	naming := make(map[string]int, len(decks))
	for _, deck := range decks {
		p, err := reading.Of(ctx, v, deck)
		if err != nil {
			return nil, err
		}
		naming[p.Path]++
	}

	pointed := make(map[string]bool, len(out))
	for at := range out {
		out[at].Decks = naming[out[at].Preset]
		pointed[out[at].Preset] = true
	}

	paths, err := u.CardFaces.Notes.OfType(ctx, v.ID, domain.TypePreset)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if pointed[path] {
			continue
		}
		// A preset no deck names has no day worked out for it, and its own
		// settings are what say whether it would schedule anything.
		one, err := reading.read(ctx, v, path)
		if err != nil {
			return nil, err
		}
		out = append(out, PresetCardsDue{
			Preset: path, Decks: naming[path], Stops: one.StopsToday,
		})
	}
	slices.SortFunc(out, func(a, b PresetCardsDue) int {
		return strings.Compare(a.Preset, b.Preset)
	})
	return out, nil
}

// getCardsDue is what the day comes to under each preset the vault's decks
// name: the budget the day of the week leaves it, and what has been answered
// under it since the day opened.
func (b *budgets) getCardsDue(due, fresh map[string]int) []PresetCardsDue {
	out := make([]PresetCardsDue, 0, len(b.left))
	for path, one := range b.left {
		out = append(out, PresetCardsDue{
			Preset: path, Cards: b.cards[path],
			Due: due[path], New: fresh[path],
			Answered:        one.spent.Answered,
			AnsweredNew:     one.spent.New,
			AnsweredReviews: one.spent.Reviews,
			Took:            one.spent.Took,
			Budget:          one.admits.Keeps, Limits: one.admits.Limits,
			Stops: one.admits.Stops,
		})
	}
	return out
}

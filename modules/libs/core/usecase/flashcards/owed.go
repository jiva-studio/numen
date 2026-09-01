package flashcards

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// Owing is what one vault's cards come to today: what it holds, what is owed,
// and what has never been asked.
type Owing struct {
	// Faces is every card the vault holds, counted once for each face it is
	// shown through.
	Faces int
	// Due is the card faces answered before and owed in the day holding now.
	// New is the ones nobody has answered. Both are held to what the budgets of
	// the day leave, so they are what a sitting will ask.
	Due   int
	New   int
	Decks []DeckOwing
	// Presets is what the day comes to under every preset the vault holds,
	// whether a deck points at it or not.
	Presets []PresetOwing
}

// DeckOwing is one deck's share of it, by the path of its file.
type DeckOwing struct {
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

// PresetOwing is one preset's day, by the path of the note it stands in. A
// preset standing in no note schedules the decks naming none.
type PresetOwing struct {
	Preset string
	// Decks is how many decks name it, whatever they hold, and Cards is how
	// many card faces stand in those decks. A deck holding no cards points at
	// its preset all the same.
	Decks int
	Cards int
	// Due and New are what the day leaves under it: the card faces owed and the
	// ones nobody has answered, held to its budget. They are what a sitting over
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
	// so Due and New are held to Closes and not to all three of these.
	Budget history.Budget
	// Closes is which of the three closes the day, and what each is called when
	// it does.
	Closes history.Closes
	// Stops is why the preset schedules nothing on this day, and empty where it
	// schedules something.
	Stops history.Stopped
}

// Owed is what a vault owes, which is what its front door shows.
type Owed struct {
	Standings Standings
	Schedules Schedules
	// Presets says which preset each deck is scheduled by. A build holding no
	// links schedules every deck by the defaults.
	Presets Presets
	Day     history.Day
	Now     func() time.Time
}

// Execute counts one vault.
//
// The log is read once here and the schedules worked out from it, so the count
// and the sitting it stands for are the one reading. Nothing is written into
// the vault: the person is shown every vault they hold, and none of them is
// written for that. What a replay came to is kept, because this is the path
// every launch waits on.
//
// A count nobody is waiting for is dropped at the next phase: reading the
// decks, reading the log and replaying it each run to their end.
func (u Owed) Execute(ctx context.Context, v domain.Vault) (Owing, error) {
	if err := ctx.Err(); err != nil {
		return Owing{}, err
	}
	standing, err := u.Standings.Execute(ctx, v)
	if err != nil {
		return Owing{}, err
	}
	if err := ctx.Err(); err != nil {
		return Owing{}, err
	}
	log, err := Log{Stores: u.Schedules.Logs}.Read(ctx, v)
	if err != nil {
		return Owing{}, err
	}
	// One reading of this vault's presets answers the schedulers, the budgets
	// and how many decks name each preset.
	reading := u.Presets.Reading()
	asks, err := u.Schedules.under(ctx, v, reading, standing)
	if err != nil {
		return Owing{}, err
	}
	schedules := u.Schedules.replayed(ctx, v, log, asks)
	if err := ctx.Err(); err != nil {
		return Owing{}, err
	}

	now := u.now()
	day, err := budgeted(
		ctx, v, reading, u.Day, standing, schedules, log,
		u.Schedules.By, u.Schedules.at, now,
	)
	if err != nil {
		return Owing{}, err
	}
	holds := day.asks(standing, schedules, u.Day, now, Over{})

	out := Owing{Faces: len(standing)}
	decks := make(map[string]*DeckOwing)
	at := func(deck string) *DeckOwing {
		one, held := decks[deck]
		if !held {
			one = &DeckOwing{Deck: deck}
			decks[deck] = one
		}
		return one
	}
	// The schedules and the presets are both in hand, so what stands learned is
	// counted off the reading that is already here.
	for _, one := range standing {
		row := at(one.Deck)
		row.Faces++
		if !schedules[one.CardFace].Seen() {
			row.Unbegun++
		}
		if asks.under(one.CardFace).Preset.Learned(schedules[one.CardFace], now) {
			row.Learned++
		}
	}
	for deck, one := range day.sat {
		at(deck).Answered = one.Answered
	}
	// What each preset leaves is counted from the same pass the deck rows are,
	// so the tile over a preset and the sitting it opens are one number.
	due, fresh := make(map[string]int), make(map[string]int)
	for _, one := range holds.seen {
		out.Due++
		at(one.Deck).Due++
		due[day.under[one.CardFace]]++
	}
	for _, one := range holds.fresh {
		out.New++
		at(one.Deck).New++
		fresh[day.under[one.CardFace]]++
	}

	for _, deck := range decks {
		out.Decks = append(out.Decks, *deck)
	}
	slices.SortFunc(out.Decks, func(a, b DeckOwing) int {
		return strings.Compare(a.Deck, b.Deck)
	})
	out.Presets, err = u.presets(ctx, v, reading, day, due, fresh)
	if err != nil {
		return Owing{}, err
	}
	return out, nil
}

// presets is every preset the vault holds: the ones its decks point at, and
// then the ones nothing points at.
//
// How many decks name a preset is counted over every deck the vault holds, so a
// deck of no cards points at its preset like any other. A preset no deck names
// stands at nothing.
func (u Owed) presets(
	ctx context.Context, v domain.Vault, reading *Reading, day *budgets,
	due, fresh map[string]int,
) ([]PresetOwing, error) {
	out := day.owing(due, fresh)
	if u.Standings.Notes == nil {
		return out, nil
	}

	decks, err := u.Standings.Notes.OfType(ctx, v.ID, domain.TypeDeck)
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

	paths, err := u.Standings.Notes.OfType(ctx, v.ID, domain.TypePreset)
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
		out = append(out, PresetOwing{
			Preset: path, Decks: naming[path], Stops: one.StopsToday,
		})
	}
	slices.SortFunc(out, func(a, b PresetOwing) int {
		return strings.Compare(a.Preset, b.Preset)
	})
	return out, nil
}

// owing is what the day comes to under each preset the vault's decks name: the
// budget the day of the week leaves it, and what has been answered under it
// since the day opened.
func (b *budgets) owing(due, fresh map[string]int) []PresetOwing {
	out := make([]PresetOwing, 0, len(b.left))
	for path, one := range b.left {
		out = append(out, PresetOwing{
			Preset: path, Cards: b.cards[path],
			Due: due[path], New: fresh[path],
			Answered:        one.spent.Answered,
			AnsweredNew:     one.spent.New,
			AnsweredReviews: one.spent.Reviews,
			Took:            one.spent.Took,
			Budget:          one.admits.Keeps, Closes: one.admits.Closes,
			Stops: one.admits.Stops,
		})
	}
	return out
}

func (u Owed) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}

package flashcards

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// budgets is what today leaves each preset a vault's decks are scheduled by,
// and which preset each card face stands under.
//
// The presets are separate scopes: one running out closes its own decks and no
// others. A preset standing in no note schedules the decks naming none.
type budgets struct {
	under map[history.CardFace]string
	left  map[string]*allowance
	// cards is how many card faces stand under each preset.
	cards map[string]int
	// faced are the card faces answered in the review day being sat.
	faced map[history.CardFace]bool
	// sat is what the review day being sat came to in each deck.
	sat map[string]history.Spent
}

// allowance is one preset's day: what the day admits, what an answer under it
// costs, how it counts, and what has gone on it already.
type allowance struct {
	admits history.Allowance
	cost   history.Cost
	counts history.Counts
	spent  history.Spent
}

// budgeted works out the day's budgets over the cards standing.
//
// What has been answered since the day opened is off it, so a second sitting
// takes up where the first left off.
func budgeted(
	ctx context.Context, v domain.Vault, reading *Reading, day history.Day,
	standing []Standing, schedules map[history.CardFace]history.Schedule,
	log Held, by history.Scheduler, at func(retention float64) history.Scheduler,
	now time.Time,
) (*budgets, error) {
	out := &budgets{
		under: make(map[history.CardFace]string, len(standing)),
		left:  make(map[string]*allowance),
		cards: make(map[string]int),
	}

	// A deck is asked once which preset schedules it, however many card faces it
	// holds, and a preset note is opened once however many decks name it.
	asked := make(map[string]string, len(standing))
	// The deck each card face stands in, which is how the day's answers are
	// grouped, and the settings each preset was read with.
	in := make(map[history.CardFace]string, len(standing))
	settings := make(map[string]history.Preset)
	// The material each preset has still to begin.
	unseen := make(map[string]int)
	for _, one := range standing {
		path, known := asked[one.Deck]
		if !known {
			p, err := reading.Of(ctx, v, one.Deck)
			if err != nil {
				return nil, err
			}
			path = p.Path
			asked[one.Deck] = path
			if _, held := settings[path]; !held {
				settings[path] = p.Preset
			}
		}
		out.under[one.CardFace] = path
		in[one.CardFace] = one.Deck
		out.cards[path]++
		if _, answered := schedules[one.CardFace]; !answered {
			unseen[path]++
		}
	}

	counting := make(map[string]history.Counts, len(asked))
	for deck, path := range asked {
		counting[deck] = settings[path].Counts
	}
	named := day.Names(now)
	out.faced = history.Faced(day, named, log.Answers)

	// A card face stands in one deck and one preset, so a preset's day is the
	// sum of the days of the decks that name it.
	out.sat = history.Sat(day, named, log.Answers, in, counting)
	spent := make(map[string]history.Spent, len(settings))
	for deck, one := range out.sat {
		at := spent[asked[deck]]
		at.Answered += one.Answered
		at.New += one.New
		at.Reviews += one.Reviews
		at.Took += one.Took
		spent[asked[deck]] = at
	}

	costed := history.CostedUnder(by, log.Answers, out.under)
	for path, p := range settings {
		cost, held := costed[path]
		if !held {
			cost = history.DefaultCost
		}
		// A date paces the day against how long a card face begun today takes to
		// be learned, worked out under the scheduler this preset's cards are
		// spaced by. No other goal reads it, and it is asked for under no other.
		left := history.Left{New: unseen[path]}
		if p.Goal == history.GoalDate {
			left.Ripens = history.Ripens(at(p.Retention), day, p, now)
		}
		out.left[path] = &allowance{
			admits: p.Admits(day, now, spent[path], left),
			cost:   cost,
			counts: p.Counts,
			spent:  spent[path],
		}
	}
	return out, nil
}

// takes reports whether the preset a card face stands under has room for it
// today, and spends the room where it has.
//
// The day is closed by the budget the preset's goal names, and every other
// budget takes no part. A preset counting in cards charges a card face the
// first time the day answers it, so a face the day has already charged comes
// round again for no count. The minutes are spent on every answer whichever way
// the preset counts.
func (b *budgets) takes(face history.CardFace, fresh bool) bool {
	one, held := b.left[b.under[face]]
	if !held || one.admits.Paused {
		return false
	}
	counted := one.counts == history.CountsShows || !b.faced[face]
	cost, left, closes := one.cost.Review, &one.admits.Reviews, one.admits.Closes.Reviews
	if fresh {
		cost, left, closes = one.cost.New, &one.admits.New, one.admits.Closes.New
	}
	if counted && closes != history.ClosedNothing && *left <= 0 {
		return false
	}
	if one.admits.Closes.Minutes != history.ClosedNothing && one.admits.Minutes < cost {
		return false
	}
	if counted {
		*left--
	}
	one.admits.Minutes -= cost
	return true
}

// asking is what a day holds of the cards standing: the ones owed, the one
// waiting longest at the front, and then the ones nobody has answered.
type asking struct {
	seen  []Standing
	fresh []Standing
}

// holds reports whether a card standing here is one of the cards the sitting is
// opened over.
//
// A sitting over a preset takes the cards of every deck pointing at it, so the
// one budget spent is that preset's.
func (b *budgets) holds(one Standing, over Over) bool {
	if over.ByPreset {
		return b.under[one.CardFace] == over.Preset
	}
	return over.Deck == "" || one.Deck == over.Deck
}

// refuses says why a preset has nothing to ask in the day being sat, in the
// person's own words.
func (b *budgets) refuses(preset string) error {
	one, scheduling := b.left[preset]
	switch {
	case !scheduling:
		return fmt.Errorf("%w: no deck of this vault is scheduled by it", ErrSchedulesNothing)
	case one.admits.Paused:
		return fmt.Errorf("%w: it is paused", ErrSchedulesNothing)
	case one.spent.Answered > 0:
		return fmt.Errorf("%w: its day is spent", ErrSchedulesNothing)
	default:
		return fmt.Errorf("%w: nothing under it is owed yet", ErrSchedulesNothing)
	}
}

// asks is what the budgets leave of the cards standing, in the order they are
// put to a person.
//
// Each preset spends its day between the debt before it and the material it has
// not begun, in the share it names. What is taken is then put in one order: the
// debt first, the card waiting longest at the front, and the cards nobody has
// answered after it, in the order they stand in their decks.
func (b *budgets) asks(
	standing []Standing, schedules map[history.CardFace]history.Schedule,
	day history.Day, now time.Time, over Over,
) asking {
	var owed, fresh []Standing
	for _, one := range standing {
		if !b.holds(one, over) {
			continue
		}
		s, answered := schedules[one.CardFace]
		switch {
		case !answered:
			fresh = append(fresh, one)
		case day.Owed(s, now):
			owed = append(owed, one)
		}
	}
	slices.SortStableFunc(owed, func(a, b Standing) int {
		return schedules[a.CardFace].Due.Compare(schedules[b.CardFace].Due)
	})

	took := b.spends(owed, fresh)
	var out asking
	for at, one := range owed {
		if took.owed[at] {
			out.seen = append(out.seen, one)
		}
	}
	for at, one := range fresh {
		if took.fresh[at] {
			out.fresh = append(out.fresh, one)
		}
	}
	return out
}

// spending is which of the cards put to a preset its day took.
type spending struct{ owed, fresh []bool }

// spends is what each preset's day takes of the debt before it and the material
// it has not begun.
//
// The two are offered one against the other in the share the preset names, and
// a side that runs out leaves the rest of the day to the other. An odd card goes
// to the debt, so a day holding one card spends it on what is already begun.
func (b *budgets) spends(owed, fresh []Standing) spending {
	out := spending{owed: make([]bool, len(owed)), fresh: make([]bool, len(fresh))}

	// The cards of each preset, in the order they stand.
	type queue struct{ owed, fresh []int }
	at := make(map[string]*queue)
	var order []string
	into := func(path string) *queue {
		one, held := at[path]
		if !held {
			one = &queue{}
			at[path] = one
			order = append(order, path)
		}
		return one
	}
	for i, one := range owed {
		q := into(b.under[one.CardFace])
		q.owed = append(q.owed, i)
	}
	for i, one := range fresh {
		q := into(b.under[one.CardFace])
		q.fresh = append(q.fresh, i)
	}
	// The presets are asked in one order, so a sitting asked twice is the same
	// sitting.
	slices.Sort(order)

	for _, path := range order {
		q := at[path]
		admits := history.Allowance{Backlog: history.AllBacklog}
		if one, held := b.left[path]; held {
			admits = one.admits
		}
		var debt, begun, i, j int
		for i < len(q.owed) || j < len(q.fresh) {
			// The next card comes from the side the share leaves short, and
			// from whichever side is left when the other is done.
			if admits.Paying(debt, begun, i < len(q.owed), j < len(q.fresh)) {
				if card := owed[q.owed[i]]; b.takes(card.CardFace, false) {
					out.owed[q.owed[i]] = true
					debt++
				}
				i++
				continue
			}
			if card := fresh[q.fresh[j]]; b.takes(card.CardFace, true) {
				out.fresh[q.fresh[j]] = true
				begun++
			}
			j++
		}
	}
	return out
}

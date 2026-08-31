package flashcards

import (
	"context"
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

// allowance is one preset's day: what it keeps, what an answer under it costs,
// what has gone on it already, and what is left of each of the three.
type allowance struct {
	budget history.Budget
	cost   history.Cost
	counts history.Counts
	spent  history.Spent
	paused bool
	// closes is which of the three budgets closes this preset's day.
	closes  history.Closes
	new     int
	reviews int
	minutes time.Duration
}

// budgeted works out the day's budgets over the cards standing.
//
// What has been answered since the day opened is off it, so a second sitting
// takes up where the first left off. A paused preset keeps nothing.
func budgeted(
	ctx context.Context, v domain.Vault, reading *Reading, day history.Day,
	standing []Standing, schedules map[history.CardFace]history.Schedule,
	log Held, by history.Scheduler, now time.Time,
) (*budgets, error) {
	out := &budgets{
		under: make(map[history.CardFace]string, len(standing)),
		left:  make(map[string]*allowance),
		cards: make(map[string]int),
	}

	// The load a day carries is the load of the review day being sat, which is
	// the day the spending is counted in.
	opened := day.Ends(now).AddDate(0, 0, -1)

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
			if _, held := out.left[path]; !held {
				settings[path] = p.Preset
				out.left[path] = &allowance{
					budget: p.Preset.On(opened.Weekday()),
					cost:   history.DefaultCost,
					counts: p.Preset.Counts,
					closes: p.Preset.Closing(),
					paused: p.Preset.Paused(day, now),
				}
			}
		}
		out.under[one.CardFace] = path
		in[one.CardFace] = one.Deck
		out.cards[path]++
		if _, answered := schedules[one.CardFace]; !answered {
			unseen[path]++
		}
	}
	// A day of a date holds the share of the material still to begin that has to
	// be begun to be through it by that day.
	for path, one := range out.left {
		if p := settings[path]; p.Goal == history.GoalDate {
			one.budget.New = paced(unseen[path], p.Days(day, now))
		}
	}

	counting := make(map[string]history.Counts, len(asked))
	for deck, path := range asked {
		counting[deck] = out.left[path].counts
	}
	named := day.Names(now)
	out.faced = history.Faced(day, named, log.Answers)

	for path, one := range history.CostedUnder(by, log.Answers, out.under) {
		out.left[path].cost = one
	}
	// A card face stands in one deck and one preset, so a preset's day is the
	// sum of the days of the decks that name it.
	out.sat = history.Sat(day, named, log.Answers, in, counting)
	for deck, one := range out.sat {
		spent := &out.left[asked[deck]].spent
		spent.Answered += one.Answered
		spent.New += one.New
		spent.Reviews += one.Reviews
		spent.Took += one.Took
	}
	for _, one := range out.left {
		one.new = one.budget.New - one.spent.New
		one.reviews = one.budget.Reviews - one.spent.Reviews
		one.minutes = time.Duration(one.budget.Minutes*float64(time.Minute)) - one.spent.Took
	}
	return out, nil
}

// paced is how much of the material a day holds when a date sets the pace: what
// is left to begin, over the days left to begin it in.
func paced(left, days int) int {
	if days <= 0 {
		return left
	}
	return (left + days - 1) / days
}

// takes reports whether the preset a card face stands under has room for it
// today, and spends the room where it has.
//
// The day is closed by the budget the preset's goal names, and every other
// budget takes no part. A preset counting in cards charges a card face the
// first time the day answers it, so a face the day has already charged comes
// round again for nothing.
func (b *budgets) takes(face history.CardFace, fresh bool) bool {
	one, held := b.left[b.under[face]]
	if !held || one.paused {
		return false
	}
	if one.counts != history.CountsShows && b.faced[face] {
		return true
	}
	cost, left, closes := one.cost.Review, &one.reviews, one.closes.Reviews
	if fresh {
		cost, left, closes = one.cost.New, &one.new, one.closes.New
	}
	if closes && *left <= 0 {
		return false
	}
	if one.closes.Minutes && one.minutes < cost {
		return false
	}
	*left--
	one.minutes -= cost
	return true
}

// asking is what a day holds of the cards standing: the ones owed, the one
// waiting longest at the front, and then the ones nobody has answered.
type asking struct {
	seen  []Standing
	fresh []Standing
}

// asks is what the budgets leave of the cards standing, in the order they are
// put to a person.
//
// The debt is paid before anything new is taken on, so a day too short for both
// is a day of cards already begun. Deck is one deck, or empty for every deck.
func (b *budgets) asks(
	standing []Standing, schedules map[history.CardFace]history.Schedule,
	day history.Day, now time.Time, deck string,
) asking {
	var owed []Standing
	for _, one := range standing {
		if deck != "" && one.Deck != deck {
			continue
		}
		if s, answered := schedules[one.CardFace]; answered && day.Owed(s, now) {
			owed = append(owed, one)
		}
	}
	slices.SortStableFunc(owed, func(a, b Standing) int {
		return schedules[a.CardFace].Due.Compare(schedules[b.CardFace].Due)
	})

	var out asking
	for _, one := range owed {
		if b.takes(one.CardFace, false) {
			out.seen = append(out.seen, one)
		}
	}
	for _, one := range standing {
		if deck != "" && one.Deck != deck {
			continue
		}
		if _, answered := schedules[one.CardFace]; answered {
			continue
		}
		if b.takes(one.CardFace, true) {
			out.fresh = append(out.fresh, one)
		}
	}
	return out
}

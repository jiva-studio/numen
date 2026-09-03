package flashcards

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
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
	under map[history.CardFaceID]string
	left  map[string]*allowance
	// decks says which preset schedules each deck, by the path of its file.
	decks map[string]string
	// cards is how many card faces stand under each preset.
	cards map[string]int
	// faced are the card faces answered in the review day being sat.
	faced map[history.CardFaceID]bool
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
	standing []Standing, schedules map[history.CardFaceID]history.Schedule,
	log Held, by history.Scheduler, at func(retention float64) history.Scheduler,
	now time.Time,
) (*budgets, error) {
	out := &budgets{
		under: make(map[history.CardFaceID]string, len(standing)),
		left:  make(map[string]*allowance),
		cards: make(map[string]int),
	}

	// A deck is asked once which preset schedules it, however many card faces it
	// holds, and a preset note is opened once however many decks name it.
	asked := make(map[string]string, len(standing))
	// The deck each card face stands in, which is how the day's answers are
	// grouped, and the settings each preset was read with.
	in := make(map[history.CardFaceID]string, len(standing))
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

	out.decks = asked
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
// today, in the deck the face stands in, and spends the room where it has.
//
// The day is closed by the budget the preset's goal names, and every other
// budget takes no part. A card is charged both to the deck's share of the day
// and to the day itself, so a deck spends its own share and the preset's
// ceiling holds over all of them. A preset counting in cards charges a card
// face the first time the day answers it, so a face the day has already charged
// comes round again for no count. The minutes are spent on every answer
// whichever way the preset counts.
func (b *budgets) takes(share *allowance, face history.CardFaceID, fresh bool) bool {
	one, held := b.left[b.under[face]]
	if !held || one.admits.Paused() {
		return false
	}
	counted := one.counts.Charges(b.faced[face])
	cost := one.cost.Review
	if fresh {
		cost = one.cost.New
	}
	if !one.room(fresh, counted, cost) || !share.room(fresh, counted, cost) {
		return false
	}
	one.spends(fresh, counted, cost)
	share.spends(fresh, counted, cost)
	return true
}

// room reports whether this day has a place left for a card face of this kind
// at this cost.
func (a *allowance) room(fresh, counted bool, cost time.Duration) bool {
	left, closes := a.admits.Reviews, a.admits.Closes.Reviews
	if fresh {
		left, closes = a.admits.New, a.admits.Closes.New
	}
	if counted && closes != history.ClosedNothing && left <= 0 {
		return false
	}
	return a.admits.Closes.Minutes == history.ClosedNothing || a.admits.Minutes >= cost
}

// spends takes a card face of this kind out of this day.
func (a *allowance) spends(fresh, counted bool, cost time.Duration) {
	if counted {
		if fresh {
			a.admits.New--
		} else {
			a.admits.Reviews--
		}
	}
	a.admits.Minutes -= cost
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
	if over.Named {
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
	case one.admits.Paused():
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
//
// The day is divided over every deck the preset schedules before anything is
// held back, and the sitting is then the slice of that division belonging to
// what it was opened over. A deck's row on the front door and what pressing
// that deck hands over are the one division.
func (b *budgets) asks(
	standing []Standing, schedules map[history.CardFaceID]history.Schedule,
	day history.Day, now time.Time, over Over,
) asking {
	var owed, fresh []Standing
	for _, one := range standing {
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
		if took.owed[at] && b.holds(one, over) {
			out.seen = append(out.seen, one)
		}
	}
	for at, one := range fresh {
		if took.fresh[at] && b.holds(one, over) {
			out.fresh = append(out.fresh, one)
		}
	}
	return out
}

// spending is which of the cards put to a preset its day took.
type spending struct{ owed, fresh []bool }

// dealt is one deck's cards under one preset, in the order they stand, and how
// far its share of the day has been walked through them.
type dealt struct {
	deck        string
	owed, fresh []int
	// seen and unseen are how far each side has been walked, and debt and begun
	// how many of each the deck has taken.
	seen, unseen int
	debt, begun  int
}

// spends is what each preset's day takes of the debt before it and the material
// it has not begun.
//
// The day is divided over the decks the preset schedules, each deck's share in
// proportion to what it owes. Within a deck the debt and the material not begun
// are offered one against the other in the share the preset names, and a side
// that runs out leaves the rest of the deck's share to the other. An odd card
// goes to the debt, so a share holding one card spends it on what is already
// begun. What no deck could use out of its own share is offered round again, so
// the day spends what it holds.
func (b *budgets) spends(owed, fresh []Standing) spending {
	out := spending{owed: make([]bool, len(owed)), fresh: make([]bool, len(fresh))}

	at := make(map[string]map[string]*dealt)
	var order []string
	into := func(path, deck string) *dealt {
		decks, held := at[path]
		if !held {
			decks = make(map[string]*dealt)
			at[path] = decks
			order = append(order, path)
		}
		one, held := decks[deck]
		if !held {
			one = &dealt{deck: deck}
			decks[deck] = one
		}
		return one
	}
	for i, one := range owed {
		q := into(b.under[one.CardFace], one.Deck)
		q.owed = append(q.owed, i)
	}
	for i, one := range fresh {
		q := into(b.under[one.CardFace], one.Deck)
		q.fresh = append(q.fresh, i)
	}
	// A deck already sat through today holds a share of the day whether or not
	// it has a card left to give, so what it took stands against its own share.
	for deck, path := range b.decks {
		if _, sat := b.sat[deck]; sat {
			into(path, deck)
		}
	}
	// The presets are asked in one order, so a sitting asked twice is the same
	// sitting.
	slices.Sort(order)

	for _, path := range order {
		one, held := b.left[path]
		if !held || one.admits.Paused() {
			continue
		}
		// The decks are handed their shares in the order their paths stand, so
		// the division does not turn on the order the vault was walked in.
		decks := make([]*dealt, 0, len(at[path]))
		for _, q := range at[path] {
			decks = append(decks, q)
		}
		slices.SortFunc(decks, func(a, b *dealt) int {
			return strings.Compare(a.deck, b.deck)
		})

		shares := b.divides(one, decks)
		for i, q := range decks {
			b.deals(&shares[i], q, owed, fresh, &out)
		}
		// What is left of the day after every deck has had its share goes to
		// the decks that still hold a card, in the order they stand.
		for _, q := range decks {
			q.seen, q.unseen = 0, 0
			over := *one
			b.deals(&over, q, owed, fresh, &out)
		}
	}
	return out
}

// divides is each deck's share of one preset's day.
//
// The whole day is handed out in proportion to what each deck owes of it, and
// what a deck has already answered today comes off that deck's own share, so a
// deck sat first spends its share and no other deck's.
func (b *budgets) divides(one *allowance, decks []*dealt) []allowance {
	reviews := make([]float64, len(decks))
	begun := make([]float64, len(decks))
	minutes := make([]float64, len(decks))
	for at, q := range decks {
		sat := b.sat[q.deck]
		reviews[at] = float64(len(q.owed) + sat.Reviews)
		begun[at] = float64(len(q.fresh) + sat.New)
		minutes[at] = float64(time.Duration(len(q.owed))*one.cost.Review +
			time.Duration(len(q.fresh))*one.cost.New + sat.Took)
	}
	keeps := one.admits.Keeps
	reviews = divided(float64(keeps.Reviews), reviews)
	begun = divided(float64(keeps.New), begun)
	minutes = divided(keeps.Minutes*float64(time.Minute), minutes)

	out := make([]allowance, len(decks))
	for at, q := range decks {
		sat := b.sat[q.deck]
		out[at] = *one
		out[at].admits.Reviews = max(0, int(reviews[at])-sat.Reviews)
		out[at].admits.New = max(0, int(begun[at])-sat.New)
		out[at].admits.Minutes = max(0, time.Duration(minutes[at])-sat.Took)
	}
	return out
}

// divided hands a budget out in proportion to what each deck owes of it.
//
// A deck owing nine times another's takes nine times the share, a deck owing
// nothing takes nothing, and no deck takes more than it owes. What the
// proportions leave over goes by the largest fraction, and decks standing equal
// take it in the order they are given.
func divided(budget float64, owes []float64) []float64 {
	out := make([]float64, len(owes))
	var total float64
	for _, one := range owes {
		total += one
	}
	if total <= 0 {
		return out
	}
	if budget >= total {
		copy(out, owes)
		return out
	}
	over := make([]int, len(owes))
	parts := make([]float64, len(owes))
	left := budget
	for at, one := range owes {
		exact := budget * one / total
		out[at] = math.Floor(exact)
		parts[at] = exact - out[at]
		left -= out[at]
		over[at] = at
	}
	slices.SortStableFunc(over, func(a, b int) int {
		return cmp.Compare(parts[b], parts[a])
	})
	for _, at := range over[:min(len(over), int(left))] {
		out[at]++
	}
	return out
}

// deals is what one deck's share of the day takes of the debt before it and the
// material it has not begun. A card another share has already taken is passed
// over, and the deck picks up where its share left off.
func (b *budgets) deals(share *allowance, q *dealt, owed, fresh []Standing, out *spending) {
	for {
		for q.seen < len(q.owed) && out.owed[q.owed[q.seen]] {
			q.seen++
		}
		for q.unseen < len(q.fresh) && out.fresh[q.fresh[q.unseen]] {
			q.unseen++
		}
		debt, begun := q.seen < len(q.owed), q.unseen < len(q.fresh)
		if !debt && !begun {
			return
		}
		// The next card comes from the side the share leaves short, and from
		// whichever side is left when the other is done.
		if share.admits.Paying(q.debt, q.begun, debt, begun) {
			if card := owed[q.owed[q.seen]]; b.takes(share, card.CardFace, false) {
				out.owed[q.owed[q.seen]] = true
				q.debt++
			}
			q.seen++
			continue
		}
		if card := fresh[q.fresh[q.unseen]]; b.takes(share, card.CardFace, true) {
			out.fresh[q.fresh[q.unseen]] = true
			q.begun++
		}
		q.unseen++
	}
}

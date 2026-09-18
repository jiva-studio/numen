package flashcards

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// budgets is what today leaves each preset a vault's decks are scheduled by,
// and which preset each card face stands under.
//
// The presets are separate scopes: one running out closes its own decks and no
// others. A preset standing in no note schedules the decks naming none.
type budgets struct {
	under map[review.CardFaceID]string
	left  map[string]*allowance
	// decks says which preset schedules each deck, by the path of its file.
	decks map[string]string
	// cards is how many card faces stand under each preset.
	cards map[string]int
	// faced are the card faces answered in the review day being sat.
	faced map[review.CardFaceID]bool
	// spentUnder is what the review day came to in each deck.
	spentUnder map[string]review.Spent
}

// allowance is one preset's day: what the day admits, what an answer under it
// costs, how it counts, and what has gone on it already.
type allowance struct {
	admits review.Allowance
	cost   review.AnswerCost
	counts review.BudgetUnit
	spent  review.Spent
}

// getBudgets works out the day's budgets over the cards standing.
//
// What has been answered since the day opened is off it, so a second session
// takes up where the first left off.
func getBudgets(
	ctx context.Context, v domain.Vault, reading *PresetReads, day review.Day,
	faces []CardFace, schedules map[review.CardFaceID]review.Schedule,
	log ReviewLog, by review.Scheduler, at func(retention float64) review.Scheduler,
	now time.Time,
) (*budgets, error) {
	out := &budgets{
		under: make(map[review.CardFaceID]string, len(faces)),
		left:  make(map[string]*allowance),
		cards: make(map[string]int),
	}

	// A deck is asked once which preset schedules it, however many card faces it
	// holds, and a preset note is opened once however many decks name it.
	asked := make(map[string]string, len(faces))
	// The deck each card face stands in, which is how the day's answers are
	// grouped, and the settings each preset was read with.
	in := make(map[review.CardFaceID]string, len(faces))
	settings := make(map[string]review.Preset)
	// The material each preset has still to begin.
	unseen := make(map[string]int)
	for _, one := range faces {
		path, known := asked[one.Deck]
		if !known {
			p, err := reading.GetForDeck(ctx, v, one.Deck)
			if err != nil {
				return nil, err
			}
			path = p.Path
			asked[one.Deck] = path
			if _, held := settings[path]; !held {
				settings[path] = p.Settings
			}
		}
		out.under[one.ID] = path
		in[one.ID] = one.Deck
		out.cards[path]++
		if _, answered := schedules[one.ID]; !answered {
			unseen[path]++
		}
	}

	out.decks = asked
	counting := make(map[string]review.BudgetUnit, len(asked))
	for deck, path := range asked {
		counting[deck] = settings[path].Counts
	}
	named := day.GetName(now)
	out.faced = review.GetFaced(day, named, log.Answers)

	// A card face stands in one deck and one preset, so a preset's day is the
	// sum of the days of the decks that name it.
	out.spentUnder = review.GetSpentUnder(day, named, log.Answers, in, counting)
	spent := make(map[string]review.Spent, len(settings))
	for deck, one := range out.spentUnder {
		at := spent[asked[deck]]
		at.Answered += one.Answered
		at.New += one.New
		at.Reviews += one.Reviews
		at.Took += one.Took
		spent[asked[deck]] = at
	}

	costed := review.GetCostUnder(by, log.Answers, out.under)
	for path, p := range settings {
		cost, held := costed[path]
		if !held {
			cost = review.DefaultCost
		}
		// A date paces the day against how long a card face begun today takes to
		// be learned, worked out under the scheduler this preset's cards are
		// spaced by. No other goal reads it, and it is asked for under no other.
		learn := 0
		if p.Goal == review.GoalDate {
			learn = review.GetRipeningDays(at(p.Retention), day, p, now)
		}
		out.left[path] = &allowance{
			admits: p.GetAllowance(day, now, spent[path], unseen[path], learn),
			cost:   cost,
			counts: p.Counts,
			spent:  spent[path],
		}
	}
	return out, nil
}

// charge reports whether the preset a card face stands under has room for it
// today, in the deck the face stands in, and spends the room where it has.
//
// The day is closed by the budget the preset's goal names, and every other
// budget charge no part. A card is charged both to the deck's share of the day
// and to the day itself, so a deck spends its own share and the preset's
// ceiling holds over all of them. A preset counting in cards charges a card
// face the first time the day answers it, so a face the day has already charged
// comes round again for no count. The minutes are spent on every answer
// whichever way the preset counts.
func (b *budgets) charge(share *allowance, face review.CardFaceID, fresh bool) bool {
	one, held := b.left[b.under[face]]
	if !held || one.admits.IsPaused() {
		return false
	}
	counted := one.counts.IsCharged(b.faced[face])
	cost := one.cost.Review
	if fresh {
		cost = one.cost.New
	}
	if !one.room(fresh, counted, cost) || !share.room(fresh, counted, cost) {
		return false
	}
	one.spend(fresh, counted, cost)
	share.spend(fresh, counted, cost)
	return true
}

// room reports whether this day has a place left for a card face of this kind
// at this cost.
func (a *allowance) room(fresh, counted bool, cost time.Duration) bool {
	left, closes := a.admits.Reviews, a.admits.Limits.Reviews
	if fresh {
		left, closes = a.admits.New, a.admits.Limits.New
	}
	if counted && closes != review.ClosedNothing && left <= 0 {
		return false
	}
	return a.admits.Limits.Minutes == review.ClosedNothing || a.admits.Minutes >= cost
}

// spend takes a card face of this kind out of this day.
func (a *allowance) spend(fresh, counted bool, cost time.Duration) {
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
	seen  []CardFace
	fresh []CardFace
}

// isAllowed reports whether a card standing here is one of the cards the
// session is opened over.
//
// A session over a preset takes the cards of every deck pointing at it, so the
// one budget spent is that preset's.
func (b *budgets) isAllowed(one CardFace, over Scope) bool {
	if over.IsNamed {
		return b.under[one.ID] == over.Preset
	}
	return over.Deck == "" || one.Deck == over.Deck
}

// getStopError says why a preset has nothing to ask in the day being sat, in
// the person's own words.
func (b *budgets) getStopError(preset string) error {
	one, scheduling := b.left[preset]
	switch {
	case !scheduling:
		return fmt.Errorf("%w: no deck of this vault is scheduled by it", ErrSchedulesNothing)
	case one.admits.IsPaused():
		return fmt.Errorf("%w: it is paused", ErrSchedulesNothing)
	case one.spent.Answered > 0:
		return fmt.Errorf("%w: its day is spent", ErrSchedulesNothing)
	default:
		return fmt.Errorf("%w: nothing under it is owed yet", ErrSchedulesNothing)
	}
}

// getAsking is what the budgets leave of the cards standing, in the order they
// are put to a person.
//
// Each preset spends its day between the debt before it and the material it has
// not begun, in the share it names. What is taken is then put in one order: the
// debt first, the card waiting longest at the front, and the cards nobody has
// answered after it, in the order they stand in their decks.
//
// The day is divided over every deck the preset schedules before anything is
// held back, and the session is then the slice of that division belonging to
// what it was opened over. A deck's row on the front door and what pressing
// that deck hands over are the one division.
func (b *budgets) getAsking(
	faces []CardFace, schedules map[review.CardFaceID]review.Schedule,
	day review.Day, now time.Time, over Scope,
) asking {
	var owed, fresh []CardFace
	for _, one := range faces {
		s, answered := schedules[one.ID]
		switch {
		case !answered:
			fresh = append(fresh, one)
		case day.IsOwed(s, now):
			owed = append(owed, one)
		}
	}
	slices.SortStableFunc(owed, func(a, b CardFace) int {
		return schedules[a.ID].Due.Compare(schedules[b.ID].Due)
	})

	took := b.spend(owed, fresh)
	var out asking
	for at, one := range owed {
		if took.owed[at] && b.isAllowed(one, over) {
			out.seen = append(out.seen, one)
		}
	}
	for at, one := range fresh {
		if took.fresh[at] && b.isAllowed(one, over) {
			out.fresh = append(out.fresh, one)
		}
	}
	return out
}

// taken is which of the cards put to a preset its day took.
type taken struct{ owed, fresh []bool }

// deckShare is one deck's cards under one preset, in the order they stand, and
// how far its share of the day has been walked through them.
type deckShare struct {
	deck        string
	owed, fresh []int
	// seen and unseen are how far each side has been walked, and debt and begun
	// how many of each the deck has taken.
	seen, unseen int
	debt, begun  int
}

// countRemaining is how many of a deck's cards no share has taken, which is
// what the deck still owes of the day.
func (q *deckShare) countRemaining(out taken) int {
	held := 0
	for _, at := range q.owed {
		if !out.owed[at] {
			held++
		}
	}
	for _, at := range q.fresh {
		if !out.fresh[at] {
			held++
		}
	}
	return held
}

// getTimeSpent is how much of the day a deck has had, in the time it took: what
// an earlier session spent of today, and what the deck's own share has just
// spent.
func (b *budgets) getTimeSpent(q *deckShare, cost review.AnswerCost) time.Duration {
	return b.spentUnder[q.deck].Took +
		time.Duration(q.debt)*cost.Review + time.Duration(q.begun)*cost.New
}

// spend is what each preset's day takes of the debt before it and the material
// it has not begun.
//
// The day is divided over the decks the preset schedules, each deck's share in
// proportion to what it owes. Within a deck the debt and the material not begun
// are offered one against the other in the share the preset names, and a side
// that runs out leaves the rest of the deck's share to the other. An odd card
// goes to the debt, so a share holding one card spend it on what is already
// begun. What no deck could use out of its own share is offered round again, so
// the day spend what it holds.
func (b *budgets) spend(owed, fresh []CardFace) taken {
	out := taken{owed: make([]bool, len(owed)), fresh: make([]bool, len(fresh))}

	at := make(map[string]map[string]*deckShare)
	var order []string
	into := func(path, deck string) *deckShare {
		decks, held := at[path]
		if !held {
			decks = make(map[string]*deckShare)
			at[path] = decks
			order = append(order, path)
		}
		one, held := decks[deck]
		if !held {
			one = &deckShare{deck: deck}
			decks[deck] = one
		}
		return one
	}
	for i, one := range owed {
		q := into(b.under[one.ID], one.Deck)
		q.owed = append(q.owed, i)
	}
	for i, one := range fresh {
		q := into(b.under[one.ID], one.Deck)
		q.fresh = append(q.fresh, i)
	}
	// A deck already sat through today holds a share of the day whether or not
	// it has a card left to give, so what it took stands against its own share.
	for deck, path := range b.decks {
		if _, spent := b.spentUnder[deck]; spent {
			into(path, deck)
		}
	}
	// The presets are asked in one order, so a session asked twice is the same
	// session.
	slices.Sort(order)

	for _, path := range order {
		one, held := b.left[path]
		if !held || one.admits.IsPaused() {
			continue
		}
		// The decks are handed their shares in the order their paths stand, so
		// the division does not turn on the order the vault was walked in.
		decks := make([]*deckShare, 0, len(at[path]))
		for _, q := range at[path] {
			decks = append(decks, q)
		}
		slices.SortFunc(decks, func(a, b *deckShare) int {
			return strings.Compare(a.deck, b.deck)
		})

		shares := b.divide(one, decks)
		for i, q := range decks {
			b.deal(&shares[i], q, owed, fresh, &out)
		}
		// What is left of the day after every deck has had its share goes to
		// the decks that still hold a card, the one holding most first: what a
		// deck has left to answer is what it still owes of the day, and the day
		// is handed out by what is owed. Decks holding the same number take it
		// in the order of the time the day has given them already, least first.
		remaining := make(map[string]int, len(decks))
		for _, q := range decks {
			remaining[q.deck] = q.countRemaining(out)
		}
		slices.SortStableFunc(decks, func(x, y *deckShare) int {
			return cmp.Or(
				cmp.Compare(remaining[y.deck], remaining[x.deck]),
				cmp.Compare(b.getTimeSpent(x, one.cost), b.getTimeSpent(y, one.cost)),
			)
		})
		for _, q := range decks {
			q.seen, q.unseen = 0, 0
			over := *one
			b.deal(&over, q, owed, fresh, &out)
		}
	}
	return out
}

// divide is each deck's share of one preset's day.
//
// The whole day is handed out in proportion to what each deck owes of it, and
// what a deck has already answered today comes off that deck's own share, so a
// deck sat first spends its share and no other deck's.
func (b *budgets) divide(one *allowance, decks []*deckShare) []allowance {
	reviews := make([]int, len(decks))
	begun := make([]int, len(decks))
	minutes := make([]int, len(decks))
	for at, q := range decks {
		spent := b.spentUnder[q.deck]
		reviews[at] = len(q.owed) + spent.Reviews
		begun[at] = len(q.fresh) + spent.New
		minutes[at] = int(time.Duration(len(q.owed))*one.cost.Review +
			time.Duration(len(q.fresh))*one.cost.New + spent.Took)
	}
	keeps := one.admits.Keeps
	reviews = review.GetShares(keeps.Reviews, reviews)
	begun = review.GetShares(keeps.New, begun)
	minutes = review.GetShares(int(keeps.Minutes*float64(time.Minute)), minutes)

	out := make([]allowance, len(decks))
	for at, q := range decks {
		spent := b.spentUnder[q.deck]
		out[at] = *one
		out[at].admits.Reviews = max(0, reviews[at]-spent.Reviews)
		out[at].admits.New = max(0, begun[at]-spent.New)
		out[at].admits.Minutes = max(0, time.Duration(minutes[at])-spent.Took)
	}
	return out
}

// deal is what one deck's share of the day takes of the debt before it and the
// material it has not begun. A card another share has already taken is passed
// over, and the deck picks up where its share left off.
func (b *budgets) deal(share *allowance, q *deckShare, owed, fresh []CardFace, out *taken) {
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
		if share.admits.IsPayingDebt(q.debt, q.begun, debt, begun) {
			if card := owed[q.owed[q.seen]]; b.charge(share, card.ID, false) {
				out.owed[q.owed[q.seen]] = true
				q.debt++
			}
			q.seen++
			continue
		}
		if card := fresh[q.fresh[q.unseen]]; b.charge(share, card.ID, true) {
			out.fresh[q.fresh[q.unseen]] = true
			q.begun++
		}
		q.unseen++
	}
}

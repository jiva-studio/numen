package flashcards

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review/projection"
)

// budgets tracks daily allowances, review card assignments, and deck review progress per preset.
type budgets struct {
	reviewCardPresets map[review.ReviewCardID]string
	allowances        map[string]*allowance
	// decks maps each deck path to its scheduling preset path.
	decks map[string]string
	// cards stores the count of review cards per preset.
	cards map[string]int
	// reviewedCards records review cards reviewed during the current day.
	reviewedCards map[review.ReviewCardID]bool
	// spentByDeck records review statistics per deck for the current day.
	spentByDeck map[string]review.Spent
}

// allowance tracks daily budget limits, review costs, and consumption for a preset.
type allowance struct {
	admits     review.Allowance
	cost       review.AnswerCost
	counts     review.BudgetUnit
	spentStats review.Spent
}

// getBudgets calculates daily budgets and allowances for all active presets.
func getBudgets(
	ctx context.Context, v domain.Vault, reads *PresetReads, day review.Day,
	reviewCards []ReviewCard, schedules map[review.ReviewCardID]review.Schedule,
	log ReviewLog, delta review.Scheduler, pos func(retention float64) review.Scheduler,
	now time.Time,
) (*budgets, error) {
	out := &budgets{
		reviewCardPresets: make(map[review.ReviewCardID]string, len(reviewCards)),
		allowances:        make(map[string]*allowance),
		cards:             make(map[string]int),
	}
	survey, err := readPresetSurvey(ctx, v, reads, reviewCards, schedules, out)
	if err != nil {
		return nil, err
	}

	out.decks = survey.presetByDeck
	deckCounts := make(map[string]review.BudgetUnit, len(survey.presetByDeck))
	for deck, path := range survey.presetByDeck {
		deckCounts[deck] = survey.settings[path].Counts
	}
	named := day.GetName(now)
	out.reviewedCards = review.GetReviewed(day, named, log.Answers)

	out.spentByDeck = review.GetSpentUnder(day, named, log.Answers, survey.in, deckCounts)
	elapsed := sumSpentByPreset(out.spentByDeck, survey.presetByDeck)

	costed := review.GetCostUnder(delta, log.Answers, out.reviewCardPresets)
	for path, p := range survey.settings {
		cost, target := costed[path]
		if !target {
			cost = review.DefaultCost
		}
		// Target date goals scale daily review allowances by learning duration.
		learn := 0
		if p.Goal == review.GoalDate {
			learn = projection.GetDaysToLearn(pos(p.Retention), day, p, now)
		}
		out.allowances[path] = &allowance{
			admits:     p.GetAllowance(day, now, elapsed[path], survey.unstarted[path], learn),
			cost:       cost,
			counts:     p.Counts,
			spentStats: elapsed[path],
		}
	}
	return out, nil
}

// presetSurvey contains preset associations and settings discovered from review cards.
type presetSurvey struct {
	// presetByDeck maps deck paths to their scheduling preset paths.
	presetByDeck map[string]string
	// in maps review card IDs to their deck paths.
	in map[review.ReviewCardID]string
	// settings maps preset paths to their review configurations.
	settings map[string]review.Preset
	// unstarted counts unreviewed review cards per preset.
	unstarted map[string]int
}

// readPresetSurvey collects preset mappings, review card counts, and settings for each deck.
func readPresetSurvey(
	ctx context.Context, v domain.Vault, reads *PresetReads, reviewCards []ReviewCard,
	schedules map[review.ReviewCardID]review.Schedule, out *budgets,
) (presetSurvey, error) {
	sv := presetSurvey{
		presetByDeck: make(map[string]string, len(reviewCards)),
		in:           make(map[review.ReviewCardID]string, len(reviewCards)),
		settings:     make(map[string]review.Preset),
		unstarted:    make(map[string]int),
	}
	for _, one := range reviewCards {
		path, recognized := sv.presetByDeck[one.Deck]
		if !recognized {
			p, err := reads.GetForDeck(ctx, v, one.Deck)
			if err != nil {
				return presetSurvey{}, err
			}
			path = p.Path
			sv.presetByDeck[one.Deck] = path
			if _, target := sv.settings[path]; !target {
				sv.settings[path] = p.Settings
			}
		}
		out.reviewCardPresets[one.ID] = path
		sv.in[one.ID] = one.Deck
		out.cards[path]++
		if _, reply := schedules[one.ID]; !reply {
			sv.unstarted[path]++
		}
	}
	return sv, nil
}

// sumSpentByPreset aggregates review consumption across decks belonging to the same preset.
func sumSpentByPreset(
	spentByDeck map[string]review.Spent, presetByDeck map[string]string,
) map[string]review.Spent {
	elapsed := make(map[string]review.Spent, len(presetByDeck))
	for deck, one := range spentByDeck {
		sum := elapsed[presetByDeck[deck]]
		sum.Answered += one.Answered
		sum.New += one.New
		sum.Reviews += one.Reviews
		sum.Took += one.Took
		elapsed[presetByDeck[deck]] = sum
	}
	return elapsed
}

// charge attempts to deduct budget allowance for reviewing a review card in its deck.
func (b *budgets) charge(share *allowance, reviewCard review.ReviewCardID, isNew bool) bool {
	one, target := b.allowances[b.reviewCardPresets[reviewCard]]
	if !target || one.admits.IsPaused() {
		return false
	}
	counted := one.counts.IsCharged(b.reviewedCards[reviewCard])
	cost := one.cost.Review
	if isNew {
		cost = one.cost.New
	}
	if !one.hasRoom(isNew, counted, cost) || !share.hasRoom(isNew, counted, cost) {
		return false
	}
	one.spend(isNew, counted, cost)
	share.spend(isNew, counted, cost)
	return true
}

// hasRoom checks whether sufficient allowance remains for the review cost.
func (a *allowance) hasRoom(isNew, counted bool, cost time.Duration) bool {
	remaining, closes := a.admits.Reviews, a.admits.Limits.Reviews
	if isNew {
		remaining, closes = a.admits.New, a.admits.Limits.New
	}
	if counted && closes != review.ClosedNothing && remaining <= 0 {
		return false
	}
	return a.admits.Limits.Minutes == review.ClosedNothing || a.admits.Minutes >= cost
}

// spend decrements remaining review and time allowances.
func (a *allowance) spend(isNew, counted bool, cost time.Duration) {
	if counted {
		if isNew {
			a.admits.New--
		} else {
			a.admits.Reviews--
		}
	}
	a.admits.Minutes -= cost
}

// asking holds due and new review cards selected for review.
type asking struct {
	owed     []ReviewCard
	newCards []ReviewCard
}

// isAllowed checks if a review card belongs to the requested review scope.
func (b *budgets) isAllowed(one ReviewCard, scope Scope) bool {
	if scope.IsNamed {
		return b.reviewCardPresets[one.ID] == scope.Preset
	}
	return scope.Deck == "" || one.Deck == scope.Deck
}

// getStopError returns the error explaining why a preset has no cards available for review.
func (b *budgets) getStopError(preset string) error {
	one, scheduling := b.allowances[preset]
	switch {
	case !scheduling:
		return fmt.Errorf("%w: no deck of this vault is scheduled by it", ErrSchedulesNothing)
	case one.admits.IsPaused():
		return fmt.Errorf("%w: it is paused", ErrSchedulesNothing)
	case one.spentStats.Answered > 0:
		return fmt.Errorf("%w: daily budget spent", ErrSchedulesNothing)
	default:
		return fmt.Errorf("%w: no cards due", ErrSchedulesNothing)
	}
}

// getAsking filters and orders due and new cards within budget constraints for the scope.
func (b *budgets) getAsking(
	reviewCards []ReviewCard, schedules map[review.ReviewCardID]review.Schedule,
	day review.Day, now time.Time, scope Scope,
) asking {
	var owed, newCards []ReviewCard
	for _, one := range reviewCards {
		s, reply := schedules[one.ID]
		switch {
		case !reply:
			newCards = append(newCards, one)
		case day.IsOwed(s, now):
			owed = append(owed, one)
		}
	}
	slices.SortStableFunc(owed, func(a, b ReviewCard) int {
		return schedules[a.ID].Due.Compare(schedules[b.ID].Due)
	})

	allocations := b.spend(owed, newCards)
	var out asking
	for item, one := range owed {
		if allocations.owed[item] && b.isAllowed(one, scope) {
			out.owed = append(out.owed, one)
		}
	}
	for entry, one := range newCards {
		if allocations.newCards[entry] && b.isAllowed(one, scope) {
			out.newCards = append(out.newCards, one)
		}
	}
	return out
}

// budgetAllocation indicates which candidate cards were accepted by the budget.
type budgetAllocation struct{ owed, newCards []bool }

// deckShare tracks review card indices and budget consumption for a deck.
type deckShare struct {
	deck           string
	owed, newCards []int
	// owedIndex and newIndex track progress through owed and new card pools.
	owedIndex, newIndex int
	debt, begun         int
}

// countRemaining returns the number of unallocated cards remaining for this deck.
func (q *deckShare) countRemaining(out budgetAllocation) int {
	target := 0
	for _, item := range q.owed {
		if !out.owed[item] {
			target++
		}
	}
	for _, entry := range q.newCards {
		if !out.newCards[entry] {
			target++
		}
	}
	return target
}

// getTimeSpent returns total review duration consumed by the deck today.
func (b *budgets) getTimeSpent(q *deckShare, cost review.AnswerCost) time.Duration {
	return b.spentByDeck[q.deck].Took +
		time.Duration(q.debt)*cost.Review + time.Duration(q.begun)*cost.New
}

// spend allocates daily preset budgets across decks and card types.
func (b *budgets) spend(owed, newCards []ReviewCard) budgetAllocation {
	out := budgetAllocation{owed: make([]bool, len(owed)), newCards: make([]bool, len(newCards))}
	shares := b.collectShares(owed, newCards)
	for _, path := range shares.order {
		one, target := b.allowances[path]
		if !target || one.admits.IsPaused() {
			continue
		}
		b.spendPreset(one, shares.getSorted(path), owed, newCards, &out)
	}
	return out
}

// shareRegistry holds deck shares grouped by preset path.
type shareRegistry struct {
	shares map[string]map[string]*deckShare
	order  []string
}

// getShare returns the deck share for a preset and deck path, allocating it if absent.
func (k *shareRegistry) getShare(path, deck string) *deckShare {
	decks, target := k.shares[path]
	if !target {
		decks = make(map[string]*deckShare)
		k.shares[path] = decks
		k.order = append(k.order, path)
	}
	one, target := decks[deck]
	if !target {
		one = &deckShare{deck: deck}
		decks[deck] = one
	}
	return one
}

// getSorted returns deck shares for a preset sorted deterministically by deck path.
func (k *shareRegistry) getSorted(path string) []*deckShare {
	decks := make([]*deckShare, 0, len(k.shares[path]))
	for _, q := range k.shares[path] {
		decks = append(decks, q)
	}
	slices.SortFunc(decks, func(a, b *deckShare) int {
		return strings.Compare(a.deck, b.deck)
	})
	return decks
}

// collectShares groups due and unreviewed cards into deck shares per preset.
func (b *budgets) collectShares(owed, newCards []ReviewCard) *shareRegistry {
	shares := &shareRegistry{shares: make(map[string]map[string]*deckShare)}
	for i, one := range owed {
		q := shares.getShare(b.reviewCardPresets[one.ID], one.Deck)
		q.owed = append(q.owed, i)
	}
	for i, one := range newCards {
		q := shares.getShare(b.reviewCardPresets[one.ID], one.Deck)
		q.newCards = append(q.newCards, i)
	}
	// Decks with prior activity today retain their share allocation.
	for deck, path := range b.decks {
		if _, elapsed := b.spentByDeck[deck]; elapsed {
			shares.getShare(path, deck)
		}
	}
	slices.Sort(shares.order)
	return shares
}

// spendPreset distributes preset allowance to decks and reallocates unused capacity.
func (b *budgets) spendPreset(
	one *allowance, decks []*deckShare, owed, newCards []ReviewCard, out *budgetAllocation,
) {
	shares := b.divide(one, decks)
	for i, q := range decks {
		b.deal(&shares[i], q, owed, newCards, out)
	}
	remaining := make(map[string]int, len(decks))
	for _, q := range decks {
		remaining[q.deck] = q.countRemaining(*out)
	}
	slices.SortStableFunc(decks, func(x, y *deckShare) int {
		return cmp.Or(
			cmp.Compare(remaining[y.deck], remaining[x.deck]),
			cmp.Compare(b.getTimeSpent(x, one.cost), b.getTimeSpent(y, one.cost)),
		)
	})
	for _, q := range decks {
		q.owedIndex, q.newIndex = 0, 0
		overVal := *one
		b.deal(&overVal, q, owed, newCards, out)
	}
}

// divide proportions daily preset allowances across decks based on demand.
func (b *budgets) divide(one *allowance, decks []*deckShare) []allowance {
	reviews := make([]int, len(decks))
	begun := make([]int, len(decks))
	minutes := make([]int, len(decks))
	for item, q := range decks {
		elapsed := b.spentByDeck[q.deck]
		reviews[item] = len(q.owed) + elapsed.Reviews
		begun[item] = len(q.newCards) + elapsed.New
		minutes[item] = int(time.Duration(len(q.owed))*one.cost.Review +
			time.Duration(len(q.newCards))*one.cost.New + elapsed.Took)
	}
	keeps := one.admits.Keeps
	reviews = review.DivideBudget(keeps.Reviews, reviews)
	begun = review.DivideBudget(keeps.New, begun)
	minutes = review.DivideBudget(int(keeps.Minutes*float64(time.Minute)), minutes)

	out := make([]allowance, len(decks))
	for entry, q := range decks {
		elapsedVal := b.spentByDeck[q.deck]
		out[entry] = *one
		out[entry].admits.Reviews = max(0, reviews[entry]-elapsedVal.Reviews)
		out[entry].admits.New = max(0, begun[entry]-elapsedVal.New)
		out[entry].admits.Minutes = max(0, time.Duration(minutes[entry])-elapsedVal.Took)
	}
	return out
}

// deal allocates cards for a deck from review and new pools within the given allowance.
func (b *budgets) deal(share *allowance, q *deckShare, owed, newCards []ReviewCard, out *budgetAllocation) {
	for {
		q.advancePastOthers(*out)
		debt, begun := q.owedIndex < len(q.owed), q.newIndex < len(q.newCards)
		if !debt && !begun {
			return
		}
		// Prioritize debt or new cards according to preset ratio policy.
		if share.admits.IsNextCardDebt(q.debt, q.begun, debt, begun) {
			if card := owed[q.owed[q.owedIndex]]; b.charge(share, card.ID, false) {
				out.owed[q.owed[q.owedIndex]] = true
				q.debt++
			}
			q.owedIndex++
			continue
		}
		if card := newCards[q.newCards[q.newIndex]]; b.charge(share, card.ID, true) {
			out.newCards[q.newCards[q.newIndex]] = true
			q.begun++
		}
		q.newIndex++
	}
}

// advancePastOthers advances pointers past cards already claimed by other decks.
func (q *deckShare) advancePastOthers(out budgetAllocation) {
	for q.owedIndex < len(q.owed) && out.owed[q.owed[q.owedIndex]] {
		q.owedIndex++
	}
	for q.newIndex < len(q.newCards) && out.newCards[q.newCards[q.newIndex]] {
		q.newIndex++
	}
}

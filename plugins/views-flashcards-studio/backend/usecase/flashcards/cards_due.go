package flashcards

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/presets"
)

// CardsDue contains vault card counts, due cards, and unreviewed card statistics for the current day.
type CardsDue struct {
	ReviewCards int
	Due         int
	New         int
	Decks       []DeckCardsDue
	Presets     []PresetCardsDue
}

// DeckCardsDue tracks card counts and review status for a single deck.
type DeckCardsDue struct {
	Deck        string
	ReviewCards int
	Due         int
	New         int
	Unbegun     int
	Answered    int
	Learned     int
}

// PresetCardsDue contains daily budget and review metrics for a preset.
type PresetCardsDue struct {
	Preset          string
	Decks           int
	Cards           int
	Due             int
	New             int
	Answered        int
	ReviewedNew     int
	ReviewedReviews int
	Took            time.Duration
	Budget          review.Budget
	Limits          review.Limits
	Stops           review.StopReason
}

// CountCardsDue calculates daily card review metrics across decks and presets in a vault.
type CountCardsDue struct {
	reviewCards ListReviewCards
	schedules   Schedules
	presets     presets.Presets
	day         review.Day
	now         ports.Clock
}

func NewCountCardsDue(
	reviewCards ListReviewCards, schedules Schedules, presets presets.Presets,
	day review.Day, now ports.Clock,
) CountCardsDue {
	return CountCardsDue{
		reviewCards: reviewCards, schedules: schedules, presets: presets, day: day, now: now,
	}
}

// Execute calculates daily review counts and preset metrics for the vault.
func (u CountCardsDue) Execute(ctx context.Context, v domain.Vault) (CardsDue, error) {
	in, err := u.readInputs(ctx, v)
	if err != nil {
		return CardsDue{}, err
	}
	holds := in.day.getAskingBudget(in.reviewCards, in.schedules, u.day, in.now, Scope{Deck: "", Preset: "", IsNamed: false})

	out := CardsDue{ReviewCards: len(in.reviewCards), Due: 0, New: 0, Decks: nil, Presets: nil}
	rows := make(deckRows)
	rows.countCards(in)
	for deck, one := range in.day.spentByDeck {
		rows.get(deck).Answered = one.Answered
	}
	due, newCount := rows.countDueAndNew(in.day, holds, &out)
	out.Decks = rows.getSortedRows()
	out.Presets, err = u.getPresets(ctx, v, in.reads, in.day, due, newCount)
	if err != nil {
		return CardsDue{}, err
	}
	return out, nil
}

// dueInputs is what one vault's count is worked out from.
type dueInputs struct {
	reviewCards []ReviewCard
	schedules   map[review.CardID]review.Schedule
	asks        assignment
	reads       *presets.Reads
	day         *budgets
	now         time.Time
}

func (u CountCardsDue) readInputs(ctx context.Context, v domain.Vault) (dueInputs, error) {
	if err := ctx.Err(); err != nil {
		return dueInputs{}, err
	}
	reviewCards, err := u.reviewCards.Execute(ctx, v)
	if err != nil {
		return dueInputs{}, err
	}
	if err := ctx.Err(); err != nil {
		return dueInputs{}, err
	}
	log, err := Log{Stores: u.schedules.logs}.Read(ctx, v)
	if err != nil {
		return dueInputs{}, err
	}
	reads := u.presets.OpenReads()
	asks, err := u.schedules.getAssignmentFrom(ctx, v, reads, reviewCards)
	if err != nil {
		return dueInputs{}, err
	}
	schedules := u.schedules.getSchedulesCached(ctx, v, log, asks)
	if err := ctx.Err(); err != nil {
		return dueInputs{}, err
	}

	now := u.now.Now()
	day, err := getBudgets(ctx, v, budgetInputs{
		reads: reads, day: u.day, reviewCards: reviewCards, schedules: schedules, log: log,
		delta: u.schedules.scheduler, pos: u.schedules.getScheduler, now: now,
	})
	// gomutants:disable-next-line BRANCH_IF reason="PresetReads cache is already populated for all reviewCard decks by getAssignmentFrom."
	if err != nil {
		return dueInputs{}, err
	}
	return dueInputs{
		reviewCards: reviewCards, schedules: schedules, asks: asks,
		reads: reads, day: day, now: now,
	}, nil
}

// deckRows is the count of each deck, by the deck's name.
type deckRows map[string]*DeckCardsDue

func (r deckRows) get(deck string) *DeckCardsDue {
	one, target := r[deck]
	if !target {
		one = &DeckCardsDue{Deck: deck, ReviewCards: 0, Due: 0, New: 0, Unbegun: 0, Answered: 0, Learned: 0}
		r[deck] = one
	}
	return one
}

func (r deckRows) countCards(in dueInputs) {
	for _, one := range in.reviewCards {
		row := r.get(one.Deck)
		row.ReviewCards++
		if in.schedules[one.ID].IsNew() {
			row.Unbegun++
		}
		if in.asks.policy(one.ID).Preset.IsLearned(in.schedules[one.ID], in.now) {
			row.Learned++
		}
	}
}

func (r deckRows) countDueAndNew(
	day *budgets, holds asking, out *CardsDue,
) (due, newCards map[string]int) {
	due, newCards = make(map[string]int), make(map[string]int)
	for _, one := range holds.dueCards {
		out.Due++
		r.get(one.Deck).Due++
		due[day.reviewCardPresets[one.ID]]++
	}
	for _, one := range holds.newCards {
		out.New++
		r.get(one.Deck).New++
		newCards[day.reviewCardPresets[one.ID]]++
	}
	return due, newCards
}

func (r deckRows) getSortedRows() []DeckCardsDue {
	var out []DeckCardsDue
	for _, deck := range r {
		out = append(out, *deck)
	}
	slices.SortFunc(out, func(a, b DeckCardsDue) int {
		return strings.Compare(a.Deck, b.Deck)
	})
	return out
}

func (u CountCardsDue) getPresets(
	ctx context.Context, v domain.Vault, reads *presets.Reads, day *budgets,
	due, newCards map[string]int,
) ([]PresetCardsDue, error) {
	out := day.getCardsDue(due, newCards)
	if u.reviewCards.notes == nil {
		return out, nil
	}

	decks, err := u.reviewCards.notes.GetPathsOfType(ctx, v.ID, format.TypeDeck)
	if err != nil {
		return nil, err
	}
	naming := make(map[string]int, len(decks))
	for _, deck := range decks {
		p, err := reads.GetForDeck(ctx, v, deck)
		if err != nil {
			return nil, err
		}
		naming[p.Path]++
	}

	pointed := make(map[string]bool, len(out))
	for item := range out {
		out[item].Decks = naming[out[item].Preset]
		pointed[out[item].Preset] = true
	}

	paths, err := u.reviewCards.notes.GetPathsOfType(ctx, v.ID, format.TypePreset)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if pointed[path] {
			continue
		}
		// Presets without referencing decks have no daily activity scheduled.
		one, err := reads.GetPreset(ctx, v, path)
		if err != nil {
			return nil, err
		}
		out = append(out, PresetCardsDue{
			Preset: path, Decks: naming[path],
			Cards:           0,
			Due:             0,
			New:             0,
			Answered:        0,
			ReviewedNew:     0,
			ReviewedReviews: 0,
			Took:            time.Duration(0),
			Budget:          review.Budget{New: 0, Reviews: 0, Minutes: 0},
			Limits:          review.Limits{New: "", Reviews: "", Minutes: "", Backlog: ""}, Stops: one.StopsToday,
		})
	}
	slices.SortFunc(out, func(a, b PresetCardsDue) int {
		return strings.Compare(a.Preset, b.Preset)
	})
	return out, nil
}

func (b *budgets) getCardsDue(due, newCards map[string]int) []PresetCardsDue {
	out := make([]PresetCardsDue, 0, len(b.allowances))
	for path, one := range b.allowances {
		out = append(out, PresetCardsDue{
			Preset: path,
			Decks:  0, Cards: b.cards[path],
			Due: due[path], New: newCards[path],
			Answered:        one.spentStats.Answered,
			ReviewedNew:     one.spentStats.New,
			ReviewedReviews: one.spentStats.Reviews,
			Took:            one.spentStats.Took,
			Budget:          one.admits.Keeps, Limits: one.admits.Limits,
			Stops: one.admits.Stops,
		})
	}
	return out
}

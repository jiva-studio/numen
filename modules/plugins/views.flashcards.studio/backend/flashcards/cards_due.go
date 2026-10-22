package flashcards

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// CardsDue contains vault card counts, due cards, and unreviewed card statistics for the current day.
type CardsDue struct {
	// Faces is the total count of card faces in the vault.
	Faces int
	// Due contains the number of review card faces scheduled for today.
	// New contains the number of unreviewed card faces allocated by today's budget.
	Due   int
	New   int
	Decks []DeckCardsDue
	// Presets contains daily review metrics for all presets in the vault.
	Presets []PresetCardsDue
}

// DeckCardsDue tracks card counts and review status for a single deck.
type DeckCardsDue struct {
	Deck  string
	Faces int
	Due   int
	New   int
	// Unbegun is the count of card faces in the deck that have never been reviewed.
	Unbegun int
	// Answered is the count of cards reviewed in the deck during the active review day.
	Answered int
	// Learned is the count of card faces meeting the retention criteria of the deck's preset.
	Learned int
}

// PresetCardsDue contains daily budget and review metrics for a preset.
type PresetCardsDue struct {
	Preset string
	// Decks is the count of decks referencing this preset. Cards is total card faces across those decks.
	Decks int
	Cards int
	// Due and New contain review and unreviewed card counts allowed by the preset budget.
	Due int
	New int
	// Answered is the count of cards reviewed during today's session, with Took recording total review duration.
	Answered        int
	AnsweredNew     int
	AnsweredReviews int
	Took            time.Duration
	// Budget defines the limits configured for this day of the week.
	Budget review.Budget
	// Limits defines the active cutoff limits for the day.
	Limits review.Limits
	// Stops indicates why the preset schedules no reviews today, or empty if active.
	Stops review.StopReason
}

// CountCardsDue calculates daily card review metrics across decks and presets in a vault.
type CountCardsDue struct {
	cardFaces ListCardFaces
	schedules Schedules
	// presets resolves presets assigned to each deck.
	presets Presets
	day     review.Day
	now     ports.Clock
}

// NewCountCardsDue constructs a CountCardsDue calculator.
func NewCountCardsDue(
	faces ListCardFaces, schedules Schedules, presets Presets,
	day review.Day, now ports.Clock,
) CountCardsDue {
	return CountCardsDue{
		cardFaces: faces, schedules: schedules, presets: presets, day: day, now: now,
	}
}

// WithCardFaces returns a copy of CountCardsDue with the given ListCardFaces.
func (u CountCardsDue) WithCardFaces(faces ListCardFaces) CountCardsDue {
	u.cardFaces = faces
	return u
}

// CardFaces returns the ListCardFaces CountCardsDue counts through.
func (u CountCardsDue) CardFaces() ListCardFaces {
	return u.cardFaces
}

// WithDay returns a copy of CountCardsDue with the given Day.
func (u CountCardsDue) WithDay(day review.Day) CountCardsDue {
	u.day = day
	return u
}

// WithNow returns a copy of CountCardsDue with the given Clock.
func (u CountCardsDue) WithNow(now ports.Clock) CountCardsDue {
	u.now = now
	return u
}

// Execute calculates daily review counts and preset metrics for the vault.
func (u CountCardsDue) Execute(ctx context.Context, v domain.Vault) (CardsDue, error) {
	in, err := u.readInputs(ctx, v)
	if err != nil {
		return CardsDue{}, err
	}
	holds := in.day.getAsking(in.faces, in.schedules, u.day, in.now, Scope{})

	out := CardsDue{Faces: len(in.faces)}
	rows := make(deckRows)
	rows.countFaces(in)
	for deck, one := range in.day.spentByDeck {
		rows.get(deck).Answered = one.Answered
	}
	due, newCount := rows.countDueAndNew(in.day, holds, &out)
	out.Decks = rows.getSorted()
	out.Presets, err = u.getPresets(ctx, v, in.reads, in.day, due, newCount)
	if err != nil {
		return CardsDue{}, err
	}
	return out, nil
}

// dueInputs is what one vault's count is worked out from.
type dueInputs struct {
	faces     []CardFace
	schedules map[review.CardFaceID]review.Schedule
	asks      assignment
	reads     *PresetReads
	day       *budgets
	now       time.Time
}

// readInputs loads card faces, review logs, and presets to compute schedules and budgets.
func (u CountCardsDue) readInputs(ctx context.Context, v domain.Vault) (dueInputs, error) {
	if err := ctx.Err(); err != nil {
		return dueInputs{}, err
	}
	faces, err := u.cardFaces.Execute(ctx, v)
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
	asks, err := u.schedules.getAssignmentFrom(ctx, v, reads, faces)
	if err != nil {
		return dueInputs{}, err
	}
	schedules := u.schedules.getSchedulesCached(ctx, v, log, asks)
	if err := ctx.Err(); err != nil {
		return dueInputs{}, err
	}

	now := u.now()
	day, err := getBudgets(
		ctx, v, reads, u.day, faces, schedules, log,
		u.schedules.by, u.schedules.getScheduler, now,
	)
	if err != nil {
		return dueInputs{}, err
	}
	return dueInputs{
		faces: faces, schedules: schedules, asks: asks,
		reads: reads, day: day, now: now,
	}, nil
}

// deckRows is the count of each deck, by the deck's name.
type deckRows map[string]*DeckCardsDue

// get is the row of a deck, created when it is new.
func (r deckRows) get(deck string) *DeckCardsDue {
	one, target := r[deck]
	if !target {
		one = &DeckCardsDue{Deck: deck}
		r[deck] = one
	}
	return one
}

// countFaces aggregates card counts, unbegun cards, and learned cards per deck.
func (r deckRows) countFaces(in dueInputs) {
	for _, one := range in.faces {
		row := r.get(one.Deck)
		row.Faces++
		if in.schedules[one.ID].IsNew() {
			row.Unbegun++
		}
		if in.asks.under(one.ID).Preset.IsLearned(in.schedules[one.ID], in.now) {
			row.Learned++
		}
	}
}

// countDueAndNew aggregates due and new card counts per deck and per preset.
func (r deckRows) countDueAndNew(
	day *budgets, holds asking, out *CardsDue,
) (due, newCards map[string]int) {
	due, newCards = make(map[string]int), make(map[string]int)
	for _, one := range holds.owed {
		out.Due++
		r.get(one.Deck).Due++
		due[day.cardFacePresets[one.ID]]++
	}
	for _, one := range holds.newCards {
		out.New++
		r.get(one.Deck).New++
		newCards[day.cardFacePresets[one.ID]]++
	}
	return due, newCards
}

// getSorted is the rows ordered by the name of the deck.
func (r deckRows) getSorted() []DeckCardsDue {
	var out []DeckCardsDue
	for _, deck := range r {
		out = append(out, *deck)
	}
	slices.SortFunc(out, func(a, b DeckCardsDue) int {
		return strings.Compare(a.Deck, b.Deck)
	})
	return out
}

// getPresets returns daily metrics for all presets in the vault.
func (u CountCardsDue) getPresets(
	ctx context.Context, v domain.Vault, reads *PresetReads, day *budgets,
	due, newCards map[string]int,
) ([]PresetCardsDue, error) {
	out := day.getCardsDue(due, newCards)
	if u.cardFaces.notes == nil {
		return out, nil
	}

	decks, err := u.cardFaces.notes.GetPathsOfType(ctx, v.ID, format.TypeDeck)
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
	for at := range out {
		out[at].Decks = naming[out[at].Preset]
		pointed[out[at].Preset] = true
	}

	paths, err := u.cardFaces.notes.GetPathsOfType(ctx, v.ID, format.TypePreset)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if pointed[path] {
			continue
		}
		// Presets without referencing decks have no daily activity scheduled.
		one, err := reads.read(ctx, v, path)
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

// getCardsDue calculates preset metrics from daily allowances and card counts.
func (b *budgets) getCardsDue(due, newCards map[string]int) []PresetCardsDue {
	out := make([]PresetCardsDue, 0, len(b.allowances))
	for path, one := range b.allowances {
		out = append(out, PresetCardsDue{
			Preset: path, Cards: b.cards[path],
			Due: due[path], New: newCards[path],
			Answered:        one.spentStats.Answered,
			AnsweredNew:     one.spentStats.New,
			AnsweredReviews: one.spentStats.Reviews,
			Took:            one.spentStats.Took,
			Budget:          one.admits.Keeps, Limits: one.admits.Limits,
			Stops:           one.admits.Stops,
		})
	}
	return out
}

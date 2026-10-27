package flashcards

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// ErrBothNamed is returned when a session specifies both a deck and a preset.
var ErrBothNamed = errors.New("a session is opened over one deck or over one preset")

// ErrSchedulesNothing indicates the preset has no cards to review today.
var ErrSchedulesNothing = errors.New("this preset schedules nothing today")

// Scope defines the boundaries of a review session.
type Scope struct {
	// Deck is the path of one deck.
	Deck string
	// Preset is the note path of the preset.
	Preset  string
	IsNamed bool
}

// QueuedReviewCard is a review card queued for review with its schedule and intervals.
type QueuedReviewCard struct {
	ReviewCard
	Schedule review.Schedule
	// Ahead maps each rating to its resulting interval from now.
	Ahead map[review.Rating]time.Duration
}

// SessionResult contains the queued cards and any unwritten decks or skipped log entries.
type SessionResult struct {
	// Queue is the list of cards to review in order.
	Queue []QueuedReviewCard
	// Unwritten lists deck paths containing cards without an id that could not be saved.
	Unwritten []string
	// Skipped is the count of unparseable review log lines.
	Skipped int
}

// Session builds and orders the cards for a review session.
type Session struct {
	cardIDs     AssignCardIDs
	reviewCards ListReviewCards
	schedules   Schedules
	presets     Presets
	day         review.Day
	now         ports.Clock
}

// NewSession constructs a Session from its dependencies.
func NewSession(
	cardIDs AssignCardIDs, reviewCards ListReviewCards, schedules Schedules, presets Presets,
	day review.Day, now ports.Clock,
) Session {
	return Session{
		cardIDs: cardIDs, reviewCards: reviewCards, schedules: schedules, presets: presets,
		day: day, now: now,
	}
}

// WithDay returns a copy of Session with the given Day.
func (u Session) WithDay(day review.Day) Session {
	u.day = day
	return u
}

// WithNow returns a copy of Session with the given Clock.
func (u Session) WithNow(now ports.Clock) Session {
	u.now = now
	return u
}

// Execute returns the ordered review queue for the given scope.
func (u Session) Execute(
	ctx context.Context, v domain.Vault, scope Scope,
) (SessionResult, error) {
	if scope.IsNamed && scope.Deck != "" {
		return SessionResult{}, ErrBothNamed
	}
	assigned, err := u.cardIDs.Execute(ctx, v)
	if err != nil {
		return SessionResult{}, err
	}
	reviewCards, err := u.reviewCards.Execute(ctx, v)
	if err != nil {
		return SessionResult{}, err
	}

	target, err := Log{Stores: u.schedules.logs}.Read(ctx, v)
	if err != nil {
		return SessionResult{}, err
	}
	reads := u.presets.OpenReads()
	asks, err := u.schedules.getAssignmentFrom(ctx, v, reads, reviewCards)
	if err != nil {
		return SessionResult{}, err
	}
	schedules := u.schedules.getSchedulesCached(ctx, v, target, asks)

	now := u.now()
	day, err := getBudgets(
		ctx, v, reads, u.day, reviewCards, schedules, target,
		u.schedules.scheduler, u.schedules.getScheduler, now,
	)
	if err != nil {
		return SessionResult{}, err
	}
	holds := day.getAsking(reviewCards, schedules, u.day, now, scope)
	if scope.IsNamed && len(holds.owed)+len(holds.newCards) == 0 {
		return SessionResult{}, day.getStopError(scope.Preset)
	}

	dueTable := review.NewDueByDay(u.day)
	for _, s := range schedules {
		dueTable.Add(s.Due)
	}

	out := SessionResult{Unwritten: assigned.Unwritten, Skipped: target.Skipped}
	out.Queue = make([]QueuedReviewCard, 0, len(holds.owed)+len(holds.newCards))
	for _, one := range holds.owed {
		s := schedules[one.ID]
		out.Queue = append(out.Queue, QueuedReviewCard{
			ReviewCard: one, Schedule: s, Ahead: getIntervals(asks.policy, dueTable, one.ID, s, now),
		})
	}
	for _, one := range holds.newCards {
		out.Queue = append(out.Queue, QueuedReviewCard{
			ReviewCard: one,
			Ahead:      getIntervals(asks.policy, dueTable, one.ID, review.Schedule{}, now),
		})
	}
	return out, nil
}

// getIntervals returns the duration until the next review for each rating.
func getIntervals(
	assign review.Assignment, dueTable *review.DueByDay, reviewCard review.ReviewCardID,
	s review.Schedule, now time.Time,
) map[review.Rating]time.Duration {
	if assign == nil {
		return nil
	}
	one := assign(reviewCard)
	if one.By == nil {
		return nil
	}
	out := make(map[review.Rating]time.Duration, 4)
	for _, r := range []review.Rating{review.Again, review.Hard, review.Good, review.Easy} {
		due := one.By.Next(s, now, r).Due
		out[r] = one.Preset.GetDueDay(dueTable, now, due).Sub(now)
	}
	return out
}

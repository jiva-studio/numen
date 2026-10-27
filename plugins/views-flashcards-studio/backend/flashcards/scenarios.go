package flashcards

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Day is where one day of review gives way to the next.
type Day = review.Day

// owes, what to ask next, and what an answer is written to.
type Scenarios struct {
	ReviewCards   ListReviewCards
	AssignCardIDs AssignCardIDs
	Schedules     Schedules
	CardsDue      CountCardsDue
	Session       Session
	Log           Log
	// Record writes down an answer in a run, and a taking back of one.
	Record Record
	// Counted is how much of a vault was answered on each day it was reviewed.
	Counted CountReviews
	// GetReviewDays is those days inside a stretch of them.
	GetReviewDays GetReviewDays
	// Presets is which preset each deck is scheduled by, and how one is read,
	// written and made.
	Presets Presets
	// Curves is what the one control of a preset evaluates to over the whole range
	// of its goal.
	Curves ProjectCurve
	// Day is where one day of review gives way to the next.
	Day review.Day
}

// Config is what the scenarios that run a vault's cards are built against.
type Config struct {
	Readers   ports.VaultReaders
	Writers   ports.VaultWriters
	Notes     ports.NoteQueries
	Links     ports.LinkQueries
	Problems  ports.ProblemQueries
	Index     func(ctx context.Context, v domain.Vault, paths []string) error
	Logs      ports.LogStores
	Schedules ports.ScheduleStore
	DayCounts ports.ScheduleStore
	DayStart  func() time.Duration
	Now       ports.Clock
	Cores     int
	Scheduler ports.ScheduleAlgorithm
}

// New builds the scenarios that run a vault's cards against this installation.
func New(cfg Config) Scenarios { return NewScenarios(cfg) }

// NewScenarios builds the scenarios that run a vault's cards against this installation.
func NewScenarios(cfg Config) Scenarios {
	reviewCards := NewListReviewCards(cfg.Readers, cfg.Notes, cfg.Links)
	reviewCards.Cores = cfg.Cores
	// A deck or card model this window writes is forgotten before the index is
	// brought level with it.
	leveled := func(ctx context.Context, v domain.Vault, paths []string) error {
		reviewCards.Invalidate(v.ID, paths...)
		if cfg.Index == nil {
			return nil
		}
		return cfg.Index(ctx, v, paths)
	}
	assigning := NewAssignCardIDs(
		cfg.Readers, cfg.Writers, cfg.Notes, cfg.Links, leveled, cfg.Now,
	)
	var starts time.Duration
	if cfg.DayStart != nil {
		starts = cfg.DayStart()
	}
	day := review.Day{Starts: starts}

	presets := NewPresets(
		cfg.Readers, cfg.Writers, cfg.Links, cfg.Notes, leveled, day, cfg.Now,
	)
	// A link the index does not carry is accounted for in what parsing turned
	// up.
	presets.Problems = cfg.Problems

	// Each card is worked out at the share of the cards its own preset asks
	// for, which is what says which preset a review card is placed under.
	var defaultScheduler review.Scheduler = review.NewFSRS()
	var createScheduler func(retention float64) review.Scheduler
	if cfg.Scheduler != nil {
		defaultScheduler = cfg.Scheduler.CreateScheduler(review.Defaults().Retention)
		createScheduler = cfg.Scheduler.CreateScheduler
	}
	schedules := NewSchedules(cfg.Logs, defaultScheduler, day, reviewCards, presets)
	schedules.Cache = cfg.Schedules
	if createScheduler != nil {
		schedules.At = createScheduler
	}

	counted := NewCountReviews(cfg.Logs, schedules, day, cfg.Now)
	counted.Cache = cfg.DayCounts

	curves := NewProjectCurve(reviewCards, schedules, presets, day, cfg.Now)
	curves.Cores = cfg.Cores
	if createScheduler != nil {
		curves.By = createScheduler
	}

	return Scenarios{
		ReviewCards:   reviewCards,
		AssignCardIDs: assigning,
		Schedules:     schedules,
		CardsDue:      NewCountCardsDue(reviewCards, schedules, presets, day, cfg.Now),
		Session:       NewSession(assigning, reviewCards, schedules, presets, day, cfg.Now),
		Log:           Log{Stores: cfg.Logs},
		Record:        NewRecord(cfg.Now),
		Counted:       counted,
		GetReviewDays: NewGetReviewDays(counted),
		Presets:       presets,
		Curves:        curves,
		Day:           day,
	}
}

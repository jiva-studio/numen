package flashcards

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presets"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Day is where one day of review gives way to the next.
type Day = review.Day

// Scenarios is the set of use cases the plugin exposes: what is due, what to ask
// next, and where an answer is written.
type Scenarios struct {
	ReviewCards   ListReviewCards
	AssignCardIDs AssignCardIDs
	Schedules     Schedules
	CardsDue      CountCardsDue
	Session       Session
	Log           Log
	Record        Record
	Counted       CountReviews
	GetReviewDays GetReviewDays
	Presets       presets.Presets
	Curves        ProjectCurve
	Day           review.Day
}

// Notes is what the scenarios built from a Config ask the index of a vault's notes.
type Notes interface {
	ports.IndexQueries
	ports.TypePathQueries
	ports.NoteRefQueries
	ports.NameQueries
}

// Config is what the scenarios that run a vault's cards are built against.
type Config struct {
	Readers   ports.VaultReaders
	Writers   ports.VaultWriters
	Notes     Notes
	Links     ports.LinkQueries
	Problems  ports.ProblemQueries
	Index     func(ctx context.Context, v domain.Vault, paths []string) error
	Logs      ports.LogStores
	Schedules ports.ScheduleStore
	DayCounts ports.ScheduleStore
	Report    func(what string, err error)
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
	// updated for it.
	updateIndex := func(ctx context.Context, v domain.Vault, paths []string) error {
		reviewCards.Invalidate(v.ID, paths...)
		if cfg.Index == nil {
			return nil
		}
		return cfg.Index(ctx, v, paths)
	}
	assigning := NewAssignCardIDs(
		cfg.Readers, cfg.Writers, cfg.Notes, cfg.Links, updateIndex, cfg.Now,
	)
	var starts time.Duration
	if cfg.DayStart != nil {
		starts = cfg.DayStart()
	}
	day := review.Day{Starts: starts, In: nil}

	deckPresets := presets.New(
		cfg.Readers, cfg.Writers, cfg.Links, cfg.Notes, updateIndex, cfg.Now,
	).WithDay(day)
	// A link the index does not carry is accounted for in what parsing turned
	// up.
	deckPresets.Problems = cfg.Problems

	// Each card is scheduled at the retention target of its own preset.
	var defaultScheduler review.Scheduler = review.NewFSRS()
	var createScheduler func(retention float64) review.Scheduler
	if cfg.Scheduler != nil {
		defaultScheduler = cfg.Scheduler.CreateScheduler(review.Defaults().Retention)
		createScheduler = cfg.Scheduler.CreateScheduler
	}
	schedules := NewSchedules(cfg.Logs, defaultScheduler, day, reviewCards, deckPresets)
	schedules.Cache = cfg.Schedules
	schedules.Report = cfg.Report
	if createScheduler != nil {
		schedules.At = createScheduler
	}

	counted := NewCountReviews(cfg.Logs, schedules, day, cfg.Now)
	counted.Cache = cfg.DayCounts
	counted.Report = cfg.Report

	curves := NewProjectCurve(reviewCards, schedules, deckPresets, day, cfg.Now)
	curves.Cores = cfg.Cores
	if createScheduler != nil {
		curves.By = createScheduler
	}

	return Scenarios{
		ReviewCards:   reviewCards,
		AssignCardIDs: assigning,
		Schedules:     schedules,
		CardsDue:      NewCountCardsDue(reviewCards, schedules, deckPresets, day, cfg.Now),
		Session:       NewSession(assigning, reviewCards, schedules, deckPresets, day, cfg.Now),
		Log:           Log{Stores: cfg.Logs},
		Record:        NewRecord(cfg.Now),
		Counted:       counted,
		GetReviewDays: NewGetReviewDays(counted),
		Presets:       deckPresets,
		Curves:        curves,
		Day:           day,
	}
}

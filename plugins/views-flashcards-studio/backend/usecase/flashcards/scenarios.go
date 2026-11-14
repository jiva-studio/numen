package flashcards

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// ReviewScenarios is what a session of review does: lists the cards, gives the ones that carry no
// card id one, orders the cards to ask, writes an answer and opens the run that keeps the answers.
type ReviewScenarios struct {
	ReviewCards   ListReviewCards
	AssignCardIDs AssignCardIDs
	Session       Session
	Record        Record
	Log           Log
}

// ScheduleScenarios is what decides when a card is asked: where each card stands, the preset of
// each deck and the curve of load a preset projects.
type ScheduleScenarios[P DeckPresets] struct {
	Schedules Schedules
	Presets   P
	Curves    ProjectCurve
}

// CountScenarios is what counts a vault: the cards due now and the reviews answered on each day.
type CountScenarios struct {
	CardsDue      CountCardsDue
	Counted       CountReviews
	GetReviewDays GetReviewDays
}

// IndexUpdate brings the named notes of a vault up to date in the index.
type IndexUpdate = edit.IndexUpdater

// DeckPresets is the presets each deck of a vault is scheduled by.
type DeckPresets interface {
	// OpenReads opens a run of reads sharing the preset notes they open.
	OpenReads() ports.PresetReads
	// HasLinks reports whether the links of a deck can be resolved. Without them every deck is
	// scheduled by the defaults.
	HasLinks() bool
	// GetBacklinks returns the links that point at the note at path.
	GetBacklinks(ctx context.Context, vault domain.VaultID, path string) ([]domain.ResolvedLink, error)
}

// Notes is what the scenarios built from a Config ask the index of a vault's notes.
type Notes interface {
	ports.IndexQueries
	ports.TypePathQueries
}

// Config is what the scenarios that run a vault's cards are built against. Day is where one day of
// review gives way to the next. Cards reads the decks and the card models, and NewDeckWriter makes
// the writer of a deck over the index update the scenarios pass it. Entropy fills the identifiers
// of the runs and the answers, and Cores is how many workers parse decks and project curves.
type Config struct {
	Readers       ports.VaultReaders
	Notes         Notes
	Cards         ports.CardNoteReader
	NewDeckWriter func(index IndexUpdate) ports.DeckWriter
	Index         IndexUpdate
	Logs          ports.LogStores
	Schedules     ports.ScheduleStore
	DayCounts     ports.ScheduleStore
	Report        func(what string, err error)
	Day           review.Day
	Now           ports.Clock
	Entropy       ports.Entropy
	Cores         int
	Scheduler     ports.ScheduleAlgorithm
}

// NewScenarios builds the scenarios that run a vault's cards against this installation, with the
// presets newPresets makes for the day of review. They share one list of review cards, so a deck,
// a card model or a preset the scenarios write is forgotten before the index is updated for it.
func NewScenarios[P DeckPresets](
	cfg Config, newPresets func(index IndexUpdate, day review.Day) P,
) (ReviewScenarios, ScheduleScenarios[P], CountScenarios) {
	reviewCards := NewListReviewCards(cfg.Readers, cfg.Notes, cfg.Cards)
	reviewCards.Cores = cfg.Cores
	updateIndex := func(ctx context.Context, v domain.Vault, paths []string) error {
		reviewCards.Invalidate(v.ID, paths...)
		if cfg.Index == nil {
			return nil
		}
		return cfg.Index(ctx, v, paths)
	}
	assigning := NewAssignCardIDs(cfg.Notes, cfg.Cards, cfg.NewDeckWriter(updateIndex))
	deckPresets := newPresets(updateIndex, cfg.Day)

	// Each card is scheduled at the retention target of its own preset.
	var defaultScheduler review.Scheduler = review.NewFSRS()
	var createScheduler func(retention float64) review.Scheduler
	if cfg.Scheduler != nil {
		defaultScheduler = cfg.Scheduler.CreateScheduler(review.Defaults().Retention)
		createScheduler = cfg.Scheduler.CreateScheduler
	}
	schedules := NewSchedules(cfg.Logs, defaultScheduler, cfg.Day, reviewCards, deckPresets)
	schedules.Cache = cfg.Schedules
	schedules.Report = cfg.Report
	if createScheduler != nil {
		schedules.At = createScheduler
	}

	counted := NewCountReviews(cfg.Logs, schedules, cfg.Day, cfg.Now)
	counted.Cache = cfg.DayCounts
	counted.Report = cfg.Report

	curves := NewProjectCurve(reviewCards, schedules, deckPresets, cfg.Day, cfg.Now)
	curves.Cores = cfg.Cores
	if createScheduler != nil {
		curves.By = createScheduler
	}

	return ReviewScenarios{
			ReviewCards:   reviewCards,
			AssignCardIDs: assigning,
			Session:       NewSession(assigning, reviewCards, schedules, deckPresets, cfg.Day, cfg.Now),
			Record:        NewRecord(cfg.Now, cfg.Entropy),
			Log:           Log{Stores: cfg.Logs, Entropy: cfg.Entropy},
		},
		ScheduleScenarios[P]{Schedules: schedules, Presets: deckPresets, Curves: curves},
		CountScenarios{
			CardsDue:      NewCountCardsDue(reviewCards, schedules, deckPresets, cfg.Day, cfg.Now),
			Counted:       counted,
			GetReviewDays: NewGetReviewDays(counted),
		}
}

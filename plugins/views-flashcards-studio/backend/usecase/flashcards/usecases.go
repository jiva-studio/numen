package flashcards

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// ReviewUseCases is what a session of review does: lists the cards, gives the ones that carry no
// card id one, orders the cards to ask, writes an answer and opens the run that keeps the answers.
type ReviewUseCases struct {
	ReviewCards   ListReviewCards
	AssignCardIDs AssignCardIDs
	Session       Session
	Record        Record
	Log           Log
}

// ScheduleUseCases is what decides when a card is asked: where each card stands, the runs of
// preset reads the schedules share, the presets of the vault and the curve of load a preset projects.
type ScheduleUseCases[P any] struct {
	Schedules       Schedules
	OpenPresetReads OpenPresetReads
	Presets         P
	Curves          ProjectCurve
}

// CountUseCases is what counts a vault: the cards due now and the reviews answered on each day.
type CountUseCases struct {
	CardsDue      CountCardsDue
	Counted       CountReviews
	GetReviewDays GetReviewDays
}

// IndexUpdate brings the named notes of a vault up to date in the index.
type IndexUpdate = edit.IndexUpdater

// OpenPresetReads opens runs of reads over the presets each deck of a vault is scheduled by.
type OpenPresetReads interface {
	Execute() ports.PresetReads
}

// NewPresets makes the presets for a day of review over the index update the use cases pass it:
// the use cases on the presets and the opener of the runs of preset reads the schedules share.
type NewPresets[P any] func(index IndexUpdate, day review.Day) (P, OpenPresetReads)

// Notes is what the use cases built from a Config ask the index of a vault's notes.
type Notes interface {
	ports.IndexQueries
	ports.TypePathQueries
}

// Config is what the use cases that run a vault's cards are built against. Day is where one day of
// review gives way to the next. Cards reads the decks and the card models, and NewDeckWriter makes
// the writer of a deck over the index update the use cases pass it. Entropy fills the identifiers
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

// NewUseCases builds the use cases that run a vault's cards against this installation, with the
// presets newPresets makes for the day of review. They share one list of review cards, so a deck,
// a card model or a preset the use cases write is forgotten before the index is updated for it.
func NewUseCases[P any](cfg Config, newPresets NewPresets[P]) (ReviewUseCases, ScheduleUseCases[P], CountUseCases) {
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
	presets, deckPresets := newPresets(updateIndex, cfg.Day)

	// Each card is scheduled at the retention target of its own preset.
	var defaultScheduler review.Scheduler = review.NewFSRS()
	var createScheduler func(retention float64) review.Scheduler
	if cfg.Scheduler != nil {
		defaultScheduler = cfg.Scheduler.CreateScheduler(review.NewDefaultPreset().Retention)
		createScheduler = cfg.Scheduler.CreateScheduler
	}
	schedules := NewSchedules(cfg.Logs, defaultScheduler, cfg.Day, reviewCards, deckPresets)
	schedules.Cache = cfg.Schedules
	schedules.Report = cfg.Report
	if createScheduler != nil {
		schedules.GetScheduler = createScheduler
	}

	counted := NewCountReviews(cfg.Logs, schedules, cfg.Day, cfg.Now)
	counted.Cache = cfg.DayCounts
	counted.Report = cfg.Report

	curves := NewProjectCurve(reviewCards, schedules, deckPresets, cfg.Day, cfg.Now)
	curves.Cores = cfg.Cores
	if createScheduler != nil {
		curves.GetScheduler = createScheduler
	}

	return ReviewUseCases{
			ReviewCards:   reviewCards,
			AssignCardIDs: assigning,
			Session:       NewSession(assigning, reviewCards, schedules, deckPresets, cfg.Day, cfg.Now),
			Record:        NewRecord(cfg.Now, cfg.Entropy),
			Log:           Log{Stores: cfg.Logs, Entropy: cfg.Entropy},
		},
		ScheduleUseCases[P]{Schedules: schedules, OpenPresetReads: deckPresets, Presets: presets, Curves: curves},
		CountUseCases{
			CardsDue:      NewCountCardsDue(reviewCards, schedules, deckPresets, cfg.Day, cfg.Now),
			Counted:       counted,
			GetReviewDays: NewGetReviewDays(counted),
		}
}

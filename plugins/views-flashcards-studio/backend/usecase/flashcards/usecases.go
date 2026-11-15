package flashcards

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/presets"
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
// preset reads the schedules share, the preset of each deck and the curve of load a preset projects.
type ScheduleUseCases struct {
	Schedules       Schedules
	OpenPresetReads presets.OpenReads
	Presets         PresetUseCases
	Curves          ProjectCurve
}

// PresetUseCases is what the presets of a vault do: read one preset, find the preset of a deck,
// list them, put a deck on one and save the settings of one.
type PresetUseCases struct {
	Read       presets.Read
	GetForDeck presets.GetForDeck
	List       presets.List
	Point      presets.Point
	Save       presets.Save
}

// CountUseCases is what counts a vault: the cards due now and the reviews answered on each day.
type CountUseCases struct {
	CardsDue      CountCardsDue
	Counted       CountReviews
	GetReviewDays GetReviewDays
}

// Notes is what the scenarios built from a Config ask the index of a vault's notes.
type Notes interface {
	ports.IndexQueries
	ports.TypePathQueries
	ports.NoteRefQueries
	ports.NameQueries
}

// Config is what the scenarios that run a vault's cards are built against. Day is where one day of
// review gives way to the next.
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
	Day       review.Day
	Now       ports.Clock
	Cores     int
	Scheduler ports.ScheduleAlgorithm
}

// NewUseCases builds the use cases that run a vault's cards against this installation. They share
// one list of review cards, so a deck or card model the review writes is forgotten before the
// index is updated for it.
func NewUseCases(cfg Config) (ReviewUseCases, ScheduleUseCases, CountUseCases) {
	reviewCards := NewListReviewCards(cfg.Readers, cfg.Notes, cfg.Links)
	reviewCards.Cores = cfg.Cores
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

	presetsConfig := presets.Config{
		Readers: cfg.Readers, Writers: cfg.Writers, Links: cfg.Links, Notes: cfg.Notes,
		Problems: cfg.Problems, Index: updateIndex, Now: cfg.Now, Day: cfg.Day,
	}
	deckPresets := presets.NewOpenReads(presetsConfig)

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

	return ReviewUseCases{
			ReviewCards:   reviewCards,
			AssignCardIDs: assigning,
			Session:       NewSession(assigning, reviewCards, schedules, deckPresets, cfg.Day, cfg.Now),
			Record:        NewRecord(cfg.Now),
			Log:           Log{Stores: cfg.Logs},
		},
		ScheduleUseCases{
			Schedules: schedules, OpenPresetReads: deckPresets, Curves: curves,
			Presets: PresetUseCases{
				Read:       presets.NewRead(presetsConfig),
				GetForDeck: presets.NewGetForDeck(presetsConfig),
				List:       presets.NewList(presetsConfig),
				Point:      presets.NewPoint(presetsConfig),
				Save:       presets.NewSave(presetsConfig),
			},
		},
		CountUseCases{
			CardsDue:      NewCountCardsDue(reviewCards, schedules, deckPresets, cfg.Day, cfg.Now),
			Counted:       counted,
			GetReviewDays: NewGetReviewDays(counted),
		}
}

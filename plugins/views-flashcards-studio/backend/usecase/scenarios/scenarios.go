// Package scenarios builds everything the plugin does to the cards of a vault, from the ports the
// host answers and the settings in force.
package scenarios

import (
	"context"
	"runtime"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards/deckedit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/neighborhood"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

// The names the schedules and the days counted are kept under in the cache the host holds.
const (
	schedulesCache = "schedules"
	daysCache      = "days"
)

// Inputs is what the scenarios are built against. Index brings the notes a window wrote up to date
// in the index, which is the host's.
type Inputs struct {
	Readers    ports.VaultReaders
	Writers    ports.VaultWriters
	Notes      flashcards.Notes
	Types      ports.NoteTypeQueries
	CardModels cards.CardModelQueries
	Links      ports.LinkQueries
	Problems   ports.ProblemQueries
	Logs       ports.LogStores
	Schedules  func(name string) ports.ScheduleStore
	Report     func(what string, err error)
	Index      func(ctx context.Context, v domain.Vault, paths []string) error
	DayStart   func() time.Duration
	Now        ports.Clock
	Cores      int
	Scheduler  ports.ScheduleAlgorithm
}

// Scenarios is everything the plugin does to the cards of a vault.
type Scenarios struct {
	Cards        cards.Scenarios
	DeckEdit     deckedit.Scenarios
	Flash        flashcards.Scenarios
	Day          review.Day
	Neighborhood neighborhood.Show
}

// New builds the scenarios. A count of cores that is not positive is the cores the process may use.
func New(in Inputs) *Scenarios {
	cores := in.Cores
	if cores <= 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Any non-positive argument queries GOMAXPROCS without changing it."
		cores = runtime.GOMAXPROCS(0)
	}
	read := cards.NewRead(in.Readers, in.Links)
	write := cards.NewWrite(in.Readers, in.Writers, in.Links, in.Index, in.Now)
	set := cards.Scenarios{
		Read:        read,
		List:        cards.NewList(in.Readers, in.CardModels),
		Write:       write,
		Create:      cards.NewCreate(in.Writers, in.Index, in.Now),
		RenameField: cards.NewRenameField(in.Readers, in.Writers, in.Notes, in.Links, in.Index, in.Now),
		BoundedRenameField: cards.NewRenameField(
			in.Readers, in.Writers, in.Notes, in.Links, in.Index, in.Now).WithMaxDecks(cards.MaxDecksPerRename),
	}
	running := flashcards.NewScenarios(flashcards.Config{
		Readers:   in.Readers,
		Writers:   in.Writers,
		Notes:     in.Notes,
		Links:     in.Links,
		Problems:  in.Problems,
		Index:     in.Index,
		Logs:      in.Logs,
		Schedules: in.Schedules(schedulesCache),
		DayCounts: in.Schedules(daysCache),
		Report:    in.Report,
		DayStart:  in.DayStart,
		Now:       in.Now,
		Cores:     cores,
		Scheduler: in.Scheduler,
	})
	return &Scenarios{
		Cards: set, DeckEdit: deckedit.NewScenarios(read, write), Flash: running, Day: running.Day,
		Neighborhood: neighborhood.NewShow(
			note.NewShowLinks(in.Links),
			struct {
				ports.NoteRefQueries
				ports.NoteTypeQueries
			}{in.Notes, in.Types},
			note.NewRead(in.Readers),
		),
	}
}

package container

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Flashcards is what everything that runs a vault's cards is called.
type Flashcards = flashcards.Scenarios

// ShowNeighbourhood builds the show neighbourhood usecase for flashcards.
func (c Config) ShowNeighbourhood(
	linked note.ShowLinks,
	notes port.NoteQueries,
	read note.Read,
) flashcards.ShowNeighbourhood {
	return flashcards.NewShowNeighbourhood(linked, notes, read)
}

// Answers opens the shelf a vault's answers are kept on. It is the same folder
// the application keeps everything else of its own in, under an area of its
// own.
func (c Config) Answers() port.DerivedStores {
	return filesystem.DerivedStores{Options: c.vaultOptions(), Area: filesystem.FlashcardsDir}
}

// GetDayStartTime is how long past midnight a day of review begins, as the settings
// hold it. A file that cannot be read begins the day where an installation
// nobody has configured begins it.
func (c Config) GetDayStartTime() time.Duration {
	held, err := c.Settings()
	if err != nil {
		return settings.DefaultDayStart()
	}
	return held.GetDayStartTime()
}

// getCores is how many decks are parsed, and how many places of a curve are
// worked out, at once: the compute plan's workers, or one per CPU when the
// container has no plan.
func (c Config) getCores() int {
	if c.ResolvedCompute.Workers > 0 {
		return c.ResolvedCompute.Workers
	}
	// gomutants:disable-next-line INTEGER_DECREMENT reason="GOMAXPROCS reads the setting for every argument below one"
	return runtime.GOMAXPROCS(0)
}

// Schedules is where the working out is remembered between launches: the folder
// the configuration names, or the platform's cache location.
func (c Config) Schedules() (port.ScheduleStore, error) {
	if c.SchedulesPath != "" {
		return appstate.SchedulesAt(c.SchedulesPath), nil
	}
	// A returned *Schedules is nil where there is no cache folder, and a nil
	// pointer in an interface is not a nil interface: the caller's check for one
	// would pass and the first call on it would panic.
	kept, err := appstate.OpenSchedules()
	if err != nil {
		return nil, err
	}
	return kept, nil
}

// OpenDayCounts is where what each day came to is remembered. It stands beside the
// schedules and not in them, because the two go out of date by different rules:
// a schedule is the whole history read again, and a day is a sum one file at a
// time.
func (c Config) OpenDayCounts() (port.ScheduleStore, error) {
	if c.SchedulesPath != "" {
		return appstate.SchedulesAt(filepath.Join(c.SchedulesPath, "days")), nil
	}
	counting, err := appstate.OpenCounting()
	if err != nil {
		return nil, err
	}
	return counting, nil
}

// Flashcards builds the scenarios against this installation.
//
// The cache is where the working out is remembered between launches. It is this
// machine's, so it stands in the platform's cache location and a machine that
// has none works the schedules out at every launch.
func (c Config) Flashcards(
	notes port.NoteQueries,
	links port.LinkQueries,
	problems port.ProblemQueries,
	index func(ctx context.Context, v domain.Vault, paths []string) error,
) flashcards.Scenarios {
	logs := c.Answers()
	now := c.Clock()
	kept, err := c.Schedules()
	if err != nil {
		// A machine that cannot say where its caches go works the schedules out
		// at every launch. That is slower and no less correct.
		c.handleError(fmt.Errorf("the schedules are worked out at every launch: %w", err))
	}

	counting, err := c.OpenDayCounts()
	if err != nil {
		c.handleError(fmt.Errorf("the days are counted again at every launch: %w", err))
	}

	return flashcards.NewScenarios(flashcards.Config{
		Readers:   c.VaultReaders(),
		Writers:   c.VaultWriters(),
		Notes:     notes,
		Links:     links,
		Problems:  problems,
		Index:     index,
		Logs:      logs,
		Schedules: kept,
		DayCounts: counting,
		DayStart:  c.GetDayStartTime,
		Now:       now,
		Cores:     c.getCores(),
		Scheduler: c.Scheduler,
	})
}

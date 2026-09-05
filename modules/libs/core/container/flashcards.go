package container

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// Flashcards is everything that runs a vault's cards: what stands in it, what it
// owes, what to ask next, and what an answer is written to.
type Flashcards struct {
	CardFaces flashcards.ListCardFaces
	Marks     flashcards.MarkCards
	Schedules flashcards.Schedules
	CardsDue  flashcards.CountCardsDue
	Session   flashcards.Session
	Log       flashcards.Log
	// Counted is how much of a vault was answered on each day it was reviewed.
	Counted flashcards.CountReviews
	// Presets is which preset each deck is scheduled by, and how one is read,
	// written and made.
	Presets flashcards.Presets
	// Curves is what the one control of a preset comes to over the whole range
	// of its goal.
	Curves flashcards.ProjectCurve
	// Day is where one day of review gives way to the next.
	Day review.Day
}

// Answers opens the shelf a vault's answers are kept on. It is the same folder
// the application keeps everything else of its own in, under an area of its
// own.
func (c Config) Answers() port.DerivedStores {
	return filesystem.DerivedStores{Options: c.VaultOptions(), Area: filesystem.FlashcardsDir}
}

// DayStarts is how long past midnight a day of review begins, as the settings
// hold it. A file that cannot be read begins the day where an installation
// nobody has configured begins it.
func (c Config) DayStarts() time.Duration {
	path, err := c.settingsFile()
	if err != nil {
		return settings.DefaultStarts()
	}
	held, err := settings.At(path)
	if err != nil {
		return settings.DefaultStarts()
	}
	return held.DayStarts()
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

// Counting is where what each day came to is remembered. It stands beside the
// schedules and not in them, because the two go out of date by different rules:
// a schedule is the whole history read again, and a day is a sum one file at a
// time.
func (c Config) Counting() (port.ScheduleStore, error) {
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
	index func(ctx context.Context, v domain.Vault, paths []string) error,
) Flashcards {
	logs := c.Answers()
	now := c.Clock()
	kept, err := c.Schedules()
	if err != nil {
		// A machine that cannot say where its caches go works the schedules out
		// at every launch. That is slower and no less correct.
		c.trouble(fmt.Errorf("the schedules are worked out at every launch: %w", err))
	}

	faces := flashcards.ListCardFaces{Readers: c.VaultReaders(), Notes: notes, Links: links}
	marking := flashcards.MarkCards{
		Readers: c.VaultReaders(), Writers: c.VaultWriters(),
		Notes: notes, Links: links, Index: index, Now: now,
	}
	day := review.Day{Starts: c.DayStarts()}

	counting, err := c.Counting()
	if err != nil {
		c.trouble(fmt.Errorf("the days are counted again at every launch: %w", err))
	}

	presets := flashcards.Presets{
		Readers: c.VaultReaders(), Writers: c.VaultWriters(),
		Links: links, Notes: notes, Index: index, Day: day, Now: now,
	}
	// A link the index does not carry is accounted for in what parsing turned
	// up, which is the same reader answering both.
	if said, holds := notes.(port.ProblemQueries); holds {
		presets.Problems = said
	}

	// Each card is worked out at the share of the cards its own preset asks
	// for, which is what says which preset a card face stands under.
	schedules := flashcards.Schedules{
		Logs: logs, Cache: kept, By: review.NewFSRS(), Day: day,
		CardFaces: faces, Presets: presets,
	}

	return Flashcards{
		CardFaces: faces,
		Marks:     marking,
		Schedules: schedules,
		CardsDue: flashcards.CountCardsDue{
			CardFaces: faces, Schedules: schedules, Presets: presets, Day: day, Now: now,
		},
		Session: flashcards.Session{
			Marks: marking, CardFaces: faces, Schedules: schedules,
			Presets: presets, Day: day, Now: now,
		},
		Log: flashcards.Log{Stores: logs},
		Counted: flashcards.CountReviews{
			Logs: logs, Cache: counting, Schedules: schedules, Day: day, Now: now,
		},
		Presets: presets,
		// How many places of a curve run at once is what this machine can run
		// at once, which is a fact only here is allowed to read.
		Curves: flashcards.ProjectCurve{
			CardFaces: faces, Schedules: schedules, Presets: presets, Day: day, Now: now,
			Cores: runtime.GOMAXPROCS(0),
		},
		Day: day,
	}
}

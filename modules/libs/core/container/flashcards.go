package container

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// Flashcards is everything that runs a vault's cards: what stands in it, what it
// owes, what to ask next, and what an answer is written to.
type Flashcards struct {
	Standings flashcards.Standings
	Marking   flashcards.Marking
	Schedules flashcards.Schedules
	Owed      flashcards.Owed
	Session   flashcards.Session
	Log       flashcards.Log
	// Counted is how much of a vault was answered on each day it was reviewed.
	Counted flashcards.Counted
	// Presets is which preset each deck is scheduled by, and how one is read,
	// written and made.
	Presets flashcards.Presets
	// Curves is what the one control of a preset comes to over the whole range
	// of its goal.
	Curves flashcards.Curves
	// Day is where one day of review gives way to the next.
	Day history.Day
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

// Kept is where the working out is remembered between launches: the folder the
// configuration names, or the platform's cache location.
func (c Config) Kept() (port.ScheduleStore, error) {
	if c.SchedulesPath != "" {
		return appstate.SchedulesAt(c.SchedulesPath), nil
	}
	return appstate.OpenSchedules()
}

// Counting is where what each day came to is remembered. It stands beside the
// schedules and not in them, because the two go out of date by different rules:
// a schedule is the whole history read again, and a day is a sum one file at a
// time.
func (c Config) Counting() (port.ScheduleStore, error) {
	if c.SchedulesPath != "" {
		return appstate.SchedulesAt(filepath.Join(c.SchedulesPath, "days")), nil
	}
	return appstate.OpenCounting()
}

// Flashcards builds the scenarios against this installation.
//
// Kept is where the working out is remembered between launches. It is a cache
// and it is this machine's, so it stands in the platform's cache location and a
// machine that has none works the schedules out at every launch.
func (c Config) Flashcards(
	notes port.NoteQueries,
	links port.LinkQueries,
	index func(ctx context.Context, v domain.Vault, paths []string) error,
) Flashcards {
	logs := c.Answers()
	kept, err := c.Kept()
	if err != nil {
		// A machine that cannot say where its caches go works the schedules out
		// at every launch. That is slower and no less correct.
		c.trouble(fmt.Errorf("the schedules are worked out at every launch: %w", err))
	}

	standing := flashcards.Standings{Readers: c.VaultReaders(), Notes: notes, Links: links}
	marking := flashcards.Marking{
		Readers: c.VaultReaders(), Writers: c.VaultWriters(),
		Notes: notes, Links: links, Index: index, Now: time.Now,
	}
	day := history.Day{Starts: c.DayStarts()}

	counting, err := c.Counting()
	if err != nil {
		c.trouble(fmt.Errorf("the days are counted again at every launch: %w", err))
	}

	presets := flashcards.Presets{
		Readers: c.VaultReaders(), Writers: c.VaultWriters(),
		Links: links, Notes: notes, Index: index, Day: day, Now: time.Now,
	}
	// A link the index does not carry is accounted for in what parsing turned
	// up, which is the same reader answering both.
	if said, holds := notes.(port.ProblemQueries); holds {
		presets.Problems = said
	}

	// Each card is worked out at the share of the cards its own preset asks
	// for, which is what says which preset a card face stands under.
	schedules := flashcards.Schedules{
		Logs: logs, Kept: kept, By: history.NewFSRS(), Day: day,
		Standings: standing, Presets: presets,
	}

	return Flashcards{
		Standings: standing,
		Marking:   marking,
		Schedules: schedules,
		Owed: flashcards.Owed{
			Standings: standing, Schedules: schedules, Presets: presets, Day: day, Now: time.Now,
		},
		Session: flashcards.Session{
			Marking: marking, Standings: standing, Schedules: schedules,
			Presets: presets, Day: day, Now: time.Now,
		},
		Log: flashcards.Log{Stores: logs},
		Counted: flashcards.Counted{
			Logs: logs, Kept: counting, Schedules: schedules, Day: day, Now: time.Now,
		},
		Presets: presets,
		Curves: flashcards.Curves{
			Standings: standing, Schedules: schedules, Presets: presets, Day: day, Now: time.Now,
		},
		Day: day,
	}
}

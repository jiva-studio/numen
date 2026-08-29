package container

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
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
	// Day is where one day of review gives way to the next.
	Day history.Day
}

// Answers opens the shelf a vault's answers are kept on. It is the same folder
// the application keeps everything else of its own in, under an area of its
// own.
func (c Config) Answers() port.DerivedStores {
	return filesystem.DerivedStores{Options: c.VaultOptions(), Area: filesystem.FlashcardsDir}
}

// Kept is where the working out is remembered between launches: the folder the
// configuration names, or the platform's cache location.
func (c Config) Kept() (port.Schedules, error) {
	if c.SchedulesPath != "" {
		return appstate.SchedulesAt(c.SchedulesPath), nil
	}
	return appstate.OpenSchedules()
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
		// at every launch. That is slower and no less correct, and it is said
		// once here so the slowness is not a mystery.
		fmt.Fprintln(os.Stderr, "numen: the schedules are worked out at every launch:", err)
	}

	standing := flashcards.Standings{Readers: c.VaultReaders(), Notes: notes, Links: links}
	marking := flashcards.Marking{
		Readers: c.VaultReaders(), Writers: c.VaultWriters(),
		Notes: notes, Links: links, Index: index, Now: time.Now,
	}
	schedules := flashcards.Schedules{Logs: logs, Kept: kept, By: history.NewFSRS()}
	day := history.Day{Starts: history.DayStarts}
	return Flashcards{
		Standings: standing,
		Marking:   marking,
		Schedules: schedules,
		Owed:      flashcards.Owed{Standings: standing, Schedules: schedules, Day: day, Now: time.Now},
		Session: flashcards.Session{
			Marking: marking, Standings: standing, Schedules: schedules, Day: day, Now: time.Now,
		},
		Log: flashcards.Log{Stores: logs},
		Day: day,
	}
}

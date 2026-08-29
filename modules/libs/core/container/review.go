package container

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/review"
)

// Review is everything that runs a vault's cards: what stands in it, what it
// owes, what to ask next, and what an answer is written to.
type Review struct {
	Standings review.Standings
	Schedules review.Schedules
	Owed      review.Owed
	Session   review.Session
	Log       review.Log
	// Day is where one day of review gives way to the next.
	Day history.Day
}

// Answers opens the shelf a vault's answers are kept on. It is the same folder
// the application keeps everything else of its own in, under an area of its
// own.
func (c Config) Answers() port.DerivedStores {
	return filesystem.DerivedStores{Options: c.VaultOptions(), Area: filesystem.ReviewDir}
}

// Review builds the scenarios against this installation.
//
// Kept is where the working out is remembered between launches. It is a cache
// and it is this machine's, so it stands in the platform's cache location and a
// machine that has none works the schedules out at every launch.
func (c Config) Review(
	notes port.NoteQueries,
	links port.LinkQueries,
	index func(ctx context.Context, v domain.Vault, paths []string) error,
) Review {
	logs := c.Answers()
	var kept port.Schedules
	if at, err := appstate.OpenSchedules(); err == nil {
		kept = at
	}

	standing := review.Standings{
		Readers: c.VaultReaders(), Writers: c.VaultWriters(),
		Notes: notes, Links: links, Index: index, Now: time.Now,
	}
	schedules := review.Schedules{Logs: logs, Kept: kept, By: history.NewFSRS()}
	day := history.Day{Starts: history.DayStarts}
	return Review{
		Standings: standing,
		Schedules: schedules,
		Owed:      review.Owed{Standings: standing, Schedules: schedules, Day: day, Now: time.Now},
		Session:   review.Session{Standings: standing, Schedules: schedules, Day: day, Now: time.Now},
		Log:       review.Log{Stores: logs},
		Day:       day,
	}
}

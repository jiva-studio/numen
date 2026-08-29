package review

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
)

// keptVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a replay and nothing else.
const keptVersion = 1

type kept struct {
	V  int    `json:"v"`
	By string `json:"by"`
	// Files are the names of the log files this was worked out from.
	Files []string       `json:"files"`
	Seats []keptSchedule `json:"seats"`
}

type keptSchedule struct {
	Card       string  `json:"card"`
	Face       string  `json:"face"`
	Due        string  `json:"due"`
	Last       string  `json:"last"`
	Reps       int     `json:"reps"`
	Lapses     int     `json:"lapses"`
	Stability  float64 `json:"stability"`
	Difficulty float64 `json:"difficulty"`
	Phase      uint8   `json:"phase"`
}

// Schedules is where the answers have left every seat of one vault.
//
// What is kept between launches is a cache: a file listing the runs it was
// worked out from. A cache filled from every run the vault now holds, by this
// scheduler, is used as it stands. Anything else is the whole history read
// again — a run arriving from another machine holds answers older than ones
// already counted, and a schedule depends on the order of the answers, so
// nothing can be added to what the later ones produced.
type Schedules struct {
	Logs port.DerivedStores
	// Kept is where the working out is remembered. A build holding none works
	// it out at every launch.
	Kept port.Schedules
	By   history.Scheduler
}

// Execute is every seat the vault's answers name, and where they leave it.
func (u Schedules) Execute(
	ctx context.Context, v domain.Vault,
) (map[history.Seat]history.Schedule, error) {
	held, err := Log{Stores: u.Logs}.Read(ctx, v)
	if err != nil {
		return nil, err
	}
	if out, ok := u.remembered(ctx, v, held.Files); ok {
		return out, nil
	}

	out := history.Replay(u.By, held.Answers)
	u.remember(ctx, v, held.Files, out)
	return out, nil
}

// remembered is what was worked out last time, when it was worked out from the
// runs the vault now holds and by the scheduler now asking.
func (u Schedules) remembered(
	ctx context.Context, v domain.Vault, files []string,
) (map[history.Seat]history.Schedule, bool) {
	if u.Kept == nil {
		return nil, false
	}
	raw, err := u.Kept.Read(ctx, v.ID)
	if err != nil {
		return nil, false
	}
	var was kept
	if err := json.Unmarshal(raw, &was); err != nil {
		return nil, false
	}
	if was.V != keptVersion || was.By != u.By.Name() || !slices.Equal(was.Files, files) {
		return nil, false
	}

	out := make(map[history.Seat]history.Schedule, len(was.Seats))
	for _, s := range was.Seats {
		due, err := history.Moment(s.Due)
		if err != nil {
			return nil, false
		}
		last, err := history.Moment(s.Last)
		if err != nil {
			return nil, false
		}
		out[history.Seat{Card: s.Card, Face: s.Face}] = history.Schedule{
			Due: due, Last: last, Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		}
	}
	return out, true
}

// remember puts the working out where the next launch will find it. A cache
// that could not be written is a launch that works it out again, so nothing
// here is reported.
func (u Schedules) remember(
	ctx context.Context, v domain.Vault, files []string, out map[history.Seat]history.Schedule,
) {
	if u.Kept == nil {
		return
	}
	now := kept{V: keptVersion, By: u.By.Name(), Files: files}
	for seat, s := range out {
		now.Seats = append(now.Seats, keptSchedule{
			Card: seat.Card, Face: seat.Face,
			Due:  s.Due.UTC().Format(history.Stamp),
			Last: s.Last.UTC().Format(history.Stamp),
			Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		})
	}
	slices.SortFunc(now.Seats, func(a, b keptSchedule) int {
		if a.Card != b.Card {
			return strings.Compare(a.Card, b.Card)
		}
		return strings.Compare(a.Face, b.Face)
	})

	raw, err := json.Marshal(now)
	if err != nil {
		return
	}
	_ = u.Kept.Write(ctx, v.ID, raw)
}

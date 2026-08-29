package flashcards

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// keptVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a replay and nothing else.
const keptVersion = 1

type kept struct {
	V  int    `json:"v"`
	By string `json:"by"`
	// Files are the log files this was worked out from, each with the length it
	// had. A run is appended to under one name, so a name alone would call a
	// cache current while the answers written after it were never counted.
	Files []keptFile     `json:"files"`
	Faces []keptSchedule `json:"faces"`
}

type keptFile struct {
	Name string `json:"name"`
	Size int    `json:"size"`
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

// Schedules is where the answers have left every card face of one vault.
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

// Execute is every card face the vault's answers name, and where they leave it.
func (u Schedules) Execute(
	ctx context.Context, v domain.Vault,
) (map[history.CardFace]history.Schedule, error) {
	log := Log{Stores: u.Logs}

	// The listing comes first, and the files are read only when the cache does
	// not answer: reading the folder is what a launch does anyway, and reading
	// every answer in it is what the cache is for.
	files, err := log.Files(ctx, v)
	if err != nil {
		return nil, err
	}
	if out, ok := u.remembered(ctx, v, files); ok {
		return out, nil
	}

	held, err := log.Read(ctx, v)
	if err != nil {
		return nil, err
	}
	return u.From(ctx, v, held), nil
}

// From is where a log that has already been read leaves every card face. A
// caller holding the answers does not read them again to be told this.
func (u Schedules) From(
	ctx context.Context, v domain.Vault, held Held,
) map[history.CardFace]history.Schedule {
	out := history.Replay(u.By, held.Answers)
	u.remember(ctx, v, held.Files, out)
	return out
}

// remembered is what was worked out last time, when it was worked out from the
// runs the vault now holds and by the scheduler now asking.
func (u Schedules) remembered(
	ctx context.Context, v domain.Vault, files []port.Stored,
) (map[history.CardFace]history.Schedule, bool) {
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
	if was.V != keptVersion || was.By != u.By.Name() || !read(was.Files, files) {
		return nil, false
	}

	out := make(map[history.CardFace]history.Schedule, len(was.Faces))
	for _, s := range was.Faces {
		due, err := history.Moment(s.Due)
		if err != nil {
			return nil, false
		}
		last, err := history.Moment(s.Last)
		if err != nil {
			return nil, false
		}
		out[history.CardFace{Card: s.Card, Face: s.Face}] = history.Schedule{
			Due: due, Last: last, Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		}
	}
	return out, true
}

// read reports whether a cache was worked out from exactly the log that now
// stands: the same files, each the length it was read at.
func read(was []keptFile, files []port.Stored) bool {
	if len(was) != len(files) {
		return false
	}
	for at, one := range was {
		if one.Name != files[at].Name || one.Size != files[at].Size {
			return false
		}
	}
	return true
}

// remember puts the working out where the next launch will find it. A cache
// that could not be written is a launch that works it out again, so nothing
// here is reported.
func (u Schedules) remember(
	ctx context.Context, v domain.Vault, files []port.Stored, out map[history.CardFace]history.Schedule,
) {
	if u.Kept == nil {
		return
	}
	now := kept{V: keptVersion, By: u.By.Name()}
	for _, one := range files {
		now.Files = append(now.Files, keptFile{Name: one.Name, Size: one.Size})
	}
	for on, s := range out {
		now.Faces = append(now.Faces, keptSchedule{
			Card: on.Card, Face: on.Face,
			Due:  s.Due.UTC().Format(history.Stamp),
			Last: s.Last.UTC().Format(history.Stamp),
			Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		})
	}
	slices.SortFunc(now.Faces, func(a, b keptSchedule) int {
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

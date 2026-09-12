package flashcards

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// keptVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a replay and nothing else.
const keptVersion = 2

type scheduleCache struct {
	V int `json:"v"`
	// By is the assignment this was worked out under: which card face stood
	// under which target. A schedule worked out under one assignment is not read
	// back under another.
	By string `json:"by"`
	// Files are the log files this was worked out from, each with the length it
	// had. A run is appended to under one name, so a name alone would call a
	// cache current while the answers written after it were never counted.
	Files []cachedFile     `json:"files"`
	Faces []cachedSchedule `json:"faces"`
}

type cachedFile struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type cachedSchedule struct {
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
	// Cache is where the working out is remembered. A build holding none works
	// it out at every launch.
	Cache port.ScheduleStore
	By    review.Scheduler
	// Day is where one day of review gives way to the next, which is what says
	// on which day a card placed by its preset lands.
	Day review.Day
	// CardFaces and Presets say which preset schedules each card face, so a
	// card is worked out at the share of the cards its own preset asks for. A
	// build holding neither works every card out by By.
	CardFaces ListCardFaces
	Presets   Presets
	// At is the scheduler asking for a share of the cards to come back. A build
	// holding none reads FSRS.
	At func(retention float64) review.Scheduler
}

// NewSchedules is what a vault's cards are placed through: where its answers
// are kept, the scheduler that places a card, where one day of review gives way
// to the next, and the card faces and presets that say which share of the cards
// each card is worked out at.
func NewSchedules(
	logs port.DerivedStores,
	by review.Scheduler,
	day review.Day,
	faces ListCardFaces,
	presets Presets,
) Schedules {
	return Schedules{Logs: logs, By: by, Day: day, CardFaces: faces, Presets: presets}
}

// assignment is which scheduler each card face is worked out by, and what that
// assignment comes to.
//
// The mark is what says a cache is out of date. A target moving, a deck
// repointed, a card moved between decks and a preset placing its cards
// differently all move it, and the whole cache is thrown away: a schedule
// depends on every answer before it, so nothing worked out under the old
// assignment can be kept.
type assignment struct {
	under review.Assignment
	mark  string
}

// plain is every card face on the one scheduler, at the preset a deck naming
// none is scheduled by.
func (u Schedules) plain() assignment {
	return assignment{
		under: review.By(u.By),
		mark:  getMark([]string{u.By.Name(), u.opening(), review.Defaults().GetPlacing()}),
	}
}

// opening is the hour a day of review begins at, as a name a cache is filed
// under. A day beginning elsewhere puts a card on another day.
func (u Schedules) opening() string {
	in := "local"
	if u.Day.In != nil {
		in = u.Day.In.String()
	}
	return "day\t" + review.Clock(u.Day.Starts) + "\t" + in
}

// getAssignment is the scheduler each card face is worked out by, over a
// reading of this vault's presets of its own.
//
// A card face whose deck names no preset is worked out at the defaults, and so
// is every card of a vault nothing has read yet.
func (u Schedules) getAssignment(ctx context.Context, v domain.Vault) (assignment, error) {
	if u.Presets.Links == nil || u.CardFaces.Notes == nil {
		return u.plain(), nil
	}
	faces, err := u.CardFaces.Execute(ctx, v)
	if errors.Is(err, ErrNotCarried) {
		return u.plain(), nil
	}
	if err != nil {
		return assignment{}, err
	}
	return u.under(ctx, v, u.Presets.Reading(), faces)
}

// under is the same, from the cards and the reading of the presets a caller
// already holds.
func (u Schedules) under(
	ctx context.Context, v domain.Vault, reading *PresetReads, faces []CardFace,
) (assignment, error) {
	out := u.plain()
	if reading == nil || reading.Links == nil {
		return out, nil
	}

	by := make(map[string]review.SchedulingPolicy)
	under := make(map[review.CardFaceID]review.SchedulingPolicy, len(faces))
	asked := make(map[string]string, len(faces))
	for _, one := range faces {
		path, known := asked[one.Deck]
		if !known {
			p, err := reading.Of(ctx, v, one.Deck)
			if err != nil {
				return assignment{}, err
			}
			path = p.Path
			asked[one.Deck] = path
			if _, held := by[path]; !held {
				by[path] = review.SchedulingPolicy{By: u.at(p.Settings.Retention), Preset: p.Settings}
			}
		}
		under[one.ID] = by[path]
	}

	marks := make([]string, 0, len(under)+2)
	marks = append(marks, u.By.Name(), u.opening())
	for face, one := range under {
		marks = append(marks, face.Card+"\t"+face.Face+"\t"+one.By.Name()+"\t"+one.Preset.GetPlacing())
	}
	out.mark = getMark(marks)
	out.under = func(face review.CardFaceID) review.SchedulingPolicy {
		if one, held := under[face]; held {
			return one
		}
		return review.SchedulingPolicy{By: u.By, Preset: review.Defaults()}
	}
	return out, nil
}

// getMark is one name for an assignment, whatever order it was walked in. It is
// a digest because a vault of many cards names many card faces, and what a mark
// is asked is whether it is the one that stands.
func getMark(lines []string) string {
	slices.Sort(lines)
	sum := sha256.New()
	for _, one := range lines {
		sum.Write([]byte(one))
		sum.Write([]byte{'\n'})
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// at is the scheduler asking for a share of the cards to come back.
func (u Schedules) at(retention float64) review.Scheduler {
	if u.At != nil {
		return u.At(retention)
	}
	return review.NewFSRSAt(retention)
}

// Execute is every card face the vault's answers name, and where they leave it.
func (u Schedules) Execute(
	ctx context.Context, v domain.Vault,
) (map[review.CardFaceID]review.Schedule, error) {
	log := Log{Stores: u.Logs}

	// The listing comes first, and the files are read only when the cache does
	// not answer: reading the folder is what a launch does anyway, and reading
	// every answer in it is what the cache is for.
	files, err := log.Files(ctx, v)
	if err != nil {
		return nil, err
	}
	asks, err := u.getAssignment(ctx, v)
	if err != nil {
		return nil, err
	}
	if out, ok := u.readRemembered(ctx, v, files, asks.mark); ok {
		return out, nil
	}

	held, err := log.Read(ctx, v)
	if err != nil {
		return nil, err
	}
	return u.replayAndRemember(ctx, v, held, asks), nil
}

// From is where a log that has already been read leaves every card face. A
// caller holding the answers does not read them again to be told this.
func (u Schedules) From(
	ctx context.Context, v domain.Vault, held ReviewLog,
) (map[review.CardFaceID]review.Schedule, error) {
	asks, err := u.getAssignment(ctx, v)
	if err != nil {
		return nil, err
	}
	return u.getSchedulesCached(ctx, v, held, asks), nil
}

// worked is where a log a caller has already read leaves the card faces of one
// assignment.
//
// The cache is filed under the assignment a whole vault stands at, and the
// caller here holds the card faces of one preset. A cache is thrown away when
// what it was worked out under changes, so the two are never one answer and the
// cache takes no part: neither read nor written.
func (u Schedules) getSchedules(held ReviewLog, asks assignment) map[review.CardFaceID]review.Schedule {
	return replaySchedules(u.Day, held, asks)
}

// getSchedulesCached is the same, with what a replay came to kept for the next
// launch.
//
// Every path through this asks the cache first. A session and the front door
// stand on the same log and the same assignment, so the second of them to run
// is told what the first worked out.
func (u Schedules) getSchedulesCached(
	ctx context.Context, v domain.Vault, held ReviewLog, asks assignment,
) map[review.CardFaceID]review.Schedule {
	if out, ok := u.readRemembered(ctx, v, held.Files, asks.mark); ok {
		return out
	}
	return u.replayAndRemember(ctx, v, held, asks)
}

// replayAndRemember works the answers out and remembers what they came to. It
// is what a caller that has already found the cache out of date asks for.
func (u Schedules) replayAndRemember(
	ctx context.Context, v domain.Vault, held ReviewLog, asks assignment,
) map[review.CardFaceID]review.Schedule {
	out := replaySchedules(u.Day, held, asks)
	u.remember(ctx, v, held.Files, asks.mark, out)
	return out
}

// replaySchedules is where the answers leave every card face, and is what a
// caller that only reads them asks for.
func replaySchedules(
	d review.Day, held ReviewLog, asks assignment,
) map[review.CardFaceID]review.Schedule {
	return held.History().Replay(d, asks.under)
}

// readRemembered is what was worked out last time, when it was worked out from
// the runs the vault now holds and under the targets now in force.
func (u Schedules) readRemembered(
	ctx context.Context, v domain.Vault, files []port.Entry, mark string,
) (map[review.CardFaceID]review.Schedule, bool) {
	if u.Cache == nil {
		return nil, false
	}
	raw, err := u.Cache.Read(ctx, v.ID)
	if err != nil {
		return nil, false
	}
	var was scheduleCache
	if err := json.Unmarshal(raw, &was); err != nil {
		return nil, false
	}
	if was.V != keptVersion || was.By != mark || !read(was.Files, files) {
		return nil, false
	}

	out := make(map[review.CardFaceID]review.Schedule, len(was.Faces))
	for _, s := range was.Faces {
		due, err := review.Moment(s.Due)
		if err != nil {
			return nil, false
		}
		last, err := review.Moment(s.Last)
		if err != nil {
			return nil, false
		}
		out[review.CardFaceID{Card: s.Card, Face: s.Face}] = review.Schedule{
			Due: due, Last: last, Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		}
	}
	return out, true
}

// read reports whether a cache was worked out from exactly the log that now
// stands: the same files, each the length it was read at.
func read(was []cachedFile, files []port.Entry) bool {
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
	ctx context.Context, v domain.Vault, files []port.Entry, mark string,
	out map[review.CardFaceID]review.Schedule,
) {
	if u.Cache == nil {
		return
	}
	now := scheduleCache{V: keptVersion, By: mark}
	for _, one := range files {
		now.Files = append(now.Files, cachedFile{Name: one.Name, Size: one.Size})
	}
	for on, s := range out {
		now.Faces = append(now.Faces, cachedSchedule{
			Card: on.Card, Face: on.Face,
			Due:  s.Due.UTC().Format(review.Stamp),
			Last: s.Last.UTC().Format(review.Stamp),
			Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		})
	}
	slices.SortFunc(now.Faces, func(a, b cachedSchedule) int {
		if a.Card != b.Card {
			return strings.Compare(a.Card, b.Card)
		}
		return strings.Compare(a.Face, b.Face)
	})

	raw, err := json.Marshal(now)
	if err != nil {
		return
	}
	_ = u.Cache.Write(ctx, v.ID, raw)
}

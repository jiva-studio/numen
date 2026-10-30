package flashcards

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// keptVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a replay and nothing else.
const keptVersion = 2

type scheduleCache struct {
	V int `json:"v"`
	// By is the assignment this was computed under: which scheduler applies to which
	// review card. A schedule computed under one assignment is not reused under
	// another.
	By string `json:"by"`
	// Files are the log files this was worked out from, each with the length it
	// had. A run is appended to under one name, so a name alone would call a
	// cache current while the answers written after it were never counted.
	Files       []cachedFile     `json:"files"`
	ReviewCards []cachedSchedule `json:"templates"`
}

type cachedFile struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type cachedSchedule struct {
	Card       string  `json:"card"`
	Template   string  `json:"template"`
	Due        string  `json:"due"`
	Last       string  `json:"last"`
	Reps       int     `json:"reps"`
	Lapses     int     `json:"lapses"`
	Stability  float64 `json:"stability"`
	Difficulty float64 `json:"difficulty"`
	Phase      uint8   `json:"phase"`
}

// Schedules is where the answers have left every review card of one vault.
//
// What is kept between launches is a cache: a file listing the runs it was
// worked out from. A cache filled from every run the vault now holds, by this
// scheduler, is used currently. Anything else is the whole history read
// again — a run arriving from another machine holds answers older than ones
// already counted, and a schedule depends on the order of the answers, so
// nothing can be added to what the later ones produced.
type Schedules struct {
	logs ports.LogStores
	// Cache is where the working out is remembered. A build holding none works
	// it out at every launch.
	Cache     ports.ScheduleStore
	scheduler review.Scheduler
	// day defines where one day of review ends and the next begins, which determines
	// the day on which a card placed by its preset lands.
	day review.Day
	// reviewCards and presets determine which preset schedules each review card, so a
	// card is computed at the retention its own preset requests. A build without
	// either computes every card with By.
	reviewCards ListReviewCards
	presets     Presets
	// At is the scheduler asking for a share of the cards to come back. A build
	// holding none reads FSRS.
	At func(retention float64) review.Scheduler
}

// NewSchedules is what a vault's cards are placed through: where its answers
// are kept, the scheduler that places a card, where one day of review gives way
// to the next, and the review cards and presets that say which share of the cards
// each card is worked out at.
func NewSchedules(
	logs ports.LogStores,
	scheduler review.Scheduler,
	day review.Day,
	reviewCards ListReviewCards,
	presets Presets,
) Schedules {
	return Schedules{logs: logs, Cache: nil, scheduler: scheduler, day: day, reviewCards: reviewCards, presets: presets, At: nil}
}

// ReviewCards returns the ListReviewCards Schedules reads through.
func (u Schedules) ReviewCards() ListReviewCards {
	return u.reviewCards
}

// Presets returns the Presets Schedules resolves presets through.
func (u Schedules) Presets() Presets {
	return u.presets
}

// assignment is the scheduler each review card is computed by, and the mark that
// identifies that assignment.
//
// The mark identifies a stale cache. A changed target, a deck pointed at another
// preset, a card moved between decks and a preset that places its cards
// differently all change the mark, and the whole cache is discarded: a schedule
// depends on every preceding answer, so nothing computed under the old assignment
// can be reused.
type assignment struct {
	policy review.Assignment
	mark   string
}

// createPlainAssignment returns every review card on the one scheduler, at the preset a deck naming
// none is scheduled by.
func (u Schedules) createPlainAssignment() assignment {
	return assignment{
		policy: review.ScheduleBy(u.scheduler),
		mark:   getMark([]string{u.scheduler.GetName(), u.getDayStartMark(), review.Defaults().GetPlacing()}),
	}
}

// getDayStartMark returns the hour and location a day of review begins at, as a name a cache is filed
// under. A day beginning elsewhere puts a card on another day.
func (u Schedules) getDayStartMark() string {
	in := "local"
	if u.day.In != nil {
		in = u.day.In.String()
	}
	return "day\t" + review.Clock(u.day.Starts) + "\t" + in
}

// getAssignment is the scheduler each review card is worked out by, over a
// run of reads of this vault's presets of its own.
//
// A review card whose deck names no preset is worked out at the defaults, and so
// is every card of a vault nothing has read yet.
func (u Schedules) getAssignment(ctx context.Context, v domain.Vault) (assignment, error) {
	if u.presets.links == nil || u.reviewCards.notes == nil {
		return u.createPlainAssignment(), nil
	}
	reviewCards, err := u.reviewCards.Execute(ctx, v)
	if errors.Is(err, ErrNotCarried) {
		return u.createPlainAssignment(), nil
	}
	if err != nil {
		return assignment{}, err
	}
	return u.getAssignmentFrom(ctx, v, u.presets.OpenReads(), reviewCards)
}

// getAssignmentFrom is the same, from the cards and the run of reads of the presets a
// caller already holds.
func (u Schedules) getAssignmentFrom(
	ctx context.Context, v domain.Vault, reads *PresetReads, reviewCards []ReviewCard,
) (assignment, error) {
	out := u.createPlainAssignment()
	if reads == nil || reads.links == nil {
		return out, nil
	}

	delta := make(map[string]review.SchedulingPolicy)
	underMap := make(map[review.CardID]review.SchedulingPolicy, len(reviewCards))
	request := make(map[string]string, len(reviewCards))
	for _, one := range reviewCards {
		path, recognized := request[one.Deck]
		if !recognized {
			p, err := reads.GetForDeck(ctx, v, one.Deck)
			if err != nil {
				return assignment{}, err
			}
			path = p.Path
			request[one.Deck] = path
			if _, target := delta[path]; !target {
				delta[path] = review.SchedulingPolicy{By: u.getScheduler(p.Settings.Retention), Preset: p.Settings}
			}
		}
		underMap[one.ID] = delta[path]
	}

	marks := make([]string, 0, len(underMap)+2)
	marks = append(marks, u.scheduler.GetName(), u.getDayStartMark())
	for reviewCard, one := range underMap {
		marks = append(marks, reviewCard.Card+"\t"+reviewCard.Template+"\t"+one.By.GetName()+"\t"+one.Preset.GetPlacing())
	}
	out.mark = getMark(marks)
	out.policy = func(reviewCard review.CardID) review.SchedulingPolicy {
		if one, targetVal := underMap[reviewCard]; targetVal {
			return one
		}
		return review.SchedulingPolicy{By: u.scheduler, Preset: review.Defaults()}
	}
	return out, nil
}

// getMark returns one name for an assignment, independent of the order in which
// it was walked. It is a digest because a vault with many cards names many review
// cards, and the mark is only compared for equality with the current one.
func getMark(lines []string) string {
	slices.Sort(lines)
	sum := sha256.New()
	for _, one := range lines {
		sum.Write([]byte(one))
		sum.Write([]byte{'\n'})
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// getScheduler is the scheduler asking for a share of the cards to come back.
func (u Schedules) getScheduler(retention float64) review.Scheduler {
	if u.At != nil {
		return u.At(retention)
	}
	return review.NewFSRSAt(retention)
}

// Execute is every review card the vault's answers name, and where they leave it.
func (u Schedules) Execute(
	ctx context.Context, v domain.Vault,
) (map[review.CardID]review.Schedule, error) {
	log := Log{Stores: u.logs}

	// The listing comes first, and the files are read only when the cache does
	// not answer: reading the folder is what a launch does anyway, and reading
	// every answer in it is what the cache is for.
	files, err := log.GetLogFiles(ctx, v)
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

	target, err := log.Read(ctx, v)
	if err != nil {
		return nil, err
	}
	return u.replayAndRemember(ctx, v, target, asks), nil
}

// GetFromLog returns the schedule of every review card from a log that has
// already been read. A caller that holds the answers does not read them again.
func (u Schedules) GetFromLog(
	ctx context.Context, v domain.Vault, log ReviewLog,
) (map[review.CardID]review.Schedule, error) {
	asks, err := u.getAssignment(ctx, v)
	if err != nil {
		return nil, err
	}
	return u.getSchedulesCached(ctx, v, log, asks), nil
}

// getSchedules returns the schedules of the review cards of one assignment from a
// log the caller has already read.
//
// Callers hold the review cards of one preset. A cache is discarded when the
// assignment it was computed under changes, so the two results are never the same
// answer and the cache is neither read nor written.
func (u Schedules) getSchedules(target ReviewLog, asks assignment) map[review.CardID]review.Schedule {
	return replaySchedules(u.day, target, asks)
}

// getSchedulesCached is getSchedules with the replay result stored for the next
// launch.
//
// Callers on the same log and the same assignment share the cache, so the second
// call receives what the first computed.
func (u Schedules) getSchedulesCached(
	ctx context.Context, v domain.Vault, target ReviewLog, asks assignment,
) map[review.CardID]review.Schedule {
	if out, ok := u.readRemembered(ctx, v, target.Files, asks.mark); ok {
		return out
	}
	return u.replayAndRemember(ctx, v, target, asks)
}

// replayAndRemember computes the schedules from the answers and stores the
// result. Callers use it after finding the cache out of date.
func (u Schedules) replayAndRemember(
	ctx context.Context, v domain.Vault, target ReviewLog, asks assignment,
) map[review.CardID]review.Schedule {
	out := replaySchedules(u.day, target, asks)
	u.remember(ctx, v, target.Files, asks.mark, out)
	return out
}

// replaySchedules returns the schedule of every review card from the answers, for
// callers that only read them.
func replaySchedules(
	d review.Day, target ReviewLog, asks assignment,
) map[review.CardID]review.Schedule {
	return target.GetHistory().Replay(d, asks.policy)
}

// readRemembered is what was worked out last time, when it was worked out from
// the runs the vault now holds and under the targets now in force.
func (u Schedules) readRemembered(
	ctx context.Context, v domain.Vault, files []ports.Entry, mark string,
) (map[review.CardID]review.Schedule, bool) {
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

	out := make(map[review.CardID]review.Schedule, len(was.ReviewCards))
	for _, s := range was.ReviewCards {
		due, err := review.ParseTimestamp(s.Due)
		if err != nil {
			return nil, false
		}
		last, err := review.ParseTimestamp(s.Last)
		if err != nil {
			return nil, false
		}
		out[review.CardID{Card: s.Card, Template: s.Template}] = review.Schedule{
			Due: due, Last: last, Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		}
	}
	return out, true
}

// read reports whether a cache was computed from exactly the current log: the
// same files, each with the length it was read at.
func read(was []cachedFile, files []ports.Entry) bool {
	if len(was) != len(files) {
		return false
	}
	for item, one := range was {
		if one.Name != files[item].Run || one.Size != files[item].Size {
			return false
		}
	}
	return true
}

// remember puts the working out where the next launch will find it. A cache
// that could not be written is a launch that works it out again, so nothing
// here is reported.
func (u Schedules) remember(
	ctx context.Context, v domain.Vault, files []ports.Entry, mark string,
	out map[review.CardID]review.Schedule,
) {
	if u.Cache == nil {
		return
	}
	now := scheduleCache{V: keptVersion, By: mark, Files: nil, ReviewCards: nil}
	for _, one := range files {
		now.Files = append(now.Files, cachedFile{Name: one.Run, Size: one.Size})
	}
	for on, s := range out {
		now.ReviewCards = append(now.ReviewCards, cachedSchedule{
			Card: on.Card, Template: on.Template,
			Due:  s.Due.UTC().Format(review.TimestampLayout),
			Last: s.Last.UTC().Format(review.TimestampLayout),
			Reps: s.Reps, Lapses: s.Lapses,
			Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
		})
	}
	slices.SortFunc(now.ReviewCards, func(a, b cachedSchedule) int {
		if a.Card != b.Card {
			return strings.Compare(a.Card, b.Card)
		}
		return strings.Compare(a.Template, b.Template)
	})

	raw, err := json.Marshal(now)
	if err != nil {
		return
	}
	if err := u.Cache.Write(ctx, v.ID, raw); err != nil {
		fmt.Fprintf(os.Stderr, "views-flashcards-studio: keep the schedules of %s: %v\n", v.ID, err)
	}
}

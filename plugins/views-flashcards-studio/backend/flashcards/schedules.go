package flashcards

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// keptVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a replay and nothing else.
const keptVersion = 2

type scheduleCache struct {
	V           int              `json:"v"`
	By          string           `json:"by"`
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
// What is kept between launches is a cache listing the runs it was worked out from. A cache
// filled from every run the vault now holds, by this scheduler, is used as it is. Anything else
// is the whole history read again: a run arriving from another machine can hold answers older
// than ones counted, and a schedule depends on the order of the answers.
type Schedules struct {
	logs        ports.LogStores
	Cache       ports.ScheduleStore
	Report      func(what string, err error)
	scheduler   review.Scheduler
	day         review.Day
	reviewCards ListReviewCards
	presets     Presets
	At          func(retention float64) review.Scheduler
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
	return Schedules{
		logs: logs, Cache: nil, Report: nil, scheduler: scheduler,
		day: day, reviewCards: reviewCards, presets: presets, At: nil,
	}
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

func (u Schedules) createPlainAssignment() assignment {
	return assignment{
		policy: review.ScheduleBy(u.scheduler),
		mark:   getMark([]string{u.scheduler.GetName(), u.getDayStartMark(), review.Defaults().GetPlacing()}),
	}
}

func (u Schedules) getDayStartMark() string {
	in := "local"
	if u.day.In != nil {
		in = u.day.In.String()
	}
	return "day\t" + review.Clock(u.day.Starts) + "\t" + in
}

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

	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Slice capacity preallocation does not affect the computed mark digest."
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

func getMark(lines []string) string {
	slices.Sort(lines)
	sum := sha256.New()
	for _, one := range lines {
		sum.Write([]byte(one))
		sum.Write([]byte{'\n'})
	}
	return hex.EncodeToString(sum.Sum(nil))
}

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

func (u Schedules) getSchedules(target ReviewLog, asks assignment) map[review.CardID]review.Schedule {
	return replaySchedules(u.day, target, asks)
}

func (u Schedules) getSchedulesCached(
	ctx context.Context, v domain.Vault, target ReviewLog, asks assignment,
) map[review.CardID]review.Schedule {
	if out, ok := u.readRemembered(ctx, v, target.Files, asks.mark); ok {
		return out
	}
	return u.replayAndRemember(ctx, v, target, asks)
}

func (u Schedules) replayAndRemember(
	ctx context.Context, v domain.Vault, target ReviewLog, asks assignment,
) map[review.CardID]review.Schedule {
	out := replaySchedules(u.day, target, asks)
	u.remember(ctx, v, target.Files, asks.mark, out)
	return out
}

func replaySchedules(
	d review.Day, target ReviewLog, asks assignment,
) map[review.CardID]review.Schedule {
	return target.GetHistory().Replay(d, asks.policy)
}

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
	// gomutants:disable-next-line BRANCH_IF reason="Marshaling scheduleCache with primitive fields cannot fail."
	if err != nil {
		return
	}
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="Cache write failure is reported without affecting the returned schedules."
	if err := u.Cache.Write(ctx, v.ID, raw); err != nil && u.Report != nil {
		u.Report("keep the schedules of "+string(v.ID), err)
	}
}

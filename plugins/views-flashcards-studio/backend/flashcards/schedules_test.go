package flashcards_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review/projection"
)

// A build without a cache computes the schedules from the answers at every
// launch and produces the same schedules as a build with a cache. The cache only
// saves work and never determines a result.
func TestSchedulesAreTheSameWithNothingCached(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	retained, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	uncached := s.schedules
	uncached.Cache = nil
	worked, err := uncached.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	if len(worked) != len(retained) {
		t.Fatalf("kept %d schedules, worked out %d", len(retained), len(worked))
	}
	if !worked[on].Due.Equal(retained[on].Due) || worked[on].Reps != retained[on].Reps {
		t.Errorf("worked out %+v, kept %+v", worked[on], retained[on])
	}
}

// A cache built from a listing in one order is recognized as fresh when the
// same runs are listed in another order at the next launch. A directory's own
// order is not promised to repeat between two listings of it.
func TestAScheduleCacheIsFreshAcrossAReorderedListing(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	for _, item := range []time.Time{
		time.Now().AddDate(0, 0, -2),
		time.Now().AddDate(0, 0, -1),
		time.Now(),
	} {
		if _, err := s.run(t, item).Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}

	counts := &loadCounts{}
	logs := countingStores{inner: &reorderedStores{LogStores: s.logs}, on: counts}
	schedules := flashcards.NewSchedules(logs, review.NewFSRS(), today, s.standings, s.presets)
	schedules.Cache = countingKept{inner: s.schedules.Cache, on: counts}

	if _, err := schedules.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if counts.Rewritten != 1 {
		t.Fatalf("the first request wrote the cache %d times, want 1", counts.Rewritten)
	}

	*counts = loadCounts{}
	if _, err := schedules.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if counts.Rewritten != 0 {
		t.Errorf("a cache fresh under a reordered listing was rewritten %d times", counts.Rewritten)
	}
}

// The cache is written in one order whatever order the schedules were worked
// out in, so two machines that answered the same cards hold the same file and a
// synchroniser has nothing to reconcile.
func TestTheCacheIsSavedInOneOrder(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	// Both review cards of one card, so the order turns on the review card and not the card.
	recognise := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	name := review.CardID{Card: "k7m2xq9fzp", Template: "Name it"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), name, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := record.Answer(t.Context(), recognise, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.schedules.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	raw, err := s.schedules.Cache.Read(t.Context(), s.vault.ID)
	if err != nil {
		t.Fatal(err)
	}
	var was plantedCache
	if err := json.Unmarshal(raw, &was); err != nil {
		t.Fatal(err)
	}

	if len(was.ReviewCards) != 2 {
		t.Fatalf("the cache holds %d review cards, want the two answered", len(was.ReviewCards))
	}
	if was.ReviewCards[0].Template != "Name it" || was.ReviewCards[1].Template != "Recognise" {
		t.Errorf("the cache holds %s then %s", was.ReviewCards[0].Template, was.ReviewCards[1].Template)
	}
}

// A cache nothing can read is nothing remembered: the answers are there, and
// the schedules are worked out from them again.
func TestACacheNothingCanReadIsWorkedOutAgain(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	want, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.schedules.Cache.Write(t.Context(), s.vault.ID, []byte("not a cache")); err != nil {
		t.Fatal(err)
	}

	got, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a cache nothing can read rejected the schedules: %v", err)
	}
	if !got[on].Due.Equal(want[on].Due) {
		t.Errorf("worked out %+v, want %+v", got[on], want[on])
	}
}

// newTargetedNotes is a vault of two presets asking for different shares of the
// cards, with a deck of one card under each. Neither evens its days out, so
// what separates two cards here is the target each is placed under.
func newTargetedNotes(high, low float64) map[string]string {
	return map[string]string{
		"Term.md": term,
		"High.md": preset(fmt.Sprintf("retention: %g\neven_load: false\n", high)),
		"Low.md":  preset(fmt.Sprintf("retention: %g\neven_load: false\n", low)),
		"decks/High.md": "---\ntype: deck\nlinks:\n" +
			"  - to: High\n    role: ref\n    type: preset\n---\n" +
			"\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n",
		"decks/Low.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Low\n    role: ref\n    type: preset\n---\n" +
			"\n## Two ^zpqrstvwxy\n\n[[Term]]\n\n### Word\n\ngam\n\n### Meaning\n\nto go\n",
	}
}

// The two review cards of the targeted vault, answered alike.
var (
	underHigh = review.CardID{Card: "k7m2xq9fzp", Template: "Say it"}
	underLow  = review.CardID{Card: "zpqrstvwxy", Template: "Say it"}
)

// answerBothAlike takes both cards through the same answers at the same
// moments, far enough for each to be learned, so what separates their schedules
// is the preset each is placed under.
func answerBothAlike(t *testing.T, s vaulted) {
	t.Helper()
	for day := range 3 {
		record := s.run(t, saturday.AddDate(0, 0, day-3))
		answer(t, record, underHigh.Card, 6*time.Second)
		answer(t, record, underLow.Card, 6*time.Second)
	}
}

// A card is scheduled at the retention its own preset requests, so two presets
// with different retention targets give the same answer different intervals.
func TestEachPresetSchedulesItsCardsAtItsOwnTarget(t *testing.T) {
	t.Parallel()
	s := openVault(t, newTargetedNotes(0.95, 0.75))
	answerBothAlike(t, s)

	got, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	high, low := got[underHigh], got[underLow]
	if high.Due.IsZero() || low.Due.IsZero() {
		t.Fatalf("the two cards came to %+v and %+v", high, low)
	}
	// A higher retention target gives shorter intervals.
	if !high.Due.Before(low.Due) {
		t.Errorf("asking for 0.95 comes round at %v and 0.75 at %v", high.Due, low.Due)
	}
}

// Moving one preset's target throws the cache away and works the schedules out
// again, so the cards under it come round somewhere else.
func TestMovingATargetWorksTheSchedulesOutAgain(t *testing.T) {
	t.Parallel()
	s := openVault(t, newTargetedNotes(0.95, 0.75))
	answerBothAlike(t, s)

	was, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	settings, err := s.presets.Read(t.Context(), s.vault, "Low.md")
	if err != nil {
		t.Fatal(err)
	}
	moved := settings.Settings
	moved.Retention = 0.95
	if _, err := s.presets.Save(t.Context(), s.vault, "Low.md", moved, domain.FileStat{}); err != nil {
		t.Fatal(err)
	}

	now, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if now[underLow].Due.Equal(was[underLow].Due) {
		t.Errorf("the target moved and the card still comes round at %v", now[underLow].Due)
	}
	// The two now ask for the same share, so they come round together.
	if !now[underLow].Due.Equal(now[underHigh].Due) {
		t.Errorf("two presets at one target come round at %v and %v",
			now[underLow].Due, now[underHigh].Due)
	}
}

// Pointing a deck at the other preset works the schedules out again. The two
// targets in force did not move; which card is placed under which did, and that is
// what the cache is held against.
func TestRepointingADeckWorksTheSchedulesOutAgain(t *testing.T) {
	t.Parallel()
	files := newTargetedNotes(0.95, 0.75)
	s := openVault(t, files)
	answerBothAlike(t, s)

	was, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if was[underHigh].Due.Equal(was[underLow].Due) {
		t.Fatalf("the two cards stand at one day, %v", was[underHigh].Due)
	}

	// The decks swap presets, so the card with the 0.95 target gets 0.75
	// and the reverse.
	write(t, s, "decks/High.md", strings.Replace(files["decks/High.md"], "to: High", "to: Low", 1))
	write(t, s, "decks/Low.md", strings.Replace(files["decks/Low.md"], "to: Low", "to: High", 1))

	now, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if !now[underHigh].Due.Equal(was[underLow].Due) || !now[underLow].Due.Equal(was[underHigh].Due) {
		t.Errorf("the decks swapped presets and the cards come round at %v and %v, want %v and %v",
			now[underHigh].Due, now[underLow].Due, was[underLow].Due, was[underHigh].Due)
	}
}

// A card moved into a deck under another preset is scheduled by that preset.
// Both decks go on holding cards and go on pointing where they pointed, so the
// two targets in force are the two that were in force.
func TestACardMovedToAnotherDeckIsScheduledByItsPreset(t *testing.T) {
	t.Parallel()
	// The card that moves is written in a form that can be cut from one deck and pasted
	// into the other.
	movedCard := "\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n"
	files := newTargetedNotes(0.95, 0.75)
	files["decks/High.md"] += "\n## Stays ^card000001\n\n[[Term]]\n\n### Word\n\nkr\n" +
		"\n### Meaning\n\nto do\n"
	s := openVault(t, files)
	answerBothAlike(t, s)

	was, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	write(t, s, "decks/High.md", strings.Replace(files["decks/High.md"], movedCard, "", 1))
	write(t, s, "decks/Low.md", files["decks/Low.md"]+movedCard)

	now, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if !now[underHigh].Due.Equal(was[underLow].Due) {
		t.Errorf("the card moved under 0.75 and comes round at %v, want %v",
			now[underHigh].Due, was[underLow].Due)
	}
}

// A vault whose decks name no preset is scheduled with the defaults, which are
// the target for the whole vault.
func TestAVaultOfNoPresetsIsScheduledAsItWas(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if _, err := s.run(t, saturday).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	got, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	// The same answers, worked out by the one scheduler and nothing else.
	direct := flashcards.NewSchedules(
		s.logs, review.NewFSRS(), today,
		flashcards.NewListReviewCards(s.env.Ports.Readers, nil, nil),
		flashcards.NewPresets(nil, nil, nil, nil, nil, today, time.Now),
	)
	want, err := direct.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || !got[on].Due.Equal(want[on].Due) {
		t.Errorf("worked out %+v, want %+v", got[on], want[on])
	}
}

// The due day of a card is the same whether the card is answered or projected.
// A preset that evens out the load moves the card to the day with no load, and
// the session and the plot move it to the same day.
//
// The card is answered twice at the hour the day opens. A card answered zero
// time after its last answer is one the projection treats as certainly recalled,
// so both paths compute the same interval and the test compares the day chosen.
func TestSchedules_ReviewedAndProjectedCardShareDueDay(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 9, 7, 4, 0, 0, 0, time.Local)
	delta := review.NewFSRSAt(0.9)
	begun := delta.Next(review.Schedule{}, when, review.Good)
	fell := delta.Next(begun, when, review.Good).Due
	if away := fell.Sub(when).Hours() / 24; away < review.EvenFrom {
		t.Fatalf("the second answer sends the card %g days away", away)
	}

	s := openVault(t, map[string]string{
		"Term.md": term,
		"Even.md": preset(fmt.Sprintf(
			"new_a_day: 0\nreviews_a_day: 9999\nretention: 0.9\neven_load: true\nload: {%s: 0}\n",
			review.DayName(fell.Weekday()))),
		"decks/Even.md": deckNaming([]string{"Even"}, 1, 0),
	})
	on := review.CardID{Card: cardID(0), Template: "Say it"}
	read, err := s.presets.Read(t.Context(), s.vault, "Even.md")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	// The projection takes the card on from where the first answer left it, and
	// its one day of review is the day that answer is given in.
	run := projection.Simulation{By: delta, Day: today, Cost: review.DefaultCost, Days: 21}
	projected, err := run.Run(t.Context(), when, read.Settings, first, 0)
	if err != nil {
		t.Fatal(err)
	}
	comes := 0
	for day, load := range projected.Load {
		if day > 0 && load > 0 {
			comes = day
			break
		}
	}

	if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	reply, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	actualDay := dayFrom(when, reply[on].Due)

	if comes == 0 || comes != actualDay {
		t.Errorf("the projection has the card back on day %d and the answer on day %d",
			comes, actualDay)
	}
	if actualDay == dayFrom(when, fell) {
		t.Errorf("the card stood on the %v it fell on", fell.Weekday())
	}
}

// dayFrom is how many days of review stand between the one holding from and the
// one holding at.
func dayFrom(from, pos time.Time) int {
	open := from
	for i := range 400 {
		if pos.Before(today.GetEnd(open)) {
			return i
		}
		open = today.GetEnd(open)
	}
	return -1
}

// A preset keeping no even load leaves a card where the scheduler puts it, so a
// card falls on a day carrying none of the load. Nothing is shown there: the
func TestACardFallingOnADayAtNoneOfTheLoadExistsOver(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 9, 7, 4, 0, 0, 0, time.Local)
	delta := review.NewFSRSAt(0.9)
	begun := delta.Next(review.Schedule{}, when, review.Good)
	fell := delta.Next(begun, when, review.Good).Due

	s := openVault(t, map[string]string{
		"Term.md": term,
		"Even.md": preset(fmt.Sprintf(
			"new_a_day: 0\nreviews_a_day: 9999\nretention: 0.9\neven_load: false\nload: {%s: 0}\n",
			review.DayName(fell.Weekday()))),
		"decks/Even.md": deckNaming([]string{"Even"}, 1, 0),
	})
	on := review.CardID{Card: cardID(0), Template: "Say it"}

	for range 2 {
		if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}
	reply, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := reply[on].Due; !got.Equal(fell) {
		t.Fatalf("the card comes round at %v, and the scheduler put it at %v", got, fell)
	}

	opens := today.GetEnd(fell).AddDate(0, 0, -1)
	if request := s.sessionAt(t, today, opens.Add(6*time.Hour)).Queue; len(request) != 0 {
		t.Errorf("a %v carrying none of the load asked %d cards", fell.Weekday(), len(request))
	}
	after := today.GetEnd(fell).Add(6 * time.Hour)
	if requestVal := s.sessionAt(t, today, after).Queue; len(requestVal) != 1 {
		t.Errorf("the day after asked %d cards, want the one standing over", len(requestVal))
	}
}

type mockScheduler struct {
	name string
}

func (m mockScheduler) GetName() string { return m.name }
func (m mockScheduler) Next(s review.Schedule, pos time.Time, _ review.Rating) review.Schedule {
	s.Last = pos
	s.Due = pos.Add(24 * time.Hour)
	s.Reps++
	return s
}
func (m mockScheduler) Endings(s review.Schedule, pos time.Time) (good, again review.Schedule) {
	good = m.Next(s, pos, review.Good)
	again = m.Next(s, pos, review.Again)
	return good, again
}
func (m mockScheduler) IsSpaced(_ review.Schedule) bool { return true }

type mockAlgorithm struct {
	calledWith []float64
}

func (m *mockAlgorithm) GetName() string { return "mock-algo" }
func (m *mockAlgorithm) CreateScheduler(retention float64) review.Scheduler {
	m.calledWith = append(m.calledWith, retention)
	return mockScheduler{name: "mock-algo"}
}

func TestScheduleAlgorithmPluggable(t *testing.T) {
	algo := &mockAlgorithm{}
	scenarios := flashcards.NewScenarios(flashcards.Config{
		Scheduler: algo,
	})

	if len(algo.calledWith) == 0 {
		t.Fatal("expected scheduler algorithm CreateScheduler to be called")
	}
	_ = scenarios
}

// A warm request over a vault of two hundred cards answered over thirty days.
// The counts below measure the work of a request and not its size, so a small
// vault gives the same counts as a large one.
const (
	loadCards  = 200
	loadDays   = 30
	loadPerDay = 20
)

// TestCountCardsDue_UsesScheduleCache covers the schedule cache. The answers are
// replayed once, and every later request is served from the replay result.
func TestCountCardsDue_UsesScheduleCache(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	first := l.countLoads(t, func() error { _, err := l.owed.Execute(ctx, l.vault); return err })
	if first.Rewritten != 1 {
		t.Errorf("the first request wrote the cache %d times, want 1", first.Rewritten)
	}

	cached := l.countLoads(t, func() error { _, err := l.owed.Execute(ctx, l.vault); return err })
	if cached.Consulted == 0 {
		t.Error("a warm request never consulted the cache")
	}
	if cached.Rewritten != 0 {
		t.Errorf("a warm request rewrote the cache %d times", cached.Rewritten)
	}
}

// person starting one waits for a listing and not for the whole history.
func TestStartingASessionConsultsTheScheduleCache(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	if _, err := l.owed.Execute(ctx, l.vault); err != nil {
		t.Fatal(err)
	}
	cached := l.countLoads(t, func() error {
		_, err := l.sat.Execute(ctx, l.vault, flashcards.Scope{})
		return err
	})
	if cached.Consulted == 0 {
		t.Error("starting a session never consulted the cache")
	}
	if cached.Rewritten != 0 {
		t.Errorf("starting a session rewrote the cache %d times", cached.Rewritten)
	}
}

// Opening the history consults the cache and leaves it currently.
func TestTheHistoryConsultsTheScheduleCache(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	if _, err := l.owed.Execute(ctx, l.vault); err != nil {
		t.Fatal(err)
	}
	cached := l.countLoads(t, func() error { _, err := l.review.Execute(ctx, l.vault); return err })
	// The day counts are kept in a cache of their own, and the first request
	// over this vault fills it.
	if cached.Rewritten > 1 {
		t.Errorf("opening the history wrote a cache %d times", cached.Rewritten)
	}
}

// A vault answered the same way is left in the same place whether the cache
// answered or the answers were replayed.
func TestTheCacheAnswersWhatAReplayAnswers(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	replayed, err := l.schedules.Execute(ctx, l.vault)
	if err != nil {
		t.Fatal(err)
	}
	remembered, err := l.schedules.Execute(ctx, l.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed) == 0 {
		t.Fatal("the answers left no review card anywhere")
	}
	if len(replayed) != len(remembered) {
		t.Fatalf("the replay left %d review cards and the cache %d",
			len(replayed), len(remembered))
	}
	for reviewCard, was := range replayed {
		if now := remembered[reviewCard]; !now.Due.Equal(was.Due) || now.Reps != was.Reps {
			t.Errorf("%+v is at %+v, and the replay left it at %+v", reviewCard, now, was)
		}
	}
}

// loadCounts is the calls one request made to the store, the cache and the
// scheduler. Each of these is work a request does once, so each is counted.
type loadCounts struct {
	// Listed is how many times the log folder was listed, and Opened how many
	// run files were read.
	Listed int
	Opened int
	// Consulted is how many times the cache was read, and Rewritten how many
	// times it was written.
	Consulted int
	Rewritten int
	// Dated is how many times the scheduler was called for the date following an
	// answer.
	Dated int
}

// countingStores counts the calls a use case makes to one vault's store.
type countingStores struct {
	inner ports.LogStores
	on    *loadCounts
}

func (s countingStores) Open(v domain.Vault) (ports.LogStore, error) {
	one, err := s.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return countingStore{inner: one, on: s.on}, nil
}

type countingStore struct {
	inner ports.LogStore
	on    *loadCounts
}

func (s countingStore) ReadLog(ctx context.Context, name string) ([]byte, error) {
	s.on.Opened++
	return s.inner.ReadLog(ctx, name)
}

func (s countingStore) ListLogs(ctx context.Context) ([]ports.Entry, error) {
	s.on.Listed++
	return s.inner.ListLogs(ctx)
}

func (s countingStore) AppendLog(ctx context.Context, name string, content []byte) error {
	return s.inner.AppendLog(ctx, name, content)
}

// countingKept counts the calls a use case makes to the cache.
type countingKept struct {
	inner ports.ScheduleStore
	on    *loadCounts
}

func (k countingKept) Read(ctx context.Context, vaultID domain.VaultID) ([]byte, error) {
	k.on.Consulted++
	return k.inner.Read(ctx, vaultID)
}

func (k countingKept) Write(ctx context.Context, vaultID domain.VaultID, content []byte) error {
	k.on.Rewritten++
	return k.inner.Write(ctx, vaultID, content)
}

// countingBy counts how often the scheduler is called for a next date.
type countingBy struct {
	inner review.Scheduler
	on    *loadCounts
}

func (b countingBy) GetName() string { return b.inner.GetName() }

func (b countingBy) Next(s review.Schedule, pos time.Time, r review.Rating) review.Schedule {
	b.on.Dated++
	return b.inner.Next(s, pos, r)
}

func (b countingBy) Endings(s review.Schedule, pos time.Time) (review.Schedule, review.Schedule) {
	b.on.Dated += 2
	return b.inner.Endings(s, pos)
}

func (b countingBy) IsSpaced(s review.Schedule) bool { return b.inner.IsSpaced(s) }

// loaded is a vault of many cards and many run files, with every call to the
// store, the cache and the scheduler counted.
type loaded struct {
	vaulted
	on          *loadCounts
	schedules   flashcards.Schedules
	logs        ports.LogStores
	owed        flashcards.CountCardsDue
	sat         flashcards.Session
	review      flashcards.CountReviews
	reviewCards int
}

// load builds a vault of cards cards, answered on days days, perDay answers to
// a day and one run file to a day.
func load(tb testing.TB, cards, days, perDay int) loaded {
	tb.Helper()
	return loadAt(tb, time.Now, cards, days, perDay)
}

// loadAt is load with every request reading the time from now.
func loadAt(tb testing.TB, now ports.Clock, cards, days, perDay int) loaded {
	tb.Helper()

	s := openVault(tb, loadDeck(cards))
	on := &loadCounts{}
	logs := countingStores{inner: s.logs, on: on}
	schedules := flashcards.NewSchedules(
		logs,
		countingBy{inner: review.NewFSRS(), on: on},
		today,
		s.standings,
		s.presets,
	)
	schedules.Cache = countingKept{inner: s.env.Ports.Schedules("reviewCards"), on: on}
	loadAnswers(tb, s, now(), cards, days, perDay)
	reviewUseCase := flashcards.NewCountReviews(logs, schedules, today, now)
	reviewUseCase.Cache = countingKept{inner: s.env.Ports.Schedules("days"), on: on}
	return loaded{
		vaulted:   s,
		on:        on,
		schedules: schedules,
		logs:      logs,
		owed: flashcards.NewCountCardsDue(
			s.standings, schedules, s.presets,
			today, now,
		),
		sat: flashcards.NewSession(
			s.assigning, s.standings, schedules,
			s.presets, today, now,
		),
		review:      reviewUseCase,
		reviewCards: cards,
	}
}

// logRead is this vault's answers, read the way a request reads them.
func (l loaded) logRead(tb testing.TB) (flashcards.ReviewLog, error) {
	tb.Helper()
	return flashcards.Log{Stores: l.logs}.Read(tb.Context(), l.vault)
}

// countLoads runs one request with the counters reset and returns the counts.
func (l loaded) countLoads(tb testing.TB, run func() error) loadCounts {
	tb.Helper()
	*l.on = loadCounts{}
	if err := run(); err != nil {
		tb.Fatal(err)
	}
	return *l.on
}

// loadDeck builds a vault with one card model, one review card and the given
// number of cards, spread over twenty decks.
func loadDeck(cards int) map[string]string {
	const decks = 20
	out := map[string]string{"Term.md": vault["Term.md"]}
	constructed := make([]*strings.Builder, decks)
	for item := range constructed {
		constructed[item] = &strings.Builder{}
		constructed[item].WriteString("---\ntype: deck\n---\n")
	}
	for entry := range cards {
		fmt.Fprintf(constructed[entry%decks],
			"\n## Card %d ^%s\n\n[[Term]]\n\n### Word\n\nword %d\n\n### Meaning\n\nmeaning %d\n",
			entry, createCardID(entry), entry, entry)
	}
	for elem, one := range constructed {
		out[fmt.Sprintf("decks/Deck%02d.md", elem)] = one.String()
	}
	return out
}

// createCardID returns one card's id, in the alphabet the application writes them in.
func createCardID(n int) string {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	out := make([]byte, 10)
	for item := range out {
		out[item] = alphabet[n%len(alphabet)]
		n /= len(alphabet)
	}
	return string(out)
}

// loadAnswers writes a run file for each of the days days of review before the
// one holding now, each holding perDay answers over the vault's cards. A day's
// answers begin at noon of that day in the zone the days are counted in, so
// all of them fall inside one day of review.
func loadAnswers(tb testing.TB, s vaulted, now time.Time, cards, days, perDay int) {
	tb.Helper()
	ctx := tb.Context()

	store, err := s.logs.Open(s.vault)
	if err != nil {
		tb.Fatal(err)
	}
	log := flashcards.Log{Stores: s.logs}
	y, m, d := today.GetDate(now).Date()
	card := 0
	for day := range days {
		when := time.Date(y, m, d-days+day, 12, 0, 0, 0, time.Local)
		run, err := log.Open(ctx, s.vault, when)
		if err != nil {
			tb.Fatal(err)
		}
		var lines []byte
		for one := range perDay {
			raw, err := review.Write(review.Answer{
				ID:         fmt.Sprintf("%06d%010d", day, one),
				ReviewCard: review.CardID{Card: createCardID(card % cards), Template: "Say it"},
				At:         when.Add(time.Duration(one) * time.Minute),
				Rating:     review.Rating(one%4 + 1),
				Took:       4 * time.Second,
			})
			if err != nil {
				tb.Fatal(err)
			}
			lines = append(lines, raw...)
			card++
		}
		if err := store.AppendLog(ctx, run.GetName(), lines); err != nil {
			tb.Fatal(err)
		}
	}
}

// The requests a window makes of a vault, over fifty thousand review cards
// answered a hundred and fifty times a day for six months.
//
// The first three are warm: the caches are filled before the clock starts, which
// is the cost of a repeated query within a session. The last is the history
// screen on a build without a counting cache, which is the cost of the first
// query after a launch.
func BenchmarkVault(b *testing.B) {
	l := load(b, 50000, 180, 150)
	ctx := b.Context()
	if _, err := l.owed.Execute(ctx, l.vault); err != nil {
		b.Fatal(err)
	}
	if _, err := l.review.Execute(ctx, l.vault); err != nil {
		b.Fatal(err)
	}

	b.Run("FrontDoor", func(b *testing.B) {
		for b.Loop() {
			if _, err := l.owed.Execute(ctx, l.vault); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Session", func(b *testing.B) {
		for b.Loop() {
			if _, err := l.sat.Execute(ctx, l.vault, flashcards.Scope{}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("History", func(b *testing.B) {
		for b.Loop() {
			if _, err := l.review.Execute(ctx, l.vault); err != nil {
				b.Fatal(err)
			}
		}
	})

	uncounted := flashcards.NewCountReviews(l.logs, l.schedules, today, time.Now)
	b.Run("HistoryUncounted", func(b *testing.B) {
		for b.Loop() {
			if _, err := uncounted.Execute(ctx, l.vault); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// counting is a vault's files with a tally of what was opened, by path.
type counting struct {
	ports.VaultReaders
	reads map[string]int
}

func (c counting) Open(v domain.Vault) (ports.VaultReader, error) {
	reader, err := c.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return countingReader{VaultReader: reader, reads: c.reads}, nil
}

type countingReader struct {
	ports.VaultReader
	reads map[string]int
}

func (c countingReader) Read(ctx context.Context, path string) ([]byte, error) {
	c.reads[path]++
	return c.VaultReader.Read(ctx, path)
}

// tallied is a vault's answers with a tally of the files read out of them.
type tallied struct {
	ports.LogStores
	reads map[string]int
}

func (s tallied) Open(v domain.Vault) (ports.LogStore, error) {
	store, err := s.LogStores.Open(v)
	if err != nil {
		return nil, err
	}
	return talliedStore{LogStore: store, reads: s.reads}, nil
}

type talliedStore struct {
	ports.LogStore
	reads map[string]int
}

func (s talliedStore) ReadLog(ctx context.Context, name string) ([]byte, error) {
	s.reads[name]++
	return s.LogStore.ReadLog(ctx, name)
}

// lookups is a vault's links with a tally of the notes whose links were looked
// up, by path.
type lookups struct {
	ports.LinkQueries
	looks map[string]int
}

func (a lookups) GetLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	a.looks[from]++
	return a.LinkQueries.GetLinks(ctx, vaultID, from)
}

// Counting a vault reads it once: every deck is opened once and queried once
// for the preset that schedules it.
func TestCountingAVaultReadsItsDecksOnce(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 8\nreviews_a_day: 45\n"),
		"decks/One.md":   deckOf("Sanskrit", 2, 0),
		"decks/Two.md":   deckOf("Sanskrit", 2, 100),
		"decks/Three.md": deckOf("", 2, 200),
	})

	reads := map[string]int{}
	standings := flashcards.NewListReviewCards(
		counting{VaultReaders: s.env.Ports.Readers, reads: reads},
		s.notes, s.links,
	)

	looks := map[string]int{}
	presets := flashcards.NewPresets(
		s.env.Ports.Readers, s.env.Ports.Writers,
		lookups{LinkQueries: s.links, looks: looks},
		s.notes, s.scan, review.Day{}, time.Now,
	)

	schedules := flashcards.NewSchedules(s.logs, review.NewFSRS(), today, standings, presets)
	owed := flashcards.NewCountCardsDue(standings, schedules, presets, today, func() time.Time { return saturday })

	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"decks/One.md", "decks/Two.md", "decks/Three.md"} {
		if reads[path] != 1 {
			t.Errorf("%s was read %d times", path, reads[path])
		}
		if looks[path] != 1 {
			t.Errorf("%s was asked for its preset %d times", path, looks[path])
		}
	}
}

// A preset note is opened once however many decks name it.
func TestAPresetIsOpenedOncePerCall(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 8\nreviews_a_day: 45\n"),
		"decks/One.md":   deckOf("Sanskrit", 2, 0),
		"decks/Two.md":   deckOf("Sanskrit", 2, 100),
		"decks/Three.md": deckOf("Sanskrit", 2, 200),
	})

	reads := map[string]int{}
	presets := flashcards.NewPresets(
		counting{VaultReaders: s.env.Ports.Readers, reads: reads},
		s.env.Ports.Writers, s.links, s.notes, s.scan, review.Day{}, time.Now,
	)
	schedules := flashcards.NewSchedules(s.logs, review.NewFSRS(), today, s.standings, presets)
	owed := flashcards.NewCountCardsDue(s.standings, schedules, presets, today, func() time.Time { return saturday })

	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if reads["Sanskrit.md"] != 1 {
		t.Errorf("the preset three decks name was opened %d times", reads["Sanskrit.md"])
	}
}

// The answers are read once at a launch: the due count and the schedules are
// both computed from that one read.
func TestTheAnswerLogIsReadOnce(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"decks/Roots.md": deckOf("", 3, 0),
	})
	answer(t, s.run(t, saturday.AddDate(0, 0, -1)), "card000000", 6*time.Second)

	reads := map[string]int{}
	schedules := flashcards.NewSchedules(
		tallied{LogStores: s.logs, reads: reads},
		review.NewFSRS(), today, s.standings, s.presets,
	)
	owed := flashcards.NewCountCardsDue(s.standings, schedules, s.presets, today, func() time.Time { return saturday })

	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	for name, times := range reads {
		if times != 1 {
			t.Errorf("%s was read %d times", name, times)
		}
	}
	if len(reads) == 0 {
		t.Error("nothing of the log was read")
	}
}

// A curve is a long walk, and a caller that has given up on it is answered with
// what it gave up on.
func TestACurveAnswersTheCallersCancellation(t *testing.T) {
	t.Parallel()
	s := openVault(t, newStudiedNotes(30))
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	ctx, stop := context.WithCancel(t.Context())
	stop()
	_, err := s.curves(noon).Execute(ctx, s.vault, "Sanskrit.md", p)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled curve said %v", err)
	}
}

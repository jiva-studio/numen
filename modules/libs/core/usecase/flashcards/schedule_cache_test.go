package flashcards_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A warm request over a vault of two hundred cards answered over thirty days.
// What is counted below is the work a request does and not its size, so a small
// vault says what a large one says.
const (
	loadCards  = 200
	loadDays   = 30
	loadPerDay = 20
)

// The front door consults the cache and does not rewrite it: the answers are
// replayed once, and every request after that is told what the replay came to.
func TestTheFrontDoorConsultsTheScheduleCache(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	first := l.countLoads(t, func() error { _, err := l.owed.Execute(ctx, l.vault); return err })
	if first.Rewritten != 1 {
		t.Errorf("the first request wrote the cache %d times, want 1", first.Rewritten)
	}

	warm := l.countLoads(t, func() error { _, err := l.owed.Execute(ctx, l.vault); return err })
	if warm.Consulted == 0 {
		t.Error("a warm request never consulted the cache")
	}
	if warm.Rewritten != 0 {
		t.Errorf("a warm request rewrote the cache %d times", warm.Rewritten)
	}
}

// Starting a session is answered from the cache like the front door, so a
// person starting one waits for a listing and not for the whole history.
func TestStartingASessionConsultsTheScheduleCache(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	if _, err := l.owed.Execute(ctx, l.vault); err != nil {
		t.Fatal(err)
	}
	warm := l.countLoads(t, func() error {
		_, err := l.sat.Execute(ctx, l.vault, flashcards.Scope{})
		return err
	})
	if warm.Consulted == 0 {
		t.Error("starting a session never consulted the cache")
	}
	if warm.Rewritten != 0 {
		t.Errorf("starting a session rewrote the cache %d times", warm.Rewritten)
	}
}

// Opening the history consults the cache and leaves it as it stands.
func TestTheHistoryConsultsTheScheduleCache(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	if _, err := l.owed.Execute(ctx, l.vault); err != nil {
		t.Fatal(err)
	}
	warm := l.countLoads(t, func() error { _, err := l.review.Execute(ctx, l.vault); return err })
	// The day counts are kept in a cache of their own, and the first request
	// over this vault fills it.
	if warm.Rewritten > 1 {
		t.Errorf("opening the history wrote a cache %d times", warm.Rewritten)
	}
}

// A vault answered the same way is left in the same place whether the cache
// answered or the answers were replayed.
func TestTheCacheAnswersWhatAReplayAnswers(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	replayed, err := l.owed.Schedules.Execute(ctx, l.vault)
	if err != nil {
		t.Fatal(err)
	}
	remembered, err := l.owed.Schedules.Execute(ctx, l.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed) == 0 {
		t.Fatal("the answers left no card face anywhere")
	}
	if len(replayed) != len(remembered) {
		t.Fatalf("the replay left %d card faces and the cache %d",
			len(replayed), len(remembered))
	}
	for face, was := range replayed {
		if now := remembered[face]; !now.Due.Equal(was.Due) || now.Reps != was.Reps {
			t.Errorf("%+v stands at %+v, and the replay left it at %+v", face, now, was)
		}
	}
}

package flashcards

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

func deckOf(n int) deckCards { return deckCards{reviewCards: make([]ReviewCard, n)} }

func keyOf(path string) deckKey { return deckKey{vault: "v", path: path} }

func TestTheLeastRecentlyUsedDeckIsForgottenAloneWhenTheCardsPassTheBound(t *testing.T) {
	t.Parallel()
	c := newCardCache()
	c.put(0, keyOf("a"), deckOf(2), 6)
	c.put(0, keyOf("b"), deckOf(2), 6)
	c.put(0, keyOf("c"), deckOf(2), 6)
	c.get(keyOf("a"))

	c.put(0, keyOf("d"), deckOf(2), 6)

	if c.hasDeck(keyOf("b")) {
		t.Error("the least recently used deck is still held")
	}
	for _, reviewCard := range []string{"a", "c", "d"} {
		if !c.hasDeck(keyOf(reviewCard)) {
			t.Errorf("deck %s was forgotten with it", reviewCard)
		}
	}
	if c.getSize() != 6 {
		t.Errorf("%d review cards held, want 6", c.getSize())
	}
}

func TestTheCardsHeldNeverPassTheBound(t *testing.T) {
	t.Parallel()
	c := newCardCache()
	for i := range 50 {
		c.put(0, keyOf(string(rune('a'+i%26))+string(rune('a'+i/26))), deckOf(1+i%4), 10)
		if c.getSize() > 10 {
			t.Fatalf("%d review cards held past a bound of 10", c.getSize())
		}
	}
}

func TestADeckContainingMoreCardsThanTheBoundIsNotStored(t *testing.T) {
	t.Parallel()
	c := newCardCache()
	c.put(0, keyOf("small"), deckOf(1), 3)
	c.put(0, keyOf("big"), deckOf(4), 3)

	if c.hasDeck(keyOf("big")) || !c.hasDeck(keyOf("small")) {
		t.Error("an oversize deck emptied the cache or was stored")
	}
}

func TestReplacingADeckCountsItsCardsOnce(t *testing.T) {
	t.Parallel()
	c := newCardCache()
	c.put(0, keyOf("a"), deckOf(3), 10)
	c.put(0, keyOf("a"), deckOf(2), 10)

	if c.getSize() != 2 {
		t.Errorf("%d review cards held, want 2", c.getSize())
	}
	c.invalidate(domain.VaultID("v"), []string{"a"})
	if c.getSize() != 0 {
		t.Errorf("%d review cards held after forgetting the only deck", c.getSize())
	}
}

// How many places of a curve run at once is what the build was told, and a
// build told nothing runs them one at a time. Reading the machine's cores here
// instead would answer whatever the machine happens to have.
func TestHowManyPlacesRunAtOnce(t *testing.T) {
	t.Parallel()
	for _, cores := range []int{0, 1, 3} {
		want := max(1, cores)
		// A place holds its ground until as many as are admitted stand beside
		// it, so what stood there at once is what the room allowed.
		let := make(chan struct{})
		var letting sync.Once
		var mu sync.Mutex
		running, most := 0, 0

		err := (ProjectCurve{Cores: cores}).computePlaces(2*Points, func(int) error {
			mu.Lock()
			running++
			if running > most {
				most = running
			}
			full := running == want
			mu.Unlock()
			if full {
				letting.Do(func() { close(let) })
			}
			<-let
			mu.Lock()
			running--
			mu.Unlock()
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if most != want {
			t.Errorf("told %d cores, %d places ran at once, want %d", cores, most, want)
		}
	}
}

// answering is a store that returns what it was given for every name.
type answering struct {
	ports.LogStore
	why error
}

func (a answering) ReadLog(context.Context, string) ([]byte, error) {
	return nil, a.why
}

// A run file nobody may open is one run of the log that could not be acted on.
// The answers in every other file are still the person's, so a schedule is
// worked out from them and says how much it did not see.
func TestARunNobodyMayOpenIsOneSkippedRun(t *testing.T) {
	store := answering{why: &os.PathError{Op: "open", Path: "run.jsonl", Err: fs.ErrPermission}}

	ran, err := Log{}.ReadFile(t.Context(), store, ports.Entry{Run: "run"})
	if err != nil {
		t.Fatalf("a file nobody may open failed the whole history: %v", err)
	}
	if !ran.IsUnreadable || ran.Skipped != 1 || len(ran.Answers) != 0 {
		t.Errorf("the run came back as %+v", ran)
	}
}

// A file another machine's synchroniser took away between the listing and the
// reading is no run at all, and nothing was skipped by it.
func TestARunTakenAwayBeforeItWasReadIsNoRun(t *testing.T) {
	store := answering{why: &os.PathError{Op: "open", Path: "run.jsonl", Err: fs.ErrNotExist}}

	ran, err := Log{}.ReadFile(t.Context(), store, ports.Entry{Run: "run"})
	if err != nil {
		t.Fatalf("a file that was gone failed the whole history: %v", err)
	}
	if !ran.IsGone || ran.Skipped != 0 {
		t.Errorf("the run came back as %+v", ran)
	}
}

// Anything else the store says is trouble the caller is told about.
func TestARunThatWouldNotBeReadForAnyOtherReasonIsTrouble(t *testing.T) {
	store := answering{why: fmt.Errorf("the disk is not there")}

	if _, err := (Log{}).ReadFile(t.Context(), store, ports.Entry{Run: "run"}); err == nil {
		t.Error("a store that could not answer was taken as a run that was read")
	}
}

// A run file another program holds open is one run of the log that could not be
// acted on. A synchroniser and a backup reader each hold a file for moments at a
// time, and the store says so in one word whatever the system underneath it is.
func TestARunHeldByAnotherProgramIsOneSkippedRun(t *testing.T) {
	store := answering{why: fmt.Errorf("run.jsonl: %w", ports.ErrHeldByAnother)}

	ran, err := Log{}.ReadFile(t.Context(), store, ports.Entry{Run: "run"})
	if err != nil {
		t.Fatalf("a file another program holds failed the whole history: %v", err)
	}
	if !ran.IsUnreadable || ran.Skipped != 1 || len(ran.Answers) != 0 {
		t.Errorf("the run came back as %+v", ran)
	}
}

// deckLoad is one deck of a preset and what it holds: cards that are owed, and
// cards nobody has begun.
type deckLoad struct {
	path          string
	owed, unbegun int
}

// newDeckLoads generates the decks of one preset, each under a path of its own.
func newDeckLoads(t *rapid.T) []deckLoad {
	decks := rapid.IntRange(1, 5).Draw(t, "decks")
	out := make([]deckLoad, decks)
	for item := range out {
		out[item] = deckLoad{
			path:    fmt.Sprintf("decks/%c.md", 'A'+item),
			owed:    rapid.IntRange(0, 8).Draw(t, "owed"),
			unbegun: rapid.IntRange(0, 8).Draw(t, "fresh"),
		}
	}
	return out
}

// getTakenCards is what a day of this budget hands over, with the decks reached
// in this order: the review cards the session takes, by name.
//
// The order stands for the order the vault was walked in. Each deck's own cards
// keep the order they stand in, which is the deck's and not the walk's. A
// budget of no minutes is a day that does not close on them.
func getTakenCards(load []deckLoad, order []int, keeps review.Budget) map[review.ReviewCardID]bool {
	limits := review.Limits{New: review.ClosedNew, Reviews: review.ClosedReviews}
	if keeps.Minutes > 0 {
		limits.Minutes = review.ClosedMinutes
	}
	day := &budgets{
		reviewCardPresets: make(map[review.ReviewCardID]string),
		allowances: map[string]*allowance{"Preset.md": {
			admits: review.Allowance{
				Keeps: keeps, New: keeps.New, Reviews: keeps.Reviews,
				Minutes: time.Duration(keeps.Minutes * float64(time.Minute)),
				Limits:  limits,
				Backlog: review.AllBacklog,
			},
			cost:   review.DefaultCost,
			counts: review.BudgetUnitCards,
		}},
		decks:         make(map[string]string),
		reviewedCards: make(map[review.ReviewCardID]bool),
		spentByDeck:   make(map[string]review.Spent),
	}

	var owed, newCards []ReviewCard
	for _, item := range order {
		one := load[item]
		day.decks[one.path] = "Preset.md"
		for card := range one.owed {
			reviewCard := ReviewCard{Deck: one.path, ID: review.ReviewCardID{
				Card: fmt.Sprintf("%s/owed/%d", one.path, card),
			}}
			day.reviewCardPresets[reviewCard.ID] = "Preset.md"
			owed = append(owed, reviewCard)
		}
		for card := range one.unbegun {
			reviewCard := ReviewCard{Deck: one.path, ID: review.ReviewCardID{
				Card: fmt.Sprintf("%s/fresh/%d", one.path, card),
			}}
			day.reviewCardPresets[reviewCard.ID] = "Preset.md"
			newCards = append(newCards, reviewCard)
		}
	}

	took := day.spend(owed, newCards)
	out := make(map[review.ReviewCardID]bool)
	for entry, one := range owed {
		if took.owed[entry] {
			out[one.ID] = true
		}
	}
	for elem, one := range newCards {
		if took.newCards[elem] {
			out[one.ID] = true
		}
	}
	return out
}

// countHandedOver is how many cards each deck was handed of a day of this
// budget.
func countHandedOver(load []deckLoad, keeps review.Budget) map[string]int {
	places := make([]int, len(load))
	for item := range places {
		places[item] = item
	}
	out := make(map[string]int, len(load))
	for reviewCard := range getTakenCards(load, places, keeps) {
		for _, one := range load {
			if strings.HasPrefix(reviewCard.Card, one.path+"/") {
				out[one.path]++
			}
		}
	}
	return out
}

// A day of one minute over a deck of one unbegun card and a deck of five: a
// card costs twenty seconds, so the shares of the minute — ten seconds and
// fifty — buy two cards in the larger deck and none in the smaller. The twenty
// seconds nobody could spend buy a third card in the larger deck, which is
// where four cards are still standing, and not the one card of the deck whose
// name sorts first.
//
// What no deck can use is offered round again, to the deck still holding most.
func TestWhatNoDeckCouldUseGoesToTheDeckContainingMost(t *testing.T) {
	t.Parallel()
	keeps := review.Budget{New: 6, Minutes: 1}
	one, five := "decks/a.md", "decks/b.md"

	got := countHandedOver([]deckLoad{{path: one, unbegun: 1}, {path: five, unbegun: 5}}, keeps)
	if want := (map[string]int{five: 3}); !maps.Equal(got, want) {
		t.Fatalf("a day of one minute handed over %v, want %v", got, want)
	}
	// The same two decks under each other's names hand over the same cards.
	got = countHandedOver([]deckLoad{{path: five, unbegun: 1}, {path: one, unbegun: 5}}, keeps)
	if want := (map[string]int{one: 3}); !maps.Equal(got, want) {
		t.Fatalf("a day of one minute over the same decks renamed handed over "+
			"%v, want %v", got, want)
	}
}

// A vault hands over the same cards however its files are walked. The decks are
// handed their shares in the order their paths stand, so the order a walk
// reached them in decides nothing.
//
// A vault divides the same day however its files are walked.
func TestTheWalkOfAVaultDoesNotChangeWhatADayPassesOver(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		load := newDeckLoads(t)
		keeps := review.Budget{
			New:     rapid.IntRange(0, 20).Draw(t, "new"),
			Reviews: rapid.IntRange(0, 20).Draw(t, "reviews"),
		}
		places := make([]int, len(load))
		for item := range places {
			places[item] = item
		}

		was := getTakenCards(load, places, keeps)
		now := getTakenCards(load, rapid.Permutation(places).Draw(t, "walk"), keeps)
		if !maps.Equal(was, now) {
			t.Fatalf("a day of %d new and %d reviews over %v hands over %v "+
				"walked in order and %v walked otherwise",
				keeps.New, keeps.Reviews, load, was, now)
		}
	})
}

// getSize returns the number of review cards held.
func (c *reviewCardCache) getSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

// hasDeck says whether the deck at key is held.
func (c *reviewCardCache) hasDeck(key deckKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, target := c.decks[key]
	return target
}

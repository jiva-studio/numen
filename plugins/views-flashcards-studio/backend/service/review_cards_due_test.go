package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// The vault list shown to the user is built from the first message, and
// counting a vault reads every deck in it. The vaults therefore arrive with
// their name and location, with no count and no deck information.
func TestTheVaultsStandBeforeAnyOfThemIsCounted(t *testing.T) {
	handler, target := newAPI(t, deck, other)

	stream, err := newFlashcardsClient(t, handler).WatchCardsDue(
		t.Context(), &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stream.Close)
	if !stream.Receive() {
		t.Fatalf("overview stream empty: %v", stream.Err())
	}

	first := stream.Msg()
	if first.Day == "" {
		t.Error("the vaults arrived in no day")
	}
	if first.Counted != nil {
		t.Errorf("a count stands beside the vaults: %+v", first.Counted)
	}
	if len(first.Vaults) != len(target) {
		t.Fatalf("overview returned %d of %d vaults", len(first.Vaults), len(target))
	}
	for item, one := range first.Vaults {
		if one.ID != string(target[item].ID) || one.Name == "" || one.Path == "" {
			t.Errorf("the vault stands as %+v", one)
		}
		if one.ReviewCards != 0 || one.Due != 0 || one.New != 0 ||
			len(one.Decks) != 0 || len(one.Presets) != 0 || one.Unread != "" {
			t.Errorf("%s arrived counted: %+v", one.Name, one)
		}
	}
}

// Each vault's count is a message of its own, so a vault of fifty thousand
// cards holds up nothing but itself.
func TestEachVaultsCountArrivesOnItsOwn(t *testing.T) {
	handler, target := newAPI(t, deck, other)

	// The vaults are read when the window opens them, and what a count evaluates to
	// is what is under test here.
	front(t, handler)

	stream, err := newFlashcardsClient(t, handler).WatchCardsDue(
		t.Context(), &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stream.Close)
	if !stream.Receive() {
		t.Fatalf("overview stream empty: %v", stream.Err())
	}

	counts := make(map[string]int, len(target))
	for stream.Receive() {
		response := stream.Msg()
		if len(response.Vaults) != 0 {
			t.Errorf("the vaults were listed again: %+v", response.Vaults)
		}
		one := response.Counted
		if one.ReviewCards != 1 || one.New != 1 {
			t.Errorf("%s comes to %+v", one.Name, one)
		}
		counts[one.ID]++
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	for _, v := range target {
		if counts[string(v.ID)] != 1 {
			t.Errorf("the vault %s was counted %d times", string(v.ID), counts[string(v.ID)])
		}
	}
}

// The counts the window is handed stand in a review day, and the window weighs
// a preset's goal against that day. A day of review begins at the hour the
// settings name, so between midnight and that hour the calendar has turned and
// the day has not.
func TestTheDayTheCountsStandInIsTheReviewDay(t *testing.T) {
	handler, _ := newAPI(t, deck)
	handler.scenarios().day = review.Day{Starts: 4 * time.Hour, In: time.UTC}

	for name, c := range map[string]struct {
		currentTime time.Time
		want        string
	}{
		"an hour past midnight": {
			time.Date(2026, 9, 5, 1, 0, 0, 0, time.UTC), "2026-09-04",
		},
		"the hour the day begins at": {
			time.Date(2026, 9, 5, 4, 0, 0, 0, time.UTC), "2026-09-05",
		},
		"the middle of the day": {
			time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC), "2026-09-05",
		},
	} {
		t.Run(name, func(t *testing.T) {
			handler.now = func() time.Time { return c.currentTime }
			if got := front(t, handler).Day; got != c.want {
				t.Errorf("at %v the counts stand in %q, want %q", c.currentTime, got, c.want)
			}
		})
	}
}

// learnedDecks is a vault of two decks: one on a preset that counts a card
// learned while it is likely to be recalled, and one naming no preset, which
// counts by the defaults and their three weeks.
var learnedDecks = map[string]string{
	"Word.md": "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"Recall.md": "---\ntype: preset\ngoal: retention\nretention: 0.9\n" +
		"learned: retention\n---\n\n# Recall\n",
	"decks/Near.md": newCardModelDeck("Word", "Recall", 2, 5000),
	"decks/Far.md":  newCardModelDeck("Word", "", 2, 5100),
}

// The count includes how much of each deck is learned, so the deck screen
// displays the core's result and applies no rule of its own.
func TestTheCountCarriesHowMuchOfADeckExistsLearned(t *testing.T) {
	handler, target := newAPI(t, learnedDecks)
	v := target[0]
	setNow(handler, firstMorning)

	// One card of each deck answered, so the two decks differ by their rule
	// alone and not by what was done to them.
	session := startSession(t, handler, v)
	reply := map[string]bool{}
	for _, one := range session.Cards {
		if reply[one.Deck] {
			continue
		}
		reply[one.Deck] = true
		if _, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
			Vault: string(v.ID), Run: session.Run,
			CardID: one.CardID, Template: one.Template,
			Rating: api.RatingGood, DurationMs: 5000,
		}); err != nil {
			t.Fatal(err)
		}
	}
	setNow(handler, firstMorning.Add(time.Minute))

	want := map[string]int32{"decks/Near.md": 1, "decks/Far.md": 0}
	for _, one := range getVaultCount(t, handler, v).Decks {
		if one.ReviewCards != 2 {
			t.Errorf("%s has %d review cards, want 2", one.Deck, one.ReviewCards)
		}
		if one.Learned != want[one.Deck] {
			t.Errorf("%s has %d learned, want %d",
				one.Deck, one.Learned, want[one.Deck])
		}
	}
}

// screen has a nought to draw.
func TestVaultCount_UnansweredVaultHasZeroLearned(t *testing.T) {
	handler, target := newAPI(t, learnedDecks)
	v := target[0]
	setNow(handler, firstMorning)

	for _, one := range getVaultCount(t, handler, v).Decks {
		if one.Learned != 0 {
			t.Errorf("%s has %d learned before review",
				one.Deck, one.Learned)
		}
	}
}

// The time the user waits when the review window opens on an installation with
// several vaults.
//
// The first two benchmarks measure the window's cost: the time to display the
// vault list, and the time until the last vault count arrives. The third
// measures counting every vault in one response.
func BenchmarkWatchCardsDue(b *testing.B) {
	const vaults, cards, days, perDay = 4, 5000, 60, 100

	target := make([]map[string]string, 0, vaults)
	for range vaults {
		target = append(target, benchDeck(cards))
	}
	handler, all := newAPI(b, target...)
	for _, v := range all {
		benchAnswers(b, handler, v, cards, days, perDay)
	}

	ctx := b.Context()
	client := reviewClient{reviewHandler: handler}

	// The schedule caches are filled before the clock starts, which is what a
	// person's second opening of a day costs.
	for _, v := range all {
		if _, err := handler.scenarios().flash.CardsDue.Execute(ctx, v); err != nil {
			b.Fatal(err)
		}
	}

	// A vault is read when the window opens it, and that is done before the
	// clock starts: what is measured here is what counting one costs.
	for _, v := range all {
		for handler.countVault(ctx, v).IsIndexing {
			time.Sleep(time.Millisecond)
		}
	}

	b.Run("TheListOnScreen", func(b *testing.B) {
		for b.Loop() {
			stream, err := client.WatchCardsDue(ctx, &api.WatchCardsDueRequest{})
			if err != nil {
				b.Fatal(err)
			}
			if !stream.Receive() {
				b.Fatal(stream.Err())
			}
			stream.Close()
		}
	})

	b.Run("EveryCountIn", func(b *testing.B) {
		for b.Loop() {
			stream, err := client.WatchCardsDue(ctx, &api.WatchCardsDueRequest{})
			if err != nil {
				b.Fatal(err)
			}
			// The body is empty because the whole stream is the measurement.
			for stream.Receive() {
			}
			if err := stream.Err(); err != nil {
				b.Fatal(err)
			}
			stream.Close()
		}
	})

	b.Run("OneAfterAnother", func(b *testing.B) {
		for b.Loop() {
			for _, v := range all {
				if one := handler.countVault(ctx, v); one.Unread != "" {
					b.Fatal(one.Unread)
				}
			}
		}
	})
}

// benchDeck is one card model with one review card and cards cards, spread
// over twenty decks like a typical vault.
func benchDeck(cards int) map[string]string {
	const decks = 20
	out := map[string]string{"Term.md": deck["Term.md"]}
	constructed := make([]*strings.Builder, decks)
	for item := range constructed {
		constructed[item] = &strings.Builder{}
		constructed[item].WriteString("---\ntype: deck\n---\n")
	}
	for entry := range cards {
		fmt.Fprintf(constructed[entry%decks],
			"\n## Card %d ^%s\n\n[[Term]]\n\n### Word\n\nword %d\n\n### Meaning\n\nmeaning %d\n",
			entry, benchMark(entry), entry, entry)
	}
	for elem, one := range constructed {
		out[fmt.Sprintf("decks/Deck%02d.md", elem)] = one.String()
	}
	return out
}

// benchMark is one card's id, in the alphabet the application writes them in.
func benchMark(n int) string {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	out := make([]byte, 10)
	for item := range out {
		out[item] = alphabet[n%len(alphabet)]
		n /= len(alphabet)
	}
	return string(out)
}

// benchAnswers writes one run file for each of the last days days, each holding
// perDay answers over the vault's cards.
func benchAnswers(b *testing.B, handler *API, v domain.Vault, cards, days, perDay int) {
	b.Helper()
	ctx := b.Context()

	pos := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	card := 0
	for day := range days {
		when := pos.Add(time.Duration(day) * 24 * time.Hour)
		run, err := handler.scenarios().flash.Log.Open(ctx, v, when)
		if err != nil {
			b.Fatal(err)
		}
		for one := range perDay {
			if err := run.Append(ctx, review.Answer{
				ID:         fmt.Sprintf("%s%06d%010d", v.ID, day, one),
				ReviewCard: review.CardID{Card: benchMark(card % cards), Template: "Say it"},
				At:         when.Add(time.Duration(one) * time.Minute),
				Rating:     review.Rating(one%4 + 1),
				Took:       4 * time.Second,
			}); err != nil {
				b.Fatal(err)
			}
			card++
		}
	}
}

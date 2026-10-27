package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1/flashcardsv1connect"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// The list a person sees is drawn from the first message, and counting a vault
// reads every deck in it. So the vaults stand there by name and by where they
// are, with nothing counted and nothing said about what they hold.
func TestTheVaultsStandBeforeAnyOfThemIsCounted(t *testing.T) {
	api, target := newAPI(t, deck, other)

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("overview stream empty: %v", stream.Err())
	}

	first := stream.Msg()
	if first.GetDay() == "" {
		t.Error("the vaults arrived in no day")
	}
	if first.GetCounted() != nil {
		t.Errorf("a count stands beside the vaults: %+v", first.GetCounted())
	}
	if len(first.GetVaults()) != len(target) {
		t.Fatalf("overview returned %d of %d vaults", len(first.GetVaults()), len(target))
	}
	for item, one := range first.GetVaults() {
		if one.GetId() != string(target[item].ID) || one.GetName() == "" || one.GetPath() == "" {
			t.Errorf("the vault stands as %+v", one)
		}
		if one.GetReviewCards() != 0 || one.GetDue() != 0 || one.GetNew() != 0 ||
			len(one.GetDecks()) != 0 || len(one.GetPresets()) != 0 || one.GetUnread() != "" {
			t.Errorf("%s arrived counted: %+v", one.GetName(), one)
		}
	}
}

// Each vault's count is a message of its own, so a vault of fifty thousand
// cards holds up nothing but itself.
func TestEachVaultsCountArrivesOnItsOwn(t *testing.T) {
	api, target := newAPI(t, deck, other)

	// The vaults are read when the window opens them, and what a count evaluates to
	// is what is under test here.
	front(t, api)

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("overview stream empty: %v", stream.Err())
	}

	counts := make(map[string]int, len(target))
	for stream.Receive() {
		response := stream.Msg()
		if len(response.GetVaults()) != 0 {
			t.Errorf("the vaults were listed again: %+v", response.GetVaults())
		}
		one := response.GetCounted()
		if one.GetReviewCards() != 1 || one.GetNew() != 1 {
			t.Errorf("%s comes to %+v", one.GetName(), one)
		}
		counts[one.GetId()]++
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
	api, _ := newAPI(t, deck)
	api.scenarios().day = review.Day{Starts: 4 * time.Hour, In: time.UTC}

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
			api.now = func() time.Time { return c.currentTime }
			if got := front(t, api).GetDay(); got != c.want {
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

// The count carries how much of each deck stands learned, so the deck screen
// draws the core's own answer and works out no rule of its own.
func TestTheCountCarriesHowMuchOfADeckExistsLearned(t *testing.T) {
	api, target := newAPI(t, learnedDecks)
	v := target[0]
	setNow(api, firstMorning)

	// One card of each deck answered, so the two decks differ by their rule
	// alone and not by what was done to them.
	session := startSession(t, api, v)
	reply := map[string]bool{}
	for _, one := range session.GetCards() {
		if reply[one.GetDeck()] {
			continue
		}
		reply[one.GetDeck()] = true
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&flashcardsv1.AnswerCardRequest{
			Vault: string(v.ID), Run: session.GetRun(),
			CardId: one.GetCardId(), Template: one.GetTemplate(),
			Rating: flashcardsv1.Rating_RATING_GOOD, TookMs: 5000,
		})); err != nil {
			t.Fatal(err)
		}
	}
	setNow(api, firstMorning.Add(time.Minute))

	want := map[string]int32{"decks/Near.md": 1, "decks/Far.md": 0}
	for _, one := range getVaultCount(t, api, v).GetDecks() {
		if one.GetReviewCards() != 2 {
			t.Errorf("%s has %d review cards, want 2", one.GetDeck(), one.GetReviewCards())
		}
		if one.GetLearned() != want[one.GetDeck()] {
			t.Errorf("%s has %d learned, want %d",
				one.GetDeck(), one.GetLearned(), want[one.GetDeck()])
		}
	}
}

// screen has a nought to draw.
func TestAVaultNobodyHasAnsweredCarriesNothingLearned(t *testing.T) {
	api, target := newAPI(t, learnedDecks)
	v := target[0]
	setNow(api, firstMorning)

	for _, one := range getVaultCount(t, api, v).GetDecks() {
		if one.GetLearned() != 0 {
			t.Errorf("%s has %d learned before review",
				one.GetDeck(), one.GetLearned())
		}
	}
}

// What a person waits for when the review window opens on an installation of
// several vaults.
//
// The first two are what the window costs: the list of vaults on screen, and
// the last of their counts landing. The third counts every vault inside one
// answer.
func BenchmarkFrontDoor(b *testing.B) {
	const vaults, cards, days, perDay = 4, 5000, 60, 100

	target := make([]map[string]string, 0, vaults)
	for range vaults {
		target = append(target, benchDeck(cards))
	}
	api, all := newAPI(b, target...)
	for _, v := range all {
		benchAnswers(b, api, v, cards, days, perDay)
	}

	ctx := b.Context()
	mux := http.NewServeMux()
	mux.Handle(flashcardsv1connect.NewFlashcardsServiceHandler(api))
	server := httptest.NewServer(mux)
	b.Cleanup(server.Close)
	client := flashcardsv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)

	// The schedule caches are filled before the clock starts, which is what a
	// person's second opening of a day costs.
	for _, v := range all {
		if _, err := api.scenarios().flash.CardsDue.Execute(ctx, v); err != nil {
			b.Fatal(err)
		}
	}

	// A vault is read when the window opens it, and that is done before the
	// clock starts: what is measured here is what counting one costs.
	for _, v := range all {
		for api.countVault(ctx, v).GetIsIndexing() {
			time.Sleep(time.Millisecond)
		}
	}

	b.Run("TheListOnScreen", func(b *testing.B) {
		for b.Loop() {
			stream, err := client.WatchCardsDue(ctx, connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
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
			stream, err := client.WatchCardsDue(ctx, connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
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
				if one := api.countVault(ctx, v); one.GetUnread() != "" {
					b.Fatal(one.GetUnread())
				}
			}
		}
	})
}

// benchDeck is one card model of one review card and cards cards, spread over twenty
// decks the way a person's own vault spreads them.
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
func benchAnswers(b *testing.B, api *API, v domain.Vault, cards, days, perDay int) {
	b.Helper()
	ctx := b.Context()

	pos := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	card := 0
	for day := range days {
		when := pos.Add(time.Duration(day) * 24 * time.Hour)
		run, err := api.scenarios().flash.Log.Open(ctx, v, when)
		if err != nil {
			b.Fatal(err)
		}
		for one := range perDay {
			if err := run.Append(ctx, review.Answer{
				ID:         fmt.Sprintf("%s%06d%010d", v.ID, day, one),
				ReviewCard: review.ReviewCardID{Card: benchMark(card % cards), Template: "Say it"},
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

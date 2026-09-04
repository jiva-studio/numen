package flashcardsui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// What a person waits for when the review window opens on an installation of
// several vaults.
//
// The first two are what the window costs: the list of vaults on screen, and
// the last of their counts landing. The third counts every vault inside one
// answer.
func BenchmarkFrontDoor(b *testing.B) {
	const vaults, cards, days, perDay = 4, 5000, 60, 100

	held := make([]map[string]string, 0, vaults)
	for range vaults {
		held = append(held, benchDeck(cards))
	}
	api, all := windowed(b, held...)
	for _, v := range all {
		benchAnswers(b, api, v, cards, days, perDay)
	}

	ctx := b.Context()
	server := httptest.NewServer(api.Serving(http.NotFoundHandler()))
	b.Cleanup(server.Close)
	client := numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)

	// The schedule caches are filled before the clock starts, which is what a
	// person's second opening of a day costs.
	for _, v := range all {
		if _, err := api.CardsDue.Execute(ctx, v); err != nil {
			b.Fatal(err)
		}
	}

	// A vault is read when the window opens it, and that is done before the
	// clock starts: what is measured here is what counting one costs.
	for _, v := range all {
		for api.counted(ctx, v).GetReading() {
			time.Sleep(time.Millisecond)
		}
	}

	b.Run("TheListOnScreen", func(b *testing.B) {
		for b.Loop() {
			stream, err := client.WatchCardsDue(ctx, connect.NewRequest(&v1.WatchCardsDueRequest{}))
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
			stream, err := client.WatchCardsDue(ctx, connect.NewRequest(&v1.WatchCardsDueRequest{}))
			if err != nil {
				b.Fatal(err)
			}
			for stream.Receive() { //nolint:revive // the whole stream is the measurement
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
				if one := api.counted(ctx, v); one.GetUnread() != "" {
					b.Fatal(one.GetUnread())
				}
			}
		}
	})
}

// benchDeck is one stencil of one face and cards cards, spread over twenty
// decks the way a person's own vault spreads them.
func benchDeck(cards int) map[string]string {
	const decks = 20
	out := map[string]string{"Term.md": deck["Term.md"]}
	built := make([]*strings.Builder, decks)
	for at := range built {
		built[at] = &strings.Builder{}
		built[at].WriteString("---\ntype: deck\n---\n")
	}
	for at := range cards {
		fmt.Fprintf(built[at%decks],
			"\n## Card %d ^%s\n\n[[Term]]\n\n### Word\n\nword %d\n\n### Meaning\n\nmeaning %d\n",
			at, benchMark(at), at, at)
	}
	for at, one := range built {
		out[fmt.Sprintf("decks/Deck%02d.md", at)] = one.String()
	}
	return out
}

// benchMark is one card's mark, in the alphabet the application writes them in.
func benchMark(n int) string {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	out := make([]byte, 10)
	for at := range out {
		out[at] = alphabet[n%len(alphabet)]
		n /= len(alphabet)
	}
	return string(out)
}

// benchAnswers writes one run file for each of the last days days, each holding
// perDay answers over the vault's cards.
func benchAnswers(b *testing.B, api *API, v domain.Vault, cards, days, perDay int) {
	b.Helper()
	ctx := b.Context()

	at := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	card := 0
	for day := range days {
		when := at.Add(time.Duration(day) * 24 * time.Hour)
		run, err := api.Log.Open(ctx, v, when)
		if err != nil {
			b.Fatal(err)
		}
		for one := range perDay {
			if err := run.Append(ctx, review.Answer{
				ID:       fmt.Sprintf("%s%06d%010d", v.ID, day, one),
				CardFace: review.CardFaceID{Card: benchMark(card % cards), Face: "Say it"},
				At:       when.Add(time.Duration(one) * time.Minute),
				Rating:   review.Rating(one%4 + 1),
				Took:     4 * time.Second,
			}); err != nil {
				b.Fatal(err)
			}
			card++
		}
	}
}

package flashcards_test

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// loadCounts is what one request asked of the store, of the cache and of the
// scheduler. Each of these is work a request is meant to do once, so each is
// counted.
type loadCounts struct {
	// Listed is how many times the log folder was listed, and Opened how many
	// run files were read.
	Listed int
	Opened int
	// Consulted is how many times the cache was read, and Rewritten how many
	// times it was written.
	Consulted int
	Rewritten int
	// Dated is how many times the scheduler was asked where an answer leaves a
	// card.
	Dated int
}

// countingStores counts what a use case asks of one vault's store.
type countingStores struct {
	inner port.DerivedStores
	on    *loadCounts
}

func (s countingStores) Open(v domain.Vault) (port.DerivedStore, error) {
	one, err := s.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return countingStore{inner: one, on: s.on}, nil
}

type countingStore struct {
	inner port.DerivedStore
	on    *loadCounts
}

func (s countingStore) Read(ctx context.Context, name string) ([]byte, error) {
	s.on.Opened++
	return s.inner.Read(ctx, name)
}

func (s countingStore) List(ctx context.Context, name string) ([]port.Entry, error) {
	s.on.Listed++
	return s.inner.List(ctx, name)
}

func (s countingStore) Open(ctx context.Context, name string) (io.ReadSeekCloser, int64, error) {
	s.on.Opened++
	return s.inner.Open(ctx, name)
}

func (s countingStore) Take(ctx context.Context, name string, from io.Reader) (int64, error) {
	return s.inner.Take(ctx, name, from)
}

func (s countingStore) Write(ctx context.Context, name string, content []byte) error {
	return s.inner.Write(ctx, name, content)
}

func (s countingStore) Append(ctx context.Context, name string, content []byte) error {
	return s.inner.Append(ctx, name, content)
}

func (s countingStore) Remove(ctx context.Context, name string) error {
	return s.inner.Remove(ctx, name)
}

func (s countingStore) Claim(ctx context.Context, name string) (func() error, error) {
	return s.inner.Claim(ctx, name)
}

// countingKept counts what a use case asks of the cache.
type countingKept struct {
	inner port.ScheduleStore
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

// countingBy counts how often the scheduler is asked for a next date.
type countingBy struct {
	inner review.Scheduler
	on    *loadCounts
}

func (b countingBy) Name() string { return b.inner.Name() }

func (b countingBy) Next(s review.Schedule, at time.Time, r review.Rating) review.Schedule {
	b.on.Dated++
	return b.inner.Next(s, at, r)
}

func (b countingBy) Endings(s review.Schedule, at time.Time) (review.Schedule, review.Schedule) {
	b.on.Dated += 2
	return b.inner.Endings(s, at)
}

func (b countingBy) Spaced(s review.Schedule) bool { return b.inner.Spaced(s) }

// loaded is a vault of many cards and many run files, with everything a request
// asks of the store, the cache and the scheduler counted.
type loaded struct {
	vaulted
	on     *loadCounts
	owed   flashcards.CountCardsDue
	sat    flashcards.Session
	review flashcards.CountReviews
	faces  int
}

// load builds a vault of cards cards, answered on days days, perDay answers to
// a day and one run file to a day.
func load(tb testing.TB, cards, days, perDay int) loaded {
	tb.Helper()

	s := openVault(tb, loadDeck(cards))
	on := &loadCounts{}
	logs := countingStores{inner: s.logs, on: on}
	schedules := flashcards.Schedules{
		Logs:      logs,
		Cache:     countingKept{inner: appstate.SchedulesAt(filepath.Join(tb.TempDir(), "faces")), on: on},
		By:        countingBy{inner: review.NewFSRS(), on: on},
		Day:       today,
		CardFaces: s.standings,
		Presets:   s.presets,
	}
	loadAnswers(tb, s, cards, days, perDay)
	return loaded{
		vaulted: s,
		on:      on,
		owed: flashcards.CountCardsDue{
			CardFaces: s.standings, Schedules: schedules, Presets: s.presets,
			Day: today, Now: time.Now,
		},
		sat: flashcards.Session{
			Marks: s.marking, CardFaces: s.standings, Schedules: schedules,
			Presets: s.presets, Day: today, Now: time.Now,
		},
		review: flashcards.CountReviews{
			Logs:      logs,
			Cache:     countingKept{inner: appstate.SchedulesAt(filepath.Join(tb.TempDir(), "days")), on: on},
			Schedules: schedules,
			Day:       today,
			Now:       time.Now,
		},
		faces: cards,
	}
}

// logRead is this vault's answers, read the way a request reads them.
func (l loaded) logRead(tb testing.TB) (flashcards.ReviewLog, error) {
	tb.Helper()
	return flashcards.Log{Stores: l.logs}.Read(tb.Context(), l.vault)
}

// countLoads runs one request with the counters cleared, and says what it asked.
func (l loaded) countLoads(tb testing.TB, run func() error) loadCounts {
	tb.Helper()
	*l.on = loadCounts{}
	if err := run(); err != nil {
		tb.Fatal(err)
	}
	return *l.on
}

// deckOf is a vault of one stencil of one face and cards cards, spread over
// twenty decks the way a person's own vault spreads them.
func loadDeck(cards int) map[string]string {
	const decks = 20
	out := map[string]string{"Term.md": vault["Term.md"]}
	built := make([]*strings.Builder, decks)
	for at := range built {
		built[at] = &strings.Builder{}
		built[at].WriteString("---\ntype: deck\n---\n")
	}
	for at := range cards {
		fmt.Fprintf(built[at%decks],
			"\n## Card %d ^%s\n\n[[Term]]\n\n### Word\n\nword %d\n\n### Meaning\n\nmeaning %d\n",
			at, loadMark(at), at, at)
	}
	for at, one := range built {
		out[fmt.Sprintf("decks/Deck%02d.md", at)] = one.String()
	}
	return out
}

// marked is one card's mark, in the alphabet the application writes them in.
func loadMark(n int) string {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	out := make([]byte, 10)
	for at := range out {
		out[at] = alphabet[n%len(alphabet)]
		n /= len(alphabet)
	}
	return string(out)
}

// answered writes a run file for each of the last days days, each holding
// perDay answers over the vault's cards.
func loadAnswers(tb testing.TB, s vaulted, cards, days, perDay int) {
	tb.Helper()
	ctx := tb.Context()

	store, err := s.logs.Open(s.vault)
	if err != nil {
		tb.Fatal(err)
	}
	log := flashcards.Log{Stores: s.logs}
	at := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	card := 0
	for day := range days {
		when := at.Add(time.Duration(day) * 24 * time.Hour)
		run, err := log.Open(ctx, s.vault, when)
		if err != nil {
			tb.Fatal(err)
		}
		var lines []byte
		for one := range perDay {
			raw, err := review.Write(review.Answer{
				ID:       fmt.Sprintf("%06d%010d", day, one),
				CardFace: review.CardFaceID{Card: loadMark(card % cards), Face: "Say it"},
				At:       when.Add(time.Duration(one) * time.Minute),
				Rating:   review.Rating(one%4 + 1),
				Took:     4 * time.Second,
			})
			if err != nil {
				tb.Fatal(err)
			}
			lines = append(lines, raw...)
			card++
		}
		if err := store.Write(ctx, run.Name(), lines); err != nil {
			tb.Fatal(err)
		}
	}
}

package flashcards_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
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
	inner ports.DerivedStores
	on    *loadCounts
}

func (s countingStores) Open(v domain.Vault) (ports.DerivedStore, error) {
	one, err := s.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return countingStore{inner: one, on: s.on}, nil
}

type countingStore struct {
	inner ports.DerivedStore
	on    *loadCounts
}

func (s countingStore) ReadArtifact(ctx context.Context, name domain.ArtifactName) ([]byte, error) {
	s.on.Opened++
	return s.inner.ReadArtifact(ctx, name)
}

func (s countingStore) ListReviewLogs(ctx context.Context) ([]ports.Entry, error) {
	s.on.Listed++
	return s.inner.ListReviewLogs(ctx)
}

func (s countingStore) AppendArtifact(ctx context.Context, name domain.ArtifactName, content []byte) error {
	return s.inner.AppendArtifact(ctx, name, content)
}

// countingKept counts what a use case asks of the cache.
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

// countingBy counts how often the scheduler is asked for a next date.
type countingBy struct {
	inner review.Scheduler
	on    *loadCounts
}

func (b countingBy) GetName() string { return b.inner.GetName() }

func (b countingBy) Next(s review.Schedule, at time.Time, r review.Rating) review.Schedule {
	b.on.Dated++
	return b.inner.Next(s, at, r)
}

func (b countingBy) Endings(s review.Schedule, at time.Time) (review.Schedule, review.Schedule) {
	b.on.Dated += 2
	return b.inner.Endings(s, at)
}

func (b countingBy) IsSpaced(s review.Schedule) bool { return b.inner.IsSpaced(s) }

// loaded is a vault of many cards and many run files, with everything a request
// asks of the store, the cache and the scheduler counted.
type loaded struct {
	vaulted
	on        *loadCounts
	schedules flashcards.Schedules
	logs      ports.DerivedStores
	owed      flashcards.CountCardsDue
	sat       flashcards.Session
	review    flashcards.CountReviews
	faces     int
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
	schedules.Cache = countingKept{inner: s.env.Ports.Schedules("faces"), on: on}
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
			s.marking, s.standings, schedules,
			s.presets, today, now,
		),
		review: reviewUseCase,
		faces:  cards,
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
		if err := store.AppendArtifact(ctx, domain.NewReviewLogName(run.GetName()), lines); err != nil {
			tb.Fatal(err)
		}
	}
}

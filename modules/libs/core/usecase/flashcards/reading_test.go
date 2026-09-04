package flashcards_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// counting is a vault's files with a tally of what was opened, by path.
type counting struct {
	port.VaultReaders
	reads map[string]int
}

func (c counting) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := c.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return countingReader{VaultReader: reader, reads: c.reads}, nil
}

type countingReader struct {
	port.VaultReader
	reads map[string]int
}

func (c countingReader) Read(ctx context.Context, path string) ([]byte, error) {
	c.reads[path]++
	return c.VaultReader.Read(ctx, path)
}

// tallied is a vault's answers with a tally of the files read out of them.
type tallied struct {
	port.DerivedStores
	reads map[string]int
}

func (s tallied) Open(v domain.Vault) (port.DerivedStore, error) {
	store, err := s.DerivedStores.Open(v)
	if err != nil {
		return nil, err
	}
	return talliedStore{DerivedStore: store, reads: s.reads}, nil
}

type talliedStore struct {
	port.DerivedStore
	reads map[string]int
}

func (s talliedStore) Read(ctx context.Context, name string) ([]byte, error) {
	s.reads[name]++
	return s.DerivedStore.Read(ctx, name)
}

// lookups is a vault's links with a tally of the notes whose links were looked
// up, by path.
type lookups struct {
	port.LinkQueries
	looks map[string]int
}

func (a lookups) Links(ctx context.Context, vaultID, from string) ([]domain.ResolvedLink, error) {
	a.looks[from]++
	return a.LinkQueries.Links(ctx, vaultID, from)
}

// Counting a vault reads it once: every deck is opened once and asked once
// which preset schedules it.
func TestCountingAVaultReadsItsDecksOnce(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 8\nreviews_a_day: 45\n"),
		"decks/One.md":   deckOf("Sanskrit", 2, 0),
		"decks/Two.md":   deckOf("Sanskrit", 2, 100),
		"decks/Three.md": deckOf("", 2, 200),
	})

	reads := map[string]int{}
	standings := s.standings
	standings.Readers = counting{VaultReaders: standings.Readers, reads: reads}

	looks := map[string]int{}
	presets := s.presets
	presets.Links = lookups{LinkQueries: presets.Links, looks: looks}

	owed := s.owedAt(today, func() time.Time { return saturday })
	owed.Standings = standings
	owed.Presets = presets

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
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 8\nreviews_a_day: 45\n"),
		"decks/One.md":   deckOf("Sanskrit", 2, 0),
		"decks/Two.md":   deckOf("Sanskrit", 2, 100),
		"decks/Three.md": deckOf("Sanskrit", 2, 200),
	})

	reads := map[string]int{}
	presets := s.presets
	presets.Readers = counting{VaultReaders: presets.Readers, reads: reads}
	owed := s.owedAt(today, func() time.Time { return saturday })
	owed.Presets = presets

	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if reads["Sanskrit.md"] != 1 {
		t.Errorf("the preset three decks name was opened %d times", reads["Sanskrit.md"])
	}
}

// The answers are read once at a launch: what is owed and the schedules it is
// counted from come out of the one reading.
func TestTheAnswerLogIsReadOnce(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"decks/Roots.md": deckOf("", 3, 0),
	})
	answer(t, s.run(t, saturday.AddDate(0, 0, -1)), "card000000", 6*time.Second)

	reads := map[string]int{}
	kept := s.kept
	kept.Logs = tallied{DerivedStores: kept.Logs, reads: reads}
	owed := s.owedAt(today, func() time.Time { return saturday })
	owed.Schedules = kept

	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	for name, times := range reads {
		if strings.HasSuffix(name, ".jsonl") && times != 1 {
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
	s := opened(t, studied(30))
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	ctx, stop := context.WithCancel(t.Context())
	stop()
	_, err := s.curves(noon).Execute(ctx, s.vault, "Sanskrit.md", p)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled curve said %v", err)
	}
}

package flashcards_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/review"
)

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
	ports.DerivedStores
	reads map[domain.ArtifactName]int
}

func (s tallied) Open(v domain.Vault) (ports.DerivedStore, error) {
	store, err := s.DerivedStores.Open(v)
	if err != nil {
		return nil, err
	}
	return talliedStore{DerivedStore: store, reads: s.reads}, nil
}

type talliedStore struct {
	ports.DerivedStore
	reads map[domain.ArtifactName]int
}

func (s talliedStore) ReadArtifact(ctx context.Context, name domain.ArtifactName) ([]byte, error) {
	s.reads[name]++
	return s.DerivedStore.ReadArtifact(ctx, name)
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

// Counting a vault reads it once: every deck is opened once and asked once
// which preset schedules it.
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
	standings := flashcards.NewListCardFaces(
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

// The answers are read once at a launch: what is owed and the schedules it is
// counted from come out of the one read.
func TestTheAnswerLogIsReadOnce(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"decks/Roots.md": deckOf("", 3, 0),
	})
	answer(t, s.run(t, saturday.AddDate(0, 0, -1)), "card000000", 6*time.Second)

	reads := map[domain.ArtifactName]int{}
	kept := flashcards.NewSchedules(
		tallied{DerivedStores: s.logs, reads: reads},
		review.NewFSRS(), today, s.standings, s.presets,
	)
	owed := flashcards.NewCountCardsDue(s.standings, kept, s.presets, today, func() time.Time { return saturday })

	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	for name, times := range reads {
		if name.File == domain.ArtifactReviewLog && times != 1 {
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

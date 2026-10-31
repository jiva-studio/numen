package flashcards_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// A vault the index has never read is not a vault holding no cards. The
func TestAVaultTheIndexHasNotReadIsNotAVaultOfNoCards(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	unread := domain.Vault{ID: "nobody-scanned-this", Name: "Unread", Path: s.vault.Path}

	_, err := s.standings.Execute(t.Context(), unread)
	if !errors.Is(err, flashcards.ErrNotCarried) {
		t.Errorf("a vault the index has not read came back with %v", err)
	}
}

// A deck that cannot be read does not remove the cards of the other decks.
// The read failure is reported in the editor.
func TestADeckThatCannotBeReadPreservesTheOthersActive(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	whole, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(whole) == 0 {
		t.Fatal("the vault stands for nothing")
	}

	// The index still lists the deck; the file behind it is gone, which is what
	if err := os.Remove(filepath.Join(s.vault.Path, "decks", "Words.md")); err != nil {
		t.Fatal(err)
	}

	remaining, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("one deck gone failed the whole vault: %v", err)
	}
	if len(remaining) == 0 {
		t.Error("one deck gone left the vault standing for nothing")
	}
	if len(remaining) >= len(whole) {
		t.Errorf("the gone deck still stands: %d review cards of %d", len(remaining), len(whole))
	}
	for _, one := range remaining {
		if one.Deck == "decks/Words.md" {
			t.Errorf("a card of the gone deck stands: %+v", one)
		}
	}
}

// readTally returns the vault's readers and counts how many times each file is
// read.
type readTally struct {
	inner ports.VaultReaders
	mu    sync.Mutex
	reads map[string]int
}

func (c *readTally) Open(v domain.Vault) (ports.VaultReader, error) {
	one, err := c.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return deckReadCounter{VaultReader: one, of: c}, nil
}

func (c *readTally) take() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.reads
	c.reads = nil
	return out
}

type deckReadCounter struct {
	ports.VaultReader
	of *readTally
}

func (r deckReadCounter) Read(ctx context.Context, path string) ([]byte, error) {
	r.of.mu.Lock()
	if r.of.reads == nil {
		r.of.reads = make(map[string]int)
	}
	r.of.reads[path]++
	r.of.mu.Unlock()
	return r.VaultReader.Read(ctx, path)
}

func openTallied(t *testing.T, notes map[string]string) (vaulted, flashcards.ListReviewCards, *readTally) {
	t.Helper()
	s := openVault(t, notes)
	reads := &readTally{inner: s.env.Ports.Readers}
	reviewCards := flashcards.NewListReviewCards(reads, s.notes, s.links)
	reviewCards.Cores = 4
	if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	reads.take()
	return s, reviewCards, reads
}

func ids(reviewCards []flashcards.ReviewCard) []string {
	out := make([]string, 0, len(reviewCards))
	for _, f := range reviewCards {
		out = append(out, f.Deck+"|"+f.ID.Card+"|"+f.ID.Template)
	}
	return out
}

func TestASecondRequestDoesNotReadADeckThatHasNotChanged(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)

	again, err := reviewCards.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := reads.take(); len(got) != 0 {
		t.Errorf("an unchanged vault was read again: %v", got)
	}
	if len(again) != 5 {
		t.Errorf("the second request stands for %d review cards, want 5", len(again))
	}
	for _, one := range again {
		front, _ := one.Lay()
		if front == "" {
			t.Errorf("a cached review card lays out nothing: %+v", one.ID)
		}
	}
}

func TestADeckWhoseSizeChangedIsReadAgain(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)

	write(t, s, "decks/Words.md", vault["decks/Words.md"]+"\n")
	if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Words.md"] != 1 {
		t.Errorf("the changed deck was read %d times: %v", got["decks/Words.md"], got)
	}
	if got["decks/Mammals.md"] != 0 {
		t.Errorf("the unchanged deck was read: %v", got)
	}
}

func TestADeckWhoseModificationTimeChangedIsReadAgain(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)

	pos := filepath.Join(s.vault.Path, "decks", "Words.md")
	later := vaulttest.Epoch.Add(time.Hour)
	if err := os.Chtimes(pos, later, later); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if got := reads.take(); got["decks/Words.md"] != 1 {
		t.Errorf("a deck touched in place was read %d times: %v", got["decks/Words.md"], got)
	}
}

func TestADeckThatIsDeletedPreservesNoEntryPrior(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)
	pos := filepath.Join(s.vault.Path, "decks", "Words.md")
	info, err := os.Stat(pos)
	if err != nil {
		t.Fatal(err)
	}

	remove(t, s, "decks/Words.md")
	remaining, err := reviewCards.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range remaining {
		if one.Deck == "decks/Words.md" {
			t.Fatalf("a deleted deck stands: %+v", one.ID)
		}
	}
	reads.take()

	// The same bytes under the same stamp are the file the entry was made from.
	write(t, s, "decks/Words.md", vault["decks/Words.md"])
	if err := os.Chtimes(pos, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	back, err := reviewCards.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := reads.take(); got["decks/Words.md"] != 1 {
		t.Errorf("a deck that came back was read %d times: %v", got["decks/Words.md"], got)
	}
	if len(back) != 5 {
		t.Errorf("the vault stands for %d review cards, want 5", len(back))
	}
}

func TestAChangedCardModelIsReadAgainWithEveryDeckSplitByIt(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)

	write(t, s, "Animal.md", vault["Animal.md"]+"\n")
	if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Mammals.md"] != 1 {
		t.Errorf("the deck cut by the changed card model was read %d times: %v", got["decks/Mammals.md"], got)
	}
	if got["decks/Words.md"] != 0 {
		t.Errorf("a deck cut by another card model was read: %v", got)
	}
}

func TestInvalidatingACardModelForgetsEveryDeckSplitByIt(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)

	reviewCards.Invalidate(s.vault.ID, "Animal.md")
	if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Mammals.md"] != 1 || got["decks/Words.md"] != 0 {
		t.Errorf("reads after forgetting a card model: %v", got)
	}
}

func TestInvalidatingADeckForgetsIt(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)

	reviewCards.Invalidate(s.vault.ID, "decks/Words.md")
	if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Words.md"] != 1 || got["decks/Mammals.md"] != 0 {
		t.Errorf("reads after forgetting a deck: %v", got)
	}
}

func TestInvalidatingAllForgetsEveryDeck(t *testing.T) {
	t.Parallel()
	s, reviewCards, reads := openTallied(t, vault)

	reviewCards.InvalidateAll()
	if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Words.md"] != 1 || got["decks/Mammals.md"] != 1 {
		t.Errorf("reads after forgetting everything: %v", got)
	}
}

func TestAMarkSavedThroughTheAppIsActiveInTheNextRequest(t *testing.T) {
	t.Parallel()
	notes := map[string]string{
		"Term.md": vault["Term.md"],
		"decks/Words.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nCompost\n",
	}
	s := openVault(t, notes)
	reviewCards := flashcards.NewListReviewCards(s.env.Ports.Readers, s.notes, s.links)
	assigning := s.assigning.WithIndex(func(ctx context.Context, v domain.Vault, paths []string) error {
		reviewCards.Invalidate(v.ID, paths...)
		return s.scan(ctx, v, paths)
	})

	before, err := reviewCards.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("a card with no card id stands: %v", ids(before))
	}
	if _, err := assigning.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	after, err := reviewCards.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 {
		t.Errorf("the marked card stands as %d review cards, want 1", len(after))
	}
}

func TestTheCardsComeBackInTheOrderOfThePathsWhateverParsesFirst(t *testing.T) {
	t.Parallel()
	notes := map[string]string{"Term.md": vault["Term.md"]}
	var paths []string
	for i := range 24 {
		path := fmt.Sprintf("decks/Deck%02d.md", i)
		paths = append(paths, path)
		notes[path] = fmt.Sprintf("---\ntype: deck\n---\n\n## Card ^q%02dwxyzpqr\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n", i, i, i)
	}
	s := openVault(t, notes)

	one := flashcards.NewListReviewCards(s.env.Ports.Readers, s.notes, s.links)
	want := ids(one.GetReviewCards(t.Context(), s.vault, paths))
	if len(want) != len(paths) {
		t.Fatalf("%d decks stand as %d review cards", len(paths), len(want))
	}

	for range 10 {
		many := flashcards.NewListReviewCards(s.env.Ports.Readers, s.notes, s.links)
		many.Cores = 64
		if got := ids(many.GetReviewCards(t.Context(), s.vault, paths)); !reflect.DeepEqual(got, want) {
			t.Fatalf("the order moved:\n got %v\nwant %v", got, want)
		}
	}
}

func TestRequestsAndInvalidationsMakeNoRaceBetweenThem(t *testing.T) {
	t.Parallel()
	s, reviewCards, _ := openTallied(t, vault)

	var running sync.WaitGroup
	for range 8 {
		running.Add(2)
		go func() {
			defer running.Done()
			for range 20 {
				got, err := reviewCards.Execute(t.Context(), s.vault)
				if err != nil || len(got) != 5 {
					t.Errorf("a request came back with %d review cards and %v", len(got), err)
					return
				}
			}
		}()
		go func() {
			defer running.Done()
			for range 20 {
				reviewCards.Invalidate(s.vault.ID, "Term.md")
				reviewCards.InvalidateAll()
			}
		}()
	}
	running.Wait()
}

func TestAPoolOfOneAndAPoolOfManyGiveTheSameCardsEveryTime(t *testing.T) {
	t.Parallel()
	notes := map[string]string{"Term.md": vault["Term.md"]}
	var paths []string
	for i := range 16 {
		path := fmt.Sprintf("decks/Deck%02d.md", i)
		paths = append(paths, path)
		notes[path] = fmt.Sprintf("---\ntype: deck\n---\n\n## Card ^q%02dwxyzpqr\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n", i, i, i)
	}
	s := openVault(t, notes)
	one := flashcards.NewListReviewCards(s.env.Ports.Readers, s.notes, s.links)
	one.Cores = 1
	want := ids(one.GetReviewCards(t.Context(), s.vault, paths))

	for _, cores := range []int{2, 3, 16, 64} {
		many := flashcards.NewListReviewCards(s.env.Ports.Readers, s.notes, s.links)
		many.Cores = cores
		for range 3 {
			if got := ids(many.GetReviewCards(t.Context(), s.vault, paths)); !reflect.DeepEqual(got, want) {
				t.Fatalf("a pool of %d gave another order:\n got %v\nwant %v", cores, got, want)
			}
		}
	}
}

func TestADeckPastTheCardBoundIsParsedAgainWhileTheOthersStay(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	reads := &readTally{inner: s.env.Ports.Readers}
	reviewCards := flashcards.NewListReviewCards(reads, s.notes, s.links)
	// Mammals yields four review cards and Words yields one.
	reviewCards.MaxCards = 3
	for range 2 {
		if _, err := reviewCards.Execute(t.Context(), s.vault); err != nil {
			t.Fatal(err)
		}
	}
	got := reads.take()
	if got["decks/Mammals.md"] != 2 || got["decks/Words.md"] != 1 {
		t.Errorf("reads under a bound of three review cards: %v", got)
	}
}

// A vault of three decks: two scheduled by a preset that gives Saturday half
// the load, and one naming no preset at all.
var scheduled = map[string]string{
	"Term.md": "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\nload: {sat: 50}\n---\n\n# Sanskrit\n",
	"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n" +
		"\n## Root ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n",
	"decks/Mantras.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n" +
		"\n## Gayatri ^zpqrstvwxy\n\n[[Term]]\n\n### Word\n\ngayatri\n\n### Meaning\n\na metre\n",
	"decks/Terms.md": "---\ntype: deck\n---\n" +
		"\n## Term ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nsutra\n\n### Meaning\n\na thread\n",
}

// saturday is a day the Sanskrit preset gives half the load, at an hour well
// inside it.
var saturday = time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local)

// What a day came to is counted under the preset each deck names, over as many
// sessions as the day held. A deck naming no preset comes under the defaults.
func TestWhatADayCameToBelowEachPresetOfAVault(t *testing.T) {
	t.Parallel()
	s := openVault(t, scheduled)

	// Two sessions of the one day, each writing a file of its own.
	morning := s.run(t, saturday)
	answer(t, morning, "k7m2xq9fzp", 6*time.Second)
	evening := s.run(t, saturday.Add(9*time.Hour))
	answer(t, evening, "zpqrstvwxy", 9*time.Second)
	answer(t, evening, "3f4g5h6j7k", 4*time.Second)

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(saturday.Add(10*time.Hour))).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := []flashcards.PresetCardsDue{
		{
			Preset: "", Decks: 1, Cards: 1, Due: 1,
			Answered: 1, ReviewedNew: 1, Took: 4 * time.Second,
			Budget: review.Budget{New: 10, Reviews: 200, Minutes: 20},
			Limits: review.Limits{
				Minutes: review.ClosedMinutes, Backlog: review.ClosedBacklog,
			},
		},
		{
			Preset: "Sanskrit.md", Decks: 2, Cards: 2, Due: 2,
			Answered: 2, ReviewedNew: 2, Took: 15 * time.Second,
			Budget: review.Budget{New: 4, Reviews: 23, Minutes: 10},
			Limits: review.Limits{
				Minutes: review.ClosedMinutes, Backlog: review.ClosedBacklog,
			},
		},
	}
	if len(counts.Presets) != len(want) {
		t.Fatalf("the day came to %+v, want %+v", counts.Presets, want)
	}
	for item, one := range want {
		if counts.Presets[item] != one {
			t.Errorf("%q came to %+v, want %+v", one.Preset, counts.Presets[item], one)
		}
	}
}

// A vault holding no preset at all is one scope: the defaults, with every deck
// under them.
func TestAVaultContainingNoPresetExistsOnTheDefaults(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":      term,
		"decks/One.md": deckOf("", 20, 0),
		"decks/Two.md": deckOf("", 20, 100),
	})

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(saturday)).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := flashcards.PresetCardsDue{
		Preset: "", Decks: 2, Cards: 40, New: 40,
		Budget: review.Defaults().GetAllowance(today, saturday, review.Spent{}, 0, 0).Keeps,
		Limits: review.Defaults().GetAllowance(today, saturday, review.Spent{}, 0, 0).Limits,
	}
	if len(counts.Presets) != 1 {
		t.Fatalf("the vault came to %+v, want the defaults alone", counts.Presets)
	}
	if counts.Presets[0] != want {
		t.Errorf("the defaults came to %+v, want %+v", counts.Presets[0], want)
	}
}

// A preset that no deck points at is still listed in the count, with no day
// counts.
func TestAPresetNoDeckPointsAtExistsInTheCount(t *testing.T) {
	t.Parallel()
	files := make(map[string]string, len(scheduled)+1)
	for path, raw := range scheduled {
		files[path] = raw
	}
	files["Empty.md"] = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 137\n---\n\n# Empty\n"
	s := openVault(t, files)

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(saturday)).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	var target *flashcards.PresetCardsDue
	for item, one := range counts.Presets {
		if one.Preset == "Empty.md" {
			target = &counts.Presets[item]
		}
	}
	if target == nil {
		t.Fatalf("the preset nothing points at is not in the count: %+v", counts.Presets)
	}
	if *target != (flashcards.PresetCardsDue{Preset: "Empty.md"}) {
		t.Errorf("it came to %+v, want a preset nothing is under", *target)
	}
}

// What was answered on another day is not what today came to, and neither is an
// answer taken back.
func TestCountCardsDue_CountsOnlyAnswersOfThatDay(t *testing.T) {
	t.Parallel()
	s := openVault(t, scheduled)

	before := s.run(t, saturday.AddDate(0, 0, -1))
	answer(t, before, "k7m2xq9fzp", 6*time.Second)

	session := s.run(t, saturday)
	provided := answer(t, session, "zpqrstvwxy", 9*time.Second)
	if _, err := session.TakeBack(t.Context(), provided); err != nil {
		t.Fatal(err)
	}

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(saturday.Add(time.Hour))).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range counts.Presets {
		if one.Answered != 0 || one.Took != 0 {
			t.Errorf("%q came to %+v, want a day nothing is on", one.Preset, one)
		}
	}
}

// Answers are counted per deck as well as per preset, so a deck without answers
// today does not take the count of another deck on the same preset.
func TestCountCardsDue_CountsAnswersPerDeck(t *testing.T) {
	t.Parallel()
	s := openVault(t, scheduled)

	// One card of one of the two decks the Sanskrit preset schedules.
	answer(t, s.run(t, saturday), "k7m2xq9fzp", 6*time.Second)

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(saturday.Add(time.Hour))).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]int{"decks/Roots.md": 1, "decks/Mantras.md": 0, "decks/Terms.md": 0}
	for _, one := range counts.Decks {
		if one.Answered != want[one.Deck] {
			t.Errorf("%s was answered %d today, want %d", one.Deck, one.Answered, want[one.Deck])
		}
	}
	for _, one := range counts.Presets {
		if one.Preset == "Sanskrit.md" && one.Answered != 1 {
			t.Errorf("the preset of the two decks was answered %d today, want 1", one.Answered)
		}
	}
}

// The count is requested for every vault, and again when the window opens, a
// session ends or a vault changes. A repeated count over unchanged answers
// replays nothing.
func TestASecondCountReadsTheSchedulesOutOfTheCache(t *testing.T) {
	t.Parallel()
	s := openVault(t, scheduled)
	answer(t, s.run(t, saturday), "k7m2xq9fzp", 6*time.Second)

	replayed := 0
	s.schedules.At = func(retention float64) review.Scheduler {
		return replaying{Scheduler: review.NewFSRSAt(retention), answers: &replayed}
	}
	counter := s.newCountCardsDueAt(today, clock.NewFake(saturday.Add(time.Hour)))

	if _, err := counter.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	first := replayed
	if first == 0 {
		t.Fatal("the first count replayed nothing, and there is no cache to have filled")
	}
	if _, err := counter.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if replayed != first {
		t.Errorf("the second count replayed %d answers, want none", replayed-first)
	}
}

// replaying is a scheduler saying how many answers were worked out through it.
type replaying struct {
	review.Scheduler
	answers *int
}

func (r replaying) Next(
	s review.Schedule, pos time.Time, rating review.Rating,
) review.Schedule {
	*r.answers++
	return r.Scheduler.Next(s, pos, rating)
}

// answer writes one correct answer for a card and returns its line. Every card
// of this vault is shown through the one review card.
func answer(t *testing.T, record sitting, card string, took time.Duration) string {
	t.Helper()
	return answerWith(t, record, card, review.Good, took)
}

// answerAgain writes one failed answer for a card, which makes the card due
// again in the same session.
func answerAgain(t *testing.T, record sitting, card string, took time.Duration) string {
	t.Helper()
	return answerWith(t, record, card, review.Again, took)
}

func answerWith(
	t *testing.T, record sitting, card string,
	rating review.Rating, took time.Duration,
) string {
	t.Helper()
	provided, err := record.Answer(
		t.Context(), review.CardID{Card: card, Template: "Say it"}, rating, took)
	if err != nil {
		t.Fatal(err)
	}
	return provided.ID
}

// A preset a deck names is pointed at whatever the deck holds, and how many
// cards stand under it is counted beside that.
func TestAnEmptyDeckStillPointsAtItsPreset(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Empty.md":       preset("new_a_day: 4\nreviews_a_day: 20\n"),
		"Unnamed.md":     preset("new_a_day: 4\nreviews_a_day: 20\n"),
		"decks/Empty.md": deckOf("Empty", 0, 0),
		"decks/Full.md":  deckOf("Empty", 3, 0),
	})

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(saturday)).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string][2]int, len(counts.Presets))
	for _, one := range counts.Presets {
		got[one.Preset] = [2]int{one.Decks, one.Cards}
	}
	want := map[string][2]int{"Empty.md": {2, 3}, "Unnamed.md": {0, 0}}
	for path, one := range want {
		if got[path] != one {
			t.Errorf("%s is named by %d decks holding %d cards, want %d and %d",
				path, got[path][0], got[path][1], one[0], one[1])
		}
	}
}

// A preset nothing but an empty deck names is still named by that deck.
func TestAPresetOnlyAnEmptyDeckNamesIsPointedAt(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Empty.md":       preset("new_a_day: 4\nreviews_a_day: 20\n"),
		"decks/Empty.md": deckOf("Empty", 0, 0),
	})

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(saturday)).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range counts.Presets {
		if one.Preset != "Empty.md" {
			continue
		}
		if one.Decks != 1 || one.Cards != 0 {
			t.Errorf("the preset is named by %d decks holding %d cards, want 1 and 0",
				one.Decks, one.Cards)
		}
		return
	}
	t.Error("the preset was not among what the vault holds")
}

// A count is dropped once the window that requested it is gone. Counting a
// vault reads every deck in it and replays its whole answer log.
//
// The scenario has no reader for the vault, so nothing past the first check can
// run.
func TestACountIsDroppedOnceTheWindowHasMissing(t *testing.T) {
	t.Parallel()
	removed, went := context.WithCancel(t.Context())
	went()

	u := flashcards.NewCountCardsDue(
		flashcards.ListReviewCards{}, flashcards.Schedules{}, flashcards.Presets{},
		review.Day{}, vaulttest.NewClock(),
	)
	_, err := u.Execute(removed, domain.Vault{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the count came back with %v", err)
	}
}

// learnedHour is an hour well inside a review day, on a day every preset here
// carries the whole of its load.
var learnedHour = time.Date(2026, 9, 2, 10, 0, 0, 0, time.Local)

// A vault of three decks: two on a preset that counts a card learned while it
// is likely to be recalled, and one naming no preset at all. A card answered a
// moment ago is learned under the first rule and not under the second, so the
// same answer counts differently in the decks either side of it.
var learnedVault = map[string]string{
	"Term.md":        term,
	"Recall.md":      preset("learned: retention\nretention: 0.9\n"),
	"decks/One.md":   deckOf("Recall", 3, 0),
	"decks/Two.md":   deckOf("Recall", 2, 10),
	"decks/Three.md": deckOf("", 2, 20),
}

// The number of learned cards in a deck is counted by the rule of the preset
// the deck names, and two decks on one preset are counted by the same rule.
func TestADecksLearnedCardsAreCountedByItsOwnPresetsRule(t *testing.T) {
	t.Parallel()
	s := openVault(t, learnedVault)

	run := s.run(t, learnedHour)
	answer(t, run, "card000000", 5*time.Second)
	answer(t, run, "card000010", 5*time.Second)
	answer(t, run, "card000020", 5*time.Second)

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(learnedHour.Add(time.Minute))).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]flashcards.DeckCardsDue{
		// The rule is a chance of recall, and a card answered a minute ago is
		// as likely to come back as a card gets.
		"decks/One.md": {Deck: "decks/One.md", ReviewCards: 3, Learned: 1},
		"decks/Two.md": {Deck: "decks/Two.md", ReviewCards: 2, Learned: 1},
		// The defaults count by an interval of three weeks, which a card
		// answered once is nowhere near.
		"decks/Three.md": {Deck: "decks/Three.md", ReviewCards: 2, Learned: 0},
	}
	for _, one := range counts.Decks {
		if one.Learned != want[one.Deck].Learned || one.ReviewCards != want[one.Deck].ReviewCards {
			t.Errorf("%s has %d learned of %d, want %d of %d",
				one.Deck, one.Learned, one.ReviewCards,
				want[one.Deck].Learned, want[one.Deck].ReviewCards)
		}
	}
	if len(counts.Decks) != len(want) {
		t.Errorf("the vault came to %d decks, want %d", len(counts.Decks), len(want))
	}
}

func TestCountCardsDue_UnansweredDeckHasZeroLearned(t *testing.T) {
	t.Parallel()
	s := openVault(t, learnedVault)

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(learnedHour)).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range counts.Decks {
		if one.Learned != 0 {
			t.Errorf("%s has %d learned before review",
				one.Deck, one.Learned)
		}
	}
}

// A deck holding no cards is counted for nothing: it has no review card to stand
// learned.
func TestADeckOfNoCardsIsCountedForNothing(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Recall.md":      preset("learned: retention\nretention: 0.9\n"),
		"decks/Empty.md": deckOf("Recall", 0, 0),
		"decks/One.md":   deckOf("Recall", 1, 0),
	})

	counts, err := s.newCountCardsDueAt(today, clock.NewFake(learnedHour)).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range counts.Decks {
		if one.Deck == "decks/Empty.md" {
			t.Errorf("a deck of no cards came to %+v", one)
		}
	}
}

// The number of unbegun review cards in a deck is counted in the same pass, so
// a deck whose review cards are all unbegun reports that on its own row.
func TestADecksUnbegunCardsAreCounted(t *testing.T) {
	t.Parallel()
	s := openVault(t, learnedVault)

	before, err := s.newCountCardsDueAt(today, clock.NewFake(learnedHour)).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range before.Decks {
		if one.Unbegun != one.ReviewCards {
			t.Errorf("%s has %d unbegun of %d, want all",
				one.Deck, one.Unbegun, one.ReviewCards)
		}
	}

	run := s.run(t, learnedHour)
	answer(t, run, "card000000", 5*time.Second)

	after, err := s.newCountCardsDueAt(today, clock.NewFake(learnedHour.Add(time.Minute))).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"decks/One.md": 2, "decks/Two.md": 2, "decks/Three.md": 2}
	for _, one := range after.Decks {
		if one.Unbegun != want[one.Deck] {
			t.Errorf("%s has %d unbegun, want %d", one.Deck, one.Unbegun, want[one.Deck])
		}
	}
}

// Answers per day are counted from the vault's own answers, and each answer
// belongs to the day it began on.
func TestCountAnswers_GroupsByDay(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	other := review.CardID{Card: "zpqrstvwxy", Template: "Recognise"}

	yesterday := vaulttest.Epoch.AddDate(0, 0, -1)
	past := s.run(t, yesterday)
	if _, err := past.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := past.Answer(t.Context(), other, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answered != 3 {
		t.Errorf("the vault holds %d answers, want 3", got.Answered)
	}
	if got.Days[today.GetName(yesterday)].Answered != 2 {
		t.Errorf("yesterday came to %d, want 2: %v", got.Days[today.GetName(yesterday)].Answered, got.Days)
	}
	if got.Days[today.GetName(vaulttest.Epoch)].Answered != 1 {
		t.Errorf("today came to %d, want 1: %v", got.Days[today.GetName(vaulttest.Epoch)].Answered, got.Days)
	}
	if got.Streak != 2 {
		t.Errorf("the streak is %d, want the two days answered", got.Streak)
	}
}

// not a failure.
func TestCountAnswers_EmptyVaultHasNoDays(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Days) != 0 || got.Streak != 0 || got.Answered != 0 {
		t.Errorf("counted %+v", got)
	}
}

// A run that has not changed is not read again. What a day came to is a sum, so
// the counting is kept by run and only what grew is counted afresh — which is
// what a schedule cannot do, because where an answer leaves a card depends on
// every answer before it.
func TestARunThatHasNotChangedIsNotCountedAgain(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}

	first := s.run(t, vaulttest.Epoch.AddDate(0, 0, -1))
	if _, err := first.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.counted.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	// The run is left at its current position and its content is replaced with a line
	// nothing can read. A count that read it again would lose the day it holds.
	target := getRunFiles(t, s)
	if len(target) != 1 {
		t.Fatalf("the vault holds %v", target)
	}
	was, err := s.env.Store.Read(t.Context(), target[0])
	if err != nil {
		t.Fatal(err)
	}
	s.env.Store.File(target[0], spaces(len(was)))

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answered != 1 {
		t.Errorf("a run of the same length was read again: %+v", got)
	}
}

// A run that grew is counted afresh, so a session's own answers are in the
// counting as they are written.
func TestARunThatGrewIsCountedAfresh(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	other := review.CardID{Card: "zpqrstvwxy", Template: "Recognise"}

	record := s.run(t, vaulttest.Epoch)
	if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if first.Answered != 1 {
		t.Fatalf("counted %+v", first)
	}

	if _, err := record.Answer(t.Context(), other, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	second, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if second.Answered != 2 {
		t.Errorf("the answer written after the counting is not in it: %+v", second)
	}
}

// One identifier is one answer, however many files carry it. A synchroniser
// that meets a conflict leaves a second copy of a run next to the first, and the
// day counts the answer once.
func TestARunCopiedBelowAnotherNameIsCountedOnce(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	writeConflictedCopies(t, s)

	// Once from the files, and again from what the first counting kept.
	for _, from := range []string{"the files", "the counting kept"} {
		got, err := s.counted.Execute(t.Context(), s.vault)
		if err != nil {
			t.Fatal(err)
		}
		if got.Answered != 1 {
			t.Errorf("counted from %s, the vault holds %d answers, want the one given",
				from, got.Answered)
		}
		if day := got.Days[today.GetName(vaulttest.Epoch)].Answered; day != 1 {
			t.Errorf("counted from %s, today came to %d, want 1: %v", from, day, got.Days)
		}
	}
}

// writeConflictedCopies puts a copy of every run beside it, under the name a
// synchroniser that met a conflict leaves.
func writeConflictedCopies(t *testing.T, s vaulted) {
	t.Helper()
	for _, name := range getRunFiles(t, s) {
		raw, err := s.env.Store.Read(t.Context(), name)
		if err != nil {
			t.Fatal(err)
		}
		stem := strings.TrimSuffix(name, ".jsonl") +
			" (conflicted copy).jsonl"
		s.env.Store.File(stem, raw)
	}
}

// getRunFiles is the files the plugin's folder of the vault holds.
func getRunFiles(t *testing.T, s vaulted) []string {
	t.Helper()
	entries, err := s.env.Store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, one := range entries {
		out = append(out, one.Name)
	}
	return out
}

// spaces is a file of one length holding nothing that reads as an answer.
func spaces(n int) []byte {
	out := make([]byte, n)
	for item := range out {
		out[item] = ' '
	}
	return out
}

// Upcoming cards are counted by the day they fall due, which covers the week
// ahead as well as the year behind.
func TestWhatIsStillToComeIsCountedByDay(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}

	// Answered easily, so it is days away and not minutes.
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Easy, 0); err != nil {
		t.Fatal(err)
	}

	schedules, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	due := schedules[on].Due

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Due[today.GetName(due)] != 1 {
		t.Errorf("the day it falls on comes to %d, want the card: %v", got.Due[today.GetName(due)], got.Due)
	}
	if len(got.Due) != 1 {
		t.Errorf("what is still to come is %v, want the one card", got.Due)
	}
}

func TestCountAnswers_EmptyVaultHasNoDueCards(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Due) != 0 {
		t.Errorf("a vault nobody answered has %v still to come", got.Due)
	}
}

// countedCache is the day counts as a test writes them: enough of the shape to
// say what the cache claims, and to say it under a version of its own.
type countedCache struct {
	V    int                `json:"v"`
	Runs []countedCachedRun `json:"runs"`
}

type countedCachedRun struct {
	Name string                  `json:"name"`
	Size int                     `json:"size"`
	Days map[string]review.Tally `json:"days"`
	IDs  []string                `json:"ids"`
}

// writeCache writes a cache over the vault's counting. The runs are the ones
// the last counting wrote, so the cache records this vault's file lengths, and the
// test supplies the per-day tallies.
func writeCache(t *testing.T, s vaulted, version int, days map[string]review.Tally) {
	t.Helper()
	if _, err := s.counted.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	raw, err := s.counted.Cache.Read(t.Context(), s.vault.ID)
	if err != nil {
		t.Fatal(err)
	}
	var was countedCache
	if err := json.Unmarshal(raw, &was); err != nil {
		t.Fatal(err)
	}
	if len(was.Runs) != 1 {
		t.Fatalf("the counting was kept from %d runs, want the one", len(was.Runs))
	}
	was.V, was.Runs[0].Days = version, days

	now, err := json.Marshal(was)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.counted.Cache.Write(t.Context(), s.vault.ID, now); err != nil {
		t.Fatal(err)
	}
}

// A cache written in another shape is not read: its contents differ from what
// this build writes, and the answers are counted again.
func TestACacheOfAnotherShapeIsCountedAfresh(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	writeCache(t, s, 0, map[string]review.Tally{"1999-01-01": {Answered: 99, Good: 99}})

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, target := got.Days["1999-01-01"]; target {
		t.Errorf("a cache of another shape was believed: %v", got.Days)
	}
	if got.Answered != 1 {
		t.Errorf("the vault holds %d answers, want the one written", got.Answered)
	}
}

// A cache in the shape this build writes is trusted. The shape, and not the
// file name, decides whether the cache is read.
func TestACacheOfThisShapeIsBelieved(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	writeCache(t, s, 2, map[string]review.Tally{"1999-01-01": {Answered: 99, Good: 99}})

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Days["1999-01-01"].Answered != 99 {
		t.Errorf("the cache was not read: %v", got.Days)
	}
}

// A build that caches nothing counts the whole log at every launch and returns
// the same result as one that caches.
func TestAVaultIsCountedWithNothingCached(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	countReviews := s.counted
	countReviews.Cache = nil

	got, err := countReviews.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answered != 1 || got.Days[today.GetName(vaulttest.Epoch)].Answered != 1 {
		t.Errorf("counted %+v", got)
	}
}

// Upcoming cards are computed from the schedules. A counting without a
// scheduler reports the answers and no upcoming cards, and does not fail.
func TestWithNoSchedulerNothingIsStillToCome(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Easy, 0); err != nil {
		t.Fatal(err)
	}

	scheds := flashcards.NewSchedules(s.logs, nil, today, flashcards.ListReviewCards{}, flashcards.Presets{})
	blind := flashcards.NewCountReviews(s.logs, scheds, today, vaulttest.NewClock())

	got, err := blind.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Due) != 0 {
		t.Errorf("what is still to come is %v, want nothing", got.Due)
	}
	if got.Answered != 1 {
		t.Errorf("the vault holds %d answers, want the one written", got.Answered)
	}
}

// A card due today is not counted as upcoming, however it was answered.
func TestACardDueTodayIsNotStillToCome(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}

	// Answered again, so it is due again within minutes and is due today.
	if _, err := s.run(t, vaulttest.Epoch).Answer(t.Context(), on, review.Again, 0); err != nil {
		t.Fatal(err)
	}

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Due) != 0 {
		t.Errorf("a card counter today is counted as still to come: %v", got.Due)
	}
}

// A caller requests a fixed range of days. Both ends are inclusive, an empty
// end is unbounded at that end, and the streak and the count cover the whole
// vault either way.
func TestGetReviewDaysKeepsTheDaysInsideTheStretch(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	now := vaulttest.Epoch
	yesterday, twoDaysAgo := now.AddDate(0, 0, -1), now.AddDate(0, 0, -2)
	for _, item := range []time.Time{twoDaysAgo, yesterday, now} {
		if _, err := s.run(t, item).Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}
	older, middle, newest := today.GetName(twoDaysAgo), today.GetName(yesterday), today.GetName(now)
	get := flashcards.NewGetReviewDays(s.counted)

	for name, one := range map[string]struct {
		from, to string
		want     []string
	}{
		"both ends":   {middle, middle, []string{middle}},
		"from alone":  {middle, "", []string{middle, newest}},
		"to alone":    {"", middle, []string{older, middle}},
		"neither end": {"", "", []string{older, middle, newest}},
		"nothing in":  {"2000-01-01", "2000-12-31", nil},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := get.Execute(t.Context(), s.vault, one.from, one.to)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Days) != len(one.want) {
				t.Fatalf("the days %v came back, want %v", got.Days, one.want)
			}
			for _, day := range one.want {
				if _, target := got.Days[day]; !target {
					t.Errorf("%s is missing from %v", day, got.Days)
				}
			}
			if got.Answered != 3 {
				t.Errorf("the vault holds %d answers, want 3", got.Answered)
			}
		})
	}
}

// busy is an index that cannot be brought level, which is what a second writer
func busy(context.Context, domain.Vault, []string) error {
	return errors.New("the index is busy")
}

// A deck that could not be written is named and left currently. The editor
// from and not left to wonder.
func TestADeckThatCouldNotBeSavedIsNamed(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)

	// The deck cannot be replaced, so the editor reports it as unwritten.
	s.env.Host.RefuseWriting("decks/Own.md", fs.ErrPermission)

	assigned, err := s.assigning.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a deck that could not be written was trouble: %v", err)
	}
	if !slices.Contains(assigned.Unwritten, "decks/Own.md") {
		t.Errorf("the deck that could not be written is %v", assigned.Unwritten)
	}
}

// A deck the index could not be brought level with was written all the same.
// Its cards carry the card ids that were minted, so they stand in this session and
// nothing has to mint them a second time.
func TestADeckSavedWithNoLevellingIsNotNamed(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)
	assigning := s.assigning.WithIndex(busy)

	assigned, err := assigning.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a levelling that failed was trouble: %v", err)
	}
	if len(assigned.Unwritten) != 0 {
		t.Fatalf("the deck was written and is named as unwritten: %v", assigned.Unwritten)
	}
	if target := read(t, s.vault, "decks/Own.md"); !strings.Contains(target, "## Leaf mould ^") {
		t.Errorf("the deck carries no card id: %q", target)
	}

	status, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 {
		t.Errorf("the marked card stands %d times, want once", len(status))
	}
}

// A vault whose decks are all written reports no unwritten decks.
func TestAVaultWhoseDecksAreSavedNamesNone(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)

	assigned, err := s.assigning.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(assigned.Unwritten) != 0 {
		t.Errorf("named %v, want none", assigned.Unwritten)
	}
}

// reading opens one index and returns the use case that reads it, together
// with a way to write another vault into it. One index for every vault is what
// the application runs, so a scoping test has somewhere to leak to.
func reading(
	t *testing.T,
) (flashcards.ShowNeighbourhood, func(notes map[string]string) domain.Vault) {
	t.Helper()

	installation := vaulttest.NewInstallation()
	add := func(notes map[string]string) domain.Vault {
		t.Helper()
		return installation.Add(t, notes)
	}

	return flashcards.NewShowNeighbourhood(
		note.NewShowLinks(installation.Queries),
		installation.Queries,
		note.NewRead(installation.Ports.Readers),
	), add
}

// readNeighbourhood is one vault of notes, and what its deck turns out to be
// joined to.
func readNeighbourhood(t *testing.T, notes map[string]string, deck string) flashcards.Neighbourhood {
	t.Helper()
	u, add := reading(t)
	return getNeighbourhood(t, u, add(notes), deck)
}

func getNeighbourhood(
	t *testing.T, u flashcards.ShowNeighbourhood, v domain.Vault, deck string,
) flashcards.Neighbourhood {
	t.Helper()
	out, err := u.Execute(t.Context(), v, deck)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// getWrittenLinks is how each entry was addressed, which is all a link that
// reached no note ever has.
func getWrittenLinks(j flashcards.Neighbourhood) []string {
	out := make([]string, 0, len(j.Notes))
	for _, one := range j.Notes {
		out = append(out, one.Written)
	}
	return out
}

func paths(j flashcards.Neighbourhood) []string {
	out := make([]string, 0, len(j.Notes))
	for _, one := range j.Notes {
		out = append(out, one.Path)
	}
	return out
}

func TestALinkSavedInsideACardArrivesWithItsText(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	one := j.Notes[0]
	if one.Path != "Migration.md" || one.Title != "Migration" {
		t.Errorf("got %q titled %q", one.Path, one.Title)
	}
	if !strings.Contains(one.Body, "Birds go south") {
		t.Errorf("body is %q", one.Body)
	}
	if one.IsBacklink {
		t.Error("the deck points at it, and the answer says otherwise")
	}
	if one.Outcome != note.Ok {
		t.Errorf("outcome %q", one.Outcome)
	}
}

// Every card names the card model it is cut by, so a deck points at its card models.
// A card model is how a card is laid out and not what it was written from.
func TestTheCardModelACardIsSplitByIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"card-models/Word.md": "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n[[card-models/Word]]\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 1 || got[0] != "Migration.md" {
		t.Fatalf("joined to %v", got)
	}
}

// A deck is joined to the notes it was written from. A deck names the preset
// that schedules it and may name another deck, and neither is a note written
// about the material. This holds on both sides: a deck pointing at this one is
// still a deck.
func TestOnlyTheNotesADeckWasSavedFromAreRead(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n---\n" +
			"\n# Sanskrit\n\nTwenty minutes a day.\n",
		"decks/Roots.md": "---\ntype: deck\n---\n\n# Roots\n\nThe verbs.\n",
		"decks/Songs.md": "---\ntype: deck\n---\n\n# Songs\n\nSung over [[decks/Birds]].\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Sanskrit\n    role: ref\n    type: preset\n" +
			"  - to: decks/Roots\n    role: ref\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nSee [[Migration]] and [[Feathers]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
		"Feathers.md":  "# Feathers\n\nWhat a wing is made of.\n",
	}, "decks/Birds.md")

	want := []string{"Migration.md", "Feathers.md"}
	got := paths(j)
	if len(got) != len(want) {
		t.Fatalf("joined to %v, want %v", got, want)
	}
	for _, path := range want {
		if !slices.Contains(got, path) {
			t.Errorf("joined to %v, want %v", got, want)
		}
	}
	// Excluded notes are never read, so none is reported as unread.
	if j.Unread != 0 {
		t.Errorf("%d of them are said to have come without their text", j.Unread)
	}
}

func TestANotePointingAtTheDeckIsNotOneTheDeckPointsAt(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
		"Migration.md":   "# Migration\n\nWorked at with [[decks/Birds]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if j.Notes[0].Path != "Migration.md" {
		t.Fatalf("got %q", j.Notes[0].Path)
	}
	if !j.Notes[0].IsBacklink {
		t.Error("a backlink was reported as something the deck points at")
	}
	if !strings.Contains(j.Notes[0].Body, "Worked at with") {
		t.Errorf("body is %q", j.Notes[0].Body)
	}
}

func TestANoteOnBothSidesIsNamedOnce(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":   "# Migration\n\nDrilled in [[decks/Birds]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if j.Notes[0].IsBacklink {
		t.Error("a note on both sides is one the deck points at")
	}
}

func TestTheDeckDoesNotStandInItsOwnList(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\nGathered in [[decks/Birds]] itself.\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nFiled under [[decks/Birds]].\n" +
			"\n### Meaning\n\nSee [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 || j.Notes[0].Path != "Migration.md" {
		t.Fatalf("joined to %v", paths(j))
	}
}

func TestADanglingLinkKeepsTheNameItWasSavedBy(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nSee [[Nowhere At All]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	one := j.Notes[0]
	if one.Written != "Nowhere At All" {
		t.Errorf("written %q", one.Written)
	}
	if one.Path != "" || one.Body != "" || one.Title != "" {
		t.Errorf("a dangling link came back with %+v", one)
	}
}

func TestOneNameThatCameLooseIsNamedOnce(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSee [[Nowhere At All]].\n" +
			"\n## Swallow ^3dkmf936tb\n\n### Word\n\nSee [[Nowhere At All]] again.\n",
	}, "decks/Birds.md")

	if got := getWrittenLinks(j); len(got) != 1 || got[0] != "Nowhere At All" {
		t.Fatalf("joined to %v", got)
	}
}

// A name shared by several notes resolves to the nearest one, and the reader
// cannot detect the ambiguity without the mark.
func TestReadNeighbourhood_MarksAmbiguousName(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md":     "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"north/Migration.md": "# Migration\n\nOne of the two.\n",
		"south/Migration.md": "# Migration\n\nThe other, answering to the same name.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if !j.Notes[0].IsAmbiguous {
		t.Errorf("%q answers to a name two notes answer to and is not said to be ambiguous",
			j.Notes[0].Path)
	}
}

// A panel of titles with no content gives no reason, so an unreachable vault is
// an error and not a list of empty entries.
func TestAVaultOutOfReachIsAnError(t *testing.T) {
	t.Parallel()
	u, add := reading(t)
	v := add(map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":   "# Migration\n\nBirds go south.\n",
	})
	if err := os.RemoveAll(v.Path); err != nil {
		t.Fatal(err)
	}

	if _, err := u.Execute(t.Context(), v, "decks/Birds.md"); err == nil {
		t.Error("a vault that is not there was read around without complaint")
	}
}

func TestAnAttachmentIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n![[asset://diagram.png]]\n\nSee [[Migration]].\n",
		"diagram.png":  "not really a picture",
		"Migration.md": "# Migration\n\nBirds go south.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 || j.Notes[0].Path != "Migration.md" {
		t.Fatalf("joined to %v", paths(j))
	}
}

// A link that resolves to nothing is retained only for a name that resolved
// earlier. An address that never named a note is not such a name, and displaying
// it as a missing note reports a question the vault does not have.
func TestAnAddressThatNamesNoNoteIsNotADanglingNote(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"https://example.org/birds\"\n    role: ref\n---\n" +
			"\n![[asset://diagram.png]]\n\nSee [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south.\n",
	}, "decks/Birds.md")

	for _, one := range j.Notes {
		if one.Path == "" && one.Written != "Migration" {
			t.Errorf("%q is drawn as a note whose name came loose", one.Written)
		}
	}
	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", getWrittenLinks(j))
	}
}

func TestALinkIntoAnotherVaultIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	// An identifier finds its note wherever it is, and where it is may be a
	// vault this reading has no reader for. Both vaults file a note at the same
	// path, so a reading that ignores which vault the link landed in shows the
	// wrong note and not nothing.
	const id = "01M02DTC80PABQQW3XS3XWDVHW"
	u, add := reading(t)
	v := add(map[string]string{
		"decks/Birds.md": "---\ntype: deck\nlinks:\n  - to: \"note://" + id +
			"\"\n    role: jump\n---\n\n# The ones with feathers\n",
		"elsewhere.md": "# A namesake at home\n",
	})
	add(map[string]string{
		"elsewhere.md": "---\nid: " + id + "\n---\n\n# In the other vault\n",
	})

	j := getNeighbourhood(t, u, v, "decks/Birds.md")
	if len(j.Notes) != 0 {
		t.Fatalf("joined to %v", paths(j))
	}
}

func TestANoteDeletedAfterTheScanDoesNotSinkTheOthers(t *testing.T) {
	t.Parallel()
	u, add := reading(t)
	v := add(map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nSee [[Migration]] and [[Feathers]].\n",
		"Migration.md":   "# Migration\n\nBirds go south.\n",
		"Feathers.md":    "# Feathers\n\nBarbs on a shaft.\n",
	})
	if err := os.Remove(filepath.Join(v.Path, "Migration.md")); err != nil {
		t.Fatal(err)
	}

	j := getNeighbourhood(t, u, v, "decks/Birds.md")
	if len(j.Notes) != 2 {
		t.Fatalf("joined to %v", paths(j))
	}
	byPath := map[string]flashcards.Neighbour{}
	for _, one := range j.Notes {
		byPath[one.Path] = one
	}
	removed := byPath["Migration.md"]
	if removed.Outcome != note.Missing || removed.Body != "" {
		t.Errorf("the deleted note came back as %+v", removed)
	}
	if !strings.Contains(byPath["Feathers.md"].Body, "Barbs on a shaft") {
		t.Errorf("the note beside it came back as %+v", byPath["Feathers.md"])
	}
}

func TestPastTheThirtiethNoteTheTextIsLeftUnread(t *testing.T) {
	t.Parallel()
	const pointing = flashcards.MostRead + 5
	notes := map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
	}
	for i := range pointing {
		name := fmt.Sprintf("Note %02d", i)
		notes[name+".md"] = "# " + name + "\n\nDrilled in [[decks/Birds]].\n"
	}
	j := readNeighbourhood(t, notes, "decks/Birds.md")

	if len(j.Notes) != pointing {
		t.Fatalf("joined to %d notes", len(j.Notes))
	}
	if j.Unread != 5 {
		t.Errorf("said %d came without their text", j.Unread)
	}
	for i, one := range j.Notes {
		read := one.Body != ""
		if want := i < flashcards.MostRead; read != want {
			t.Errorf("note %d of %d: read=%v", i, len(j.Notes), read)
		}
		if one.Title == "" {
			t.Errorf("note %d came back unnamed", i)
		}
	}
}

func TestADeckJoinedToNothingIsAnEmptyAnswer(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 0 || j.Unread != 0 {
		t.Fatalf("got %+v", j)
	}
}

func TestOneVaultsDeckIsNotJoinedToAnothersNotes(t *testing.T) {
	t.Parallel()
	// Both vaults file their deck at the same path, so a query that forgets its
	u, add := reading(t)
	feathers := add(map[string]string{
		"decks/Deck.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":  "# Migration\n\nBirds go south when the days shorten.\n",
	})
	sums := add(map[string]string{
		"decks/Deck.md": "---\ntype: deck\n---\n\nCut from [[Arithmetic]].\n",
		"Arithmetic.md": "# Arithmetic\n\nRemainders divide integers evenly.\n",
	})

	first := getNeighbourhood(t, u, feathers, "decks/Deck.md")
	if len(first.Notes) != 1 || first.Notes[0].Path != "Migration.md" {
		t.Fatalf("the first vault is joined to %v", paths(first))
	}
	if strings.Contains(first.Notes[0].Body, "Remainders") {
		t.Errorf("the first vault read the second's note: %q", first.Notes[0].Body)
	}

	second := getNeighbourhood(t, u, sums, "decks/Deck.md")
	if len(second.Notes) != 1 || second.Notes[0].Path != "Arithmetic.md" {
		t.Fatalf("the second vault is joined to %v", paths(second))
	}
	if strings.Contains(second.Notes[0].Body, "days shorten") {
		t.Errorf("the second vault read the first's note: %q", second.Notes[0].Body)
	}
}

// A deck points at the preset that schedules it, so the preset is among the
// notes the deck is joined to. A preset defines the schedule and is not a source
// of the cards.
func TestThePresetADeckIsScheduledByIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"presets/Steady.md": "---\ntype: preset\ngoal: retention\nretention: 0.95\n---\n" +
			"\n# Steady\n\nA slower climb through the same cards.\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"presets/Steady\"\n    role: ref\n    type: preset\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 1 || got[0] != "Migration.md" {
		t.Fatalf("joined to %v", got)
	}
}

// A preset written into a note that points back at the deck is read like any
// other note: the deck's own link determines which preset schedules it.
func TestAPresetIsLeftOutOfEveryCardsPanel(t *testing.T) {
	t.Parallel()
	j := readNeighbourhood(t, map[string]string{
		"presets/Steady.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n---\n" +
			"\n# Steady\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"presets/Steady\"\n    role: ref\n    type: preset\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n\n### Meaning\n\nA bird.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 0 {
		t.Fatalf("joined to %v", got)
	}
}

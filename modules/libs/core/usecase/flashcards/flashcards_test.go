package flashcards_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// One vault of two stencils and two decks. The animal stencil has two faces, so
// each of its cards is two card faces; the term stencil has one.
var vault = map[string]string{
	"Animal.md": "---\ntype: stencil\nfields:\n  - Name\n  - Height\n---\n" +
		"\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n" +
		"\n## Name it\n\n### Front\n\nWhat is {{Height}} at the shoulder?\n\n### Back\n\n{{Name}}\n",
	"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"decks/Mammals.md": "---\ntype: deck\n---\n" +
		"\n# The ones with fur\n" +
		"\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n" +
		"\n## Otter ^zpqrstvwxy\n\n[[Animal]]\n\n### Name\n\nOtter\n\n### Height\n\nabout 12\"\n",
	"decks/Words.md": "---\ntype: deck\n---\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// vaulted is one vault on disk, scanned, and the scenarios built over it.
type vaulted struct {
	vault     domain.Vault
	standings flashcards.ListCardFaces
	presets   flashcards.Presets
	marking   flashcards.MarkCards
	kept      flashcards.Schedules
	counted   flashcards.CountReviews
	logs      filesystem.DerivedStores
	// scan brings the index level with what the vault now holds.
	scan func(ctx context.Context, v domain.Vault, paths []string) error
}

func openVault(t testing.TB, notes map[string]string) vaulted {
	t.Helper()
	ctx := t.Context()

	db, err := index.Open(ctx, indexfile.Path(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := vaults.Scan{
		Readers: filesystem.VaultReaders{}, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.NoteQueries(), Maintenance: db.Maintenance(),
	}
	v := testsupport.NewVault(t, notes)
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}
	scanned := func(ctx context.Context, v domain.Vault, _ []string) error {
		_, err := scan.Execute(ctx, v)
		return err
	}

	logs := filesystem.DerivedStores{Area: filesystem.FlashcardsDir}
	standings := flashcards.NewListCardFaces(
		filesystem.VaultReaders{}, db.NoteQueries(), db.NoteQueries())
	presets := flashcards.NewPresets(
		filesystem.VaultReaders{}, filesystem.VaultWriters{},
		db.NoteQueries(), db.NoteQueries(), scanned, review.Day{}, time.Now)
	presets.Problems = db.NoteQueries()
	// Each card is worked out at the share of the cards its own preset asks
	// for, which is how the application builds this.
	schedules := flashcards.NewSchedules(logs, review.NewFSRS(), today, standings, presets)
	schedules.Cache = appstate.SchedulesAt(filepath.Join(t.TempDir(), "flashcards"))

	counted := flashcards.NewCountReviews(logs, schedules, today, time.Now)
	counted.Cache = appstate.SchedulesAt(filepath.Join(t.TempDir(), "days"))

	return vaulted{
		vault:     v,
		standings: standings,
		presets:   presets,
		marking: flashcards.NewMarkCards(
			filesystem.VaultReaders{}, filesystem.VaultWriters{},
			db.NoteQueries(), db.NoteQueries(), scanned, time.Now),
		kept:    schedules,
		counted: counted,
		logs:    logs,
		scan:    scanned,
	}
}

// write puts a file into the vault and brings the index level with it, which is
// what a person editing their own note in another window leaves behind.
func write(t *testing.T, s vaulted, path, body string) {
	t.Helper()
	at := filepath.Join(s.vault.Path, filepath.FromSlash(path))
	if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.scan(t.Context(), s.vault, []string{path}); err != nil {
		t.Fatal(err)
	}
}

// remove takes a file out of the vault and brings the index level with it,
// which is what a person deleting their own note in another window leaves
// behind.
func remove(t *testing.T, s vaulted, path string) {
	t.Helper()
	if err := os.Remove(filepath.Join(s.vault.Path, filepath.FromSlash(path))); err != nil {
		t.Fatal(err)
	}
	if err := s.scan(t.Context(), s.vault, []string{path}); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, v domain.Vault, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func (s vaulted) run(t *testing.T, at time.Time) flashcards.Record {
	t.Helper()
	run, err := flashcards.Log{Stores: s.logs}.Open(t.Context(), s.vault, at)
	if err != nil {
		t.Fatal(err)
	}
	return flashcards.Record{Run: run, Now: func() time.Time { return at }}
}

// runNamed is a run whose file is named for one instant and whose answers were
// given at another, which is what a run written on another machine and carried
// here by a synchroniser looks like.
func (s vaulted) runNamed(t *testing.T, named, given time.Time) flashcards.Record {
	t.Helper()
	run, err := flashcards.Log{Stores: s.logs}.Open(t.Context(), s.vault, named)
	if err != nil {
		t.Fatal(err)
	}
	return flashcards.Record{Run: run, Now: func() time.Time { return given }}
}

func (s vaulted) newCountCardsDue(day review.Day) flashcards.CountCardsDue {
	return s.newCountCardsDueAt(day, time.Now)
}

func (s vaulted) newCountCardsDueAt(day review.Day, now func() time.Time) flashcards.CountCardsDue {
	return flashcards.CountCardsDue{
		CardFaces: s.standings, Schedules: s.kept, Presets: s.presets, Day: day, Now: now,
	}
}

// session is a session whose presets read nothing, so every deck of the vault
// is scheduled by the defaults.
func (s vaulted) session(day review.Day) flashcards.Session {
	return flashcards.NewSession(
		s.marking, s.standings, s.kept,
		flashcards.NewPresets(nil, nil, nil, nil, nil, day, time.Now), day, time.Now,
	)
}

// today is the day a session is counted in, on this machine.
var today = review.Day{Starts: review.DayStarts}

// A card stands once for each face of the stencil that cuts it, because each
// face asks a different thing.
func TestACardStandsOnceForEachFaceOfItsStencil(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 5 {
		t.Fatalf("two cards of two faces and one of one face is five, got %d", len(stood))
	}

	faces := map[string]int{}
	for _, one := range stood {
		faces[one.ID.Face]++
		front, back := one.Lay()
		if one.ID.Card == "" || front == "" || back == "" {
			t.Errorf("nothing laid out for %+v", one.ID)
		}
	}
	for face, want := range map[string]int{"Recognise": 2, "Name it": 2, "Say it": 1} {
		if faces[face] != want {
			t.Errorf("the face %q stands %d times, want %d", face, faces[face], want)
		}
	}
}

// The face is laid out with this card's values, so what a person is shown is
// their own writing.
func TestAFaceIsLaidOutWithTheCardsOwnValues(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range stood {
		if one.ID.Card != "k7m2xq9fzp" || one.ID.Face != "Name it" {
			continue
		}
		front, back := one.Lay()
		if front != "What is about 45\" at the shoulder?" || back != "Llama" {
			t.Errorf("laid out front %q, back %q", front, back)
		}
		if one.Heading != "Llama" || one.Section != "The ones with fur" {
			t.Errorf("heading %q under %q", one.Heading, one.Section)
		}
		return
	}
	t.Error("the card was not among what stands in the vault")
}

// handwritten is a deck of one card that carries no mark, which is what a deck
// typed by hand is until the application writes it.
var handwritten = map[string]string{
	"Term.md": vault["Term.md"],
	"decks/Own.md": "---\ntype: deck\n---\n" +
		"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
}

// A card with no mark has nothing an answer could be recorded against, so
// nothing stands for it. Reading what a vault holds writes nothing.
func TestACardWithNoMarkStandsForNothing(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)
	before := read(t, s.vault, "decks/Own.md")

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 0 {
		t.Errorf("stands = %+v, want nothing", stood)
	}
	if after := read(t, s.vault, "decks/Own.md"); after != before {
		t.Errorf("the deck was written\n was %q\n now %q", before, after)
	}
}

// mixed is a deck a person wrote and edited: prose of their own, a section, one
// card that already carries a mark and one that does not.
var mixed = map[string]string{
	"Term.md": vault["Term.md"],
	"decks/Mine.md": "---\ntype: deck\n---\n" +
		"\nWhat I keep meaning to learn, and never do.\n" +
		"\n# The ones from the garden\n" +
		"\n## Leaf mould ^k7m2xq9fzp\n\n[[Term]]\n" +
		"\n### Слово\n\nLeaf mould\n\n### Значение\n\nCompost made of fallen leaves\n" +
		"\n## Loam\n\n[[Term]]\n" +
		"\n### Слово\n\nLoam\n\n### Значение\n\nSand, silt and clay in the right measure\n",
}

// Marking changes the marks and nothing else. The deck is the person's own
// writing, and a mark minted over one a card already carries would orphan every
// answer that card has ever been given.
func TestMarkingChangesTheMarksAndNothingElse(t *testing.T) {
	t.Parallel()
	s := openVault(t, mixed)
	before := read(t, s.vault, "decks/Mine.md")

	if _, err := s.marking.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	after := read(t, s.vault, "decks/Mine.md")

	if !strings.Contains(after, "## Leaf mould ^k7m2xq9fzp") {
		t.Error("the mark the card already carried is not the mark it carries now")
	}
	for _, kept := range []string{
		"What I keep meaning to learn, and never do.",
		"# The ones from the garden",
		"Sand, silt and clay in the right measure",
		"### Слово",
	} {
		if !strings.Contains(after, kept) {
			t.Errorf("%q is not in the deck any more", kept)
		}
	}

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 2 {
		t.Fatalf("two cards of a one-faced stencil stand twice, got %d", len(stood))
	}

	if after == before {
		t.Error("the deck held a card with no mark and was not written")
	}

	// And a deck whose cards all carry marks is not written again. The bytes of
	// a faithful rewrite are the same bytes, so what says it was left alone is
	// the file's own time: a write nobody needed wakes every watcher on the
	// vault and sends the deck through the index again.
	was := getModTime(t, s.vault, "decks/Mine.md")
	again, err := s.marking.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Unwritten) != 0 {
		t.Errorf("could not write %v", again.Unwritten)
	}
	if now := getModTime(t, s.vault, "decks/Mine.md"); !now.Equal(was) {
		t.Errorf("a deck with nothing to mark was written at %v, having stood at %v", now, was)
	}
	if now := read(t, s.vault, "decks/Mine.md"); now != after {
		t.Errorf("a deck with nothing to mark was changed\n was %q\n now %q", after, now)
	}
}

// getModTime is when a file of the vault was last written.
func getModTime(t *testing.T, v domain.Vault, path string) time.Time {
	t.Helper()
	at, err := os.Stat(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return at.ModTime()
}

// A person starting a session on a vault has its cards given marks, and what stands
// then is what they are asked.
func TestSessionDownToAVaultMarksItsCards(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)

	if _, err := s.marking.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 {
		t.Fatalf("one card of a one-faced stencil stands once, got %d", len(stood))
	}
	if stood[0].ID.Card == "" {
		t.Error("the card came back with no mark to record an answer against")
	}
}

// Counting what a vault owes writes nothing to it: a person opening the window
// is shown every vault they hold, and none of them is written to for that.
func TestCountingAVaultWritesNothingToIt(t *testing.T) {
	t.Parallel()
	s := openVault(t, handwritten)
	before := read(t, s.vault, "decks/Own.md")

	if _, err := s.newCountCardsDue(today).Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if after := read(t, s.vault, "decks/Own.md"); after != before {
		t.Errorf("the deck was written\n was %q\n now %q", before, after)
	}
}

// A card whose wikilink reaches a note that is not a stencil is a card no face
// shows. It is left out, and the rest of the vault is reviewed.
func TestACardWithNoStencilIsLeftOut(t *testing.T) {
	t.Parallel()
	notes := map[string]string{
		"Term.md":    vault["Term.md"],
		"Weather.md": "# Weather\n\nAn ordinary note.\n",
		"decks/Mixed.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n" +
			"\n## Rain ^m9n8b7v6c5\n\n[[Weather]]\n\n### Word\n\nRain\n",
	}
	s := openVault(t, notes)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 || stood[0].ID.Card != "3f4g5h6j7k" {
		t.Errorf("stands = %+v, want the one card whose stencil is one", stood)
	}
}

// A face missing a front or a back lays out nothing, so nothing is asked through
// it. The stencil's other faces are asked as usual.
func TestAFaceMissingASideIsNotAsked(t *testing.T) {
	t.Parallel()
	notes := map[string]string{
		"Half.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n" +
			"\n## Half a face\n\n### Front\n\n{{Meaning}}\n",
		"decks/Half.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould ^3f4g5h6j7k\n\n[[Half]]\n\n### Word\n\nLeaf mould\n" +
			"\n### Meaning\n\nLeaves\n",
	}
	s := openVault(t, notes)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 || stood[0].ID.Face != "Say it" {
		t.Errorf("stands = %+v, want the one face with both its sides", stood)
	}
}

// A card that leaves a field empty is asked like any other: the placeholder lays
// out as nothing, and the card is still somebody's card.
func TestACardLeavingAFieldEmptyIsStillAsked(t *testing.T) {
	t.Parallel()
	notes := map[string]string{
		"Term.md": vault["Term.md"],
		"decks/Empty.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\n",
	}
	s := openVault(t, notes)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 {
		t.Fatalf("stands = %+v, want the one card", stood)
	}
	if front, back := stood[0].Lay(); front != "Leaf mould" || back != "" {
		t.Errorf("laid out front %q, back %q", front, back)
	}
}

// A vault nobody has answered owes everything it holds, as cards nobody has
// reached.
func TestAVaultNobodyAnsweredOwesEverythingAsNew(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	owing, err := s.newCountCardsDue(today).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if owing.Faces != 5 || owing.New != 5 || owing.Due != 0 {
		t.Errorf("owing = %+v, want five card faces and five of them new", owing)
	}
	if len(owing.Decks) != 2 {
		t.Errorf("counted %d decks, want 2", len(owing.Decks))
	}
}

// A card put days away is neither owed nor asked until those days are up. What
// is owed is the day the schedule falls on, and a card answered easily is not
// that day's.
func TestACardPutDaysAwayIsNotOwedToday(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Easy, 0); err != nil {
		t.Fatal(err)
	}

	owing, err := s.newCountCardsDue(today).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if owing.Faces != 5 || owing.New != 4 || owing.Due != 0 {
		t.Errorf("owing = %+v, want the answered face counted as neither new nor due", owing)
	}
	for _, deck := range owing.Decks {
		if deck.Deck == "decks/Mammals.md" && (deck.New != 3 || deck.Due != 0) {
			t.Errorf("the deck it stands in comes to %+v", deck)
		}
	}

	session, err := s.session(today).Execute(t.Context(), s.vault, flashcards.Scope{})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range session.Queue {
		if one.ID == on {
			t.Error("the card is days away and was asked today")
		}
	}
}

// The day the schedule falls on is the day it is owed, whatever hour it falls
// at: a card put days away comes back when those days are up.
func TestACardComesBackOnTheDayItsScheduleFallsOn(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	schedules, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	due := schedules[on].Due
	if due.IsZero() {
		t.Fatal("the answer left no schedule")
	}

	// The window is opened on the day the card falls on, and it is owed.
	owed := s.newCountCardsDue(today)
	owed.Now = func() time.Time { return due }
	owing, err := owed.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if owing.Due != 1 {
		t.Errorf("owing = %+v on the day it falls on, want the card owed", owing)
	}

	// And the day before it, it is not.
	owed.Now = func() time.Time { return due.AddDate(0, 0, -1) }
	before, err := owed.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if before.Due != 0 {
		t.Errorf("owing = %+v the day before it falls on, want nothing owed", before)
	}
}

// An answer is written to the vault's own folder, and it is what the next launch
// reads: the card is no longer one nobody has reached.
func TestAnAnswerIsWrittenDownAndReadBack(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	when := time.Now()
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 4*time.Second); err != nil {
		t.Fatal(err)
	}

	schedules, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	got, held := schedules[on]
	if !held {
		t.Fatalf("the answer left no schedule: %+v", schedules)
	}
	if !got.IsSeen() || got.Reps != 1 {
		t.Errorf("schedule = %+v, want one answer behind it", got)
	}
	if !got.Due.After(when) {
		t.Errorf("due %v, which is not after the answer at %v", got.Due, when)
	}
}

// An answer taken back is not counted, and the card is one nobody has reached
// again.
func TestAnAnswerTakenBackLeavesTheCardUnanswered(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	when := time.Now()
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	record := s.run(t, when)
	given, err := record.Answer(t.Context(), on, review.Good, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := record.TakeBack(t.Context(), given.ID); err != nil {
		t.Fatal(err)
	}

	schedules, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, held := schedules[on]; held {
		t.Errorf("the answer was taken back and the schedule stands: %+v", schedules)
	}
}

// A rating outside the four is refused: nothing here guesses what a person meant
// to say about their own recall.
func TestAnAnswerOutsideTheFourIsRefused(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	record := s.run(t, time.Now())

	if _, err := record.Answer(
		t.Context(), review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}, review.Rating(9), 0,
	); err == nil {
		t.Error("a rating of nine was written down")
	}
}

// A session puts the cards owed and answered before in front of the ones nobody
// has reached, and the one waiting longest at the very front.
func TestASessionAsksWhatIsOwedBeforeWhatIsNew(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	long := time.Now().Add(-30 * 24 * time.Hour)
	recent := time.Now().Add(-3 * 24 * time.Hour)

	waited := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	lately := review.CardFaceID{Card: "zpqrstvwxy", Face: "Recognise"}
	if _, err := s.run(t, long).Answer(t.Context(), waited, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, recent).Answer(t.Context(), lately, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	session, err := s.session(today).Execute(t.Context(), s.vault, flashcards.Scope{})
	if err != nil {
		t.Fatal(err)
	}
	asked := session.Queue
	if len(asked) < 2 {
		t.Fatalf("asked %d", len(asked))
	}
	if asked[0].ID != waited {
		t.Errorf("asked %+v first, want the one waiting longest", asked[0].ID)
	}
	for i, one := range asked {
		if one.Schedule.IsSeen() {
			continue
		}
		for _, after := range asked[i:] {
			if after.Schedule.IsSeen() {
				t.Errorf("a card nobody reached stands at %d, in front of one that was", i)
				break
			}
		}
		break
	}
}

// A session over one deck asks that deck's cards and no others.
func TestASessionOverOneDeckAsksThatDeckAlone(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	session, err := s.session(today).Execute(t.Context(), s.vault, flashcards.OverDeck("decks/Words.md"))
	if err != nil {
		t.Fatal(err)
	}
	asked := session.Queue
	if len(asked) != 1 || asked[0].Deck != "decks/Words.md" {
		t.Errorf("asked %+v, want the one card of that deck", asked)
	}
}

// A cache read back says what a replay says, to the last number it carries. It
// stands in for reading the answers, so a cache that answered differently would
// be a card sent away on a day nobody worked out.
func TestACacheReadBackSaysWhatTheAnswersSay(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	when := time.Now().Add(-72 * time.Hour)

	record := s.run(t, when)
	for _, r := range []review.Rating{review.Good, review.Again, review.Hard, review.Easy} {
		if _, err := record.Answer(t.Context(), on, r, time.Second); err != nil {
			t.Fatal(err)
		}
	}

	// The first working out fills the cache; the second is answered from it.
	if _, err := s.kept.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	cached, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	replayed := review.Replay(today, review.NewFSRS(), held.Answers)

	if len(cached) != len(replayed) {
		t.Fatalf("the cache holds %d card faces and the answers say %d", len(cached), len(replayed))
	}
	for face, want := range replayed {
		if got := cached[face]; got != want {
			t.Errorf("the cache says %+v for %+v, the answers say %+v", got, face, want)
		}
	}
}

// A cache of a shape this build does not know is thrown away, because what it
// means is what the build that wrote it meant.
func TestACacheOfAShapeThisBuildDoesNotKnowIsThrownAway(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	invented := review.CardFaceID{Card: "nobodyhasit", Face: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.kept.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	rewrite(t, s, func(was *plantedCache) {
		was.V += 1
		was.Faces = append(was.Faces, plantedFace{
			Card: invented.Card, Face: invented.Face,
			Due: formatTime(time.Now()), Last: formatTime(time.Now()), Reps: 7,
		})
	})

	got, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, held := got[invented]; held {
		t.Error("a cache of another shape was read")
	}
	if got[on].Reps != 1 {
		t.Errorf("the answers were not read again: %+v", got[on])
	}
}

// What was worked out is kept between launches, and a run the cache has not seen
// is the whole history read again.
func TestARunTheCacheHasNotSeenIsCountedIn(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	when := time.Now().Add(-24 * time.Hour)

	if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if first[on].Reps != 1 {
		t.Fatalf("one answer left %d behind it", first[on].Reps)
	}

	// A second run, which is a second file: the kind of thing that arrives from
	// another machine after the first was already counted.
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	second, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if second[on].Reps != 2 {
		t.Errorf("two answers left %d behind them", second[on].Reps)
	}
}

// A session appends to one file all evening, so what was worked out before an
// answer was written is out of date the moment it is. A cache that went by the
// names alone would call itself current and never count the rest of the file.
func TestAnAnswerAppendedToARunAlreadyCountedIsCountedIn(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	record := s.run(t, time.Now())

	if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if first[on].Reps != 1 {
		t.Fatalf("one answer left %d behind it", first[on].Reps)
	}

	// The same session, the same file: a second answer appended under the name
	// the cache already knows.
	if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	second, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if second[on].Reps != 2 {
		t.Errorf("two answers left %d behind them", second[on].Reps)
	}
}

// An answer taken back in the session it was given in is not counted, for the
// same reason: the line is appended to a file already read.
func TestAnAnswerTakenBackInTheSameRunIsNotCounted(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	record := s.run(t, time.Now())

	given, err := record.Answer(t.Context(), on, review.Good, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.kept.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	if _, err := record.TakeBack(t.Context(), given.ID); err != nil {
		t.Fatal(err)
	}
	after, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, held := after[on]; held {
		t.Errorf("the answer was taken back and the schedule stands: %+v", after[on])
	}
}

// A cache filled by another scheduler is thrown away and the answers are read
// again, because the numbers one scheduler carries are its own.
func TestACacheFilledByAnotherSchedulerIsThrownAway(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	invented := review.CardFaceID{Card: "nobodyhasit", Face: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	// A cache another scheduler left, current in every other way and saying
	// what no reading of the answers could: the card answered seven times, and
	// a card face the vault has never held.
	other := s.kept
	other.By = named{Scheduler: review.NewFSRS(), name: "another-one"}
	if _, err := other.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	rewrite(t, s, func(was *plantedCache) {
		was.Faces = append(was.Faces, plantedFace{
			Card: invented.Card, Face: invented.Face,
			Due: formatTime(time.Now()), Last: formatTime(time.Now()), Reps: 1,
		})
		for at := range was.Faces {
			was.Faces[at].Reps = 7
		}
	})

	got, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got[on].Reps != 1 {
		t.Errorf("the answers were not read again: %+v", got[on])
	}
	if _, held := got[invented]; held {
		t.Error("a card face only the other scheduler's cache names came back")
	}
}

// plantedCache is the cache file as a test writes one, which is the shape the
// scheduler reads and no more of it than a test needs.
type plantedCache struct {
	V     int           `json:"v"`
	By    string        `json:"by"`
	Files []plantedFile `json:"files"`
	Faces []plantedFace `json:"faces"`
}

type plantedFile struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type plantedFace struct {
	Card string `json:"card"`
	Face string `json:"face"`
	Due  string `json:"due"`
	Last string `json:"last"`
	Reps int    `json:"reps"`
}

func formatTime(at time.Time) string { return at.UTC().Format(review.Stamp) }

// rewrite changes the cache a vault holds, so that a test can say what a cache
// claims and see whether it was believed.
func rewrite(t *testing.T, s vaulted, change func(*plantedCache)) {
	t.Helper()
	raw, err := s.kept.Cache.Read(t.Context(), s.vault.ID)
	if err != nil {
		t.Fatal(err)
	}
	var was plantedCache
	if err := json.Unmarshal(raw, &was); err != nil {
		t.Fatal(err)
	}
	change(&was)
	now, err := json.Marshal(was)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.kept.Cache.Write(t.Context(), s.vault.ID, now); err != nil {
		t.Fatal(err)
	}
}

// named is a scheduler under another name, which is what a cache is judged
// against.
type named struct {
	review.Scheduler
	name string
}

func (n named) GetName() string { return n.name }

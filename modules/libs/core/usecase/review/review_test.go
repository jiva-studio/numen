package review_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/review"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
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
	standings review.Standings
	kept      review.Schedules
	logs      filesystem.DerivedStores
}

func opened(t *testing.T, notes map[string]string) vaulted {
	t.Helper()
	ctx := t.Context()

	db, err := index.Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := usecase.Scan{
		Readers: filesystem.Readers{}, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.NoteQueries(), Maintenance: db.Statistics(),
	}
	v := testsupport.NewVault(t, notes)
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}
	scanned := func(ctx context.Context, v domain.Vault, _ []string) error {
		_, err := scan.Execute(ctx, v)
		return err
	}

	logs := filesystem.DerivedStores{Area: filesystem.ReviewDir}
	return vaulted{
		vault: v,
		standings: review.Standings{
			Readers: filesystem.Readers{}, Writers: filesystem.Writers{},
			Notes: db.NoteQueries(), Links: db.NoteQueries(),
			Index: scanned, Now: time.Now,
		},
		kept: review.Schedules{
			Logs: logs,
			Kept: appstate.SchedulesAt(filepath.Join(t.TempDir(), "review")),
			By:   history.NewFSRS(),
		},
		logs: logs,
	}
}

func (s vaulted) run(t *testing.T, at time.Time) review.Record {
	t.Helper()
	run, err := review.Log{Stores: s.logs}.Open(t.Context(), s.vault, at)
	if err != nil {
		t.Fatal(err)
	}
	return review.Record{Run: run, Now: func() time.Time { return at }}
}

func (s vaulted) owed(day history.Day) review.Owed {
	return review.Owed{Standings: s.standings, Schedules: s.kept, Day: day, Now: time.Now}
}

func (s vaulted) session(day history.Day) review.Session {
	return review.Session{Standings: s.standings, Schedules: s.kept, Day: day, Now: time.Now}
}

// today is the day a session is counted in, on this machine.
var today = history.Day{Starts: history.DayStarts}

// A card stands once for each face of the stencil that cuts it, because each
// face asks a different thing.
func TestACardStandsOnceForEachFaceOfItsStencil(t *testing.T) {
	s := opened(t, vault)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 5 {
		t.Fatalf("two cards of two faces and one of one face is five, got %d", len(stood))
	}

	faces := map[string]int{}
	for _, one := range stood {
		faces[one.CardFace.Face]++
		front, back := one.Lay()
		if one.CardFace.Card == "" || front == "" || back == "" {
			t.Errorf("nothing laid out for %+v", one.CardFace)
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
	s := opened(t, vault)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range stood {
		if one.CardFace.Card != "k7m2xq9fzp" || one.CardFace.Face != "Name it" {
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

// A deck typed by hand carries no marks. It is written once, which mints one for
// every card in it, and what stands comes from reading it again.
func TestADeckOfCardsWithNoMarksIsGivenThem(t *testing.T) {
	notes := map[string]string{
		"Term.md": vault["Term.md"],
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	}
	s := opened(t, notes)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 {
		t.Fatalf("one card of a one-faced stencil stands once, got %d", len(stood))
	}
	if stood[0].CardFace.Card == "" {
		t.Error("the card came back with no mark to record an answer against")
	}
}

// A card whose wikilink reaches a note that is not a stencil is a card no face
// shows. It is left out, and the rest of the vault is reviewed.
func TestACardWithNoStencilIsLeftOut(t *testing.T) {
	notes := map[string]string{
		"Term.md":    vault["Term.md"],
		"Weather.md": "# Weather\n\nAn ordinary note.\n",
		"decks/Mixed.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n" +
			"\n## Rain ^m9n8b7v6c5\n\n[[Weather]]\n\n### Word\n\nRain\n",
	}
	s := opened(t, notes)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 || stood[0].CardFace.Card != "3f4g5h6j7k" {
		t.Errorf("stands = %+v, want the one card whose stencil is one", stood)
	}
}

// A face missing a front or a back lays out nothing, so nothing is asked through
// it. The stencil's other faces are asked as usual.
func TestAFaceMissingASideIsNotAsked(t *testing.T) {
	notes := map[string]string{
		"Half.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n" +
			"\n## Half a face\n\n### Front\n\n{{Meaning}}\n",
		"decks/Half.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould ^3f4g5h6j7k\n\n[[Half]]\n\n### Word\n\nLeaf mould\n" +
			"\n### Meaning\n\nLeaves\n",
	}
	s := opened(t, notes)

	stood, err := s.standings.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(stood) != 1 || stood[0].CardFace.Face != "Say it" {
		t.Errorf("stands = %+v, want the one face with both its sides", stood)
	}
}

// A card that leaves a field empty is asked like any other: the placeholder lays
// out as nothing, and the card is still somebody's card.
func TestACardLeavingAFieldEmptyIsStillAsked(t *testing.T) {
	notes := map[string]string{
		"Term.md": vault["Term.md"],
		"decks/Empty.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\n",
	}
	s := opened(t, notes)

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
	s := opened(t, vault)

	owing, err := s.owed(today).Execute(t.Context(), s.vault)
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

// An answer is written to the vault's own folder, and it is what the next launch
// reads: the card is no longer one nobody has reached.
func TestAnAnswerIsWrittenDownAndReadBack(t *testing.T) {
	s := opened(t, vault)
	when := time.Now()
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, when).Answer(t.Context(), on, history.Good, 4*time.Second); err != nil {
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
	if !got.Seen() || got.Reps != 1 {
		t.Errorf("schedule = %+v, want one answer behind it", got)
	}
	if !got.Due.After(when) {
		t.Errorf("due %v, which is not after the answer at %v", got.Due, when)
	}
}

// An answer taken back is not counted, and the card is one nobody has reached
// again.
func TestAnAnswerTakenBackLeavesTheCardUnanswered(t *testing.T) {
	s := opened(t, vault)
	when := time.Now()
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	record := s.run(t, when)
	given, err := record.Answer(t.Context(), on, history.Good, time.Second)
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
	s := opened(t, vault)
	record := s.run(t, time.Now())

	if _, err := record.Answer(
		t.Context(), history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}, history.Rating(9), 0,
	); err == nil {
		t.Error("a rating of nine was written down")
	}
}

// A session puts the cards owed and answered before in front of the ones nobody
// has reached, and the one waiting longest at the very front.
func TestASessionAsksWhatIsOwedBeforeWhatIsNew(t *testing.T) {
	s := opened(t, vault)
	long := time.Now().Add(-30 * 24 * time.Hour)
	recent := time.Now().Add(-3 * 24 * time.Hour)

	waited := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	lately := history.CardFace{Card: "zpqrstvwxy", Face: "Recognise"}
	if _, err := s.run(t, long).Answer(t.Context(), waited, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, recent).Answer(t.Context(), lately, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	asked, err := s.session(today).Execute(t.Context(), s.vault, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) < 2 {
		t.Fatalf("asked %d", len(asked))
	}
	if asked[0].CardFace != waited {
		t.Errorf("asked %+v first, want the one waiting longest", asked[0].CardFace)
	}
	for i, one := range asked {
		if one.Schedule.Seen() {
			continue
		}
		for _, after := range asked[i:] {
			if after.Schedule.Seen() {
				t.Errorf("a card nobody reached stands at %d, in front of one that was", i)
				break
			}
		}
		break
	}
}

// A session over one deck asks that deck's cards and no others.
func TestASessionOverOneDeckAsksThatDeckAlone(t *testing.T) {
	s := opened(t, vault)

	asked, err := s.session(today).Execute(t.Context(), s.vault, "decks/Words.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0].Deck != "decks/Words.md" {
		t.Errorf("asked %+v, want the one card of that deck", asked)
	}
}

// What was worked out is kept between launches, and a run the cache has not seen
// is the whole history read again.
func TestARunTheCacheHasNotSeenIsCountedIn(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	when := time.Now().Add(-24 * time.Hour)

	if _, err := s.run(t, when).Answer(t.Context(), on, history.Good, 0); err != nil {
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
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
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

// A cache filled by another scheduler is thrown away and the answers are read
// again, because the numbers one scheduler carries are its own.
func TestACacheFilledByAnotherSchedulerIsThrownAway(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.kept.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	other := s.kept
	other.By = named{Scheduler: history.NewFSRS(), name: "another-one"}
	got, err := other.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got[on].Reps != 1 {
		t.Errorf("the answers were not read again: %+v", got[on])
	}
}

// named is a scheduler under another name, which is what a cache is judged
// against.
type named struct {
	history.Scheduler
	name string
}

func (n named) Name() string { return n.name }

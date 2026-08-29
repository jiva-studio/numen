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
// each of its cards is two seats; the term stencil has one.
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

// standing is one vault on disk, scanned, and the scenarios built over it.
type standing struct {
	vault   domain.Vault
	seats   review.Seats
	kept    review.Schedules
	logs    filesystem.DerivedStores
	scanned func(context.Context, domain.Vault, []string) error
}

func opened(t *testing.T, notes map[string]string) standing {
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
	return standing{
		vault: v,
		seats: review.Seats{
			Readers: filesystem.Readers{}, Writers: filesystem.Writers{},
			Notes: db.NoteQueries(), Links: db.NoteQueries(),
			Index: scanned, Now: time.Now,
		},
		kept: review.Schedules{
			Logs: logs,
			Kept: appstate.SchedulesAt(filepath.Join(t.TempDir(), "review")),
			By:   history.NewFSRS(),
		},
		logs:    logs,
		scanned: scanned,
	}
}

func (s standing) run(t *testing.T, at time.Time) review.Record {
	t.Helper()
	run, err := review.Log{Stores: s.logs}.Open(t.Context(), s.vault, at)
	if err != nil {
		t.Fatal(err)
	}
	return review.Record{Run: run, Now: func() time.Time { return at }}
}

// A card is one seat for each face of the stencil that cuts it, because each
// face asks a different thing.
func TestACardStandsOnceForEachFaceOfItsStencil(t *testing.T) {
	s := opened(t, vault)

	seats, err := s.seats.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(seats) != 5 {
		t.Fatalf("two cards of two faces and one of one is five seats, got %d", len(seats))
	}

	faces := map[string]int{}
	for _, seat := range seats {
		faces[seat.Seat.Face]++
		if seat.Seat.Card == "" || seat.Front == "" || seat.Back == "" {
			t.Errorf("a seat with nothing laid out: %+v", seat)
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
func TestASeatIsLaidOutWithTheCardsOwnValues(t *testing.T) {
	s := opened(t, vault)

	seats, err := s.seats.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, seat := range seats {
		if seat.Seat.Card != "k7m2xq9fzp" || seat.Seat.Face != "Name it" {
			continue
		}
		if seat.Front != "What is about 45\" at the shoulder?" || seat.Back != "Llama" {
			t.Errorf("laid out front %q, back %q", seat.Front, seat.Back)
		}
		if seat.Heading != "Llama" || seat.Section != "The ones with fur" {
			t.Errorf("heading %q under %q", seat.Heading, seat.Section)
		}
		return
	}
	t.Error("the card was not among the seats")
}

// A deck typed by hand carries no marks. It is written once, which mints one
// for every card in it, and the seats come from reading it again.
func TestADeckOfCardsWithNoMarksIsGivenThem(t *testing.T) {
	notes := map[string]string{
		"Term.md": vault["Term.md"],
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	}
	s := opened(t, notes)

	seats, err := s.seats.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(seats) != 1 {
		t.Fatalf("one card of a one-faced stencil is one seat, got %d", len(seats))
	}
	if seats[0].Seat.Card == "" {
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

	seats, err := s.seats.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(seats) != 1 || seats[0].Seat.Card != "3f4g5h6j7k" {
		t.Errorf("seats = %+v, want the one card whose stencil is one", seats)
	}
}

// A vault nobody has answered owes every seat it holds, as a card nobody has
// reached.
func TestAVaultNobodyAnsweredOwesEverySeatAsNew(t *testing.T) {
	s := opened(t, vault)

	owing, err := review.Owed{
		Seats: s.seats, Schedules: s.kept,
		Day: history.Day{Starts: history.DayStarts},
		Now: func() time.Time { return time.Now() },
	}.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if owing.Seats != 5 || owing.New != 5 || owing.Due != 0 {
		t.Errorf("owing = %+v, want five seats and five of them new", owing)
	}
	if len(owing.Decks) != 2 {
		t.Errorf("counted %d decks, want 2", len(owing.Decks))
	}
}

// An answer is written to the vault's own folder, and it is what the next
// launch reads: the card is no longer one nobody has reached.
func TestAnAnswerIsWrittenDownAndReadBack(t *testing.T) {
	s := opened(t, vault)
	when := time.Now()
	seat := history.Seat{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, when).Answer(t.Context(), seat, history.Good, 4*time.Second); err != nil {
		t.Fatal(err)
	}

	schedules, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	got, held := schedules[seat]
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
	seat := history.Seat{Card: "k7m2xq9fzp", Face: "Recognise"}

	record := s.run(t, when)
	given, err := record.Answer(t.Context(), seat, history.Good, time.Second)
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
	if _, held := schedules[seat]; held {
		t.Errorf("the answer was taken back and the schedule stands: %+v", schedules)
	}
}

// A rating outside the four is refused: nothing here guesses what a person
// meant to say about their own recall.
func TestAnAnswerOutsideTheFourIsRefused(t *testing.T) {
	s := opened(t, vault)
	record := s.run(t, time.Now())

	if _, err := record.Answer(
		t.Context(), history.Seat{Card: "k7m2xq9fzp", Face: "Recognise"}, history.Rating(9), 0,
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

	waited := history.Seat{Card: "k7m2xq9fzp", Face: "Recognise"}
	lately := history.Seat{Card: "zpqrstvwxy", Face: "Recognise"}
	if _, err := s.run(t, long).Answer(t.Context(), waited, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, recent).Answer(t.Context(), lately, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	asked, err := review.Session{
		Seats: s.seats, Schedules: s.kept,
		Day: history.Day{Starts: history.DayStarts},
		Now: time.Now,
	}.Execute(t.Context(), s.vault, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) < 2 {
		t.Fatalf("asked %d seats", len(asked))
	}
	if asked[0].Seat != waited {
		t.Errorf("asked %+v first, want the one waiting longest", asked[0].Seat)
	}
	for i, one := range asked {
		if !one.Schedule.Seen() {
			for _, after := range asked[i:] {
				if after.Schedule.Seen() {
					t.Errorf("a card nobody reached stands at %d, in front of one that was", i)
					break
				}
			}
			break
		}
	}
}

// A session over one deck asks that deck's cards and no others.
func TestASessionOverOneDeckAsksThatDeckAlone(t *testing.T) {
	s := opened(t, vault)

	asked, err := review.Session{
		Seats: s.seats, Schedules: s.kept,
		Day: history.Day{Starts: history.DayStarts},
		Now: time.Now,
	}.Execute(t.Context(), s.vault, "decks/Words.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0].Deck != "decks/Words.md" {
		t.Errorf("asked %+v, want the one card of that deck", asked)
	}
}

// What was worked out is kept between launches, and a run the cache has not
// seen is the whole history read again.
func TestARunTheCacheHasNotSeenIsCountedIn(t *testing.T) {
	s := opened(t, vault)
	seat := history.Seat{Card: "k7m2xq9fzp", Face: "Recognise"}
	when := time.Now().Add(-24 * time.Hour)

	if _, err := s.run(t, when).Answer(t.Context(), seat, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if first[seat].Reps != 1 {
		t.Fatalf("one answer left %d behind it", first[seat].Reps)
	}

	// A second run, which is a second file: the kind of thing that arrives from
	// another machine after the first was already counted.
	if _, err := s.run(t, time.Now()).Answer(t.Context(), seat, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	second, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if second[seat].Reps != 2 {
		t.Errorf("two answers left %d behind them", second[seat].Reps)
	}
}

// A cache filled by another scheduler is thrown away and the answers are read
// again, because the numbers one scheduler carries are its own.
func TestACacheFilledByAnotherSchedulerIsThrownAway(t *testing.T) {
	s := opened(t, vault)
	seat := history.Seat{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), seat, history.Good, 0); err != nil {
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
	if got[seat].Reps != 1 {
		t.Errorf("the answers were not read again: %+v", got[seat])
	}
}

// named is a scheduler under another name, which is what a cache is judged
// against.
type named struct {
	history.Scheduler
	name string
}

func (n named) Name() string { return n.name }

package flashcardsui

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// deck is one vault of one stencil and one deck, whose single card stands
// through one face.
var deck = map[string]string{
	"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"decks/Words.md": "---\ntype: deck\n---\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// other is a second vault sharing nothing with the first: another stencil,
// another face, another mark, another word. A claim about one vault not
// reaching another is worth nothing when both hold the same cards.
var other = map[string]string{
	"Bird.md": "---\ntype: stencil\nfields:\n  - Call\n  - Who\n---\n" +
		"\n## Whose call\n\n### Front\n\n{{Call}}\n\n### Back\n\n{{Who}}\n",
	"decks/Calls.md": "---\ntype: deck\n---\n" +
		"\n## Yaffle ^zpqrstvwxy\n\n[[Bird]]\n\n### Call\n\nA laugh across the field\n" +
		"\n### Who\n\nGreen woodpecker\n",
}

// registry is the vaults an installation holds, as the API asks for them.
type registry struct{ held []domain.Vault }

func (r registry) All() ([]domain.Vault, error) { return r.held, nil }
func (r registry) Save(domain.Vault) error      { return nil }
func (r registry) Remove(string) error          { return nil }
func (r registry) Opened(string) error          { return nil }

func (r registry) Find(string) (domain.Vault, bool, error) { return domain.Vault{}, false, nil }
func (r registry) Last() (domain.Vault, bool, error)       { return domain.Vault{}, false, nil }

// windowed is the API as the window builds it, over vaults of a test's own.
func windowed(t *testing.T, vaults ...map[string]string) (*API, []domain.Vault) {
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
	held := make([]domain.Vault, 0, len(vaults))
	for _, notes := range vaults {
		v := testsupport.NewVault(t, notes)
		if _, err := scan.Execute(ctx, v); err != nil {
			t.Fatal(err)
		}
		held = append(held, v)
	}

	// Every location is the test's own: the schedules are a cache, and a test
	// that let it fall to the platform's would fill the machine's.
	cfg := container.Config{
		RegistryPath:  filepath.Join(t.TempDir(), "vaults.json"),
		SchedulesPath: filepath.Join(t.TempDir(), "flashcards"),
	}
	running := cfg.Flashcards(db.NoteQueries(), db.NoteQueries(), nil)
	return &API{
		Registry:  registry{held: held},
		Owed:      running.Owed,
		Session:   running.Session,
		Schedules: running.Schedules,
		Log:       running.Log,
		Counted:   running.Counted,
		Now:       time.Now,
	}, held
}

// started is a sitting opened on one vault, and what it holds to ask.
func started(t *testing.T, api *API, v domain.Vault) *v1.StartResponse {
	t.Helper()
	out, err := api.Start(t.Context(), connect.NewRequest(&v1.StartRequest{VaultId: v.ID}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg
}

// A run belongs to the vault it was opened on. An answer naming another vault's
// run is refused rather than written into a history it has no part in.
func TestARunIsAnsweredOnlyOnTheVaultItWasOpenedOn(t *testing.T) {
	api, held := windowed(t, deck, other)
	one, two := held[0], held[1]

	sitting := started(t, api, one)
	if len(sitting.GetAsked()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := sitting.GetAsked()[0]

	_, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
		VaultId: two.ID,
		Run:     sitting.GetRun(),
		Card:    card.GetCard(),
		Face:    card.GetFace(),
		Rating:  v1.Rating_RATING_GOOD,
	}))
	if err == nil {
		t.Fatal("an answer named against another vault's run was written")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}

	// And nothing of it reached the other vault's folder.
	if held := runs(t, two); len(held) != 0 {
		t.Errorf("the vault holds %v", held)
	}
}

// runs is the files the vault's answers folder holds.
func runs(t *testing.T, v domain.Vault) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(v.Path, ".numen", "flashcards"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// A sitting that is over is over: the run it wrote is never appended to again,
// so a page holding its name from an hour ago writes nothing.
func TestARunIsClosedByTheNextSittingOnItsVault(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	was := started(t, api, v)
	now := started(t, api, v)
	if was.GetRun() == now.GetRun() {
		t.Fatal("a second sitting wrote to the file the first opened")
	}

	card := now.GetAsked()[0]
	_, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
		VaultId: v.ID,
		Run:     was.GetRun(),
		Card:    card.GetCard(),
		Face:    card.GetFace(),
		Rating:  v1.Rating_RATING_GOOD,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("the sitting was over and the answer was refused with %v", connect.CodeOf(err))
	}
	if !errors.Is(err, ErrNoRun) {
		t.Errorf("refused with %v", err)
	}
}

// A vault is answered on its own. What a person did in one is not what another
// owes, and the two vaults here share no mark, no face and no word, so nothing
// could pass between them by looking alike.
func TestAnAnswerInOneVaultLeavesTheOtherOwingWhatItDid(t *testing.T) {
	api, held := windowed(t, deck, other)
	one, two := held[0], held[1]

	before, err := api.Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	was := counted(t, before.Msg, two.ID)

	sitting := started(t, api, one)
	for _, card := range sitting.GetAsked() {
		if _, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
			VaultId: one.ID, Run: sitting.GetRun(),
			Card: card.GetCard(), Face: card.GetFace(),
			Rating: v1.Rating_RATING_EASY,
		})); err != nil {
			t.Fatal(err)
		}
	}

	after, err := api.Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if now := counted(t, after.Msg, two.ID); now.GetNew() != was.GetNew() ||
		now.GetDue() != was.GetDue() || now.GetFaces() != was.GetFaces() {
		t.Errorf("the other vault came to %+v, having come to %+v", now, was)
	}
	if now := counted(t, after.Msg, one.ID); now.GetNew() != 0 || now.GetDue() != 0 {
		t.Errorf("the answered vault still owes %+v", now)
	}
	// Nothing was written into the other vault's folder either.
	if names := runs(t, two); len(names) != 0 {
		t.Errorf("the other vault holds %v", names)
	}
}

// counted is one vault out of what the front door answered.
func counted(t *testing.T, said *v1.OwingResponse, id string) *v1.VaultOwing {
	t.Helper()
	for _, one := range said.GetVaults() {
		if one.GetVaultId() == id {
			return one
		}
	}
	t.Fatalf("the front door did not count %s", id)
	return nil
}

// An answer is written to the run's own file, and the identifier it comes back
// with is what taking it back names.
func TestAnAnswerIsWrittenAndCanBeTakenBack(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	sitting := started(t, api, v)
	card := sitting.GetAsked()[0]

	given, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
		VaultId: v.ID,
		Run:     sitting.GetRun(),
		Card:    card.GetCard(),
		Face:    card.GetFace(),
		Rating:  v1.Rating_RATING_GOOD,
		TookMs:  1200,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if given.Msg.GetAnswer() == "" {
		t.Fatal("the answer came back with nothing to take it back by")
	}

	if _, err := api.TakeBack(t.Context(), connect.NewRequest(&v1.TakeBackRequest{
		VaultId: v.ID,
		Run:     sitting.GetRun(),
		Answer:  given.Msg.GetAnswer(),
	})); err != nil {
		t.Fatal(err)
	}

	if names := runs(t, v); len(names) != 1 {
		t.Errorf("a sitting writes one file, the vault holds %v", names)
	}
}

// An answer outside the four is the caller's mistake and is refused as one.
func TestAnAnswerOutsideTheFourIsRefused(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]
	sitting := started(t, api, v)
	card := sitting.GetAsked()[0]

	_, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
		VaultId: v.ID, Run: sitting.GetRun(),
		Card: card.GetCard(), Face: card.GetFace(),
		Rating: v1.Rating_RATING_UNSPECIFIED,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}

// A vault the index does not carry is counted as nothing and says why, and the
// other vaults are counted all the same.
func TestAVaultNothingHasReadSaysSoAndTheRestAreCounted(t *testing.T) {
	api, held := windowed(t, deck)
	unread := testsupport.NewVault(t, deck)
	api.Registry = registry{held: append(held, unread)}

	out, err := api.Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Msg.GetVaults()) != 2 {
		t.Fatalf("counted %d vaults", len(out.Msg.GetVaults()))
	}
	for _, one := range out.Msg.GetVaults() {
		if one.GetVaultId() == unread.ID {
			if one.GetUnread() == "" {
				t.Error("a vault nothing has read is listed as one holding no cards")
			}
			continue
		}
		if one.GetUnread() != "" || one.GetNew() == 0 {
			t.Errorf("the scanned vault came back %+v", one)
		}
	}
}

// Sitting down to a vault nothing has read is refused, and says which
// application reads a vault.
func TestSittingDownToAVaultNothingHasReadIsRefused(t *testing.T) {
	api, _ := windowed(t)
	unread := testsupport.NewVault(t, deck)
	api.Registry = registry{held: []domain.Vault{unread}}

	_, err := api.Start(t.Context(), connect.NewRequest(&v1.StartRequest{VaultId: unread.ID}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("refused with %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "editor") {
		t.Errorf("says %q, and not which application reads a vault", err)
	}
}

// A question about a vault the installation does not hold is refused.
func TestAQuestionAboutAVaultNobodyHoldsIsRefused(t *testing.T) {
	api, _ := windowed(t)
	_, err := api.Start(t.Context(), connect.NewRequest(&v1.StartRequest{VaultId: "nothing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
	if !errors.Is(err, ErrNoVault) {
		t.Errorf("refused with %v", err)
	}
}

// What each of the four would do to the card is worked out with the card, so a
// person choosing between them is shown what they are choosing between.
func TestACardIsAskedWithWhatEachAnswerWouldDoToIt(t *testing.T) {
	api, held := windowed(t, deck)
	sitting := started(t, api, held[0])
	ahead := sitting.GetAsked()[0].GetAhead()

	if ahead == nil {
		t.Fatal("the card was asked without saying where the four would leave it")
	}
	if !(ahead.GetAgain() < ahead.GetHard() &&
		ahead.GetHard() < ahead.GetGood() &&
		ahead.GetGood() < ahead.GetEasy()) {
		t.Errorf("the four come round at %+v", ahead)
	}
}

// The policy the window is held to, written out here as well as in the handler.
//
// A card is HTML from the person's own vault, and this line is what stands
// behind the allowlist it is drawn through: no script, no form submitted
// anywhere, and no request off the machine.
func TestTheWindowIsHeldToOnePolicy(t *testing.T) {
	const held = "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; " +
		"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
		"form-action 'none'; frame-ancestors 'none'"
	if policy != held {
		t.Errorf("the policy reads %q", policy)
	}

	handler := (&API{}).Serving(http.NotFoundHandler())
	for _, path := range []string{"", "/", "/index.html", "/built/index.css"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.URL.Path = path
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, r)
		if said := out.Header().Get("Content-Security-Policy"); said != held {
			t.Errorf("%q is held to %q", path, said)
		}
	}
}

// The four ratings are carried across as themselves, and nothing else is a
// rating.
func TestTheFourRatingsAreCarriedAcross(t *testing.T) {
	for said, want := range map[v1.Rating]history.Rating{
		v1.Rating_RATING_AGAIN: history.Again,
		v1.Rating_RATING_HARD:  history.Hard,
		v1.Rating_RATING_GOOD:  history.Good,
		v1.Rating_RATING_EASY:  history.Easy,
	} {
		if got := rating(said); got != want {
			t.Errorf("%v came across as %v", said, got)
		}
	}
	if got := rating(v1.Rating_RATING_UNSPECIFIED); got.Valid() {
		t.Errorf("nothing said came across as %v", got)
	}
}

// A deck that could not be given marks is named to the person, so they know
// which of their cards are not in front of them.
func TestASittingSaysWhichDecksItCouldNotMark(t *testing.T) {
	api, held := windowed(t, map[string]string{
		"Term.md": deck["Term.md"],
		// The card carries no mark, so the deck is written to give it one.
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	})
	v := held[0]

	// A deck nothing may write is a deck that keeps its cards out of the
	// sitting, and it is named.
	at := filepath.Join(v.Path, "decks", "Own.md")
	if err := os.Chmod(filepath.Dir(at), 0o500); err != nil {
		t.Skipf("this filesystem does not refuse a write: %v", err)
	}
	t.Cleanup(func() { os.Chmod(filepath.Dir(at), 0o700) })

	sitting := started(t, api, v)
	if len(sitting.GetUnwritten()) != 1 || sitting.GetUnwritten()[0] != "decks/Own.md" {
		t.Errorf("the sitting says %v could not be marked", sitting.GetUnwritten())
	}
	if len(sitting.GetAsked()) != 0 {
		t.Errorf("a card with no mark was asked: %+v", sitting.GetAsked())
	}
}

// Flashcards works on any vault the installation holds without one being opened
// first, because a person owes what they owe across all of them.
func TestEveryVaultIsCountedOnTheFrontDoor(t *testing.T) {
	api, held := windowed(t, deck, other)

	out, err := api.Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Msg.GetVaults()) != len(held) {
		t.Fatalf("counted %d of %d vaults", len(out.Msg.GetVaults()), len(held))
	}
	for _, one := range out.Msg.GetVaults() {
		if one.GetFaces() != 1 || one.GetNew() != 1 {
			t.Errorf("%s comes to %+v", one.GetName(), one)
		}
	}
}

// Counting the front door writes nothing into any vault: a mark is minted when
// a person sits down to a vault, and not for every vault at every launch.
func TestCountingTheFrontDoorWritesIntoNoVault(t *testing.T) {
	handwritten := map[string]string{
		"Term.md": deck["Term.md"],
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	}
	api, held := windowed(t, handwritten)
	at := filepath.Join(held[0].Path, "decks", "Own.md")

	before, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{})); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("the deck was written\n was %q\n now %q", before, after)
	}
}

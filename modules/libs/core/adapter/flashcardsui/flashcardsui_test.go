package flashcardsui

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
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

// registry is the vaults an installation holds, as the API asks for them, and
// which of them a window was last opened on.
type registry struct {
	held []domain.Vault
	last string
}

func (r registry) All() ([]domain.Vault, error) { return r.held, nil }
func (r registry) Save(domain.Vault) error      { return nil }
func (r registry) Remove(string) error          { return nil }
func (r registry) Opened(string) error          { return nil }

func (r registry) Find(string) (domain.Vault, bool, error) { return domain.Vault{}, false, nil }

func (r registry) Last() (domain.Vault, bool, error) {
	for _, v := range r.held {
		if v.ID == r.last {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, nil
}

// windowed is the API as the window builds it, over vaults of a test's own.
func windowed(t testing.TB, vaults ...map[string]string) (*API, []domain.Vault) {
	t.Helper()
	ctx := t.Context()

	// Every location is the test's own: the schedules are a cache, and a test
	// that let it fall to the platform's would fill the machine's.
	cfg := container.Config{
		IndexPath:     filepath.Join(t.TempDir(), "index.db"),
		RegistryPath:  filepath.Join(t.TempDir(), "vaults.json"),
		SchedulesPath: filepath.Join(t.TempDir(), "flashcards"),
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := usecase.Scan{
		Readers: filesystem.Readers{}, Vaults: db.Vaults(),
		Notes: db.NotesCutAt(cfg.Cutting()),
		Known: db.Queries(), Maintenance: db.Maintenance(),
	}
	held := make([]domain.Vault, 0, len(vaults))
	for _, notes := range vaults {
		v := testsupport.NewVault(t, notes)
		if _, err := scan.Execute(ctx, v); err != nil {
			t.Fatal(err)
		}
		held = append(held, v)
	}

	// The window levels the index itself, so what it writes is what the next
	// question is answered from.
	running := cfg.Flashcards(db.Queries(), db.Links(), cfg.Level(db))
	return &API{
		Registry:  registry{held: held},
		Owed:      running.Owed,
		Session:   running.Session,
		Schedules: running.Schedules,
		Log:       running.Log,
		Counted:   running.Counted,
		Joined: flashcards.Around{
			Linked: note.ShowLinks{Links: db.Links()},
			Notes:  db.Queries(),
			Reads:  note.Read{Readers: filesystem.Readers{}},
		},
		Presets: running.Presets,
		Curves:  running.Curves,
		Notes:   db.Queries(),
		Day:     running.Day,
		Now:     time.Now,
	}, held
}

// serving is the window's own client, over a server of the test's own. The
// front door is a stream, and a stream is asked for the way the page asks for
// it.
func serving(t *testing.T, api *API) numenv1connect.FlashcardsServiceClient {
	t.Helper()
	server := httptest.NewServer(api.Serving(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)
}

// front is the whole front door: the vaults it opens on, with each count
// filled into the row it belongs to as it arrives.
func front(t *testing.T, api *API) *v1.OwingResponse {
	t.Helper()
	stream, err := serving(t, api).Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	out, at, first := &v1.OwingResponse{}, map[string]int{}, true
	for stream.Receive() {
		said := stream.Msg()
		if first {
			first = false
			out.Day, out.Vaults = said.GetDay(), said.GetVaults()
			for where, one := range out.GetVaults() {
				at[one.GetVaultId()] = where
			}
			continue
		}
		one := said.GetCounted()
		where, listed := at[one.GetVaultId()]
		if !listed {
			t.Fatalf("a count arrived for %s, which the front door did not list", one.GetVaultId())
		}
		out.Vaults[where] = one
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return out
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

	was := counted(t, front(t, api), two.ID)

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

	after := front(t, api)
	if now := counted(t, after, two.ID); now.GetNew() != was.GetNew() ||
		now.GetDue() != was.GetDue() || now.GetFaces() != was.GetFaces() {
		t.Errorf("the other vault came to %+v, having come to %+v", now, was)
	}
	if now := counted(t, after, one.ID); now.GetNew() != 0 || now.GetDue() != 0 {
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

	out := front(t, api)
	if len(out.GetVaults()) != 2 {
		t.Fatalf("counted %d vaults", len(out.GetVaults()))
	}
	for _, one := range out.GetVaults() {
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

	// The service's own routes are the service's, and reading around a deck is
	// one of them: a procedure that fell through to the files would answer a
	// page where the window expects an answer.
	for _, procedure := range []string{
		numenv1connect.FlashcardsServiceAroundProcedure,
		numenv1connect.FlashcardsServiceOwingProcedure,
	} {
		r := httptest.NewRequest(http.MethodGet, procedure, nil)
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, r)
		if out.Code == http.StatusNotFound {
			t.Errorf("%q is not served", procedure)
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

	out := front(t, api)
	if len(out.GetVaults()) != len(held) {
		t.Fatalf("counted %d of %d vaults", len(out.GetVaults()), len(held))
	}
	for _, one := range out.GetVaults() {
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
	front(t, api)
	after, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("the deck was written\n was %q\n now %q", before, after)
	}
}

// pointed is the one deck of a vault scheduled by a preset of its own.
var pointed = map[string]string{
	"Term.md": deck["Term.md"],
	"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\n---\n\n# Sanskrit\n",
	"decks/Words.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// The front door says what today came to under each preset the vault's decks
// name: what was answered under it, how long that took, and what the day holds.
func TestTheFrontDoorSaysWhatTodayCameToUnderEachPreset(t *testing.T) {
	api, held := windowed(t, pointed)
	v := held[0]

	sitting := started(t, api, v)
	if len(sitting.GetAsked()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := sitting.GetAsked()[0]
	if _, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
		VaultId: v.ID, Run: sitting.GetRun(),
		Card: card.GetCard(), Face: card.GetFace(),
		Rating: v1.Rating_RATING_GOOD, TookMs: 6000,
	})); err != nil {
		t.Fatal(err)
	}

	presets := front(t, api).GetVaults()[0].GetPresets()
	if len(presets) != 1 {
		t.Fatalf("the vault came to %+v, want the one preset", presets)
	}
	one := presets[0]
	if one.GetPreset() != "Sanskrit.md" || one.GetAnswered() != 1 || one.GetTookMs() != 6000 {
		t.Errorf("the day came to %+v", one)
	}
	if one.GetNew() != 8 || one.GetReviews() != 45 || one.GetMinutes() != 20 {
		t.Errorf("the day holds %+v", one)
	}
}

// A deck naming no preset comes under the defaults, which is what a vault
// holding no preset at all comes to.
func TestADeckNamingNoPresetComesUnderTheDefaults(t *testing.T) {
	api, _ := windowed(t, deck)

	presets := front(t, api).GetVaults()[0].GetPresets()
	if len(presets) != 1 {
		t.Fatalf("the vault came to %+v, want the defaults alone", presets)
	}
	one, defaults := presets[0], history.Defaults()
	if one.GetPreset() != "" || one.GetAnswered() != 0 {
		t.Errorf("the day came to %+v", one)
	}
	if one.GetNew() != int32(defaults.NewADay) || one.GetReviews() != int32(defaults.ReviewsADay) {
		t.Errorf("the day holds %+v", one)
	}
}

// twoPresets is one vault studying two subjects: a preset of one card a day, a
// preset of two, and a deck of three cards under each.
var twoPresets = map[string]string{
	"Term.md": deck["Term.md"],
	"Roots.md": "---\ntype: preset\ngoal: retention\nnew_a_day: 1\nreviews_a_day: 0\n" +
		"minutes_a_day: 0\n---\n\n# Roots\n",
	"Mantras.md": "---\ntype: preset\ngoal: retention\nnew_a_day: 2\nreviews_a_day: 0\n" +
		"minutes_a_day: 0\n---\n\n# Mantras\n",
	"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Roots\n    role: ref\n    type: preset\n---\n" +
		"\n## bhu ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n" +
		"\n## gam ^zpqrstvwxy\n\n[[Term]]\n\n### Word\n\ngam\n\n### Meaning\n\nto go\n" +
		"\n## kr ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nkr\n\n### Meaning\n\nto do\n",
	"decks/Mantras.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Mantras\n    role: ref\n    type: preset\n---\n" +
		"\n## one ^m4n5b6v7c8\n\n[[Term]]\n\n### Word\n\ngayatri\n\n### Meaning\n\na metre\n" +
		"\n## two ^q1w2e3r4t5\n\n[[Term]]\n\n### Word\n\nmaha\n\n### Meaning\n\ngreat\n" +
		"\n## three ^y6u7i8o9p0\n\n[[Term]]\n\n### Word\n\nsanti\n\n### Meaning\n\npeace\n",
}

// A sitting over a vault of two presets is the union of them: each deck is held
// to its own preset's budget, and one preset running out closes its own decks.
func TestASittingOverTwoPresetsIsTheUnionOfTheirBudgets(t *testing.T) {
	api, held := windowed(t, twoPresets)

	got := make(map[string]int)
	for _, one := range started(t, api, held[0]).GetAsked() {
		got[one.GetDeck()]++
	}
	want := map[string]int{"decks/Roots.md": 1, "decks/Mantras.md": 2}
	for path, cards := range want {
		if got[path] != cards {
			t.Errorf("%s was asked %d cards, want %d", path, got[path], cards)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the sitting held %v, want %v", got, want)
	}
}

// The front door says what each of a vault's presets holds today, and neither
// of them carries the other's budget.
func TestTheFrontDoorSaysWhatEachPresetOfAVaultHolds(t *testing.T) {
	api, _ := windowed(t, twoPresets)

	got := make(map[string]int32)
	for _, one := range front(t, api).GetVaults()[0].GetPresets() {
		got[one.GetPreset()] = one.GetNew()
	}
	want := map[string]int32{"Mantras.md": 2, "Roots.md": 1}
	for path, cards := range want {
		if got[path] != cards {
			t.Errorf("%s holds %d new cards a day, want %d", path, got[path], cards)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the vault came to %v, want %v", got, want)
	}
}

// carded is a deck of as many cards, pointing at the preset named.
// Each deck is written from a mark of its own, because a mark is what names a
// card and two decks writing one mark are writing one card.
func carded(at string, cards, from int) string {
	out := "---\ntype: deck\n"
	if at != "" {
		out += "links:\n  - to: " + at + "\n    role: ref\n    type: preset\n"
	}
	out += "---\n"
	for i := from; i < from+cards; i++ {
		out += fmt.Sprintf(
			"\n## Card %d ^card%06d\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n",
			i, i, i, i)
	}
	return out
}

// backlogged is a vault of more material than a day of its preset carries, and
// a deck beside it on the defaults.
var backlogged = map[string]string{
	"Term.md": deck["Term.md"],
	"Steady.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 10\n" +
		"new_a_day: 12\nreviews_a_day: 1\nretention: 0.9\ncounts: cards\n---\n\n# Steady\n",
	"decks/Steady.md": carded("Steady", 60, 0),
	"decks/Loose.md":  carded("", 20, 1000),
}

// The deck screen and the preset tab are one arithmetic, under every goal.
//
// The window asks Owing for what its decks offer today and Curve for the
// picture over a preset's range, and both are asked here as the window asks
// them: the settings the curve is drawn for are the ones the vault holds. Where
// the preset stands on that picture is the day the deck screen offers wherever
// the control moves today, and the reason a day closed names the budget its own
// goal steers and no other.
func TestTheDeckScreenAndThePresetTabAgreeUnderEveryGoal(t *testing.T) {
	for _, one := range []struct {
		what   string
		goal   history.Goal
		by     time.Time
		closed history.Closed
		never  []history.Closed
		// learned is the rule the goal is worked out against, where the goal
		// reads one. Six days are too few to carry a card past an interval of
		// three weeks, and a date paces the day for the cards that can get
		// there.
		learned history.Rule
		// sameDay is whether the control moves today, so that what the tab
		// draws where it stands is the day the deck screen offers.
		sameDay bool
	}{
		{
			what: "minutes", goal: history.GoalMinutes,
			closed: history.ClosedMinutes, sameDay: true,
			never: []history.Closed{
				history.ClosedNew, history.ClosedReviews, history.ClosedDate,
			},
		},
		{
			what: "retention", goal: history.GoalRetention,
			closed: history.ClosedNew,
			never:  []history.Closed{history.ClosedMinutes, history.ClosedDate},
		},
		{
			what: "a date", goal: history.GoalDate, learned: history.RuleRetention,
			by: time.Now().AddDate(0, 0, 6), closed: history.ClosedDate, sameDay: true,
			never: []history.Closed{
				history.ClosedNew, history.ClosedReviews, history.ClosedMinutes,
			},
		},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, held := windowed(t, backlogged)
			v := held[0]

			p := asWritten(t, api, v, "Steady.md")
			p.Goal, p.By = one.goal, one.by
			if one.learned != "" {
				p.Rule = one.learned
			}
			writtenBack(t, api, v, "Steady.md", p)

			offers := 0
			for _, deck := range owing(t, api, v).GetDecks() {
				if deck.GetDeck() == "decks/Steady.md" {
					offers = int(deck.GetDue() + deck.GetNew())
				}
			}
			if offers == 0 {
				t.Fatal("the deck screen offers nothing to compare")
			}

			drawn := pictured(t, api, v, "Steady.md", p)
			at := drawn.GetNow().GetAt()
			if at < 0 {
				t.Fatalf("the preset stands nowhere on its own curve: %+v", drawn.GetNow())
			}
			if got := int(drawn.GetAt()[at].GetReviews()); one.sameDay && got != offers {
				t.Errorf("the deck screen offers %d cards and the tab draws %d", offers, got)
			}
			if got := drawn.GetAt()[at].GetClosed(); !slices.Contains(got, string(one.closed)) {
				t.Errorf("the day closed on %q, want %q among them", got, one.closed)
			}
			for i, point := range drawn.GetAt() {
				for _, never := range one.never {
					if slices.Contains(point.GetClosed(), string(never)) {
						t.Errorf("at %v the day closed on %q, which its goal does not name",
							drawn.GetGrid()[i], never)
					}
				}
			}
		})
	}
}

// Under a goal of minutes a day long enough for the whole of the material is
// closed by nothing: the material itself ran out, and no card limit is named.
func TestALongEnoughDayIsClosedByNothing(t *testing.T) {
	api, held := windowed(t, backlogged)
	v := held[0]

	drawn := pictured(t, api, v, "Steady.md", asWritten(t, api, v, "Steady.md"))
	if got := drawn.GetAt()[len(drawn.GetAt())-1].GetClosed(); len(got) != 0 {
		t.Errorf("the longest day on the range is closed by %q", got)
	}
}

// The day suggested under a goal of minutes means what it says: the shortest
// day that asks everything the day holds, on a vault carrying a backlog.
//
// A shorter day leaves cards standing, so no place before it asks as much, and
// the shortest day on the range is never the answer to a vault behind on its
// reviews.
func TestTheSuggestedDayIsTheShortestThatAsksEverything(t *testing.T) {
	api, held := windowed(t, lived)
	v := held[0]
	lives(t, api, v, 14)
	standing(api, firstMorning.AddDate(0, 0, 14))

	p := asWritten(t, api, v, "Sanskrit.md")
	p.Goal = history.GoalMinutes
	writtenBack(t, api, v, "Sanskrit.md", p)

	drawn := pictured(t, api, v, "Sanskrit.md", p)
	at := int(drawn.GetSuggested().GetAt())
	if at < 0 {
		t.Fatalf("nothing is suggested: %+v", drawn.GetSuggested())
	}
	if at == 0 {
		t.Error("the shortest day on the range is suggested, and this vault is behind")
	}

	whole := drawn.GetAt()[len(drawn.GetAt())-1].GetReviews()
	if got := drawn.GetAt()[at].GetReviews(); got != whole {
		t.Errorf("the suggested day asks %v cards, and a day of any length asks %v", got, whole)
	}
	if got := drawn.GetAt()[at].GetClosed(); len(got) != 0 {
		t.Errorf("the suggested day closed on %q, and a day that asks everything closes on nothing",
			got)
	}
	for i := range at {
		if drawn.GetAt()[i].GetReviews() >= whole {
			t.Errorf("%v minutes a day already asks %v cards, and %v is suggested",
				drawn.GetGrid()[i], drawn.GetAt()[i].GetReviews(), drawn.GetGrid()[at])
		}
	}
}

// What is overdue is a fact about the vault, and how long it takes to clear is
// a fact about the setting being chosen.
//
// The overdue count stands over the whole curve and does not move when the goal
// does. What the deck screen owes today is that backlog and the cards falling
// due today besides, so it is never the smaller of the two.
func TestWhatIsOverdueStandsOverTheWholeCurve(t *testing.T) {
	api, held := windowed(t, lived)
	v := held[0]
	lives(t, api, v, 14)
	standing(api, firstMorning.AddDate(0, 0, 14))

	owes := 0
	for _, one := range owing(t, api, v).GetDecks() {
		for _, deck := range underSanskrit {
			if one.GetDeck() == deck {
				owes += int(one.GetDue())
			}
		}
	}

	p := asWritten(t, api, v, "Sanskrit.md")
	p.Goal = history.GoalMinutes
	writtenBack(t, api, v, "Sanskrit.md", p)
	byMinutes := pictured(t, api, v, "Sanskrit.md", p)

	overdue := int(byMinutes.GetOverdue())
	if overdue == 0 {
		t.Fatal("nothing stands overdue, and this vault was left alone for days")
	}
	if overdue > owes {
		t.Errorf("%d card faces stand overdue and the deck screen owes %d today",
			overdue, owes)
	}

	// The same vault under another goal is the same backlog.
	p.Goal = history.GoalRetention
	if got := int(pictured(t, api, v, "Sanskrit.md", p).GetOverdue()); got != overdue {
		t.Errorf("steered by its retention the same vault stands %d overdue, and by its "+
			"minutes %d", got, overdue)
	}

	// A shorter day is longer about clearing it, or never gets there at all.
	short, long := byMinutes.GetAt()[0], byMinutes.GetAt()[len(byMinutes.GetAt())-1]
	if long.GetClears() <= 0 {
		t.Fatalf("the longest day on the range clears the backlog in %d days",
			long.GetClears())
	}
	if short.GetClears() != -1 && short.GetClears() <= long.GetClears() {
		t.Errorf("%v minutes a day clears in %d and %v minutes a day in %d",
			byMinutes.GetGrid()[0], short.GetClears(),
			byMinutes.GetGrid()[len(byMinutes.GetGrid())-1], long.GetClears())
	}
}

// A vault nobody has answered has nothing overdue, and no place of its curve
// has a backlog to clear.
func TestAVaultNobodyHasAnsweredHasNothingOverdue(t *testing.T) {
	api, held := windowed(t, backlogged)
	v := held[0]

	drawn := pictured(t, api, v, "Steady.md", asWritten(t, api, v, "Steady.md"))
	if got := drawn.GetOverdue(); got != 0 {
		t.Errorf("%d card faces stand overdue in a vault nobody has answered", got)
	}
	for i, one := range drawn.GetAt() {
		if one.GetClears() != 0 {
			t.Errorf("%v minutes a day clears nothing in %d days",
				drawn.GetGrid()[i], one.GetClears())
		}
	}
}

// spread is a lived-in vault of three scopes: a preset closing its day on the
// minutes, one closing it on the counts, and the defaults over two decks, so a
// budget shared between decks is one of the three.
var spread = map[string]string{
	"Both.md": bothWays,
	"One.md":  oneWay,
	// Its counts stand at what an earlier goal left them, and its minutes are
	// what it is steered by now.
	"Timed.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 6\n" +
		"new_a_day: 2\nreviews_a_day: 1\nretention: 0.9\ncounts: cards\n---\n\n# Timed\n",
	"Counted.md": "---\ntype: preset\ngoal: retention\nretention: 0.9\n" +
		"new_a_day: 8\nreviews_a_day: 30\nminutes_a_day: 15\ncounts: cards\n---\n\n# Counted\n",
	"decks/Verbs.md": written("Both", "Timed", 100, 0),
	"decks/Nouns.md": written("One", "Timed", 80, 1000),
	"decks/Roots.md": written("One", "Counted", 100, 2000),
	"decks/Loose.md": written("One", "", 60, 3000),
	"decks/Odds.md":  written("One", "", 60, 4000),
}

// The tile over a preset and the sitting it opens are one number.
//
// A preset is the whole scope of its own budget, so what the count leaves under
// it is what a sitting over it asks. The count carries that figure, because a
// window working one out of the budget would be reading limits the goal may not
// even name.
func TestThePresetTileAndTheSittingItOpensAreOneNumber(t *testing.T) {
	api, held := windowed(t, spread)
	v := held[0]
	lives(t, api, v, 14)
	standing(api, firstMorning.AddDate(0, 0, 14))

	said := owing(t, api, v)
	// Which decks each preset schedules, so what a sitting asks can be checked
	// against the rows it was gathered from.
	under := make(map[string][]string)
	rows := make(map[string]*v1.DeckOwing, len(said.GetDecks()))
	for _, one := range said.GetDecks() {
		rows[one.GetDeck()] = one
		p, err := api.Presets.Of(t.Context(), v, one.GetDeck())
		if err != nil {
			t.Fatal(err)
		}
		under[p.Path] = append(under[p.Path], one.GetDeck())
	}

	scopes, clamped := 0, 0
	for _, one := range said.GetPresets() {
		if one.GetCards() == 0 {
			continue
		}
		scopes++
		want := int(one.GetOwedDue() + one.GetOwedNew())
		if want == 0 {
			t.Errorf("the tile over %q offers nothing to compare", one.GetPreset())
		}

		sat, err := api.Start(t.Context(), connect.NewRequest(&v1.StartRequest{
			VaultId: v.ID, Preset: naming(one.GetPreset()),
		}))
		if err != nil {
			t.Fatal(err)
		}
		if got := len(sat.Msg.GetAsked()); got != want {
			t.Errorf("the tile over %q says %d and the sitting it opens asks %d",
				one.GetPreset(), want, got)
		}

		// What the count leaves under a preset is what its decks were left,
		// gathered by the one pass.
		due, fresh := 0, 0
		for _, deck := range under[one.GetPreset()] {
			due += int(rows[deck].GetDue())
			fresh += int(rows[deck].GetNew())
		}
		if due != int(one.GetOwedDue()) || fresh != int(one.GetOwedNew()) {
			t.Errorf("the decks under %q come to %d owed and %d new, the preset to %d and %d",
				one.GetPreset(), due, fresh, one.GetOwedDue(), one.GetOwedNew())
		}

		// A figure worked out from the budget instead would be held to counts
		// the goal need not name, and this vault is where that shows.
		if min(due, int(one.GetReviews()))+min(fresh, int(one.GetNew())) < want {
			clamped++
		}
	}
	if scopes != 3 {
		t.Errorf("the vault holds %d scopes with cards under them, want three", scopes)
	}
	if clamped == 0 {
		t.Error("no preset here is held short by its counts, and the tile's arithmetic is not under test")
	}
}

// The count never allocates past the budget it is spending.
//
// A budget the goal does not name is not the one being spent: it stands in the
// file as the person left it, and a day may hand out far more than it says. A
// budget the goal does name is a wall, and nothing goes over it.
func TestTheCountNeverAllocatesPastTheBudgetItSpends(t *testing.T) {
	for _, one := range []struct {
		what  string
		notes map[string]string
	}{{"lived", lived}, {"spread", spread}} {
		t.Run(one.what, func(t *testing.T) {
			api, held := windowed(t, one.notes)
			v := held[0]
			lives(t, api, v, 14)
			now := firstMorning.AddDate(0, 0, 14)
			standing(api, now)

			past := 0
			for _, said := range owing(t, api, v).GetPresets() {
				if said.GetCards() == 0 {
					continue
				}
				due, fresh := int(said.GetOwedDue()), int(said.GetOwedNew())
				kept, keptNew := int(said.GetReviews()), int(said.GetNew())

				if said.GetClosesReviews() != "" && due > kept {
					t.Errorf("%q allocated %d owed against the %d reviews it keeps",
						said.GetPreset(), due, kept)
				}
				if said.GetClosesNew() != "" && fresh > keptNew {
					t.Errorf("%q allocated %d new against the %d it keeps",
						said.GetPreset(), fresh, keptNew)
				}
				// A count standing in the file that the goal does not name is
				// no wall, and this is the vault where that shows.
				if said.GetClosesReviews() == "" && due > kept {
					past++
				}
				if said.GetClosesNew() == "" && fresh > keptNew {
					past++
				}
			}
			if past == 0 {
				t.Error("no preset here hands out past a budget its goal leaves idle, " +
					"and the distinction is not under test")
			}
		})
	}
}

// The curve carries the pile as it stands and the pile day by day, so a band
// under it can be drawn without asking again as the knob moves.
//
// Overdue, unbegun and the cards whose day is still to come are three separate
// counts of the one material, and the band begins where the overdue pile does.
func TestTheCurveCarriesTheBacklogDayByDay(t *testing.T) {
	api, held := windowed(t, lived)
	v := held[0]
	lives(t, api, v, 14)
	standing(api, firstMorning.AddDate(0, 0, 14))

	p := asWritten(t, api, v, "Sanskrit.md")
	p.Goal = history.GoalMinutes
	writtenBack(t, api, v, "Sanskrit.md", p)
	drawn := pictured(t, api, v, "Sanskrit.md", p)

	overdue, unbegun, cards := drawn.GetOverdue(), drawn.GetUnbegun(), drawn.GetCards()
	if overdue == 0 || unbegun == 0 {
		t.Fatalf("the vault stands %d overdue and %d unbegun, and neither is under test",
			overdue, unbegun)
	}
	if overdue+unbegun > cards {
		t.Errorf("%d overdue and %d unbegun of %d card faces", overdue, unbegun, cards)
	}

	for i, one := range drawn.GetAt() {
		band := one.GetBacklog()
		if len(band) == 0 {
			t.Fatalf("%v carries no backlog to draw", drawn.GetGrid()[i])
		}
		// A day too short to carry what falls due adds to the pile, so the
		// band rises as well as falls, and never below nothing.
		for day, standing := range band {
			if standing < 0 {
				t.Errorf("%v stands %d behind on day %d", drawn.GetGrid()[i], standing, day)
			}
		}
		// The day named as the clearing is the day the band reaches nothing.
		first := int32(-1)
		for day, standing := range band {
			if standing == 0 {
				first = int32(day) + 1
				break
			}
		}
		if one.GetClears() != first {
			t.Errorf("%v clears on day %d and its band reaches nothing on day %d",
				drawn.GetGrid()[i], one.GetClears(), first)
		}
	}

	// A longer day is never further behind than a shorter one on the same day.
	short, long := drawn.GetAt()[0].GetBacklog(), drawn.GetAt()[len(drawn.GetAt())-1].GetBacklog()
	for day := range min(len(short), len(long)) {
		if long[day] > short[day] {
			t.Errorf("on day %d the longest day on the range stands %d behind and the "+
				"shortest %d", day, long[day], short[day])
		}
	}
}

// A curve of retention plots what the target costs week after week.
//
// A target does nothing to today, so the height is the load over the days
// projected: it climbs with the target where nothing binds, and stands still at
// every place where a count binds at every one of them.
func TestTheRetentionCurvePlotsWhatTheTargetCosts(t *testing.T) {
	for _, one := range []struct {
		what             string
		newADay, reviews int
		climbs           bool
	}{
		// A day of one review is a day the target cannot spend, whatever it is.
		{"a count binding at every place", 12, 1, false},
		{"counts that never bind", 50, 200, true},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, held := windowed(t, lived)
			v := held[0]
			lives(t, api, v, 14)
			standing(api, firstMorning.AddDate(0, 0, 14))

			p := asWritten(t, api, v, "Sanskrit.md")
			p.Goal = history.GoalRetention
			p.NewADay, p.ReviewsADay = one.newADay, one.reviews
			writtenBack(t, api, v, "Sanskrit.md", p)
			drawn := pictured(t, api, v, "Sanskrit.md", p)

			least := drawn.GetAt()[0].GetMinutes()
			most := drawn.GetAt()[len(drawn.GetAt())-1].GetMinutes()
			if least <= 0 {
				t.Fatalf("the easiest target costs %v minutes a day", least)
			}
			switch {
			case one.climbs && most <= least*2:
				t.Errorf("asking for %.2f of it back costs %v minutes a day and asking "+
					"for %.2f costs %v", drawn.GetGrid()[len(drawn.GetGrid())-1], most,
					drawn.GetGrid()[0], least)
			case !one.climbs && most != least:
				t.Errorf("a day the count closes at every place costs %v minutes at one "+
					"end of the range and %v at the other", least, most)
			}

			// What the cost buys climbs with it, and is a share of the material.
			for i, at := range drawn.GetAt() {
				if at.GetRetained() < 0 || at.GetRetained() > 1 {
					t.Errorf("%.2f leaves %v of the material in the head",
						drawn.GetGrid()[i], at.GetRetained())
				}
			}
			kept := drawn.GetAt()[len(drawn.GetAt())-1].GetRetained() -
				drawn.GetAt()[0].GetRetained()
			if one.climbs && kept <= 0 {
				t.Errorf("asking for more of it back kept %v more of it", kept)
			}
		})
	}
}

// sat is what a sitting over one preset came to: how many cards, and how they
// divided between the debt and the material it had not begun.
type sat struct{ asked, owed, fresh int }

// sitting opens a sitting over one preset, the way pressing its tile does.
func sitting(t *testing.T, api *API, v domain.Vault, preset string) sat {
	t.Helper()
	out, err := api.Start(t.Context(), connect.NewRequest(&v1.StartRequest{
		VaultId: v.ID, Preset: naming(preset),
	}))
	if err != nil {
		t.Fatal(err)
	}
	held := sat{asked: len(out.Msg.GetAsked())}
	for _, one := range out.Msg.GetAsked() {
		if !one.GetSeen() {
			held.fresh++
			continue
		}
		held.owed++
	}
	return held
}

// The share of a day that goes to the debt moves the sitting and the projection
// alike, and moves them together.
//
// A share is read where a day is spent, which is one place, so the cards a
// person is asked for and the cards the picture is drawn from are the same
// cards. Were the projection to read its own rule, moving the share would
// change tomorrow's sitting and leave the curve and the overdue band standing.
func TestTheBacklogShareMovesTheSittingAndTheProjectionTogether(t *testing.T) {
	held := make(map[int]sat, 2)
	bands := make(map[int][]int32, 2)

	for _, share := range []int{100, 0} {
		api, vaults := windowed(t, lived)
		v := vaults[0]
		lives(t, api, v, 20)
		standing(api, firstMorning.AddDate(0, 0, 20))

		p := asWritten(t, api, v, "Sanskrit.md")
		p.Goal, p.Backlog = history.GoalMinutes, share
		writtenBack(t, api, v, "Sanskrit.md", p)

		// The control is left on a place of its own grid, so the picture is
		// drawn for the day the vault is held to.
		drawn := pictured(t, api, v, "Sanskrit.md", p)
		value := drawn.GetGrid()[2]
		p.MinutesADay = int(value)
		writtenBack(t, api, v, "Sanskrit.md", p)

		drawn = pictured(t, api, v, "Sanskrit.md", p)
		at := 2
		if drawn.GetGrid()[at] != value {
			t.Fatalf("the control was left at %v and the grid holds %v there",
				value, drawn.GetGrid()[at])
		}

		one := sitting(t, api, v, "Sanskrit.md")
		if got := int(drawn.GetAt()[at].GetReviews()); got != one.asked {
			t.Errorf("giving the debt %d of the day, the sitting asks %d cards and the "+
				"picture draws %d", share, one.asked, got)
		}
		held[share], bands[share] = one, drawn.GetAt()[at].GetBacklog()
	}

	// Each share put its own side of the day first: paying the debt filled the
	// day with it, and putting it last spent the day on new cards.
	if held[100].owed == 0 || held[0].fresh == 0 {
		t.Fatalf("paying the debt first asked %+v and putting it last %+v, and neither "+
			"side is under test", held[100], held[0])
	}
	// The two shares are two different days, in the sitting and in the picture
	// both. A setting inert in either half would show as one of these matching.
	if held[100].owed == held[0].owed {
		t.Errorf("paying the debt first asked %+v and putting it last %+v",
			held[100], held[0])
	}
	if held[100].owed <= held[0].owed {
		t.Errorf("paying the debt first asked %d owed and putting it last %d",
			held[100].owed, held[0].owed)
	}
	if slices.Equal(bands[100], bands[0]) {
		t.Error("the overdue band is the same whether the debt is paid first or last")
	}
}

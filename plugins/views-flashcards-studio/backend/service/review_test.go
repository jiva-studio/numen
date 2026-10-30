package service

import (
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1/flashcardsv1connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/presetwire"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/schematest"
)

// deck is one vault of one card model and one deck, whose single card stands
// through one review card.
var deck = map[string]string{
	"Term.md": "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"decks/Words.md": "---\ntype: deck\n---\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// other is a second vault sharing nothing with the first: another card model,
// another review card, another card id, another word. A claim about one vault not
// reaching another is worth nothing when both hold the same cards.
var other = map[string]string{
	"Bird.md": "---\ntype: card-model\nfields:\n  - Call\n  - Who\n---\n" +
		"\n## Whose call\n\n### Front\n\n{{Call}}\n\n### Back\n\n{{Who}}\n",
	"decks/Calls.md": "---\ntype: deck\n---\n" +
		"\n## Yaffle ^zpqrstvwxy\n\n[[Bird]]\n\n### Call\n\nA laugh across the field\n" +
		"\n### Who\n\nGreen woodpecker\n",
}

// API is the review as these tests ask it.
type API = reviewHandler

// newAPI is the review over vaults of a test's own, all of them known to the host.
func newAPI(t testing.TB, notes ...map[string]string) (*API, []domain.Vault) {
	t.Helper()
	api, target, _ := newAPIOn(t, notes...)
	return api, target
}

// newAPIOn is newAPI together with the host the review works through.
func newAPIOn(t testing.TB, notes ...map[string]string) (*API, []domain.Vault, vaulttest.Installation) {
	t.Helper()
	installation := vaulttest.NewInstallation()
	target := make([]domain.Vault, 0, len(notes))
	for _, one := range notes {
		target = append(target, installation.Add(t, one))
	}
	backend := New(Config{Vault: installation.Host, Store: installation.Host.Store()})
	return &reviewHandler{Backend: backend}, target, installation
}

// newFlashcardsClient returns a test client connected to the service.
func newFlashcardsClient(t *testing.T, api *API) flashcardsv1connect.FlashcardsServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(flashcardsv1connect.NewFlashcardsServiceHandler(api))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return flashcardsv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)
}

// front polls WatchCardsDue until all vaults complete indexing.
func front(t *testing.T, api *API) *flashcardsv1.WatchCardsDueResponse {
	t.Helper()
	client := newFlashcardsClient(t, api)

	var out *flashcardsv1.WatchCardsDueResponse
	for pos := time.Now(); time.Since(pos) < 30*time.Second; {
		out = readCardsDue(t, client)
		if !slices.ContainsFunc(out.GetVaults(), (*flashcardsv1.VaultCardsDue).GetIsIndexing) {
			return out
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("a vault was still being read")
	return out
}

// readCardsDue collects the full response from WatchCardsDue stream.
func readCardsDue(
	t *testing.T, client flashcardsv1connect.FlashcardsServiceClient,
) *flashcardsv1.WatchCardsDueResponse {
	t.Helper()
	stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, stream)

	out, pos, first := &flashcardsv1.WatchCardsDueResponse{}, map[string]int{}, true
	for stream.Receive() {
		response := stream.Msg()
		if first {
			first = false
			out.Day, out.Vaults = response.GetDay(), response.GetVaults()
			for where, one := range out.GetVaults() {
				pos[one.GetId()] = where
			}
			continue
		}
		one := response.GetCounted()
		if one == nil {
			// The list again, which the stream says while a count is still
			// running so that a page which has gone is found.
			continue
		}
		where, items := pos[one.GetId()]
		if !items {
			t.Fatalf("unexpected vault %s in count response", one.GetId())
		}
		out.Vaults[where] = one
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// startSession is a session opened on one vault, and what it holds to ask.
func startSession(t *testing.T, api *API, v domain.Vault) *flashcardsv1.StartSessionResponse {
	t.Helper()
	out, err := api.StartSession(t.Context(),
		connect.NewRequest(&flashcardsv1.StartSessionRequest{Vault: string(v.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg
}

// A run belongs to the vault it was opened on, and an answer naming another
// vault's run fails.
func TestARunIsAnsweredOnlyOnTheVaultItWasOpenedOn(t *testing.T) {
	api, target, installation := newAPIOn(t, deck, other)
	one, two := target[0], target[1]

	session := startSession(t, api, one)
	if len(session.GetCards()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := session.GetCards()[0]

	_, err := api.AnswerCard(t.Context(), connect.NewRequest(&flashcardsv1.AnswerCardRequest{
		Vault:    string(two.ID),
		Run:      session.GetRun(),
		CardId:   card.GetCardId(),
		Template: card.GetTemplate(),
		Rating:   flashcardsv1.Rating_RATING_GOOD,
	}))
	if err == nil {
		t.Fatal("an answer named against another vault's run was written")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}

	// And nothing of it reached the other vault's folder.
	if targetVal := getRunFiles(t, installation.Host.Store().With(two.ID)); len(targetVal) != 0 {
		t.Errorf("the vault holds %v", targetVal)
	}
}

// getRunFiles is the files the plugin's folder of a vault holds.
func getRunFiles(t *testing.T, store host.Store) []string {
	t.Helper()
	entries, err := store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name)
	}
	return out
}

// A session that is over is over: the run it wrote is never appended to again,
// so a page holding its name from an hour ago writes nothing.
func TestARunIsClosedByTheNextSessionOnItsVault(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]

	was := startSession(t, api, v)
	now := startSession(t, api, v)
	if was.GetRun() == now.GetRun() {
		t.Fatal("a second session wrote to the file the first opened")
	}

	card := now.GetCards()[0]
	_, err := api.AnswerCard(t.Context(), connect.NewRequest(&flashcardsv1.AnswerCardRequest{
		Vault:    string(v.ID),
		Run:      was.GetRun(),
		CardId:   card.GetCardId(),
		Template: card.GetTemplate(),
		Rating:   flashcardsv1.Rating_RATING_GOOD,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("the session was over and the answer failed with %v", connect.CodeOf(err))
	}
	if !errors.Is(err, ErrNoRun) {
		t.Errorf("failed with %v", err)
	}
}

// A vault is answered on its own. What a person did in one is not what another
// owes, and the two vaults here share no card id, no review card and no word, so nothing
// could pass between them by looking alike.
func TestAnAnswerInOneVaultPreservesTheOtherOwingWhatItDid(t *testing.T) {
	api, target, installation := newAPIOn(t, deck, other)
	one, two := target[0], target[1]

	was := findVaultCount(t, front(t, api), string(two.ID))

	session := startSession(t, api, one)
	for _, card := range session.GetCards() {
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&flashcardsv1.AnswerCardRequest{
			Vault: string(one.ID), Run: session.GetRun(),
			CardId: card.GetCardId(), Template: card.GetTemplate(),
			Rating: flashcardsv1.Rating_RATING_EASY,
		})); err != nil {
			t.Fatal(err)
		}
	}

	after := front(t, api)
	if now := findVaultCount(t, after, string(two.ID)); now.GetNew() != was.GetNew() ||
		now.GetDue() != was.GetDue() || now.GetReviewCards() != was.GetReviewCards() {
		t.Errorf("the other vault came to %+v, having come to %+v", now, was)
	}
	if now := findVaultCount(t, after, string(one.ID)); now.GetNew() != 0 || now.GetDue() != 0 {
		t.Errorf("the answered vault still owes %+v", now)
	}
	// Nothing was written into the other vault's folder either.
	if names := getRunFiles(t, installation.Host.Store().With(two.ID)); len(names) != 0 {
		t.Errorf("the other vault holds %v", names)
	}
}

// findVaultCount returns the vault matching id from the response.
func findVaultCount(t *testing.T, resp *flashcardsv1.WatchCardsDueResponse, id string) *flashcardsv1.VaultCardsDue {
	t.Helper()
	for _, one := range resp.GetVaults() {
		if one.GetId() == id {
			return one
		}
	}
	t.Fatalf("vault %s not found in response", id)
	return nil
}

// An answer is written to the run's own file, and the identifier it comes back
// with is what taking it back names.
func TestAnAnswerIsSavedAndCanBeTakenBack(t *testing.T) {
	api, target, installation := newAPIOn(t, deck)
	v := target[0]

	session := startSession(t, api, v)
	card := session.GetCards()[0]

	provided, err := api.AnswerCard(t.Context(), connect.NewRequest(&flashcardsv1.AnswerCardRequest{
		Vault:    string(v.ID),
		Run:      session.GetRun(),
		CardId:   card.GetCardId(),
		Template: card.GetTemplate(),
		Rating:   flashcardsv1.Rating_RATING_GOOD,
		TookMs:   1200,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if provided.Msg.GetAnswer() == "" {
		t.Fatal("the answer came back with nothing to take it back by")
	}

	if _, err := api.TakeBackAnswer(t.Context(), connect.NewRequest(&flashcardsv1.TakeBackAnswerRequest{
		Vault:  string(v.ID),
		Run:    session.GetRun(),
		Answer: provided.Msg.GetAnswer(),
	})); err != nil {
		t.Fatal(err)
	}

	if names := getRunFiles(t, installation.Host.Store().With(v.ID)); len(names) != 1 {
		t.Errorf("a session writes one file, the vault holds %v", names)
	}
}

// An answer outside the four is the caller's mistake and fails as one.
func TestAnAnswerOutsideTheFourIsRefused(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]
	session := startSession(t, api, v)
	card := session.GetCards()[0]

	_, err := api.AnswerCard(t.Context(), connect.NewRequest(&flashcardsv1.AnswerCardRequest{
		Vault: string(v.ID), Run: session.GetRun(),
		CardId: card.GetCardId(), Template: card.GetTemplate(),
		Rating: flashcardsv1.Rating_RATING_UNSPECIFIED,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
}

// Every vault the review counts is in the index before it is counted: a vault being read into the
// index has nothing to show, and its numbers arrive with the count that follows.
func TestAVaultIsIndexedBeforeItIsCounted(t *testing.T) {
	api, target, installation := newAPIOn(t, deck, deck)
	installation.Of(target[1].ID).SetIndexState(host.IndexState{Status: host.Indexing})

	// Nothing is counted from a walk half done, so the row of a vault being read says so.
	rows := map[string]*flashcardsv1.VaultCardsDue{}
	for _, one := range readCardsDue(t, newFlashcardsClient(t, api)).GetVaults() {
		rows[one.GetId()] = one
	}
	if got := rows[string(target[1].ID)]; !got.GetIsIndexing() || got.GetUnread() != "" {
		t.Errorf("a vault being read came back %+v", got)
	}
	if got := rows[string(target[0].ID)]; got.GetIsIndexing() || got.GetNew() == 0 {
		t.Errorf("a vault the index carries came back %+v", got)
	}

	// The reading finishes and the count that follows it holds the cards.
	installation.Of(target[1].ID).SetIndexState(host.IndexState{Status: host.Indexed})
	for _, one := range front(t, api).GetVaults() {
		if one.GetUnread() != "" || one.GetNew() == 0 {
			t.Errorf("a vault came back %+v", one)
		}
	}
}

// A vault the index could not read is uncounted, and says why nothing could be counted.
func TestAVaultTheIndexCouldNotReadIsUncountedAndReturnsWhy(t *testing.T) {
	api, target, installation := newAPIOn(t, deck)
	installation.Of(target[0].ID).SetIndexState(host.IndexState{Status: host.IndexFailed, Reason: "the disk is full"})

	one := front(t, api).GetVaults()[0]

	if one.GetIsIndexing() || one.GetUnread() != "the disk is full" {
		t.Errorf("a vault nothing could read came back %+v", one)
	}
}

// A session opened on a vault the index does not carry fails, and says so.
func TestSessionDownToAVaultTheIndexDoesNotCarryIsRefused(t *testing.T) {
	api, target, installation := newAPIOn(t, deck)
	installation.Of(target[0].ID).SetIndexState(host.IndexState{Status: host.Indexing})

	_, err := api.StartSession(t.Context(),
		connect.NewRequest(&flashcardsv1.StartSessionRequest{Vault: string(target[0].ID)}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("failed with %v: %v", connect.CodeOf(err), err)
	}
	if !errors.Is(err, flashcards.ErrNotCarried) {
		t.Errorf("says %q, and not that the index does not carry the vault", err)
	}
}

// A question about a vault the installation does not hold fails.
func TestAQuestionAboutAVaultNobodyContainsIsRefused(t *testing.T) {
	api, _ := newAPI(t)
	_, err := api.StartSession(t.Context(),
		connect.NewRequest(&flashcardsv1.StartSessionRequest{Vault: "nothing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
	if !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("failed with %v", err)
	}
}

// What each of the four would do to the card is worked out with the card, so a
// person choosing between them is shown what they are choosing between.
func TestACardIsAskedWithWhatEachAnswerWouldDoToIt(t *testing.T) {
	api, target := newAPI(t, deck)
	session := startSession(t, api, target[0])
	ahead := session.GetCards()[0].GetIntervals()

	if ahead == nil {
		t.Fatal("the card was asked without saying where the four would leave it")
	}
	if !(ahead.GetAgain() < ahead.GetHard() &&
		ahead.GetHard() < ahead.GetGood() &&
		ahead.GetGood() < ahead.GetEasy()) {
		t.Errorf("the four come round at %+v", ahead)
	}
}

// The four ratings are carried across as themselves, and nothing else is a
// rating.
func TestTheFourRatingsAreCarriedAcross(t *testing.T) {
	for response, want := range map[flashcardsv1.Rating]review.Rating{
		flashcardsv1.Rating_RATING_AGAIN: review.Again,
		flashcardsv1.Rating_RATING_HARD:  review.Hard,
		flashcardsv1.Rating_RATING_GOOD:  review.Good,
		flashcardsv1.Rating_RATING_EASY:  review.Easy,
	} {
		if got := parseRating(response); got != want {
			t.Errorf("%v came across as %v", response, got)
		}
	}
	if got := parseRating(flashcardsv1.Rating_RATING_UNSPECIFIED); got.Valid() {
		t.Errorf("nothing said came across as %v", got)
	}
}

// A deck that could not be given card ids is named to the person, so they know
// which of their cards are not in front of them.
func TestASessionReturnsWhichDecksItCouldNotMark(t *testing.T) {
	api, target, installation := newAPIOn(t, map[string]string{
		"Term.md": deck["Term.md"],
		// The card carries no card id, so the deck is written to give it one.
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	})
	v := target[0]

	// A deck nothing may write is a deck that keeps its cards out of the
	// session, and it is named.
	installation.Of(v.ID).RefuseWriting("decks/Own.md", fs.ErrPermission)

	session := startSession(t, api, v)
	if len(session.GetUnwritten()) != 1 || session.GetUnwritten()[0] != "decks/Own.md" {
		t.Errorf("the session says %v could not be marked", session.GetUnwritten())
	}
	if len(session.GetCards()) != 0 {
		t.Errorf("a card with no card id was asked: %+v", session.GetCards())
	}
}

// TestEveryVaultIsCountedOnTheFrontDoor verifies all vaults are reported.
func TestEveryVaultIsCountedOnTheFrontDoor(t *testing.T) {
	api, target := newAPI(t, deck, other)

	out := front(t, api)
	if len(out.GetVaults()) != len(target) {
		t.Fatalf("counted %d of %d vaults", len(out.GetVaults()), len(target))
	}
	for _, one := range out.GetVaults() {
		if one.GetReviewCards() != 1 || one.GetNew() != 1 {
			t.Errorf("got %s = %+v, want Review cards=1, New=1", one.GetName(), one)
		}
	}
}

// Counting cards does not modify vault files.
func TestCountingTheFrontDoorWritesIntoNoVault(t *testing.T) {
	handwritten := map[string]string{
		"Term.md": deck["Term.md"],
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	}
	api, target := newAPI(t, handwritten)
	pos := filepath.Join(target[0].Path, "decks", "Own.md")

	before, err := os.ReadFile(pos)
	if err != nil {
		t.Fatal(err)
	}
	front(t, api)
	after, err := os.ReadFile(pos)
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

// WatchCardsDue returns daily progress and limits for each preset.
func TestTheFrontDoorReturnsWhatTodayCameToBelowEachPreset(t *testing.T) {
	api, target := newAPI(t, pointed)
	v := target[0]

	session := startSession(t, api, v)
	if len(session.GetCards()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := session.GetCards()[0]
	if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&flashcardsv1.AnswerCardRequest{
		Vault: string(v.ID), Run: session.GetRun(),
		CardId: card.GetCardId(), Template: card.GetTemplate(),
		Rating: flashcardsv1.Rating_RATING_GOOD, TookMs: 6000,
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
// holding no preset at all evaluates to.
func TestADeckNamingNoPresetReachesBelowTheDefaults(t *testing.T) {
	api, _ := newAPI(t, deck)

	presets := front(t, api).GetVaults()[0].GetPresets()
	if len(presets) != 1 {
		t.Fatalf("the vault came to %+v, want the defaults alone", presets)
	}
	one, defaults := presets[0], review.Defaults()
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

// A session over a vault of two presets is the union of them: each deck is held
// to its own preset's budget, and one preset running out closes its own decks.
func TestASessionOverTwoPresetsIsTheUnionOfTheirBudgets(t *testing.T) {
	api, target := newAPI(t, twoPresets)

	got := make(map[string]int)
	for _, one := range startSession(t, api, target[0]).GetCards() {
		got[one.GetDeck()]++
	}
	want := map[string]int{"decks/Roots.md": 1, "decks/Mantras.md": 2}
	for path, cards := range want {
		if got[path] != cards {
			t.Errorf("%s was asked %d cards, want %d", path, got[path], cards)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the session held %v, want %v", got, want)
	}
}

// WatchCardsDue returns settings for each preset independently.
func TestTheFrontDoorReturnsWhatEachPresetOfAVaultContains(t *testing.T) {
	api, _ := newAPI(t, twoPresets)

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

// newDeck is a deck of as many cards, pointing at the preset named.
// Each deck is written from a card id of its own, because a card id is what names a
// card and two decks writing one card id are writing one card.
func newDeck(pos string, cards, from int) string {
	out := "---\ntype: deck\n"
	if pos != "" {
		out += "links:\n  - to: " + pos + "\n    role: ref\n    type: preset\n"
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
	"decks/Steady.md": newDeck("Steady", 60, 0),
	"decks/Loose.md":  newDeck("", 20, 1000),
}

// The deck screen and the preset tab are one arithmetic, under every goal.
//
// The window asks CardsDue for what its decks offer today and Curve for the
// picture over a preset's range, and both are asked here as the window asks
// them: the settings the curve is drawn for are the ones the vault holds. Where
// the preset relies on that picture is the day the deck screen offers wherever
// the control moves today, and the reason a day closed names the budget its own
// goal steers and no other.
func TestTheDeckScreenAndThePresetTabAgreeBelowEveryGoal(t *testing.T) {
	for _, one := range []struct {
		what   string
		goal   review.Goal
		byDate time.Time
		closed review.BudgetName
		never  []review.BudgetName
		// learned is the rule the goal is worked out against, where the goal
		// reads one. Six days are too few to carry a card past an interval of
		// three weeks, and a date paces the day for the cards that can get
		// there.
		learned review.LearnedRule
		// isSameDay is whether the control moves today, so that what the tab
		// draws at its current position is the day the deck screen offers.
		isSameDay bool
	}{
		{
			what: "minutes", goal: review.GoalMinutes,
			closed: review.ClosedMinutes, isSameDay: true,
			never: []review.BudgetName{
				review.ClosedNew, review.ClosedReviews, review.ClosedDate,
			},
		},
		{
			what: "retention", goal: review.GoalRetention,
			closed: review.ClosedNew,
			never:  []review.BudgetName{review.ClosedMinutes, review.ClosedDate},
		},
		{
			what: "a date", goal: review.GoalDate, learned: review.RuleRetention,
			byDate: time.Now().AddDate(0, 0, 6), closed: review.ClosedDate, isSameDay: true,
			never: []review.BudgetName{
				review.ClosedNew, review.ClosedReviews, review.ClosedMinutes,
			},
		},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, target := newAPI(t, backlogged)
			v := target[0]

			p := readPresetConfig(t, api, v, "Steady.md")
			p.Goal, p.By = one.goal, one.byDate
			if one.learned != "" {
				p.Rule = one.learned
			}
			writePreset(t, api, v, "Steady.md", p)

			offers := 0
			for _, deck := range getVaultCount(t, api, v).GetDecks() {
				if deck.GetDeck() == "decks/Steady.md" {
					offers = int(deck.GetDue() + deck.GetNew())
				}
			}
			if offers == 0 {
				t.Fatal("the deck screen offers nothing to compare")
			}

			curveResp := getCurve(t, api, v, "Steady.md", p)
			pos := curveResp.GetNow().GetAt()
			if pos < 0 {
				t.Fatalf("the preset stands nowhere on its own curve: %+v", curveResp.GetNow())
			}
			if got := int(curveResp.GetAt()[pos].GetReviews()); one.isSameDay && got != offers {
				t.Errorf("the deck screen offers %d cards and the tab draws %d", offers, got)
			}
			if got := curveResp.GetAt()[pos].GetClosed(); !slices.Contains(got, presetwire.ConvertBudgetToWire(one.closed)) {
				t.Errorf("the day closed on %v, want %q among them", got, one.closed)
			}
			for i, point := range curveResp.GetAt() {
				for _, never := range one.never {
					if slices.Contains(point.GetClosed(), presetwire.ConvertBudgetToWire(never)) {
						t.Errorf("at %v the day closed on %q, which its goal does not name",
							curveResp.GetGrid()[i], never)
					}
				}
			}
		})
	}
}

// Under a goal of minutes a day long enough for the whole of the material is
// closed by nothing: the material itself ran out, and no card limit is named.
func TestALongEnoughDayIsClosedByNothing(t *testing.T) {
	api, target := newAPI(t, backlogged)
	v := target[0]

	curveResp := getCurve(t, api, v, "Steady.md", readPresetConfig(t, api, v, "Steady.md"))
	if got := curveResp.GetAt()[len(curveResp.GetAt())-1].GetClosed(); len(got) != 0 {
		t.Errorf("the longest day on the range is closed by %q", got)
	}
}

// The day suggested under a goal of minutes means what it says: the shortest
// day that asks everything the day holds, on a vault carrying a backlog.
//
// A shorter day is cut short by the clock, and the shortest day on the range is
// never the answer to a vault behind on its reviews. The cards a shorter day
// asks may already be the whole count: a card the day comes back to is the one
// card, and the showings it takes are what the clock runs out on.
func TestTheSuggestedDayIsTheShortestThatAsksEverything(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
	runDays(t, api, v, 14)
	setNow(api, firstMorning.AddDate(0, 0, 14))

	p := readPresetConfig(t, api, v, "Sanskrit.md")
	p.Goal = review.GoalMinutes
	writePreset(t, api, v, "Sanskrit.md", p)

	curveResp := getCurve(t, api, v, "Sanskrit.md", p)
	pos := int(curveResp.GetSuggested().GetAt())
	if pos < 0 {
		t.Fatalf("nothing is suggested: %+v", curveResp.GetSuggested())
	}
	if pos == 0 {
		t.Error("the shortest day on the range is suggested, and this vault is behind")
	}

	whole := curveResp.GetAt()[len(curveResp.GetAt())-1].GetReviews()
	if got := curveResp.GetAt()[pos].GetReviews(); got != whole {
		t.Errorf("the suggested day asks %v cards, and a day of any length asks %v", got, whole)
	}
	if got := curveResp.GetAt()[pos].GetClosed(); len(got) != 0 {
		t.Errorf("the suggested day closed on %v, and a day that asks everything closes on nothing",
			got)
	}
	for i := range pos {
		got := curveResp.GetAt()[i].GetClosed()
		if !slices.Contains(got, presetwire.ConvertBudgetToWire(review.ClosedMinutes)) {
			t.Errorf("%v minutes a day closed on %v, and %v is suggested",
				curveResp.GetGrid()[i], got, curveResp.GetGrid()[pos])
		}
	}
}

// What is overdue is a fact about the vault, and how long it takes to clear is
// a fact about the setting being chosen.
//
// The overdue count applies across the whole curve and does not change with the goal.
func TestWhatIsOverdueExistsOverTheWholeCurve(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
	runDays(t, api, v, 14)
	setNow(api, firstMorning.AddDate(0, 0, 14))

	owes := 0
	for _, one := range getVaultCount(t, api, v).GetDecks() {
		for _, deck := range underSanskrit {
			if one.GetDeck() == deck {
				owes += int(one.GetDue())
			}
		}
	}

	p := readPresetConfig(t, api, v, "Sanskrit.md")
	p.Goal = review.GoalMinutes
	writePreset(t, api, v, "Sanskrit.md", p)
	byMinutes := getCurve(t, api, v, "Sanskrit.md", p)

	overdue := int(byMinutes.GetOverdue())
	if overdue == 0 {
		t.Fatal("got overdue == 0, want overdue > 0 for abandoned vault")
	}
	if overdue > owes {
		t.Errorf("got overdue = %d, want <= %d (deck due count)", overdue, owes)
	}

	// The same vault under another goal is the same backlog.
	p.Goal = review.GoalRetention
	if got := int(getCurve(t, api, v, "Sanskrit.md", p).GetOverdue()); got != overdue {
		t.Errorf("got overdue = %d for GoalRetention, want %d (from GoalMinutes)", got, overdue)
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

// A vault with no review history has nothing overdue.
func TestAVaultNobodyHasAnsweredHasNothingOverdue(t *testing.T) {
	api, target := newAPI(t, backlogged)
	v := target[0]

	curveResp := getCurve(t, api, v, "Steady.md", readPresetConfig(t, api, v, "Steady.md"))
	if got := curveResp.GetOverdue(); got != 0 {
		t.Errorf("got overdue = %d, want 0", got)
	}
	for i, one := range curveResp.GetAt() {
		if one.GetClears() != 0 {
			t.Errorf("at %v minutes a day: got clears = %d, want 0",
				curveResp.GetGrid()[i], one.GetClears())
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
	"decks/Verbs.md": newCardModelDeck("Both", "Timed", 100, 0),
	"decks/Nouns.md": newCardModelDeck("One", "Timed", 80, 1000),
	"decks/Roots.md": newCardModelDeck("One", "Counted", 100, 2000),
	"decks/Loose.md": newCardModelDeck("One", "", 60, 3000),
	"decks/Odds.md":  newCardModelDeck("One", "", 60, 4000),
}

// The tile over a preset and the session it opens are one number.
//
// A preset is the whole scope of its own budget, so what the count leaves under
// it is what a session over it asks, and the count carries that figure.
func TestThePresetTileAndTheSessionItOpensAreOneNumber(t *testing.T) {
	api, target := newAPI(t, spread)
	v := target[0]
	runDays(t, api, v, 14)
	setNow(api, firstMorning.AddDate(0, 0, 14))

	response := getVaultCount(t, api, v)
	// Which decks each preset schedules, so what a session asks can be checked
	// against the rows it was gathered from.
	underMap := make(map[string][]string)
	rows := make(map[string]*flashcardsv1.DeckCardsDue, len(response.GetDecks()))
	for _, one := range response.GetDecks() {
		rows[one.GetDeck()] = one
		p, err := api.scenarios().flash.Presets.GetForDeck(t.Context(), v, one.GetDeck())
		if err != nil {
			t.Fatal(err)
		}
		underMap[p.Path] = append(underMap[p.Path], one.GetDeck())
	}

	scopes, clamped := 0, 0
	for _, one := range response.GetPresets() {
		if one.GetCards() == 0 {
			continue
		}
		scopes++
		want := int(one.GetOwedDue() + one.GetOwedNew())
		if want == 0 {
			t.Errorf("the tile over %q offers nothing to compare", one.GetPreset())
		}

		sessionResp, err := api.StartSession(t.Context(), connect.NewRequest(&flashcardsv1.StartSessionRequest{
			Vault: string(v.ID), Preset: newPresetName(one.GetPreset()),
		}))
		if err != nil {
			t.Fatal(err)
		}
		if got := len(sessionResp.Msg.GetCards()); got != want {
			t.Errorf("deck %q: got session card count = %d, want %d",
				one.GetPreset(), got, want)
		}

		// What the count leaves under a preset is what its decks were left,
		// gathered by the one pass.
		due, newCards := 0, 0
		for _, deck := range underMap[one.GetPreset()] {
			due += int(rows[deck].GetDue())
			newCards += int(rows[deck].GetNew())
		}
		if due != int(one.GetOwedDue()) || newCards != int(one.GetOwedNew()) {
			t.Errorf("deck %q: got due=%d, new=%d; want preset OwedDue=%d, OwedNew=%d",
				one.GetPreset(), due, newCards, one.GetOwedDue(), one.GetOwedNew())
		}

		// Budget allocation restricts counts when configured.
		if min(due, int(one.GetReviews()))+min(newCards, int(one.GetNew())) < want {
			clamped++
		}
	}
	if scopes != 3 {
		t.Errorf("got scopes = %d, want 3", scopes)
	}
	if clamped == 0 {
		t.Error("no preset clamped by counts")
	}
}

// The count never allocates past the budget it is spending.
//
// Only the budget active for the selected goal constrains daily allocations.
func TestTheCountNeverAllocatesPastTheBudgetItSpends(t *testing.T) {
	for _, one := range []struct {
		what  string
		notes map[string]string
	}{{"lived", lived}, {"spread", spread}} {
		t.Run(one.what, func(t *testing.T) {
			api, target := newAPI(t, one.notes)
			v := target[0]
			runDays(t, api, v, 14)
			now := firstMorning.AddDate(0, 0, 14)
			setNow(api, now)

			past := 0
			for _, response := range getVaultCount(t, api, v).GetPresets() {
				if response.GetCards() == 0 {
					continue
				}
				due, newCards := int(response.GetOwedDue()), int(response.GetOwedNew())
				keptReviews, keptNew := int(response.GetReviews()), int(response.GetNew())

				if response.GetClosesReviews() != "" && due > keptReviews {
					t.Errorf("%q allocated %d owed against the %d reviews it keeps",
						response.GetPreset(), due, keptReviews)
				}
				if response.GetClosesNew() != "" && newCards > keptNew {
					t.Errorf("%q allocated %d new against the %d it keeps",
						response.GetPreset(), newCards, keptNew)
				}
				// A count standing in the file that the goal does not name is
				// no wall, and this is the vault where that shows.
				if response.GetClosesReviews() == "" && due > keptReviews {
					past++
				}
				if response.GetClosesNew() == "" && newCards > keptNew {
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

// The curve carries the pile currently and the pile day by day, so a band
// under it can be drawn without asking again as the knob moves.
//
// Overdue, unbegun and the cards whose day is still to come are three separate
// counts of the one material, and the band begins where the overdue pile does.
func TestTheCurveCarriesTheBacklogDayByDay(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
	runDays(t, api, v, 14)
	setNow(api, firstMorning.AddDate(0, 0, 14))

	p := readPresetConfig(t, api, v, "Sanskrit.md")
	p.Goal = review.GoalMinutes
	writePreset(t, api, v, "Sanskrit.md", p)
	curveResp := getCurve(t, api, v, "Sanskrit.md", p)

	overdue, unbegun, cards := curveResp.GetOverdue(), curveResp.GetUnbegun(), curveResp.GetCards()
	if overdue == 0 || unbegun == 0 {
		t.Fatalf("the vault stands %d overdue and %d unbegun, and neither is under test",
			overdue, unbegun)
	}
	if overdue+unbegun > cards {
		t.Errorf("%d overdue and %d unbegun of %d review cards", overdue, unbegun, cards)
	}

	for i, one := range curveResp.GetAt() {
		band := one.GetBacklog()
		if len(band) == 0 {
			t.Fatalf("%v carries no backlog to draw", curveResp.GetGrid()[i])
		}
		// A day too short to carry what falls due adds to the pile, so the
		// band rises as well as falls, and never below nothing.
		for day, backlog := range band {
			if backlog < 0 {
				t.Errorf("%v stands %d behind on day %d", curveResp.GetGrid()[i], backlog, day)
			}
		}
		// The day named as the clearing is the day the band reaches nothing.
		first := int32(-1)
		for day, backlog := range band {
			if backlog == 0 {
				first = int32(day) + 1
				break
			}
		}
		if one.GetClears() != first {
			t.Errorf("%v clears on day %d and its band reaches nothing on day %d",
				curveResp.GetGrid()[i], one.GetClears(), first)
		}
	}

	// A longer day is never further behind than a shorter one on the same day.
	short, long := curveResp.GetAt()[0].GetBacklog(), curveResp.GetAt()[len(curveResp.GetAt())-1].GetBacklog()
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
		counts           review.BudgetUnit
		isClimbing       bool
	}{
		// A budget of one showing is a day the target cannot spend, whatever it
		// is. Counting cards it could: the one card is asked again for nothing,
		// and how often it comes back is the target's own answer.
		{"a count binding at every place", 12, 1, review.BudgetUnitShows, false},
		{"counts that never bind", 50, 200, review.BudgetUnitCards, true},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, target := newAPI(t, lived)
			v := target[0]
			runDays(t, api, v, 14)
			setNow(api, firstMorning.AddDate(0, 0, 14))

			p := readPresetConfig(t, api, v, "Sanskrit.md")
			p.Goal = review.GoalRetention
			p.NewADay, p.ReviewsADay = one.newADay, one.reviews
			p.Counts = one.counts
			writePreset(t, api, v, "Sanskrit.md", p)
			curveResp := getCurve(t, api, v, "Sanskrit.md", p)

			least := curveResp.GetAt()[0].GetMinutes()
			most := curveResp.GetAt()[len(curveResp.GetAt())-1].GetMinutes()
			if least <= 0 {
				t.Fatalf("the easiest target costs %v minutes a day", least)
			}
			switch {
			case one.isClimbing && most <= least*2:
				t.Errorf("asking for %.2f of it back costs %v minutes a day and asking "+
					"for %.2f costs %v", curveResp.GetGrid()[len(curveResp.GetGrid())-1], most,
					curveResp.GetGrid()[0], least)
			case !one.isClimbing && most != least:
				t.Errorf("a day the count closes at every place costs %v minutes at one "+
					"end of the range and %v at the other", least, most)
			}

			// What the cost buys climbs with it, and is a share of the material.
			for i, item := range curveResp.GetAt() {
				if item.GetRetained() < 0 || item.GetRetained() > 1 {
					t.Errorf("%.2f leaves %v of the material in the head",
						curveResp.GetGrid()[i], item.GetRetained())
				}
			}
			retentionDelta := curveResp.GetAt()[len(curveResp.GetAt())-1].GetRetained() -
				curveResp.GetAt()[0].GetRetained()
			if one.isClimbing && retentionDelta <= 0 {
				t.Errorf("asking for more of it back kept %v more of it", retentionDelta)
			}
		})
	}
}

// sessionCounts holds review counts for a session.
type sessionCounts struct{ total, due, unbegun int }

// session opens a session for one preset.
func session(t *testing.T, api *API, v domain.Vault, preset string) sessionCounts {
	t.Helper()
	out, err := api.StartSession(t.Context(), connect.NewRequest(&flashcardsv1.StartSessionRequest{
		Vault: string(v.ID), Preset: newPresetName(preset),
	}))
	if err != nil {
		t.Fatal(err)
	}
	target := sessionCounts{total: len(out.Msg.GetCards())}
	for _, one := range out.Msg.GetCards() {
		if one.GetIsNew() {
			target.unbegun++
			continue
		}
		target.due++
	}
	return target
}

// The backlog share moves the session and the projection together.
func TestTheBacklogShareMovesTheSessionAndTheProjectionTogether(t *testing.T) {
	target := make(map[int]sessionCounts, 2)
	bands := make(map[int][]int32, 2)

	for _, share := range []int{100, 0} {
		api, made := newAPI(t, lived)
		v := made[0]
		runDays(t, api, v, 20)
		setNow(api, firstMorning.AddDate(0, 0, 20))

		p := readPresetConfig(t, api, v, "Sanskrit.md")
		p.Goal, p.Backlog = review.GoalMinutes, share
		writePreset(t, api, v, "Sanskrit.md", p)

		// Evaluate curve for the preset.
		curveResp := getCurve(t, api, v, "Sanskrit.md", p)
		value := curveResp.GetGrid()[2]
		p.MinutesADay = int(value)
		writePreset(t, api, v, "Sanskrit.md", p)

		curveResp = getCurve(t, api, v, "Sanskrit.md", p)
		pos := 2
		if curveResp.GetGrid()[pos] != value {
			t.Fatalf("got grid[%d] = %v, want %v", pos, curveResp.GetGrid()[pos], value)
		}

		one := session(t, api, v, "Sanskrit.md")
		if got := int(curveResp.GetAt()[pos].GetReviews()); got != one.total {
			t.Errorf("share %d: got reviews = %d, want %d", share, got, one.total)
		}
		target[share], bands[share] = one, curveResp.GetAt()[pos].GetBacklog()
	}

	if target[100].due == 0 || target[0].unbegun == 0 {
		t.Fatalf("target[100].due = %d, target[0].unbegun = %d, want non-zero",
			target[100].due, target[0].unbegun)
	}
	if target[100].due == target[0].due {
		t.Errorf("target[100].due == target[0].due (%d), want different",
			target[100].due)
	}
	if target[100].due <= target[0].due {
		t.Errorf("got target[100].due = %d <= target[0].due = %d, want >",
			target[100].due, target[0].due)
	}
	if slices.Equal(bands[100], bands[0]) {
		t.Error("bands[100] and bands[0] are equal, want different")
	}
}

// TestEveryRatingIsRead. A rating the reading does not name comes through as
// zero, which the use case refuses; a person who pressed a button would be told
// the answer was not one.
func TestEveryRatingIsRead(t *testing.T) {
	schematest.CheckHandled(t, func(r flashcardsv1.Rating) bool { return parseRating(r) != 0 })
}

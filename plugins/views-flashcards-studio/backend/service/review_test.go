package service

import (
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/enumtest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/presetwire"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// deck is one vault of one card model and one deck, whose single card yields one
// review card.
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
	handler, target, _ := newAPIOn(t, notes...)
	return handler, target
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

// newFlashcardsClient returns the client the window uses over the review.
func newFlashcardsClient(_ *testing.T, handler *API) reviewClient {
	return reviewClient{reviewHandler: handler}
}

// front polls WatchCardsDue until all vaults complete indexing.
func front(t *testing.T, handler *API) *api.WatchCardsDueResponse {
	t.Helper()
	client := newFlashcardsClient(t, handler)

	var out *api.WatchCardsDueResponse
	for pos := time.Now(); time.Since(pos) < 30*time.Second; {
		out = readCardsDue(t, client)
		if !slices.ContainsFunc(out.Vaults, func(one *api.VaultCardsDue) bool { return one.IsIndexing }) {
			return out
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("a vault was still being read")
	return out
}

// readCardsDue collects the full response from WatchCardsDue stream.
func readCardsDue(
	t *testing.T, client reviewClient,
) *api.WatchCardsDueResponse {
	t.Helper()
	stream, err := client.WatchCardsDue(t.Context(), &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	out, pos, first := &api.WatchCardsDueResponse{}, map[string]int{}, true
	for stream.Receive() {
		response := stream.Msg()
		if first {
			first = false
			out.Day, out.Vaults = response.Day, response.Vaults
			for where, one := range out.Vaults {
				pos[one.ID] = where
			}
			continue
		}
		one := response.Counted
		if one == nil {
			// A repeated list, which the stream emits while a count is still running so
			// that a closed page is detected.
			continue
		}
		where, items := pos[one.ID]
		if !items {
			t.Fatalf("unexpected vault %s in count response", one.ID)
		}
		out.Vaults[where] = one
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// startSession is a session opened on one vault, and what it holds to ask.
func startSession(t *testing.T, handler *API, v domain.Vault) *api.StartSessionResponse {
	t.Helper()
	out, err := handler.StartSession(t.Context(),
		&api.StartSessionRequest{Vault: string(v.ID)})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A run belongs to the vault it was opened on, and an answer naming another
// vault's run fails.
func TestRecordReview_ErrorsOnRunOfOtherVault(t *testing.T) {
	handler, target, installation := newAPIOn(t, deck, other)
	one, two := target[0], target[1]

	session := startSession(t, handler, one)
	if len(session.Cards) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := session.Cards[0]

	_, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
		Vault:    string(two.ID),
		Run:      session.Run,
		CardID:   card.CardID,
		Template: card.Template,
		Rating:   api.RatingGood,
	})
	if err == nil {
		t.Fatal("an answer named against another vault's run was written")
	}
	if codeOf(err) != server.CodeNotFound {
		t.Errorf("failed with %v", codeOf(err))
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
	handler, target := newAPI(t, deck)
	v := target[0]

	was := startSession(t, handler, v)
	now := startSession(t, handler, v)
	if was.Run == now.Run {
		t.Fatal("a second session wrote to the file the first opened")
	}

	card := now.Cards[0]
	_, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
		Vault:    string(v.ID),
		Run:      was.Run,
		CardID:   card.CardID,
		Template: card.Template,
		Rating:   api.RatingGood,
	})
	if codeOf(err) != server.CodeNotFound {
		t.Fatalf("the session was over and the answer failed with %v", codeOf(err))
	}
	if !hasCause(err, ErrNoRun) {
		t.Errorf("failed with %v", err)
	}
}

// A vault is answered on its own. Work done in one vault does not count toward
// another: the two vaults share no card id, no review card and no word, so nothing
// can pass between them by similarity.
func TestAnAnswerInOneVaultPreservesTheOtherOwingWhatItDid(t *testing.T) {
	handler, target, installation := newAPIOn(t, deck, other)
	one, two := target[0], target[1]

	was := findVaultCount(t, front(t, handler), string(two.ID))

	session := startSession(t, handler, one)
	for _, card := range session.Cards {
		if _, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
			Vault: string(one.ID), Run: session.Run,
			CardID: card.CardID, Template: card.Template,
			Rating: api.RatingEasy,
		}); err != nil {
			t.Fatal(err)
		}
	}

	after := front(t, handler)
	if now := findVaultCount(t, after, string(two.ID)); now.New != was.New ||
		now.Due != was.Due || now.ReviewCards != was.ReviewCards {
		t.Errorf("the other vault came to %+v, having come to %+v", now, was)
	}
	if now := findVaultCount(t, after, string(one.ID)); now.New != 0 || now.Due != 0 {
		t.Errorf("the answered vault still owes %+v", now)
	}
	// Nothing was written into the other vault's folder either.
	if names := getRunFiles(t, installation.Host.Store().With(two.ID)); len(names) != 0 {
		t.Errorf("the other vault holds %v", names)
	}
}

// findVaultCount returns the vault matching id from the response.
func findVaultCount(t *testing.T, resp *api.WatchCardsDueResponse, id string) *api.VaultCardsDue {
	t.Helper()
	for _, one := range resp.Vaults {
		if one.ID == id {
			return one
		}
	}
	t.Fatalf("vault %s not found in response", id)
	return nil
}

// An answer is written to the run's own file, and the identifier it comes back
// with is what taking it back names.
func TestAnAnswerIsSavedAndCanBeTakenBack(t *testing.T) {
	handler, target, installation := newAPIOn(t, deck)
	v := target[0]

	session := startSession(t, handler, v)
	card := session.Cards[0]

	provided, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
		Vault:      string(v.ID),
		Run:        session.Run,
		CardID:     card.CardID,
		Template:   card.Template,
		Rating:     api.RatingGood,
		DurationMs: 1200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if provided.Answer == "" {
		t.Fatal("the answer came back with nothing to take it back by")
	}

	if _, err := handler.UndoReview(t.Context(), &api.UndoReviewRequest{
		Vault:  string(v.ID),
		Run:    session.Run,
		Answer: provided.Answer,
	}); err != nil {
		t.Fatal(err)
	}

	if names := getRunFiles(t, installation.Host.Store().With(v.ID)); len(names) != 1 {
		t.Errorf("a session writes one file, the vault holds %v", names)
	}
}

// An answer outside the four is the caller's mistake and fails as one.
func TestAnAnswerOutsideTheFourIsRefused(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]
	session := startSession(t, handler, v)
	card := session.Cards[0]

	_, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
		Vault: string(v.ID), Run: session.Run,
		CardID: card.CardID, Template: card.Template,
		Rating: api.RatingUnspecified,
	})
	if codeOf(err) != server.CodeInvalidArgument {
		t.Errorf("failed with %v", codeOf(err))
	}
}

// Every vault the review counts is in the index before it is counted: a vault being read into the
// index has nothing to show, and its numbers arrive with the count that follows.
func TestAVaultIsIndexedBeforeItIsCounted(t *testing.T) {
	handler, target, installation := newAPIOn(t, deck, deck)
	installation.Of(target[1].ID).SetIndexState(host.IndexState{Status: host.Indexing})

	// A vault whose index walk is incomplete has no count, and its row reports that.
	rows := map[string]*api.VaultCardsDue{}
	for _, one := range readCardsDue(t, newFlashcardsClient(t, handler)).Vaults {
		rows[one.ID] = one
	}
	if got := rows[string(target[1].ID)]; !got.IsIndexing || got.Unread != "" {
		t.Errorf("a vault being read came back %+v", got)
	}
	if got := rows[string(target[0].ID)]; got.IsIndexing || got.New == 0 {
		t.Errorf("a vault the index carries came back %+v", got)
	}

	// The reading finishes and the count that follows it holds the cards.
	installation.Of(target[1].ID).SetIndexState(host.IndexState{Status: host.Indexed})
	for _, one := range front(t, handler).Vaults {
		if one.Unread != "" || one.New == 0 {
			t.Errorf("a vault came back %+v", one)
		}
	}
}

// A vault the index could not read is uncounted, and the row reports why.
func TestAVaultTheIndexCouldNotReadIsUncountedAndReturnsWhy(t *testing.T) {
	handler, target, installation := newAPIOn(t, deck)
	installation.Of(target[0].ID).SetIndexState(host.IndexState{Status: host.IndexFailed, Reason: "the disk is full"})

	one := front(t, handler).Vaults[0]

	if one.IsIndexing || one.Unread != "the disk is full" {
		t.Errorf("a vault nothing could read came back %+v", one)
	}
}

// A session opened on a vault the index does not carry fails with an error.
func TestSessionDownToAVaultTheIndexDoesNotCarryIsRefused(t *testing.T) {
	handler, target, installation := newAPIOn(t, deck)
	installation.Of(target[0].ID).SetIndexState(host.IndexState{Status: host.Indexing})

	_, err := handler.StartSession(t.Context(),
		&api.StartSessionRequest{Vault: string(target[0].ID)})
	if codeOf(err) != server.CodeFailedPrecondition {
		t.Fatalf("failed with %v: %v", codeOf(err), err)
	}
	if !hasCause(err, flashcards.ErrNotCarried) {
		t.Errorf("says %q, and not that the index does not carry the vault", err)
	}
}

// A question about a vault the installation does not hold fails.
func TestAQuestionAboutAVaultNobodyContainsIsRefused(t *testing.T) {
	handler, _ := newAPI(t)
	_, err := handler.StartSession(t.Context(),
		&api.StartSessionRequest{Vault: "nothing"})
	if codeOf(err) != server.CodeNotFound {
		t.Errorf("failed with %v", codeOf(err))
	}
	if !hasCause(err, host.ErrUnknownVault) {
		t.Errorf("failed with %v", err)
	}
}

// What each of the four would do to the card is worked out with the card, so a
// person choosing between them is shown what they are choosing between.
func TestStartSession_CardsCarryRatingIntervals(t *testing.T) {
	handler, target := newAPI(t, deck)
	session := startSession(t, handler, target[0])
	ahead := session.Cards[0].Intervals

	if ahead == nil {
		t.Fatal("the card was asked without saying where the four would leave it")
	}
	if !(ahead.Again < ahead.Hard &&
		ahead.Hard < ahead.Good &&
		ahead.Good < ahead.Easy) {
		t.Errorf("the four come round at %+v", ahead)
	}
}

// The four ratings are carried across as themselves, and nothing else is a
// rating.
func TestTheFourRatingsAreCarriedAcross(t *testing.T) {
	for response, want := range map[api.Rating]review.Rating{
		api.RatingAgain: review.Again,
		api.RatingHard:  review.Hard,
		api.RatingGood:  review.Good,
		api.RatingEasy:  review.Easy,
	} {
		if got := parseRating(response); got != want {
			t.Errorf("%v came across as %v", response, got)
		}
	}
	if got := parseRating(api.RatingUnspecified); got.Valid() {
		t.Errorf("nothing said came across as %v", got)
	}
}

// A deck that could not be assigned card ids is reported to the user, so the
// missing cards are identifiable.
func TestASessionReturnsWhichDecksItCouldNotMark(t *testing.T) {
	handler, target, installation := newAPIOn(t, map[string]string{
		"Term.md": deck["Term.md"],
		// The card carries no card id, so the deck is written to give it one.
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	})
	v := target[0]

	// A deck nothing may write is a deck that keeps its cards out of the
	// session, and it is named.
	installation.Of(v.ID).RefuseWriting("decks/Own.md", fs.ErrPermission)

	session := startSession(t, handler, v)
	if len(session.Unwritten) != 1 || session.Unwritten[0] != "decks/Own.md" {
		t.Errorf("the session says %v could not be marked", session.Unwritten)
	}
	if len(session.Cards) != 0 {
		t.Errorf("a card with no card id was asked: %+v", session.Cards)
	}
}

// TestWatchCardsDue_CountsEveryVault verifies all vaults are reported.
func TestWatchCardsDue_CountsEveryVault(t *testing.T) {
	handler, target := newAPI(t, deck, other)

	out := front(t, handler)
	if len(out.Vaults) != len(target) {
		t.Fatalf("counted %d of %d vaults", len(out.Vaults), len(target))
	}
	for _, one := range out.Vaults {
		if one.ReviewCards != 1 || one.New != 1 {
			t.Errorf("got %s = %+v, want Review cards=1, New=1", one.Name, one)
		}
	}
}

// Counting cards does not modify vault files.
func TestWatchCardsDue_DoesNotModifyVaultFiles(t *testing.T) {
	handwritten := map[string]string{
		"Term.md": deck["Term.md"],
		"decks/Own.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	}
	handler, target := newAPI(t, handwritten)
	pos := filepath.Join(target[0].Path, "decks", "Own.md")

	before, err := os.ReadFile(pos)
	if err != nil {
		t.Fatal(err)
	}
	front(t, handler)
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
func TestWatchCardsDue_ReturnsDailyProgressPerPreset(t *testing.T) {
	handler, target := newAPI(t, pointed)
	v := target[0]

	session := startSession(t, handler, v)
	if len(session.Cards) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := session.Cards[0]
	if _, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
		Vault: string(v.ID), Run: session.Run,
		CardID: card.CardID, Template: card.Template,
		Rating: api.RatingGood, DurationMs: 6000,
	}); err != nil {
		t.Fatal(err)
	}

	presets := front(t, handler).Vaults[0].Presets
	if len(presets) != 1 {
		t.Fatalf("the vault came to %+v, want the one preset", presets)
	}
	one := presets[0]
	if one.Preset != "Sanskrit.md" || one.Reviewed != 1 || one.DurationMs != 6000 {
		t.Errorf("the day came to %+v", one)
	}
	if one.New != 8 || one.Reviews != 45 || one.Minutes != 20 {
		t.Errorf("the day holds %+v", one)
	}
}

// A deck naming no preset comes under the defaults, which is what a vault
// holding no preset at all evaluates to.
func TestADeckNamingNoPresetReachesBelowTheDefaults(t *testing.T) {
	handler, _ := newAPI(t, deck)

	presets := front(t, handler).Vaults[0].Presets
	if len(presets) != 1 {
		t.Fatalf("the vault came to %+v, want the defaults alone", presets)
	}
	one, defaults := presets[0], review.Defaults()
	if one.Preset != "" || one.Reviewed != 0 {
		t.Errorf("the day came to %+v", one)
	}
	if one.New != int32(defaults.NewADay) || one.Reviews != int32(defaults.ReviewsADay) {
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
	handler, target := newAPI(t, twoPresets)

	got := make(map[string]int)
	for _, one := range startSession(t, handler, target[0]).Cards {
		got[one.Deck]++
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
func TestWatchCardsDue_ReturnsSettingsPerPreset(t *testing.T) {
	handler, _ := newAPI(t, twoPresets)

	got := make(map[string]int32)
	for _, one := range front(t, handler).Vaults[0].Presets {
		got[one.Preset] = one.New
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
// The deck screen calls CardsDue for what its decks offer today, and the preset
// tab calls Curve for the curve over a preset's range. Both are called here as the
// application calls them: the settings the curve is computed for are the ones the
// vault holds. The curve at the preset's setting is the day the deck screen offers
// whenever the control is at today, and the reason a day is closed names the budget
// its own goal steers and no other.
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
		// isSameDay is whether the control is at today, so the curve at its current
		// position equals the day the deck screen offers.
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
			handler, target := newAPI(t, backlogged)
			v := target[0]

			p := readPresetConfig(t, handler, v, "Steady.md")
			p.Goal, p.By = one.goal, one.byDate
			if one.learned != "" {
				p.Rule = one.learned
			}
			writePreset(t, handler, v, "Steady.md", p)

			offers := 0
			for _, deck := range getVaultCount(t, handler, v).Decks {
				if deck.Deck == "decks/Steady.md" {
					offers = int(deck.Due + deck.New)
				}
			}
			if offers == 0 {
				t.Fatal("the deck screen offers nothing to compare")
			}

			curveResp := getCurve(t, handler, v, "Steady.md", p)
			pos := curveResp.Now.At
			if pos < 0 {
				t.Fatalf("the preset stands nowhere on its own curve: %+v", curveResp.Now)
			}
			if got := int(curveResp.At[pos].Reviews); one.isSameDay && got != offers {
				t.Errorf("the deck screen offers %d cards and the tab draws %d", offers, got)
			}
			if got := curveResp.At[pos].Closed; !slices.Contains(got, presetwire.ConvertBudgetToWire(one.closed)) {
				t.Errorf("the day closed on %v, want %q among them", got, one.closed)
			}
			for i, point := range curveResp.At {
				for _, never := range one.never {
					if slices.Contains(point.Closed, presetwire.ConvertBudgetToWire(never)) {
						t.Errorf("at %v the day closed on %q, which its goal does not name",
							curveResp.Grid[i], never)
					}
				}
			}
		})
	}
}

// Under a goal of minutes a day long enough for the whole of the material is
// closed by nothing: the material itself ran out, and no card limit is named.
func TestGetCurve_LongDayHasNoClosingLimit(t *testing.T) {
	handler, target := newAPI(t, backlogged)
	v := target[0]

	curveResp := getCurve(t, handler, v, "Steady.md", readPresetConfig(t, handler, v, "Steady.md"))
	if got := curveResp.At[len(curveResp.At)-1].Closed; len(got) != 0 {
		t.Errorf("the longest day on the range is closed by %q", got)
	}
}

// The day suggested under a goal of minutes is the shortest day that presents every
// card the day holds, on a vault with a backlog.
//
// A shorter day is cut short by the clock, so the shortest day on the range is
// never the suggestion for a vault behind on its reviews. The cards a shorter day
// presents may already be the whole count: a card that returns within the day is the
// same card, and the repeated showings it takes exhaust the clock.
func TestTheSuggestedDayIsTheShortestThatAsksEverything(t *testing.T) {
	handler, target := newAPI(t, lived)
	v := target[0]
	runDays(t, handler, v, 14)
	setNow(handler, firstMorning.AddDate(0, 0, 14))

	p := readPresetConfig(t, handler, v, "Sanskrit.md")
	p.Goal = review.GoalMinutes
	writePreset(t, handler, v, "Sanskrit.md", p)

	curveResp := getCurve(t, handler, v, "Sanskrit.md", p)
	pos := int(curveResp.Suggested.At)
	if pos < 0 {
		t.Fatalf("nothing is suggested: %+v", curveResp.Suggested)
	}
	if pos == 0 {
		t.Error("the shortest day on the range is suggested, and this vault is behind")
	}

	whole := curveResp.At[len(curveResp.At)-1].Reviews
	if got := curveResp.At[pos].Reviews; got != whole {
		t.Errorf("the suggested day asks %v cards, and a day of any length asks %v", got, whole)
	}
	if got := curveResp.At[pos].Closed; len(got) != 0 {
		t.Errorf("the suggested day closed on %v, and a day that asks everything closes on nothing",
			got)
	}
	for i := range pos {
		got := curveResp.At[i].Closed
		if !slices.Contains(got, presetwire.ConvertBudgetToWire(review.ClosedMinutes)) {
			t.Errorf("%v minutes a day closed on %v, and %v is suggested",
				curveResp.Grid[i], got, curveResp.Grid[pos])
		}
	}
}

// What is overdue is a fact about the vault, and how long it takes to clear is
// a fact about the setting being chosen.
//
// The overdue count applies across the whole curve and does not change with the goal.
func TestWhatIsOverdueExistsOverTheWholeCurve(t *testing.T) {
	handler, target := newAPI(t, lived)
	v := target[0]
	runDays(t, handler, v, 14)
	setNow(handler, firstMorning.AddDate(0, 0, 14))

	owes := 0
	for _, one := range getVaultCount(t, handler, v).Decks {
		for _, deck := range underSanskrit {
			if one.Deck == deck {
				owes += int(one.Due)
			}
		}
	}

	p := readPresetConfig(t, handler, v, "Sanskrit.md")
	p.Goal = review.GoalMinutes
	writePreset(t, handler, v, "Sanskrit.md", p)
	byMinutes := getCurve(t, handler, v, "Sanskrit.md", p)

	overdue := int(byMinutes.Overdue)
	if overdue == 0 {
		t.Fatal("got overdue == 0, want overdue > 0 for abandoned vault")
	}
	if overdue > owes {
		t.Errorf("got overdue = %d, want <= %d (deck due count)", overdue, owes)
	}

	// The same vault under another goal is the same backlog.
	p.Goal = review.GoalRetention
	if got := int(getCurve(t, handler, v, "Sanskrit.md", p).Overdue); got != overdue {
		t.Errorf("got overdue = %d for GoalRetention, want %d (from GoalMinutes)", got, overdue)
	}

	// A shorter day is longer about clearing it, or never gets there at all.
	short, long := byMinutes.At[0], byMinutes.At[len(byMinutes.At)-1]
	if long.Clears <= 0 {
		t.Fatalf("the longest day on the range clears the backlog in %d days",
			long.Clears)
	}
	if short.Clears != -1 && short.Clears <= long.Clears {
		t.Errorf("%v minutes a day clears in %d and %v minutes a day in %d",
			byMinutes.Grid[0], short.Clears,
			byMinutes.Grid[len(byMinutes.Grid)-1], long.Clears)
	}
}

// A vault with no review history has nothing overdue.
func TestGetCurve_UnansweredVaultHasNothingOverdue(t *testing.T) {
	handler, target := newAPI(t, backlogged)
	v := target[0]

	curveResp := getCurve(t, handler, v, "Steady.md", readPresetConfig(t, handler, v, "Steady.md"))
	if got := curveResp.Overdue; got != 0 {
		t.Errorf("got overdue = %d, want 0", got)
	}
	for i, one := range curveResp.At {
		if one.Clears != 0 {
			t.Errorf("at %v minutes a day: got clears = %d, want 0",
				curveResp.Grid[i], one.Clears)
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
// A preset is the whole scope of its own budget, so the cards the count leaves
// for it are the cards a session over it presents, and the count reports that figure.
func TestThePresetTileAndTheSessionItOpensAreOneNumber(t *testing.T) {
	handler, target := newAPI(t, spread)
	v := target[0]
	runDays(t, handler, v, 14)
	setNow(handler, firstMorning.AddDate(0, 0, 14))

	response := getVaultCount(t, handler, v)
	// Which decks each preset schedules, so the cards a session presents can be checked
	// against the rows they were collected from.
	underMap := make(map[string][]string)
	rows := make(map[string]*api.DeckCardsDue, len(response.Decks))
	for _, one := range response.Decks {
		rows[one.Deck] = one
		p, err := handler.scenarios().flash.Presets.GetForDeck(t.Context(), v, one.Deck)
		if err != nil {
			t.Fatal(err)
		}
		underMap[p.Path] = append(underMap[p.Path], one.Deck)
	}

	scopes, clamped := 0, 0
	for _, one := range response.Presets {
		if one.Cards == 0 {
			continue
		}
		scopes++
		want := int(one.OwedDue + one.OwedNew)
		if want == 0 {
			t.Errorf("the tile over %q offers nothing to compare", one.Preset)
		}

		sessionResp, err := handler.StartSession(t.Context(), &api.StartSessionRequest{
			Vault: string(v.ID), Preset: newPresetName(one.Preset),
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := len(sessionResp.Cards); got != want {
			t.Errorf("deck %q: got session card count = %d, want %d",
				one.Preset, got, want)
		}

		// What the count leaves under a preset is what its decks were left,
		// gathered by the one pass.
		due, newCards := 0, 0
		for _, deck := range underMap[one.Preset] {
			due += int(rows[deck].Due)
			newCards += int(rows[deck].New)
		}
		if due != int(one.OwedDue) || newCards != int(one.OwedNew) {
			t.Errorf("deck %q: got due=%d, new=%d; want preset OwedDue=%d, OwedNew=%d",
				one.Preset, due, newCards, one.OwedDue, one.OwedNew)
		}

		// Budget allocation restricts counts when configured.
		if min(due, int(one.Reviews))+min(newCards, int(one.New)) < want {
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
			handler, target := newAPI(t, one.notes)
			v := target[0]
			runDays(t, handler, v, 14)
			now := firstMorning.AddDate(0, 0, 14)
			setNow(handler, now)

			past := 0
			for _, response := range getVaultCount(t, handler, v).Presets {
				if response.Cards == 0 {
					continue
				}
				due, newCards := int(response.OwedDue), int(response.OwedNew)
				keptReviews, keptNew := int(response.Reviews), int(response.New)

				if response.ClosesReviews != "" && due > keptReviews {
					t.Errorf("%q allocated %d owed against the %d reviews it keeps",
						response.Preset, due, keptReviews)
				}
				if response.ClosesNew != "" && newCards > keptNew {
					t.Errorf("%q allocated %d new against the %d it keeps",
						response.Preset, newCards, keptNew)
				}
				// A count standing in the file that the goal does not name is
				// no wall, and this is the vault where that shows.
				if response.ClosesReviews == "" && due > keptReviews {
					past++
				}
				if response.ClosesNew == "" && newCards > keptNew {
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
	handler, target := newAPI(t, lived)
	v := target[0]
	runDays(t, handler, v, 14)
	setNow(handler, firstMorning.AddDate(0, 0, 14))

	p := readPresetConfig(t, handler, v, "Sanskrit.md")
	p.Goal = review.GoalMinutes
	writePreset(t, handler, v, "Sanskrit.md", p)
	curveResp := getCurve(t, handler, v, "Sanskrit.md", p)

	overdue, unbegun, cards := curveResp.Overdue, curveResp.Unbegun, curveResp.Cards
	if overdue == 0 || unbegun == 0 {
		t.Fatalf("the vault stands %d overdue and %d unbegun, and neither is under test",
			overdue, unbegun)
	}
	if overdue+unbegun > cards {
		t.Errorf("%d overdue and %d unbegun of %d review cards", overdue, unbegun, cards)
	}

	for i, one := range curveResp.At {
		band := one.Backlog
		if len(band) == 0 {
			t.Fatalf("%v carries no backlog to draw", curveResp.Grid[i])
		}
		// A day too short to carry what falls due adds to the pile, so the
		// band rises as well as falls, and never below nothing.
		for day, backlog := range band {
			if backlog < 0 {
				t.Errorf("%v stands %d behind on day %d", curveResp.Grid[i], backlog, day)
			}
		}
		// The day reported as the clearing day is the first day on which the band is zero.
		first := int32(-1)
		for day, backlog := range band {
			if backlog == 0 {
				first = int32(day) + 1
				break
			}
		}
		if one.Clears != first {
			t.Errorf("%v clears on day %d and its band reaches nothing on day %d",
				curveResp.Grid[i], one.Clears, first)
		}
	}

	// A longer day is never further behind than a shorter one on the same day.
	short, long := curveResp.At[0].Backlog, curveResp.At[len(curveResp.At)-1].Backlog
	for day := range min(len(short), len(long)) {
		if long[day] > short[day] {
			t.Errorf("on day %d the longest day on the range stands %d behind and the "+
				"shortest %d", day, long[day], short[day])
		}
	}
}

// A curve of retention plots what the target costs week after week.
//
// A target does not affect today, so the height is the load over the projected
// days: it increases with the target where no count binds, and is constant at
// every place where a count binds on every projected day.
func TestTheRetentionCurvePlotsWhatTheTargetCosts(t *testing.T) {
	for _, one := range []struct {
		what             string
		newADay, reviews int
		counts           review.BudgetUnit
		isClimbing       bool
	}{
		// A budget of one showing is a day the target cannot spend, whatever the
		// target. Counting cards it could: the one card is presented again for no gain, and
		// how often it returns is determined by the target.
		{"a count binding at every place", 12, 1, review.BudgetUnitShows, false},
		{"counts that never bind", 50, 200, review.BudgetUnitCards, true},
	} {
		t.Run(one.what, func(t *testing.T) {
			handler, target := newAPI(t, lived)
			v := target[0]
			runDays(t, handler, v, 14)
			setNow(handler, firstMorning.AddDate(0, 0, 14))

			p := readPresetConfig(t, handler, v, "Sanskrit.md")
			p.Goal = review.GoalRetention
			p.NewADay, p.ReviewsADay = one.newADay, one.reviews
			p.Counts = one.counts
			writePreset(t, handler, v, "Sanskrit.md", p)
			curveResp := getCurve(t, handler, v, "Sanskrit.md", p)

			least := curveResp.At[0].Minutes
			most := curveResp.At[len(curveResp.At)-1].Minutes
			if least <= 0 {
				t.Fatalf("the easiest target costs %v minutes a day", least)
			}
			switch {
			case one.isClimbing && most <= least*2:
				t.Errorf("asking for %.2f of it back costs %v minutes a day and asking "+
					"for %.2f costs %v", curveResp.Grid[len(curveResp.Grid)-1], most,
					curveResp.Grid[0], least)
			case !one.isClimbing && most != least:
				t.Errorf("a day the count closes at every place costs %v minutes at one "+
					"end of the range and %v at the other", least, most)
			}

			// What the cost buys climbs with it, and is a share of the material.
			for i, item := range curveResp.At {
				if item.Retained < 0 || item.Retained > 1 {
					t.Errorf("%.2f leaves %v of the material in the head",
						curveResp.Grid[i], item.Retained)
				}
			}
			retentionDelta := curveResp.At[len(curveResp.At)-1].Retained -
				curveResp.At[0].Retained
			if one.isClimbing && retentionDelta <= 0 {
				t.Errorf("asking for more of it back kept %v more of it", retentionDelta)
			}
		})
	}
}

// sessionCounts holds review counts for a session.
type sessionCounts struct{ total, due, unbegun int }

// session opens a session for one preset.
func session(t *testing.T, handler *API, v domain.Vault, preset string) sessionCounts {
	t.Helper()
	out, err := handler.StartSession(t.Context(), &api.StartSessionRequest{
		Vault: string(v.ID), Preset: newPresetName(preset),
	})
	if err != nil {
		t.Fatal(err)
	}
	target := sessionCounts{total: len(out.Cards)}
	for _, one := range out.Cards {
		if one.IsNew {
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
		handler, made := newAPI(t, lived)
		v := made[0]
		runDays(t, handler, v, 20)
		setNow(handler, firstMorning.AddDate(0, 0, 20))

		p := readPresetConfig(t, handler, v, "Sanskrit.md")
		p.Goal, p.Backlog = review.GoalMinutes, share
		writePreset(t, handler, v, "Sanskrit.md", p)

		// Evaluate curve for the preset.
		curveResp := getCurve(t, handler, v, "Sanskrit.md", p)
		value := curveResp.Grid[2]
		p.MinutesADay = int(value)
		writePreset(t, handler, v, "Sanskrit.md", p)

		curveResp = getCurve(t, handler, v, "Sanskrit.md", p)
		pos := 2
		if curveResp.Grid[pos] != value {
			t.Fatalf("got grid[%d] = %v, want %v", pos, curveResp.Grid[pos], value)
		}

		one := session(t, handler, v, "Sanskrit.md")
		if got := int(curveResp.At[pos].Reviews); got != one.total {
			t.Errorf("share %d: got reviews = %d, want %d", share, got, one.total)
		}
		target[share], bands[share] = one, curveResp.At[pos].Backlog
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

// TestEveryRatingIsRead. A rating the parser does not recognize is parsed as
// zero, which the use case rejects; a button press would be reported as not an
// answer.
func TestEveryRatingIsRead(t *testing.T) {
	enumtest.CheckHandled(t, api.RatingValues, func(r api.Rating) bool { return parseRating(r) != 0 })
}

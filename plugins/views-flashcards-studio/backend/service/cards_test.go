package service_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1/flashcardsv1connect"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service"
)

// A card model says what a card has and a deck holds the cards. What the window
// asks about either is what these say.

// cutting is a vault with the notes given in it, and a client asking about its
// decks and its card models the way the window does.
type cutting struct {
	client flashcardsv1connect.CardsServiceClient
	root   string
}

func newCutting(t *testing.T, notes map[string]string) *cutting {
	t.Helper()

	env := vaulttest.New(t, notes)
	backend := service.New(service.Config{Vault: env.Host, Store: env.Store})

	route, handler := flashcardsv1connect.NewCardsServiceHandler(backend.Cards())
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &cutting{
		client: flashcardsv1connect.NewCardsServiceClient(server.Client(), server.URL),
		root:   env.Vault.Path,
	}
}

// onDisk is what the vault holds at a path.
func onDisk(t *testing.T, root, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

const animal = "---\ntype: card-model\nfields:\n  - Name\n  - Height\n---\n\n" +
	"## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n"

// cardRun is the cards of a deck as somebody wrote them, the middle one under
// no wikilink, so it names no card model and the problem stands against that one.
const cardRun = "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n\n" +
	"## Alpaca ^3n8vr4tqch\n\nsomebody's prose\n\n### Name\n\nAlpaca\n\n### Height\n\nabout 36\"\n\n" +
	"## Vicuña ^9wq2xr4t8h\n\n[[Animal]]\n\n### Name\n\nVicuña\n\n### Height\n\nabout 34\"\n"

const threeCards = "---\ntype: deck\n---\n\n" + cardRun

// dividedRun is a deck a person divided, every card of it whole: each stands
// under the section it was written in, each carries a card id, and each names the
// card model its heading is read through.
const dividedRun = "# Camelids\n\nwhat a person wrote about their own deck\n\n" +
	"## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n\n" +
	"# Others\n\n## Vicuña ^9wq2xr4t8h\n\n[[Animal]]\n\n### Name\n\nVicuña\n\n### Height\n\nabout 34\"\n"

const dividedDeck = "---\ntype: deck\n---\n\n" + dividedRun

// getFieldValue is what a card holds under one field. A card has no name, so this is
// how a test says which card it is looking at.
func getFieldValue(card *flashcardsv1.Card, field string) string {
	for _, v := range card.GetValues() {
		if v.GetField() == field {
			return v.GetText()
		}
	}
	return ""
}

func deck(t *testing.T, f *cutting, path string) *flashcardsv1.ReadDeckResponse {
	t.Helper()
	answer, err := f.client.ReadDeck(t.Context(), connect.NewRequest(&flashcardsv1.ReadDeckRequest{Path: path}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg
}

// TestAProblemStandsAgainstTheCardItIsAbout. A card is addressed by where it
// stands, so a problem's position is an index into the cards the same answer
// carried.
func TestAProblemExistsAgainstTheCardItIsAbout(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": threeCards,
	})

	read := deck(t, f, "Animals.md")
	target := read.GetDeck().GetCards()
	if len(target) != 3 {
		t.Fatalf("the deck came back with %d cards", len(target))
	}
	problems := read.GetDeck().GetProblems()
	if len(problems) != 1 {
		t.Fatalf("the deck came back with %d problems: %+v", len(problems), problems)
	}
	if fault := problems[0].GetKind(); fault != flashcardsv1.ProblemKind_PROBLEM_KIND_CARD_WITHOUT_A_CARD_MODEL {
		t.Errorf("the card with no wikilink under it is reported as %v", fault)
	}
	pos := problems[0].GetCard()
	if problems[0].Card == nil {
		t.Fatal("a problem about a card came back standing against none")
	}
	if name := getFieldValue(target[pos], "Name"); name != "Alpaca" {
		t.Errorf("problem targets card %d (%q)", pos, name)
	}
}

// TestACardComesBackWithTheMarkItIsAddressedBy. A card has no name, so a client
// that cannot read a card's id cannot say which card it means.
func TestACardReachesBackWithTheMarkItIsAddressedBy(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": threeCards,
	})

	target := deck(t, f, "Animals.md").GetDeck().GetCards()
	if len(target) != 3 {
		t.Fatalf("the deck came back with %d cards", len(target))
	}
	cardIDs := []string{"k7m2xq9fzp", "3n8vr4tqch", "9wq2xr4t8h"}
	for item, card := range target {
		if card.GetCardId() != cardIDs[item] {
			t.Errorf("card %d carries the card id %q, and the file writes %q",
				item, card.GetCardId(), cardIDs[item])
		}
	}
}

// TestACardSaysWhichSectionItStandsUnder. A deck of two hundred cards is a list
// a person divides, and which run a card falls in is the deck's and not the
// card's own text.
func TestACardReturnsWhichSectionItExistsBelow(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Guanaco ^p4r7t2wxk9\n\n[[Animal]]\n\n### Name\n\nGuanaco\n\n" +
			"# Camelids\n\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n" +
			"# Others\n\nwhat a person wrote about this run of it\n",
	})

	read := deck(t, f, "Animals.md").GetDeck()
	sections := read.GetSections()
	if len(sections) != 2 || sections[0].GetName() != "Camelids" || sections[1].GetName() != "Others" {
		t.Fatalf("the deck came back with the sections %+v", sections)
	}
	if lead := sections[1].GetPreamble(); lead != "what a person wrote about this run of it" {
		t.Errorf("what a person wrote under the second section came back as %q", lead)
	}
	target := read.GetCards()
	if len(target) != 2 {
		t.Fatalf("the deck came back with %d cards", len(target))
	}
	if target[0].SectionIndex != nil {
		t.Errorf("the card standing above the first section came back under section %d",
			target[0].GetSectionIndex())
	}
	if target[1].SectionIndex == nil || target[1].GetSectionIndex() != 0 {
		t.Errorf("the card under the first section came back as %+v", target[1].SectionIndex)
	}
}

// TestTwoCardsOfOneMarkStandAgainstBoth. Which of the two a person meant is a
// thing only they know, so both are shown and both are marked.
func TestTwoCardsOfOneMarkStandAgainstBoth(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n" +
			"## Alpaca ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nAlpaca\n",
	})

	problems := deck(t, f, "Animals.md").GetDeck().GetProblems()
	if len(problems) != 2 {
		t.Fatalf("two cards of one card id came back as %d problems: %+v", len(problems), problems)
	}
	for item, problem := range problems {
		if fault := problem.GetKind(); fault != flashcardsv1.ProblemKind_PROBLEM_KIND_CARD_ID_CARRIED_TWICE {
			t.Errorf("the problem against card %d is %v", item, fault)
		}
		if problem.Card == nil || problem.GetCard() != int32(item) {
			t.Errorf("the problem for card %d targets %+v", item, problem.Card)
		}
	}
}

// TestWritingADeckLeavesAloneOneThatChangedSinceItWasRead. Somebody editing
// their own deck outranks a client that read it, thought about it and arrived
// late.
func TestWritingADeckPreservesAloneOneThatChangedSinceItWasRead(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": threeCards,
	})

	read := deck(t, f, "Animals.md")
	// The person writes their own deck while the client is thinking about what
	// it read.
	theirs := threeCards + "\n## Guanaco\n\n[[Animal]]\n\n### Height\n\nabout 43\"\n"
	if err := os.WriteFile(filepath.Join(f.root, "Animals.md"), []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}

	answer, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&flashcardsv1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Cards:    read.GetDeck().GetCards()[:1],
		Tail:     read.GetDeck().GetTail(),
		Baseline: read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_STALE {
		t.Errorf("a write over a deck the person had edited answered %+v", answer.Msg)
	}
	if target := onDisk(t, f.root, "Animals.md"); target != theirs {
		t.Errorf("the deck on disk is now %q", target)
	}
}

// TestADeckWrittenBackKeepsTheCardsItHeld. Every card and every section the
// client did not touch arrives on the other side as the bytes it went in as,
// the prose under a card's wikilink and under a section's heading among them.
func TestADeckSavedBackKeepsTheCardsItHeld(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": dividedDeck,
	})

	read := deck(t, f, "Animals.md")
	answer, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&flashcardsv1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Sections: read.GetDeck().GetSections(),
		Cards:    read.GetDeck().GetCards(),
		Tail:     read.GetDeck().GetTail(),
		Baseline: read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing a deck straight back answered %+v", answer.Msg)
	}
	if target := onDisk(t, f.root, "Animals.md"); !strings.HasSuffix(target, dividedRun) {
		t.Errorf("the deck came back as %q", target)
	}
}

// TestAHeadingOfTwoLinesWritesNoSecondCard. A heading is one line, so what a
// client sends as one is cut where the lines under it begin. A second `##` in
// there is a card the next read adopts and gives a card id to, and nobody wrote
// it.
func TestAHeadingOfTwoLinesWritesNoSecondCard(t *testing.T) {
	f := newCutting(t, map[string]string{"Animal.md": animal, "Animals.md": dividedDeck})

	read := deck(t, f, "Animals.md")
	cards := read.GetDeck().GetCards()
	cards[0].Heading = "Question\n\n## Injected ^m9n8b7v6c5\n\n[[Animal]]\n\n### Name\n\nsmuggled"
	sections := read.GetDeck().GetSections()
	sections[1].Name = "Others\n\n## Also injected"

	answer, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&flashcardsv1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Sections: sections,
		Cards:    cards,
		Tail:     read.GetDeck().GetTail(),
		Baseline: read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing the deck answered %+v", answer.Msg)
	}

	after := deck(t, f, "Animals.md")
	if target := len(after.GetDeck().GetCards()); target != 2 {
		t.Errorf("the deck now holds %d cards, and two were written", target)
	}
	if targetVal := onDisk(t, f.root, "Animals.md"); strings.Contains(targetVal, "Injected") {
		t.Errorf("a card was smuggled into the file: %q", targetVal)
	}
}

// TestACardUnderASectionTheDeckDoesNotHoldIsRefused. Writing it under whichever
// section stands last moves somebody's card and says nothing about it.
func TestACardBelowASectionTheDeckDoesNotHoldIsRefused(t *testing.T) {
	f := newCutting(t, map[string]string{"Animal.md": animal, "Animals.md": dividedDeck})

	read := deck(t, f, "Animals.md")
	before := onDisk(t, f.root, "Animals.md")
	cards := read.GetDeck().GetCards()
	nowhere := int32(99)
	cards[0].SectionIndex = &nowhere

	_, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&flashcardsv1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Sections: read.GetDeck().GetSections(),
		Cards:    cards,
		Tail:     read.GetDeck().GetTail(),
		Baseline: read.GetAt(),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("the write answered %v", err)
	}
	if target := onDisk(t, f.root, "Animals.md"); target != before {
		t.Errorf("the deck was written anyway: %q", target)
	}
}

// TestACardWithNoCardModelKeepsTheHeadingItStandsUnder. Nothing can say which of
// that card's fields is first, so nothing can write its heading again: it is
// left exactly currently, and a person who changed nothing sees the file
// they wrote.
func TestACardWithNoCardModelKeepsTheHeadingItExistsBelow(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": threeCards,
	})

	read := deck(t, f, "Animals.md")
	if target := read.GetDeck().GetCards()[1].GetHeading(); target != "Alpaca" {
		t.Fatalf("the card under no wikilink came back under the heading %q", target)
	}
	answer, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&flashcardsv1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Sections: read.GetDeck().GetSections(),
		Cards:    read.GetDeck().GetCards(),
		Tail:     read.GetDeck().GetTail(),
		Baseline: read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing a deck straight back answered %+v", answer.Msg)
	}
	if targetVal := onDisk(t, f.root, "Animals.md"); !strings.HasSuffix(targetVal, cardRun) {
		t.Errorf("the deck came back as %q", targetVal)
	}
}

// TestAListOfCardModelsCutShortSaysHowManyTheVaultHolds. A list that stops at a
// ceiling and says nothing about it reads as all there is.
func TestAListOfCardModelsSplitShortReturnsHowManyTheVaultContains(t *testing.T) {
	f := newCutting(t, map[string]string{
		"Animal.md": animal,
		"Term.md":   "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n\n## Recall\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
		"Place.md":  "---\ntype: card-model\nfields:\n  - Place\n  - Country\n---\n\n## Recall\n\n### Front\n\n{{Place}}\n\n### Back\n\n{{Country}}\n",
		"Loose.md":  "# An ordinary note\n",
	})

	answer, err := f.client.ListCardModels(t.Context(),
		connect.NewRequest(&flashcardsv1.ListCardModelsRequest{Limit: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if target := answer.Msg.GetCardModels(); len(target) != 2 {
		t.Fatalf("a list asked for two came back with %d", len(target))
	}
	if total := answer.Msg.GetTotal(); total != 3 {
		t.Errorf("the vault holds three card models and the answer says %d", total)
	}
	if fields := answer.Msg.GetCardModels()[0].GetFields(); len(fields) != 2 || fields[0] != "Name" {
		t.Errorf("the first card model asks for %v", fields)
	}
}

// TestADeckMadeIsADeckToRead. A deck is made where there was no file, and it
// says it is a deck from the moment it exists, so the read that follows is a
// deck's read and not a note's.
func TestADeckMadeIsADeckToRead(t *testing.T) {
	f := newCutting(t, map[string]string{"Animal.md": animal})

	answer, err := f.client.CreateDeck(t.Context(), connect.NewRequest(&flashcardsv1.CreateDeckRequest{
		Title: "Camelids", Path: "decks",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("making a deck answered %v", code)
	}
	if path := answer.Msg.GetPath(); path != "decks/Camelids.md" {
		t.Fatalf("the deck was filed at %q", path)
	}
	if target := onDisk(t, f.root, "decks/Camelids.md"); !strings.Contains(target, "type: deck\n") {
		t.Errorf("the file does not say what it is: %q", target)
	}

	read := deck(t, f, "decks/Camelids.md")
	if code := read.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the deck just made failed with %v", code)
	}
	if cards := read.GetDeck().GetCards(); len(cards) != 0 {
		t.Errorf("a deck of no cards came back with %+v", cards)
	}
}

// TestACardModelMadeDeclaresTheFieldsItWasGiven. A card model is made with the
// fields a card cut by it is asked for, the first of which names the card.
func TestACardModelMadeDeclaresTheFieldsItWasGiven(t *testing.T) {
	f := newCutting(t, nil)

	answer, err := f.client.CreateCardModel(t.Context(), connect.NewRequest(&flashcardsv1.CreateCardModelRequest{
		Title: "Bird", Path: "cards", Fields: []string{"Species", "Wingspan"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("making a card model answered %v", code)
	}
	if path := answer.Msg.GetPath(); path != "cards/Bird.md" {
		t.Fatalf("the card model was filed at %q", path)
	}

	read, err := f.client.ReadCardModel(t.Context(), connect.NewRequest(&flashcardsv1.ReadCardModelRequest{
		Path: "cards/Bird.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := read.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the card model just made failed with %v", code)
	}
	if fields := read.Msg.GetCardModel().GetFields(); len(fields) != 2 || fields[0] != "Species" {
		t.Errorf("the card model asks for %v", fields)
	}
}

// TestRenamingAFieldReachesTheDecksThatCardModelCuts. A field's name is written
// where the card model declares it and as a heading in every card that card model
// cuts, so the rename reaches them all and says which decks it wrote.
func TestRenamingAFieldReachesTheDecksThatCardModelSplits(t *testing.T) {
	f := newCutting(t, map[string]string{
		"cards/Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Llama\n\n[[cards/Animal]]\n\n### Height\n\nabout 45\"\n",
	})

	answer, err := f.client.RenameCardModelField(t.Context(), connect.NewRequest(&flashcardsv1.RenameCardModelFieldRequest{
		Path: "cards/Animal.md", From: "Height", To: "Shoulder height",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the rename answered %v", code)
	}
	if decks := answer.Msg.GetDecks(); len(decks) != 1 || decks[0] != "Animals.md" {
		t.Fatalf("the rename says it wrote %v", decks)
	}
	if cards := answer.Msg.GetCards(); cards != 1 {
		t.Errorf("cards = %d, want the one card that card model cuts", cards)
	}
	if len(answer.Msg.GetNotWritten()) != 0 {
		t.Errorf("not written = %+v", answer.Msg.GetNotWritten())
	}
	if target := onDisk(t, f.root, "cards/Animal.md"); !strings.Contains(target, "  - Shoulder height\n") {
		t.Errorf("the card model was not renamed: %q", target)
	}
	if targetVal := onDisk(t, f.root, "Animals.md"); !strings.Contains(targetVal, "### Shoulder height\n") {
		t.Errorf("the card was not renamed: %q", targetVal)
	}
}

// `fields` and in the braces of every template that places it, and both are the one
// file, so one write carries both and the card model declares what its templates place.
func TestRenamingAFieldWritesTheTemplatesOfThatCardModel(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	read, err := f.client.ReadCardModel(t.Context(), connect.NewRequest(&flashcardsv1.ReadCardModelRequest{
		Path: "cards/Animal.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	answer, err := f.client.RenameCardModelField(t.Context(), connect.NewRequest(&flashcardsv1.RenameCardModelFieldRequest{
		Path: "cards/Animal.md", From: "Height", To: "Shoulder height", Baseline: read.Msg.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the rename answered %+v", answer.Msg)
	}

	after, err := f.client.ReadCardModel(t.Context(), connect.NewRequest(&flashcardsv1.ReadCardModelRequest{
		Path: "cards/Animal.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	templates := after.Msg.GetCardModel().GetTemplates()
	if len(templates) != 1 || templates[0].GetBack() != "{{Shoulder height}}" {
		t.Errorf("the template places %+v", templates)
	}
	if problems := after.Msg.GetCardModel().GetProblems(); len(problems) != 0 {
		t.Errorf("the card model declares something its templates do not place: %+v", problems)
	}
	// One write, so the fingerprint that came back is the file on disk and the
	// next write of it lands.
	if answer.Msg.GetAt().GetSize() != int64(len(onDisk(t, f.root, "cards/Animal.md"))) {
		t.Errorf("the fingerprint is not the file: %+v", answer.Msg.GetAt())
	}
}

// TestRenamingAFieldLeavesAloneACardModelThatChangedSinceItWasRead. Somebody
// editing their own card model outranks a client that read it, thought about it
// and arrived late.
func TestRenamingAFieldPreservesAloneACardModelThatChangedSinceItWasRead(t *testing.T) {
	f := newCutting(t, map[string]string{
		"cards/Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Llama\n\n[[cards/Animal]]\n\n### Height\n\nabout 45\"\n",
	})

	read, err := f.client.ReadCardModel(t.Context(), connect.NewRequest(&flashcardsv1.ReadCardModelRequest{
		Path: "cards/Animal.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	// The person writes their own card model while the client is thinking about
	// what it read.
	theirs := animal + "\n## Name it\n\n### Front\n\n{{Height}}\n\n### Back\n\n{{Name}}\n"
	if err := os.WriteFile(
		filepath.Join(f.root, "cards", "Animal.md"), []byte(theirs), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	deckBefore := onDisk(t, f.root, "Animals.md")

	answer, err := f.client.RenameCardModelField(t.Context(), connect.NewRequest(&flashcardsv1.RenameCardModelFieldRequest{
		Path: "cards/Animal.md", From: "Height", To: "Shoulder height", Baseline: read.Msg.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_STALE {
		t.Errorf("a rename over a card model the person had edited answered %+v", answer.Msg)
	}
	if target := onDisk(t, f.root, "cards/Animal.md"); target != theirs {
		t.Errorf("the card model on disk is now %q", target)
	}
	// The card model leads, so a rename it rejected reaches no deck.
	if targetVal := onDisk(t, f.root, "Animals.md"); targetVal != deckBefore {
		t.Errorf("a deck was written\n was %q\n now %q", deckBefore, targetVal)
	}
}

// TestADeckIsRefusedWhereTheNoteIsACardModel. Two files must agree for a card to
// be drawn, and a client handed the wrong one is told which it got.
func TestADeckIsRefusedWhereTheNoteIsACardModel(t *testing.T) {
	f := newCutting(t, map[string]string{"Animal.md": animal})

	read := deck(t, f, "Animal.md")
	if code := read.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_NOT_A_DECK {
		t.Errorf("a card model asked for as a deck answered %v", code)
	}
	if read.GetDeck() != nil {
		t.Error("a rejected deck came back with cards on it")
	}
}

// TestACardNamesItsCardModelTheWayALinkNamesANote. The wikilink under a card's
// heading is an ordinary link, so a path from the root and a name carrying an
// alias reach the same card model, and the answer says where that card model is filed.
func TestACardNamesItsCardModelTheWayALinkNamesANote(t *testing.T) {
	f := newCutting(t, map[string]string{
		"cards/Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Llama\n\n[[cards/Animal]]\n\n### Name\n\nLlama\n\n" +
			"## Alpaca\n\n[[Animal|животное]]\n\n### Name\n\nAlpaca\n\n" +
			"## Vicuña\n\n[[Nowhere]]\n\n### Name\n\nVicuña\n",
	})

	target := deck(t, f, "Animals.md").GetDeck().GetCards()
	if len(target) != 3 {
		t.Fatalf("the deck came back with %d cards", len(target))
	}
	filed := map[string]string{"Llama": "cards/Animal.md", "Alpaca": "cards/Animal.md", "Vicuña": ""}
	for _, card := range target {
		name := getFieldValue(card, "Name")
		if pos := card.GetCardModelPath(); pos != filed[name] {
			t.Errorf("the card model of %s is filed at %q, want %q", name, pos, filed[name])
		}
	}
	if content := target[0].GetCardModelLink(); content != "cards/Animal" {
		t.Errorf("the wikilink of the first card came back as %q", content)
	}
}

// card model is what the window reads a card model as.
func cardModel(t *testing.T, f *cutting, path string) *flashcardsv1.ReadCardModelResponse {
	t.Helper()
	answer, err := f.client.ReadCardModel(t.Context(), connect.NewRequest(&flashcardsv1.ReadCardModelRequest{
		Path: path,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg
}

// TestACardModelIsWrittenWithItsTemplatesAndItsFields. The templates are the body and the
// fields are one key of the frontmatter, and both are the one file, so what
// comes back is the file holding both.
func TestACardModelIsSavedWithItsTemplatesAndItsFields(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	read := cardModel(t, f, "cards/Animal.md")
	templates := read.GetCardModel().GetTemplates()
	templates[0].Back = "{{Height}} at the shoulder"

	answer, err := f.client.WriteCardModel(t.Context(), connect.NewRequest(&flashcardsv1.WriteCardModelRequest{
		Path:      "cards/Animal.md",
		Fields:    read.GetCardModel().GetFields(),
		Templates: templates,
		Preamble:  read.GetCardModel().GetPreamble(),
		Tail:      read.GetCardModel().GetTail(),
		Baseline:  read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing a card model answered %+v", answer.Msg)
	}

	target := onDisk(t, f.root, "cards/Animal.md")
	if !strings.Contains(target, "{{Height}} at the shoulder") {
		t.Errorf("the template is not the one written: %q", target)
	}
	if !strings.Contains(target, "fields:\n  - Name\n  - Height\n") {
		t.Errorf("the card model declares something else: %q", target)
	}
	// The fingerprint that comes back is the file on disk, so the write that
	// follows this one lands.
	if answer.Msg.GetAt().GetSize() != int64(len(target)) {
		t.Errorf("the fingerprint is not the file: %+v", answer.Msg.GetAt())
	}
}

// TestACardModelIsWrittenWithTheFieldsItNowDeclares. A person adding a field
// writes the card model's frontmatter and its templates in the one act.
func TestACardModelIsSavedWithTheFieldsItNowDeclares(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	read := cardModel(t, f, "cards/Animal.md")
	answer, err := f.client.WriteCardModel(t.Context(), connect.NewRequest(&flashcardsv1.WriteCardModelRequest{
		Path:      "cards/Animal.md",
		Fields:    []string{"Name", "Height", "Life span"},
		Templates: read.GetCardModel().GetTemplates(),
		Preamble:  read.GetCardModel().GetPreamble(),
		Tail:      read.GetCardModel().GetTail(),
		Baseline:  read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing a card model answered %+v", answer.Msg)
	}

	after := cardModel(t, f, "cards/Animal.md")
	if fields := after.GetCardModel().GetFields(); len(fields) != 3 || fields[2] != "Life span" {
		t.Errorf("the card model asks for %v", fields)
	}
	if answer.Msg.GetAt().GetSize() != int64(len(onDisk(t, f.root, "cards/Animal.md"))) {
		t.Errorf("the fingerprint is not the file: %+v", answer.Msg.GetAt())
	}
}

// TestWritingACardModelThatChangedSinceItWasReadWritesNothing. Changed says
// nothing was written, so the templates are not on disk either and what the person
// wrote is the file.
func TestWritingACardModelThatChangedSinceItWasReadWritesNothing(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	read := cardModel(t, f, "cards/Animal.md")
	// The person writes their own card model while the client is thinking about
	// what it read.
	theirs := animal + "\n## Name it\n\n### Front\n\n{{Height}}\n\n### Back\n\n{{Name}}\n"
	if err := os.WriteFile(
		filepath.Join(f.root, "cards", "Animal.md"), []byte(theirs), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	templates := read.GetCardModel().GetTemplates()
	templates[0].Front = "{{Species}}"
	answer, err := f.client.WriteCardModel(t.Context(), connect.NewRequest(&flashcardsv1.WriteCardModelRequest{
		Path:      "cards/Animal.md",
		Fields:    []string{"Species", "Height"},
		Templates: templates,
		Preamble:  read.GetCardModel().GetPreamble(),
		Tail:      read.GetCardModel().GetTail(),
		Baseline:  read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != flashcardsv1.ErrorCode_ERROR_CODE_STALE {
		t.Fatalf("a write over a card model the person had edited answered %+v", answer.Msg)
	}
	if target := onDisk(t, f.root, "cards/Animal.md"); target != theirs {
		t.Errorf("the card model on disk is now %q", target)
	}
}

// TestWritingACardModelWhereTheVaultHoldsNoNoteIsRefused. A write puts fields and
// templates into a card model that is there, and a card model is put in the vault by name.
func TestWritingACardModelWhereTheVaultContainsNoNoteIsRefused(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	answer, err := f.client.WriteCardModel(t.Context(), connect.NewRequest(&flashcardsv1.WriteCardModelRequest{
		Path:   "cards/Nowhere.md",
		Fields: []string{"Name"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("writing a card model where the vault holds no note answered %v", code)
	}
	if _, err := os.Stat(filepath.Join(f.root, "cards", "Nowhere.md")); err == nil {
		t.Error("a file was made where the write failed")
	}
}

// TestACardModelMadeOnANameAlreadyTakenIsRefused. Nothing is written over: the
func TestACardModelMadeOnANameAlreadyTakenIsRefused(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	answer, err := f.client.CreateCardModel(t.Context(), connect.NewRequest(&flashcardsv1.CreateCardModelRequest{
		Title: "Animal", Path: "cards", Fields: []string{"Species"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("making a card model on a name already taken answered %v", code)
	}
	if target := onDisk(t, f.root, "cards/Animal.md"); target != animal {
		t.Errorf("the card model that was there is now %q", target)
	}
}

// TestADeckMadeOnANameAlreadyTakenIsRefused. What holds for a card model holds for
// a deck.
func TestADeckMadeOnANameAlreadyTakenIsRefused(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	answer, err := f.client.CreateDeck(t.Context(), connect.NewRequest(&flashcardsv1.CreateDeckRequest{
		Title: "Animal", Path: "cards",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("making a deck on a name already taken answered %v", code)
	}
	if target := onDisk(t, f.root, "cards/Animal.md"); target != animal {
		t.Errorf("the file that was there is now %q", target)
	}
}

// TestAPathThatLeavesTheVaultIsTheClientsToCorrect. Nothing escapes the vault
// either way; what is being said is whose mistake it was, and a path the client
// built is the client's.
func TestAPathThatPreservesTheVaultIsTheClientsToCorrect(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	_, err := f.client.ReadDeck(t.Context(), connect.NewRequest(&flashcardsv1.ReadDeckRequest{
		Path: "../../etc/passwd",
	}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("reading a deck outside the vault answered %v (%v)", got, err)
	}

	_, err = f.client.ReadCardModel(t.Context(), connect.NewRequest(&flashcardsv1.ReadCardModelRequest{
		Path: "../../etc/passwd",
	}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("reading a card model outside the vault answered %v (%v)", got, err)
	}

	_, err = f.client.CreateDeck(t.Context(), connect.NewRequest(&flashcardsv1.CreateDeckRequest{
		Title: "Elsewhere", Path: "../elsewhere",
	}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("making a deck outside the vault answered %v (%v)", got, err)
	}
}

// TestARenameOntoANameTheCardModelDeclaresIsTheClientsToCorrect. A field a
// card model already declares is a name the client is holding wrongly, and it is
// told so.
func TestARenameOntoANameTheCardModelDeclaresIsTheClientsToCorrect(t *testing.T) {
	f := newCutting(t, map[string]string{"cards/Animal.md": animal})

	_, err := f.client.RenameCardModelField(t.Context(),
		connect.NewRequest(&flashcardsv1.RenameCardModelFieldRequest{
			Path: "cards/Animal.md", From: "Height", To: "Name",
		}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("renaming a field onto a declared name answered %v (%v)", got, err)
	}
	if target := onDisk(t, f.root, "cards/Animal.md"); target != animal {
		t.Errorf("the card model is now %q", target)
	}
}

// taken.
func TestAFolderThatIsAFileIsAnAnswerAPersonCanActOn(t *testing.T) {
	f := newCutting(t, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.CreateDeck(t.Context(), connect.NewRequest(&flashcardsv1.CreateDeckRequest{
		Title: "Camelids", Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != flashcardsv1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("making a deck under a file answered %v", code)
	}
	if target := onDisk(t, f.root, "Entropy.md"); target != "# Entropy\n" {
		t.Errorf("the file in the way is now %q", target)
	}
}

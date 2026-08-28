package webui_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// A stencil says what a card has and a deck holds the cards. What the window
// asks about either is what these say.

// cutting is a vault with the notes given in it, and a client asking about its
// decks and its stencils the way the window does.
type cutting struct {
	client numenv1connect.CardsServiceClient
	root   string
}

func dealing(t *testing.T, notes map[string]string) *cutting {
	t.Helper()

	f := quitting(t, nil, notes)
	scanned(t, f)

	route, handler := numenv1connect.NewCardsServiceHandler(f.opened.API)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &cutting{
		client: numenv1connect.NewCardsServiceClient(server.Client(), server.URL),
		root:   f.root,
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

const animal = "---\ntype: stencil\nfields:\n  - Name\n  - Height\n---\n\n" +
	"## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n"

// cardRun is the cards of a deck as somebody wrote them, the middle one under
// no wikilink, so it names no stencil and the problem stands against that one.
const cardRun = "## Llama\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n\n" +
	"## Alpaca\n\nsomebody's prose\n\n### Height\n\nabout 36\"\n\n" +
	"## Vicuña\n\n[[Animal]]\n\n### Height\n\nabout 34\"\n"

const threeCards = "---\ntype: deck\n---\n\n" + cardRun

func deck(t *testing.T, f *cutting, path string) *v1.ReadDeckResponse {
	t.Helper()
	answer, err := f.client.ReadDeck(t.Context(), connect.NewRequest(&v1.ReadDeckRequest{Path: path}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg
}

// TestAProblemStandsAgainstTheCardItIsAbout. A card is addressed by where it
// stands, so a problem's position is an index into the cards the same answer
// carried.
func TestAProblemStandsAgainstTheCardItIsAbout(t *testing.T) {
	f := dealing(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": threeCards,
	})

	read := deck(t, f, "Animals.md")
	held := read.GetDeck().GetCards()
	if len(held) != 3 {
		t.Fatalf("the deck came back with %d cards", len(held))
	}
	problems := read.GetDeck().GetProblems()
	if len(problems) != 1 {
		t.Fatalf("the deck came back with %d problems: %+v", len(problems), problems)
	}
	if fault := problems[0].GetFault(); fault != v1.Fault_FAULT_CARD_WITHOUT_A_STENCIL {
		t.Errorf("the card with no wikilink under it is reported as %v", fault)
	}
	at := problems[0].GetCard()
	if problems[0].Card == nil {
		t.Fatal("a problem about a card came back standing against none")
	}
	if name := held[at].GetName(); name != "Alpaca" {
		t.Errorf("the problem stands against card %d, which is %q", at, name)
	}
}

// TestWritingADeckLeavesAloneOneThatChangedSinceItWasRead. Somebody editing
// their own deck outranks a client that read it, thought about it and arrived
// late.
func TestWritingADeckLeavesAloneOneThatChangedSinceItWasRead(t *testing.T) {
	f := dealing(t, map[string]string{
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

	answer, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&v1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Cards:    read.GetDeck().GetCards()[:1],
		Tail:     read.GetDeck().GetTail(),
		Seen:     read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !answer.Msg.GetChanged() {
		t.Error("a write over a deck the person had edited was not answered as changed")
	}
	if held := onDisk(t, f.root, "Animals.md"); held != theirs {
		t.Errorf("the deck on disk is now %q", held)
	}
}

// TestADeckWrittenBackKeepsTheCardsItHeld. Every card the client did not touch
// arrives on the other side as the bytes it went in as, the prose under a
// card's wikilink among them.
func TestADeckWrittenBackKeepsTheCardsItHeld(t *testing.T) {
	f := dealing(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": threeCards,
	})

	read := deck(t, f, "Animals.md")
	answer, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&v1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Cards:    read.GetDeck().GetCards(),
		Tail:     read.GetDeck().GetTail(),
		Seen:     read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetChanged() || answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("writing a deck straight back answered %+v", answer.Msg)
	}
	if held := onDisk(t, f.root, "Animals.md"); !strings.HasSuffix(held, cardRun) {
		t.Errorf("the deck came back as %q", held)
	}
}

// TestAListOfStencilsCutShortSaysHowManyTheVaultHolds. A list that stops at a
// ceiling and says nothing about it reads as all there is.
func TestAListOfStencilsCutShortSaysHowManyTheVaultHolds(t *testing.T) {
	f := dealing(t, map[string]string{
		"Animal.md": animal,
		"Term.md":   "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n\n## Recall\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
		"Place.md":  "---\ntype: stencil\nfields:\n  - Place\n  - Country\n---\n\n## Recall\n\n### Front\n\n{{Place}}\n\n### Back\n\n{{Country}}\n",
		"Loose.md":  "# An ordinary note\n",
	})

	answer, err := f.client.Stencils(t.Context(), connect.NewRequest(&v1.StencilsRequest{Limit: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if held := answer.Msg.GetStencils(); len(held) != 2 {
		t.Fatalf("a list asked for two came back with %d", len(held))
	}
	if held := answer.Msg.GetHeld(); held != 3 {
		t.Errorf("the vault holds three stencils and the answer says %d", held)
	}
	if fields := answer.Msg.GetStencils()[0].GetFields(); len(fields) != 2 || fields[0] != "Name" {
		t.Errorf("the first stencil asks for %v", fields)
	}
}

// TestADeckMadeIsADeckToRead. A deck is made where there was no file, and it
// says it is a deck from the moment it exists, so the read that follows is a
// deck's read and not a note's.
func TestADeckMadeIsADeckToRead(t *testing.T) {
	f := dealing(t, map[string]string{"Animal.md": animal})

	answer, err := f.client.MakeDeck(t.Context(), connect.NewRequest(&v1.MakeDeckRequest{
		Title: "Camelids", Folder: "decks",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("making a deck answered %v", refusal)
	}
	if path := answer.Msg.GetPath(); path != "decks/Camelids.md" {
		t.Fatalf("the deck was filed at %q", path)
	}
	if held := onDisk(t, f.root, "decks/Camelids.md"); !strings.Contains(held, "type: deck\n") {
		t.Errorf("the file does not say what it is: %q", held)
	}

	read := deck(t, f, "decks/Camelids.md")
	if refusal := read.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the deck just made was refused with %v", refusal)
	}
	if cards := read.GetDeck().GetCards(); len(cards) != 0 {
		t.Errorf("a deck of no cards came back with %+v", cards)
	}
}

// TestAStencilMadeDeclaresTheFieldsItWasGiven. A stencil is made with the
// fields a card cut by it is asked for, the first of which names the card.
func TestAStencilMadeDeclaresTheFieldsItWasGiven(t *testing.T) {
	f := dealing(t, nil)

	answer, err := f.client.MakeStencil(t.Context(), connect.NewRequest(&v1.MakeStencilRequest{
		Title: "Bird", Folder: "cards", Fields: []string{"Species", "Wingspan"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("making a stencil answered %v", refusal)
	}
	if path := answer.Msg.GetPath(); path != "cards/Bird.md" {
		t.Fatalf("the stencil was filed at %q", path)
	}

	read, err := f.client.ReadStencil(t.Context(), connect.NewRequest(&v1.ReadStencilRequest{
		Path: "cards/Bird.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := read.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the stencil just made was refused with %v", refusal)
	}
	if fields := read.Msg.GetStencil().GetFields(); len(fields) != 2 || fields[0] != "Species" {
		t.Errorf("the stencil asks for %v", fields)
	}
}

// TestRenamingAFieldReachesTheDecksThatStencilCuts. A field's name is written
// where the stencil declares it and as a heading in every card that stencil
// cuts, so the rename reaches them all and says which decks it wrote.
func TestRenamingAFieldReachesTheDecksThatStencilCuts(t *testing.T) {
	f := dealing(t, map[string]string{
		"cards/Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Llama\n\n[[cards/Animal]]\n\n### Height\n\nabout 45\"\n",
	})

	answer, err := f.client.RenameField(t.Context(), connect.NewRequest(&v1.RenameFieldRequest{
		Path: "cards/Animal.md", From: "Height", To: "Shoulder height",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the rename answered %v", refusal)
	}
	if decks := answer.Msg.GetDecks(); len(decks) != 1 || decks[0] != "Animals.md" {
		t.Fatalf("the rename says it wrote %v", decks)
	}
	if cards := answer.Msg.GetCards(); cards != 1 {
		t.Errorf("cards = %d, want the one card that stencil cuts", cards)
	}
	if len(answer.Msg.GetNotWritten()) != 0 {
		t.Errorf("not written = %+v", answer.Msg.GetNotWritten())
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); !strings.Contains(held, "  - Shoulder height\n") {
		t.Errorf("the stencil was not renamed: %q", held)
	}
	if held := onDisk(t, f.root, "Animals.md"); !strings.Contains(held, "### Shoulder height\n") {
		t.Errorf("the card was not renamed: %q", held)
	}
}

// TestRenamingAFieldWritesTheFacesOfThatStencil. A field's name stands in
// `fields` and in the braces of every face that places it, and both are the one
// file, so one write carries both and the stencil declares what its faces place.
func TestRenamingAFieldWritesTheFacesOfThatStencil(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	read, err := f.client.ReadStencil(t.Context(), connect.NewRequest(&v1.ReadStencilRequest{
		Path: "cards/Animal.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	answer, err := f.client.RenameField(t.Context(), connect.NewRequest(&v1.RenameFieldRequest{
		Path: "cards/Animal.md", From: "Height", To: "Shoulder height", Seen: read.Msg.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetChanged() || answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the rename answered %+v", answer.Msg)
	}

	after, err := f.client.ReadStencil(t.Context(), connect.NewRequest(&v1.ReadStencilRequest{
		Path: "cards/Animal.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	faces := after.Msg.GetStencil().GetFaces()
	if len(faces) != 1 || faces[0].GetBack() != "{{Shoulder height}}" {
		t.Errorf("the face places %+v", faces)
	}
	if problems := after.Msg.GetStencil().GetProblems(); len(problems) != 0 {
		t.Errorf("the stencil declares something its faces do not place: %+v", problems)
	}
	// One write, so the fingerprint that came back is the file on disk and the
	// next write of it lands.
	if answer.Msg.GetAt().GetSize() != int64(len(onDisk(t, f.root, "cards/Animal.md"))) {
		t.Errorf("the fingerprint is not the file: %+v", answer.Msg.GetAt())
	}
}

// TestRenamingAFieldLeavesAloneAStencilThatChangedSinceItWasRead. Somebody
// editing their own stencil outranks a client that read it, thought about it
// and arrived late.
func TestRenamingAFieldLeavesAloneAStencilThatChangedSinceItWasRead(t *testing.T) {
	f := dealing(t, map[string]string{
		"cards/Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Llama\n\n[[cards/Animal]]\n\n### Height\n\nabout 45\"\n",
	})

	read, err := f.client.ReadStencil(t.Context(), connect.NewRequest(&v1.ReadStencilRequest{
		Path: "cards/Animal.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	// The person writes their own stencil while the client is thinking about
	// what it read.
	theirs := animal + "\n## Name it\n\n### Front\n\n{{Height}}\n\n### Back\n\n{{Name}}\n"
	if err := os.WriteFile(
		filepath.Join(f.root, "cards", "Animal.md"), []byte(theirs), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	deckBefore := onDisk(t, f.root, "Animals.md")

	answer, err := f.client.RenameField(t.Context(), connect.NewRequest(&v1.RenameFieldRequest{
		Path: "cards/Animal.md", From: "Height", To: "Shoulder height", Seen: read.Msg.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !answer.Msg.GetChanged() {
		t.Errorf("a rename over a stencil the person had edited answered %+v", answer.Msg)
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); held != theirs {
		t.Errorf("the stencil on disk is now %q", held)
	}
	// The stencil leads, so a rename it refused reaches no deck.
	if held := onDisk(t, f.root, "Animals.md"); held != deckBefore {
		t.Errorf("a deck was written\n was %q\n now %q", deckBefore, held)
	}
}

// TestADeckIsRefusedWhereTheNoteIsAStencil. Two files must agree for a card to
// be drawn, and a client handed the wrong one is told which it got.
func TestADeckIsRefusedWhereTheNoteIsAStencil(t *testing.T) {
	f := dealing(t, map[string]string{"Animal.md": animal})

	read := deck(t, f, "Animal.md")
	if refusal := read.GetRefusal(); refusal != v1.Refusal_REFUSAL_NOT_A_DECK {
		t.Errorf("a stencil asked for as a deck answered %v", refusal)
	}
	if read.GetDeck() != nil {
		t.Error("a refused deck came back with cards on it")
	}
}

// TestACardNamesItsStencilTheWayALinkNamesANote. The wikilink under a card's
// heading is an ordinary link, so a path from the root and a name carrying an
// alias reach the same stencil, and the answer says where that stencil is filed.
func TestACardNamesItsStencilTheWayALinkNamesANote(t *testing.T) {
	f := dealing(t, map[string]string{
		"cards/Animal.md": animal,
		"Animals.md": "---\ntype: deck\n---\n\n" +
			"## Llama\n\n[[cards/Animal]]\n\n### Height\n\nabout 45\"\n\n" +
			"## Alpaca\n\n[[Animal|животное]]\n\n### Height\n\nabout 36\"\n\n" +
			"## Vicuña\n\n[[Nowhere]]\n\n### Height\n\nabout 34\"\n",
	})

	held := deck(t, f, "Animals.md").GetDeck().GetCards()
	if len(held) != 3 {
		t.Fatalf("the deck came back with %d cards", len(held))
	}
	filed := map[string]string{"Llama": "cards/Animal.md", "Alpaca": "cards/Animal.md", "Vicuña": ""}
	for _, card := range held {
		if at := card.GetStencilAt(); at != filed[card.GetName()] {
			t.Errorf("the stencil of %s is filed at %q, want %q",
				card.GetName(), at, filed[card.GetName()])
		}
	}
	if written := held[0].GetStencil(); written != "cards/Animal" {
		t.Errorf("the wikilink of the first card came back as %q", written)
	}
}

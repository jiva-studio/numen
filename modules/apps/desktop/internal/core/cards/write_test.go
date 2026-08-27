package cards_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

// deck is the awkward one every write is tested against: a key nobody owns, a
// card of a stencil and a card of none, and fields written in an order of the
// person's own.
const deck = "---\n" +
	"id: 01J8F3K2M9QRSTVWXYZ012\n" +
	"type: deck\n" +
	"mine: keep me verbatim\n" +
	"---\n" +
	"\n" +
	"Cards I am learning.\n" +
	"\n" +
	"## Llama\n" +
	"\n" +
	"[[Animal]]\n" +
	"\n" +
	"Asked of me by Anna.\n" +
	"\n" +
	"### Life span\n" +
	"\n" +
	"about 20 years\n" +
	"\n" +
	"### Height\n" +
	"\n" +
	"about 45\"\n" +
	"\n" +
	"## шраддха\n" +
	"\n" +
	"[[Термин]]\n" +
	"\n" +
	"### Значение\n" +
	"\n" +
	"вера\n"

// A deck opened and not changed comes back byte for byte. Anything less is a
// diff the person did not ask for, on every save, forever.
func TestOpenAndWriteChangesNothing(t *testing.T) {
	for name, raw := range map[string]string{
		"the deck":          deck,
		"crlf":              strings.ReplaceAll(deck, "\n", "\r\n"),
		"no frontmatter":    "## Llama\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n",
		"no trailing break": "---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]",
		"empty deck":        "---\ntype: deck\n---\n",
		"bom":               "\xef\xbb\xbf---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n",
		"comments and order": "---\n# a note to myself\nzebra: 1\n\ntype: deck\n---\n" +
			"\n## Llama\n\n\n\n[[Animal]]\n\n\n### Height\n\n\nabout 45\"\n\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "Animals.md")
			if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}

			f, err := cards.OpenDeck(before)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			f.Deck(domain.FileRef{Path: "Animals.md"})
			if err := os.WriteFile(path, f.Bytes(), 0o600); err != nil {
				t.Fatalf("write back: %v", err)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("the round trip changed the deck\n want %q\n  got %q", before, after)
			}
		})
	}
}

func setValue(t *testing.T, raw, card, field, value string) string {
	t.Helper()
	f, err := cards.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetValue(card, field, value); err != nil {
		t.Fatalf("set %s of %s: %v", field, card, err)
	}
	return string(f.Bytes())
}

// Everything the write did not touch — the frontmatter and its key order, the
// preamble, the lead, the cards around it — is the person's, and comes out as
// it went in.
func TestSetValueChangesOneValueAndNothingElse(t *testing.T) {
	got := setValue(t, deck, "Llama", "Height", "about 46\"")

	if want := strings.Replace(deck, "about 45\"", "about 46\"", 1); got != want {
		t.Errorf("the write reached further than the value\n want %q\n  got %q", want, got)
	}
}

// A card carries its fields in the order the person wrote them, and a write
// does not tidy them into the order the stencil declares.
func TestSetValueKeepsTheOrderTheFieldsWereWrittenIn(t *testing.T) {
	raw := setValue(t, deck, "Llama", "Life span", "about 25 years")

	f, err := cards.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	card, ok := f.Deck(domain.FileRef{}).Card("Llama")
	if !ok {
		t.Fatal("the card is gone")
	}
	if got := fieldsOf(card); !slices.Equal(got, []string{"Life span", "Height"}) {
		t.Errorf("fields = %v, want the order they were written in", got)
	}
	if got, _ := card.Value("Life span"); got != "about 25 years" {
		t.Errorf("Life span = %q", got)
	}
}

// A field the card does not carry yet is written at the end of that card, which
// is the one place that moves nothing already in it.
func TestSetValueAddsAFieldAtTheEndOfTheCard(t *testing.T) {
	got := setValue(t, deck, "Llama", "Weight", "about 130 kg")

	want := strings.Replace(deck, "about 45\"\n\n## шраддха",
		"about 45\"\n\n### Weight\n\nabout 130 kg\n\n## шраддха", 1)
	if got != want {
		t.Errorf("field added wrong\n want %q\n  got %q", want, got)
	}

	// And at the end of the file, where nothing follows it.
	got = setValue(t, deck, "шраддха", "Источник", "[[Бхагавад-гита 4.39]]")
	if want := deck + "\n### Источник\n\n[[Бхагавад-гита 4.39]]\n"; got != want {
		t.Errorf("field added wrong at the end\n want %q\n  got %q", want, got)
	}
}

// A file written with carriage returns is written back with them.
func TestTheFilesOwnLineEndingIsWhatIsWritten(t *testing.T) {
	raw := strings.ReplaceAll(deck, "\n", "\r\n")
	got := setValue(t, raw, "Llama", "Weight", "about 130 kg")

	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare break was written into a file of carriage returns: %q", got)
	}
	want := strings.Replace(raw, "about 45\"\r\n\r\n## шраддха",
		"about 45\"\r\n\r\n### Weight\r\n\r\nabout 130 kg\r\n\r\n## шраддха", 1)
	if got != want {
		t.Errorf("crlf write\n want %q\n  got %q", want, got)
	}
}

func TestSetValueOfACardTheDeckDoesNotHold(t *testing.T) {
	f, err := cards.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetValue("Alpaca", "Height", "about 35\""); !errors.Is(err, cards.ErrNoSuchCard) {
		t.Errorf("err = %v, want ErrNoSuchCard", err)
	}
	if string(f.Bytes()) != deck {
		t.Error("a card that is not there was written anyway")
	}
}

// A deck whose frontmatter cannot be read is never written: repairing it means
// guessing at what the person wrote.
func TestADeckThatCannotBeReadIsNotOpened(t *testing.T) {
	if _, err := cards.OpenDeck([]byte("---\ntype: [deck\n---\n\n## Llama\n")); err == nil {
		t.Error("a broken frontmatter block opened for writing")
	}
}

// The first field's value is the heading, so writing it rewrites the heading
// line and no heading below it. Everything else in the file is the person's.
func TestSetNameRewritesTheHeadingAndNothingElse(t *testing.T) {
	f, err := cards.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetName("Llama", "Lama glama"); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := string(f.Bytes())
	if want := strings.Replace(deck, "## Llama\n", "## Lama glama\n", 1); got != want {
		t.Errorf("the write reached further than the heading\n want %q\n  got %q", want, got)
	}
	// The heading is the whole of a card's identity, so the card answers to the
	// name it now stands under.
	read := f.Deck(domain.FileRef{})
	if _, held := read.Card("Llama"); held {
		t.Error("the deck still holds a card of the old name")
	}
	card, held := read.Card("Lama glama")
	if !held {
		t.Fatal("the card is gone")
	}
	if got := fieldsOf(card); !slices.Equal(got, []string{"Life span", "Height"}) {
		t.Errorf("fields = %v, want the headings under it left where they were", got)
	}
}

// A file written with carriage returns keeps them when a heading is rewritten,
// and the deck is the file it was once the name is written back.
func TestSetNameOnADeckOfCarriageReturns(t *testing.T) {
	raw := strings.ReplaceAll(deck, "\n", "\r\n")
	f, err := cards.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetName("Llama", "Lama glama"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if want := strings.Replace(raw, "## Llama\r\n", "## Lama glama\r\n", 1); string(f.Bytes()) != want {
		t.Errorf("crlf write\n want %q\n  got %q", want, string(f.Bytes()))
	}

	if err := f.SetName("Lama glama", "Llama"); err != nil {
		t.Fatalf("set back: %v", err)
	}
	if got := string(f.Bytes()); got != raw {
		t.Errorf("the name written back is not the file it was\n want %q\n  got %q", raw, got)
	}
}

func TestSetNameOfACardTheDeckDoesNotHold(t *testing.T) {
	f, err := cards.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetName("Alpaca", "Vicuña"); !errors.Is(err, cards.ErrNoSuchCard) {
		t.Errorf("err = %v, want ErrNoSuchCard", err)
	}
	if string(f.Bytes()) != deck {
		t.Error("a card that is not there was written anyway")
	}
}

func TestAddCard(t *testing.T) {
	f, err := cards.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(cards.Card{
		Name:    "Alpaca",
		Stencil: "Animal",
		Lead:    "From the same trip.",
		Values: []cards.Value{
			{Field: "Height", Text: "about 35\""},
			{Field: "Life span", Text: ""},
		},
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	want := deck + "\n## Alpaca\n\n[[Animal]]\n\nFrom the same trip.\n\n### Height\n\nabout 35\"\n\n### Life span\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("card added wrong\n want %q\n  got %q", want, got)
	}

	card, ok := f.Deck(domain.FileRef{}).Card("Alpaca")
	if !ok {
		t.Fatal("the card that was written cannot be read back")
	}
	if card.Stencil != "Animal" || card.Lead != "From the same trip." {
		t.Errorf("card = %+v", card)
	}
	if got := fieldsOf(card); !slices.Equal(got, []string{"Height", "Life span"}) {
		t.Errorf("fields = %v", got)
	}
}

// A field renamed in a stencil is renamed in the cards that stencil cuts, and
// the value under the heading is left as it was.
func TestRenameFieldReachesTheCardsOfThatStencilAlone(t *testing.T) {
	f, err := cards.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetValue("шраддха", "Height", "no such thing"); err != nil {
		t.Fatalf("set: %v", err)
	}

	renamed, err := f.RenameField("Animal", "Height", "Shoulder height")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed != 1 {
		t.Errorf("renamed = %d, want the one card cut by that stencil", renamed)
	}

	deck := f.Deck(domain.FileRef{})
	llama, _ := deck.Card("Llama")
	if got := fieldsOf(llama); !slices.Equal(got, []string{"Life span", "Shoulder height"}) {
		t.Errorf("fields = %v", got)
	}
	if got, _ := llama.Value("Shoulder height"); got != "about 45\"" {
		t.Errorf("the value moved with the heading: %q", got)
	}
	// The card of another stencil keeps the field of that name.
	other, _ := deck.Card("шраддха")
	if got, ok := other.Value("Height"); !ok || got != "no such thing" {
		t.Errorf("a card of another stencil was renamed: %v", fieldsOf(other))
	}
}

const stencil = "---\n" +
	"type: stencil\n" +
	"mine: keep me verbatim\n" +
	"fields:\n" +
	"  - Name\n" +
	"  - Height\n" +
	"  - Life span\n" +
	"---\n" +
	"\n" +
	"## Recognise\n" +
	"\n" +
	"### Front\n" +
	"\n" +
	"{{Name}}\n" +
	"\n" +
	"### Back\n" +
	"\n" +
	"{{Height}}\n" +
	"\n" +
	"## Name it\n" +
	"\n" +
	"### Front\n" +
	"\n" +
	"Which animal is {{Height}}?\n" +
	"\n" +
	"### Back\n" +
	"\n" +
	"{{Name}}\n"

func TestSetFaceWritesOneFace(t *testing.T) {
	f, err := cards.OpenStencil([]byte(stencil))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetFace("Recognise", "![[llama.jpg]]", "**{{Name}}**, {{Life span}}"); err != nil {
		t.Fatalf("set: %v", err)
	}

	want := strings.Replace(stencil,
		"## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
		"## Recognise\n\n### Front\n\n![[llama.jpg]]\n\n### Back\n\n**{{Name}}**, {{Life span}}\n", 1)
	if got := string(f.Bytes()); got != want {
		t.Errorf("face written wrong\n want %q\n  got %q", want, got)
	}
}

func TestSetFaceWritesAFaceTheStencilDoesNotCarry(t *testing.T) {
	f, err := cards.OpenStencil([]byte(stencil))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetFace("Spell it", "{{Height}}", "{{Name}}"); err != nil {
		t.Fatalf("set: %v", err)
	}

	want := stencil + "\n## Spell it\n\n### Front\n\n{{Height}}\n\n### Back\n\n{{Name}}\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("face added wrong\n want %q\n  got %q", want, got)
	}

	n := markdown.Parse(domain.FileRef{Path: "Animal.md"}, f.Bytes())
	if got := faceNames(f.Stencil(n)); !slices.Equal(got, []string{"Recognise", "Name it", "Spell it"}) {
		t.Errorf("faces = %v", got)
	}
}

// A field is added, renamed and taken away in the frontmatter, and the keys
// around it come out of every one of those as the bytes they went in as.
func TestWritingFieldsLeavesTheRestOfTheFrontmatterAlone(t *testing.T) {
	f, err := cards.OpenStencil([]byte(stencil))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetFields([]string{"Name", "Height", "Life span", "Weight"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := f.RenameField("Height", "Shoulder height"); err != nil {
		t.Fatalf("rename: %v", err)
	}

	want := strings.Replace(stencil,
		"fields:\n  - Name\n  - Height\n  - Life span\n",
		"fields:\n  - Name\n  - Shoulder height\n  - Life span\n  - Weight\n", 1)
	if got := string(f.Bytes()); got != want {
		t.Errorf("fields written wrong\n want %q\n  got %q", want, got)
	}
	if got := f.Fields(); !slices.Equal(got, []string{"Name", "Shoulder height", "Life span", "Weight"}) {
		t.Errorf("fields = %v", got)
	}

	// What the stencil says is read from the file in front of the writer, so a
	// face still placing the old name is a placeholder nobody declares.
	n := markdown.Parse(domain.FileRef{Path: "Animal.md"}, f.Bytes())
	read := f.Stencil(n)
	if !slices.Equal(read.Fields, []string{"Name", "Shoulder height", "Life span", "Weight"}) {
		t.Errorf("read fields = %v", read.Fields)
	}
	placed := 0
	for _, p := range read.Problems {
		if p.Check == cards.CheckPlaceholder && p.Field == "Height" {
			placed++
		}
	}
	if placed != 2 {
		t.Errorf("the faces still placing Height were not reported: %+v", read.Problems)
	}
}

// A stencil declaring no such field is not written to.
func TestRenamingAFieldNobodyDeclares(t *testing.T) {
	f, err := cards.OpenStencil([]byte(stencil))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RenameField("Girth", "Waist"); !errors.Is(err, cards.ErrNoSuchField) {
		t.Fatalf("rename = %v, want it refused", err)
	}
	if got := string(f.Bytes()); got != stencil {
		t.Errorf("the file was written: %q", got)
	}
}

// The wikilink under a card's heading is an ordinary link, so an alias on it is
// the person's and is no part of what it names.
func TestACardIsCutByWhatItsLinkPointsAt(t *testing.T) {
	written := "---\ntype: deck\n---\n\n## Llama\n\n[[Animal|the beast]]\n\n### Height\n\nabout 45\"\n"
	f, err := cards.OpenDeck([]byte(written))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	renamed, err := f.RenameField("Animal", "Height", "Shoulder height")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed != 1 {
		t.Fatalf("renamed = %d, want the card the link points at", renamed)
	}
	if got, _ := f.Deck(domain.FileRef{}).Card("Llama"); got.Stencil != "Animal|the beast" {
		t.Errorf("the link was rewritten: %q", got.Stencil)
	}
}

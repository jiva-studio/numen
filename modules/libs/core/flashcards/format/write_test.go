package format_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
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
	"# The ones with fur\n" +
	"\n" +
	"## Llama ^k7m2xq9fzp\n" +
	"\n" +
	"[[Animal]]\n" +
	"\n" +
	"Written on the seed packet.\n" +
	"\n" +
	"### Life span\n" +
	"\n" +
	"about 20 years\n" +
	"\n" +
	"### Height\n" +
	"\n" +
	"about 45\"\n" +
	"\n" +
	"## компост ^zpqrstvwxy\n" +
	"\n" +
	"[[Термин]]\n" +
	"\n" +
	"### Значение\n" +
	"\n" +
	"перегной\n"

// A deck opened and not changed comes back byte for byte, and the cards it
// holds are the ones in the file. Anything less is a diff the person did not
// ask for, on every save, forever.
func TestOpenAndWriteChangesNothing(t *testing.T) {
	for name, one := range map[string]struct {
		raw   string
		cards []string
	}{
		"the deck":       {deck, []string{"Llama", "компост"}},
		"crlf":           {strings.ReplaceAll(deck, "\n", "\r\n"), []string{"Llama", "компост"}},
		"no frontmatter": {"## Llama\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n", []string{"Llama"}},
		"no trailing break": {
			"---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]", []string{"Llama"},
		},
		"no trailing break under a card's heading": {
			"---\ntype: deck\n---\n\n## Llama", []string{"Llama"},
		},
		"no trailing break under a field's heading": {
			"---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n\n### Height", []string{"Llama"},
		},
		"empty deck": {"---\ntype: deck\n---\n", nil},
		"bom":        {"\xef\xbb\xbf---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n", []string{"Llama"}},
		"comments and order": {"---\n# a note to myself\nzebra: 1\n\ntype: deck\n---\n" +
			"\n## Llama\n\n\n\n[[Animal]]\n\n\n### Height\n\n\nabout 45\"\n\n\n", []string{"Llama"}},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "Animals.md")
			if err := os.WriteFile(path, []byte(one.raw), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}

			f, err := format.OpenDeck(before)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if got := names(f.Deck(domain.Fingerprint{Path: "Animals.md"})); !slices.Equal(got, one.cards) {
				t.Errorf("cards = %v, want %v", got, one.cards)
			}
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

// The marks the two cards of the deck above carry.
const (
	llama   = "k7m2xq9fzp"
	compost = "zpqrstvwxy"
)

func setValue(t *testing.T, raw string, card domain.CardID, field, value string) string {
	t.Helper()
	f, err := format.OpenDeck([]byte(raw))
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
	got := setValue(t, deck, llama, "Height", "about 46\"")

	if want := strings.Replace(deck, "about 45\"", "about 46\"", 1); got != want {
		t.Errorf("the write reached further than the value\n want %q\n  got %q", want, got)
	}
}

// A card carries its fields in the order the person wrote them, and a write
// does not tidy them into the order the stencil declares.
func TestSetValueKeepsTheOrderTheFieldsWereWrittenIn(t *testing.T) {
	raw := setValue(t, deck, llama, "Life span", "about 25 years")

	f, err := format.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	card := f.Deck(domain.Fingerprint{}).Cards[0]
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
	got := setValue(t, deck, llama, "Weight", "about 130 kg")

	want := strings.Replace(deck, "about 45\"\n\n## компост",
		"about 45\"\n\n### Weight\n\nabout 130 kg\n\n## компост", 1)
	if got != want {
		t.Errorf("field added wrong\n want %q\n  got %q", want, got)
	}

	// And at the end of the file, where nothing follows it.
	got = setValue(t, deck, compost, "Источник", "[[Компостная куча]]")
	if want := deck + "\n### Источник\n\n[[Компостная куча]]\n"; got != want {
		t.Errorf("field added wrong at the end\n want %q\n  got %q", want, got)
	}
}

// A file written with carriage returns is written back with them.
func TestTheFilesOwnLineEndingIsWhatIsWritten(t *testing.T) {
	raw := strings.ReplaceAll(deck, "\n", "\r\n")
	got := setValue(t, raw, llama, "Weight", "about 130 kg")

	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare break was written into a file of carriage returns: %q", got)
	}
	want := strings.Replace(raw, "about 45\"\r\n\r\n## компост",
		"about 45\"\r\n\r\n### Weight\r\n\r\nabout 130 kg\r\n\r\n## компост", 1)
	if got != want {
		t.Errorf("crlf write\n want %q\n  got %q", want, got)
	}
}

func TestSetValueOfACardTheDeckDoesNotHold(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetValue("wxyz01234t", "Height", "about 35\""); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("err = %v, want ErrNoSuchCard", err)
	}
	if string(f.Bytes()) != deck {
		t.Error("a card that is not there was written anyway")
	}
}

// A deck whose frontmatter cannot be read is never written: repairing it means
// guessing at what the person wrote.
func TestADeckThatCannotBeReadIsNotOpened(t *testing.T) {
	if _, err := format.OpenDeck([]byte("---\ntype: [deck\n---\n\n## Llama\n")); err == nil {
		t.Error("a broken frontmatter block opened for writing")
	}
}

// A card is addressed by its mark, and two cards of one heading are told apart
// by nothing else. Writing the second reaches the second.
func TestSetValueReachesTheCardOfThatMarkAlone(t *testing.T) {
	raw := "---\ntype: deck\n---\n" +
		"\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n" +
		"\n## Llama ^zpqrstvwxy\n\n[[Animal]]\n\n### Height\n\nabout 46\"\n"
	got := setValue(t, raw, "zpqrstvwxy", "Height", "about 47\"")

	if want := strings.Replace(raw, "about 46\"", "about 47\"", 1); got != want {
		t.Errorf("the write reached the wrong card\n want %q\n  got %q", want, got)
	}
}

// A section is made at the end of the deck, given another name where it stands,
// and taken away. Taking one away takes its heading and nothing else: the cards
// that stood under it stay where they are.
func TestTheSplicesASectionNeeds(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddSection(format.Section{Name: "The ones without", Lead: "Added last."}); err != nil {
		t.Fatalf("add: %v", err)
	}
	made := deck + "\n# The ones without\n\nAdded last.\n"
	if got := string(f.Bytes()); got != made {
		t.Errorf("section made wrong\n want %q\n  got %q", made, got)
	}

	if err := f.RenameSection(0, "The furred ones"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	renamed := strings.Replace(made, "# The ones with fur\n", "# The furred ones\n", 1)
	if got := string(f.Bytes()); got != renamed {
		t.Errorf("section renamed wrong\n want %q\n  got %q", renamed, got)
	}

	if err := f.RemoveSection(0); err != nil {
		t.Fatalf("remove: %v", err)
	}
	removed := strings.Replace(renamed, "# The furred ones\n\n", "", 1)
	if got := string(f.Bytes()); got != removed {
		t.Errorf("section removed wrong\n want %q\n  got %q", removed, got)
	}
	read := f.Deck(domain.Fingerprint{})
	if len(read.Cards) != 2 {
		t.Errorf("cards = %+v, want the cards left where they were", read.Cards)
	}
	if read.Cards[0].Section != format.NoSection {
		t.Errorf("the card stands under section %d", read.Cards[0].Section)
	}
}

// A section the deck does not hold is not written to.
func TestASectionTheDeckDoesNotHold(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RenameSection(4, "Nowhere"); !errors.Is(err, format.ErrNoSuchSection) {
		t.Errorf("rename = %v, want ErrNoSuchSection", err)
	}
	if err := f.RemoveSection(4); !errors.Is(err, format.ErrNoSuchSection) {
		t.Errorf("remove = %v, want ErrNoSuchSection", err)
	}
	if string(f.Bytes()) != deck {
		t.Error("a section that is not there was written anyway")
	}
}

// Every field is written under a heading of its own, the first included, and
// the heading line carries the card's mark.
func TestAddCard(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{
		Heading: "Alpaca",
		Mark:    "m9n8b7v6c5",
		Stencil: "Animal",
		Lead:    "From the same trip.",
		Values: []format.Value{
			{Field: "Name", Text: "Alpaca"},
			{Field: "Height", Text: "about 35\""},
			{Field: "Life span", Text: ""},
		},
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	want := deck + "\n## Alpaca ^m9n8b7v6c5\n\n[[Animal]]\n\nFrom the same trip.\n\n" +
		"### Name\n\nAlpaca\n\n### Height\n\nabout 35\"\n\n### Life span\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("card added wrong\n want %q\n  got %q", want, got)
	}

	card, err := f.Deck(domain.Fingerprint{}).Card("m9n8b7v6c5")
	if err != nil {
		t.Fatalf("the card that was written cannot be read back: %v", err)
	}
	if card.Heading != "Alpaca" || card.Stencil != "Animal" || card.Lead != "From the same trip." {
		t.Errorf("card = %+v", card)
	}
	if got := fieldsOf(card); !slices.Equal(got, []string{"Name", "Height", "Life span"}) {
		t.Errorf("fields = %v", got)
	}
}

// A heading is a projection and a mark is minted where the deck is written, so
// a card handed over with neither is written with neither.
func TestAddCardWithNoHeadingAndNoMark(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{Stencil: "Animal"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if want := deck + "\n##\n\n[[Animal]]\n"; string(f.Bytes()) != want {
		t.Errorf("card added wrong\n want %q\n  got %q", want, string(f.Bytes()))
	}
}

// A field renamed in a stencil is renamed in the cards that stencil cuts, and
// the value under the heading is left as it was.
func TestRenameFieldReachesTheCardsOfThatStencilAlone(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetValue(compost, "Height", "no such thing"); err != nil {
		t.Fatalf("set: %v", err)
	}

	renamed, err := f.RenameField(
		map[string]string{"Animal": "Animal.md", "Term": "Term.md"},
		"Animal.md", "Height", "Shoulder height")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed != 1 {
		t.Errorf("renamed = %d, want the one card cut by that stencil", renamed)
	}

	deck := f.Deck(domain.Fingerprint{})
	llama := deck.Cards[0]
	if got := fieldsOf(llama); !slices.Equal(got, []string{"Life span", "Shoulder height"}) {
		t.Errorf("fields = %v", got)
	}
	if got, _ := llama.Value("Shoulder height"); got != "about 45\"" {
		t.Errorf("the value moved with the heading: %q", got)
	}
	// The card of another stencil keeps the field of that name.
	other := deck.Cards[1]
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

// A field is added, renamed and taken away in the frontmatter, and the keys
// around it come out of every one of those as the bytes they went in as.
func TestWritingFieldsLeavesTheRestOfTheFrontmatterAlone(t *testing.T) {
	f, err := format.OpenStencil([]byte(stencil))
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
	want = strings.ReplaceAll(want, "{{Height}}", "{{Shoulder height}}")
	if got := string(f.Bytes()); got != want {
		t.Errorf("fields written wrong\n want %q\n  got %q", want, got)
	}
	if got := f.Fields(); !slices.Equal(got, []string{"Name", "Shoulder height", "Life span", "Weight"}) {
		t.Errorf("fields = %v", got)
	}

	// A field's name is written where the stencil declares it and in the braces
	// of every face that places it, so the stencil that comes out declares what
	// its faces place.
	n := markdown.Parse(domain.Fingerprint{Path: "Animal.md"}, f.Bytes())
	read := f.Stencil(n)
	if !slices.Equal(read.Fields, []string{"Name", "Shoulder height", "Life span", "Weight"}) {
		t.Errorf("read fields = %v", read.Fields)
	}
	for _, p := range read.Problems {
		if p.Check == format.CheckPlaceholder {
			t.Errorf("a face places a name the stencil does not declare: %+v", p)
		}
	}
}

// A stencil declaring no such field is not written to.
func TestRenamingAFieldNobodyDeclares(t *testing.T) {
	f, err := format.OpenStencil([]byte(stencil))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RenameField("Girth", "Waist"); !errors.Is(err, format.ErrNoSuchField) {
		t.Fatalf("rename = %v, want it refused", err)
	}
	if got := string(f.Bytes()); got != stencil {
		t.Errorf("the file was written: %q", got)
	}
}

// A field's name is a name of its own, so a stencil already declaring the name
// asked for is not written: the file would declare it twice, every face would
// place the same field, and the values under the old heading would be shown by
// nothing.
func TestRenamingAFieldOntoAFieldTheStencilDeclares(t *testing.T) {
	f, err := format.OpenStencil([]byte(stencil))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RenameField("Height", "Life span"); !errors.Is(err, format.ErrFieldTaken) {
		t.Fatalf("rename = %v, want it refused", err)
	}
	if got := string(f.Bytes()); got != stencil {
		t.Errorf("the file was written: %q", got)
	}
}

// A face is written with the sides it has. A face missing one is a face that
// lays out nothing, and it is still that face once it has been written.
func TestAFaceIsWrittenWithTheSidesItHas(t *testing.T) {
	f, err := format.OpenStencil([]byte(stencil))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddFace(format.FaceTemplate{Name: "Half a face", Front: "Where does {{Name}} live?"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	want := stencil + "\n## Half a face\n\n### Front\n\nWhere does {{Name}} live?\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("face written wrong\n want %q\n  got %q", want, got)
	}

	read := f.Stencil(markdown.Parse(domain.Fingerprint{Path: "Animal.md"}, f.Bytes()))
	var missing int
	for _, p := range read.Problems {
		if p.Check == format.CheckFaceSide {
			missing++
		}
	}
	if missing != 1 {
		t.Errorf("problems = %+v, want the face that lays out nothing reported", read.Problems)
	}
}

// The wikilink under a card's heading is an ordinary link, so a card is cut by
// the note that link lands on, whatever the brackets spell.
func TestACardIsCutByWhatItsLinkPointsAt(t *testing.T) {
	written := "---\ntype: deck\n---\n\n## Llama\n\n[[Animal|the beast]]\n\n### Height\n\nabout 45\"\n"
	f, err := format.OpenDeck([]byte(written))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	renamed, err := f.RenameField(
		map[string]string{"Animal|the beast": "cards/Animal.md"},
		"cards/Animal.md", "Height", "Shoulder height")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed != 1 {
		t.Fatalf("renamed = %d, want the card the link points at", renamed)
	}
	if got := f.Deck(domain.Fingerprint{}).Cards[0]; got.Stencil != "Animal|the beast" {
		t.Errorf("the link was rewritten: %q", got.Stencil)
	}
}

// A machine must not choose between two cards of one mark: which of the two a
// person meant is a thing only they know, and a write that guesses puts what
// they typed into the other one.
func TestSetValueRefusesADeckOfTwoCardsOfOneMark(t *testing.T) {
	raw := "---\ntype: deck\n---\n" +
		"\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n" +
		"\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Height\n\nabout 46\"\n"
	f, err := format.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetValue("k7m2xq9fzp", "Height", "about 47\""); !errors.Is(err, format.ErrTwoCards) {
		t.Errorf("set = %v, want ErrTwoCards", err)
	}
	if string(f.Bytes()) != raw {
		t.Error("a card was written anyway")
	}
	if _, err := f.Deck(domain.Fingerprint{}).Card("k7m2xq9fzp"); !errors.Is(err, format.ErrTwoCards) {
		t.Errorf("card = %v, want ErrTwoCards", err)
	}
}

// A heading is one line. A heading composed with a break in it carries the
// lines under it into the file, where the next read takes a second `##` for a
// second card and mints it a mark.
func TestAHeadingAndASectionsNameAreCutAtTheFirstBreak(t *testing.T) {
	f, err := format.OpenDeck([]byte("---\ntype: deck\n---\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{
		Heading: "Question\n\n## Injected ^k7m2xq9fzp\n\n[[Term]]\n\n### Q\n\nsmuggled",
		Stencil: "Animal",
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := f.AddSection(format.Section{Name: "Roots\n\n## Also injected"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	read := f.Deck(domain.Fingerprint{})
	if len(read.Cards) != 1 {
		t.Fatalf("cards = %+v, want the one card that was written", read.Cards)
	}
	if read.Cards[0].Heading != "Question" {
		t.Errorf("heading = %q, want it cut at the first break", read.Cards[0].Heading)
	}
	if len(read.Sections) != 1 || read.Sections[0].Name != "Roots" {
		t.Errorf("sections = %+v, want the one that was written", read.Sections)
	}
}

// A field's name is a heading too, so a break in one is cut where a break in a
// card's heading is.
func TestAFieldsNameIsCutAtTheFirstBreak(t *testing.T) {
	f, err := format.OpenDeck([]byte("---\ntype: deck\n---\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{
		Heading: "Question",
		Stencil: "Animal",
		Values: []format.Value{{
			Field: "Answer\n\n## Injected ^k7m2xq9fzp\n\n[[Term]]",
			Text:  "smuggled",
		}},
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	read := f.Deck(domain.Fingerprint{})
	if len(read.Cards) != 1 {
		t.Fatalf("cards = %+v, want the one card that was written", read.Cards)
	}
	if got := read.Cards[0].Values; len(got) != 1 || got[0].Field != "Answer" {
		t.Errorf("values = %+v, want the one field, cut at the first break", got)
	}
}

// Nothing follows the last section, so taking its heading away leaves the break
// the file ends with and no blank line above it.
func TestRemovingTheLastSectionLeavesNoBlankLine(t *testing.T) {
	raw := "---\ntype: deck\n---\n\n# The ones with fur\n\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n" +
		"\n# The ones without\n"
	f, err := format.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RemoveSection(1); err != nil {
		t.Fatalf("remove: %v", err)
	}
	want := "---\ntype: deck\n---\n\n# The ones with fur\n\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("section removed wrong\n want %q\n  got %q", want, got)
	}
}

// The mark is minted where the deck is made whole, so that is what knows it and
// what says which card was given which.
func TestWholeSaysWhichCardsItMinted(t *testing.T) {
	body := "\n## Llama\n\n[[Animal]]\n\n### Name\n\nLlama\n" +
		"\n## Alpaca ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nAlpaca\n" +
		"\n## Vicuña\n\n[[Animal]]\n\n### Name\n\nVicuña\n"
	minting := []domain.CardID{"zpqrstvwxy", "m9n8b7v6c5"}
	mint := func() (domain.CardID, error) {
		out := minting[0]
		minting = minting[1:]
		return out, nil
	}

	_, minted, err := format.Whole(body, map[string]format.Stencil{
		"Animal": {Fields: []string{"Name"}},
	}, mint)
	if err != nil {
		t.Fatalf("whole: %v", err)
	}
	want := []format.Minted{{Card: 0, Mark: "m9n8b7v6c5"}, {Card: 2, Mark: "zpqrstvwxy"}}
	if !slices.Equal(minted, want) {
		t.Errorf("minted = %+v, want %+v", minted, want)
	}
}

// A generator that cannot answer says which card was left without a mark, so
// the write above can name the deck it happened in.
func TestAMarkThatCouldNotBeMintedSaysWhichCard(t *testing.T) {
	body := "\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n" +
		"\n## Alpaca\n\n[[Animal]]\n\n### Name\n\nAlpaca\n"
	broken := func() (domain.CardID, error) { return "", errors.New("no randomness") }

	_, _, err := format.Whole(body, nil, broken)
	if err == nil {
		t.Fatal("a deck was made whole with no mark to give")
	}
	if !strings.Contains(err.Error(), "1") || !strings.Contains(err.Error(), "no randomness") {
		t.Errorf("err = %v, want it to say which card carried none", err)
	}
}

// A deck's body is the preamble as it arrived, its sections and its cards in
// the order they stand, and the tail below the last value. A section no card
// stands under is written where it stands.
func TestDeckBodyWritesTheSectionsNoCardStandsUnder(t *testing.T) {
	body, err := format.DeckBody(format.Deck{
		Preamble: "Cards I am learning.",
		Sections: []format.Section{{Name: "Empty"}, {Name: "The ones with fur"}, {Name: "Last"}},
		Cards: []format.Card{{
			Heading: "Llama", Mark: "k7m2xq9fzp", Stencil: "Animal", Section: 1,
			Values: []format.Value{{Field: "Name", Text: "Llama"}},
		}},
	})
	if err != nil {
		t.Fatalf("body: %v", err)
	}
	want := "Cards I am learning.\n\n# Empty\n\n# The ones with fur\n\n## Llama ^k7m2xq9fzp\n\n" +
		"[[Animal]]\n\n### Name\n\nLlama\n\n# Last"
	if body != want {
		t.Errorf("body\n want %q\n  got %q", want, body)
	}
}

// A card standing under a section the deck does not hold is refused. Writing it
// under whichever section stands last moves somebody's card without saying so.
func TestDeckBodyRefusesACardUnderASectionTheDeckDoesNotHold(t *testing.T) {
	_, err := format.DeckBody(format.Deck{
		Sections: []format.Section{{Name: "One"}, {Name: "Two"}},
		Cards:    []format.Card{{Heading: "Llama", Stencil: "Animal", Section: 99}},
	})
	if !errors.Is(err, format.ErrNoSuchSection) {
		t.Errorf("body = %v, want ErrNoSuchSection", err)
	}
}

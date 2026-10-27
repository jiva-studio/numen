package format_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// deck is the awkward one every write is tested against: a key nobody owns, a
// card of a card model and a card of none, and fields written in an order of the
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
			if got := cardHeadings(f.Deck(domain.FileStat{Path: "Animals.md"})); !slices.Equal(got, one.cards) {
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

// The card ids the two cards of the deck above carry.
const (
	llama   = "k7m2xq9fzp"
	compost = "zpqrstvwxy"
)

func setValue(t *testing.T, raw string, card cardid.ID, field, value string) string {
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
// does not tidy them into the order the card model declares.
func TestSetValueKeepsTheOrderTheFieldsWereSavedIn(t *testing.T) {
	raw := setValue(t, deck, llama, "Life span", "about 25 years")

	f, err := format.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	card := f.Deck(domain.FileStat{}).Cards[0]
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

// removeValue is a field taken off a card, and the file as it now stands.
func removeValue(t *testing.T, raw string, card cardid.ID, field string) string {
	t.Helper()
	f, err := format.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RemoveValue(card, field); err != nil {
		t.Fatalf("remove %s of %s: %v", field, card, err)
	}
	return string(f.Bytes())
}

// Taking a field off a card takes its heading and its value and closes the gap,
// and every other byte of the file is the byte it was.
func TestRemoveValueConsumesTheFieldAndNothingElse(t *testing.T) {
	got := removeValue(t, deck, llama, "Life span")

	want := strings.Replace(deck, "### Life span\n\nabout 20 years\n\n", "", 1)
	if got != want {
		t.Errorf("the field was taken off wrong\n want %q\n  got %q", want, got)
	}
}

// The last field of the last card has nothing under it, so what is above it
// ends on the one break a file ends with.
func TestRemoveValueOfTheLastFieldPreservesNoBlankLine(t *testing.T) {
	got := removeValue(t, deck, compost, "Значение")

	want := strings.Replace(deck, "\n\n### Значение\n\nперегной\n", "\n", 1)
	if got != want {
		t.Errorf("the last field was taken off wrong\n want %q\n  got %q", want, got)
	}
}

// A field the card writes nothing under fails, and nothing is written.
// The card model may well declare it: what is not there is what the person wrote.
func TestRemoveValueOfAFieldTheCardDoesNotWrite(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RemoveValue(llama, "Weight"); !errors.Is(err, format.ErrNoSuchValue) {
		t.Errorf("err = %v, want ErrNoSuchValue", err)
	}
	if err := f.RemoveValue("wxyz01234t", "Height"); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("err = %v, want ErrNoSuchCard", err)
	}
	if string(f.Bytes()) != deck {
		t.Error("a field that is not there was taken off anyway")
	}
}

// A file written with carriage returns is written back with them.
func TestTheFilesOwnLineEndingIsWhatIsSaved(t *testing.T) {
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

// A card is addressed by its card id, and two cards of one heading are told apart
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

// A section is made at the end of the deck, given another name at its current position,
// and taken away. Taking one away takes its heading and nothing else: the cards
// that stood under it stay where they are.
func TestTheSplicesASectionNeeds(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddSection(
		format.Section{Name: "The ones without", Preamble: "Added last."},
	); err != nil {
		t.Fatalf("add: %v", err)
	}
	made := deck + "\n# The ones without\n\nAdded last.\n"
	if got := string(f.Bytes()); got != made {
		t.Errorf("section made wrong\n want %q\n  got %q", made, got)
	}

	if err := f.RenameSection(0, "The furred ones"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	expected := strings.Replace(made, "# The ones with fur\n", "# The furred ones\n", 1)
	if got := string(f.Bytes()); got != expected {
		t.Errorf("section renamed wrong\n want %q\n  got %q", expected, got)
	}

	if err := f.RemoveSection(0); err != nil {
		t.Fatalf("remove: %v", err)
	}
	removed := strings.Replace(expected, "# The furred ones\n\n", "", 1)
	if got := string(f.Bytes()); got != removed {
		t.Errorf("section removed wrong\n want %q\n  got %q", removed, got)
	}
	read := f.Deck(domain.FileStat{})
	if len(read.Cards) != 2 {
		t.Errorf("cards = %+v, want the cards left where they were", read.Cards)
	}
	if read.Cards[0].Section != format.NoSection {
		t.Errorf("card is under section %d", read.Cards[0].Section)
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
// the heading line carries the card's id.
func TestAddCard(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{
		Heading:       "Alpaca",
		CardID:        "m9n8b7v6c5",
		CardModelLink: "Animal",
		Preamble:      "From the same trip.",
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

	card, err := f.Deck(domain.FileStat{}).Card("m9n8b7v6c5")
	if err != nil {
		t.Fatalf("the card that was written cannot be read back: %v", err)
	}
	if card.Heading != "Alpaca" || card.CardModelLink != "Animal" ||
		card.Preamble != "From the same trip." {
		t.Errorf("card = %+v", card)
	}
	if got := fieldsOf(card); !slices.Equal(got, []string{"Name", "Height", "Life span"}) {
		t.Errorf("fields = %v", got)
	}
}

// A heading is a projection and a card id is given where the deck is written, so
// a card handed over with neither is written with neither.
func TestAddCardWithNoHeadingAndNoMark(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{CardModelLink: "Animal"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if want := deck + "\n##\n\n[[Animal]]\n"; string(f.Bytes()) != want {
		t.Errorf("card added wrong\n want %q\n  got %q", want, string(f.Bytes()))
	}
}

// A field renamed in a card model is renamed in the cards that card model cuts, and
// the value under the heading is left as it was.
func TestRenameFieldReachesTheCardsOfThatCardModelAlone(t *testing.T) {
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
		t.Errorf("renamed = %d, want the one card cut by that card model", renamed)
	}

	deck := f.Deck(domain.FileStat{})
	llama := deck.Cards[0]
	if got := fieldsOf(llama); !slices.Equal(got, []string{"Life span", "Shoulder height"}) {
		t.Errorf("fields = %v", got)
	}
	if got, _ := llama.Value("Shoulder height"); got != "about 45\"" {
		t.Errorf("the value moved with the heading: %q", got)
	}
	// The card of another card model keeps the field of that name.
	other := deck.Cards[1]
	if got, ok := other.Value("Height"); !ok || got != "no such thing" {
		t.Errorf("a card of another card model was renamed: %v", fieldsOf(other))
	}
}

const cardModel = "---\n" +
	"type: card-model\n" +
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
func TestWritingFieldsPreservesTheRestOfTheFrontmatterAlone(t *testing.T) {
	f, err := format.OpenCardModel([]byte(cardModel))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetFields([]string{"Name", "Height", "Life span", "Weight"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := f.RenameField("Height", "Shoulder height"); err != nil {
		t.Fatalf("rename: %v", err)
	}

	want := strings.Replace(cardModel,
		"fields:\n  - Name\n  - Height\n  - Life span\n",
		"fields:\n  - Name\n  - Shoulder height\n  - Life span\n  - Weight\n", 1)
	want = strings.ReplaceAll(want, "{{Height}}", "{{Shoulder height}}")
	if got := string(f.Bytes()); got != want {
		t.Errorf("fields written wrong\n want %q\n  got %q", want, got)
	}
	if got := f.Fields(); !slices.Equal(got, []string{"Name", "Shoulder height", "Life span", "Weight"}) {
		t.Errorf("fields = %v", got)
	}

	// A field's name is written where the card model declares it and in the braces
	// of every template that places it, so the card model that comes out declares what
	// its templates place.
	n := markdown.Parse(domain.FileStat{Path: "Animal.md"}, f.Bytes())
	read := format.ReadCardModel(n)
	if !slices.Equal(read.Fields, []string{"Name", "Shoulder height", "Life span", "Weight"}) {
		t.Errorf("read fields = %v", read.Fields)
	}
	for _, p := range read.Problems {
		if p.Fault == format.FaultPlaceholder {
			t.Errorf("a template places a name the card model does not declare: %+v", p)
		}
	}
}

// A card model declaring no such field is not written to.
func TestRenamingAFieldNobodyDeclares(t *testing.T) {
	f, err := format.OpenCardModel([]byte(cardModel))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RenameField("Girth", "Waist"); !errors.Is(err, format.ErrNoSuchField) {
		t.Fatalf("rename = %v, want it rejected", err)
	}
	if got := string(f.Bytes()); got != cardModel {
		t.Errorf("the file was written: %q", got)
	}
}

// A field's name is a name of its own, so a card model already declaring the name
// asked for is not written: the file would declare it twice, every template would
// place the same field, and the values under the old heading would be shown by
// nothing.
func TestRenamingAFieldOntoAFieldTheCardModelDeclares(t *testing.T) {
	f, err := format.OpenCardModel([]byte(cardModel))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RenameField("Height", "Life span"); !errors.Is(err, format.ErrFieldTaken) {
		t.Fatalf("rename = %v, want it rejected", err)
	}
	if got := string(f.Bytes()); got != cardModel {
		t.Errorf("the file was written: %q", got)
	}
}

// A template is written with the sides it has. A template missing one is a template that
// lays out nothing, and it is still that template once it has been written.
func TestATemplateIsSavedWithTheSidesItHas(t *testing.T) {
	f, err := format.OpenCardModel([]byte(cardModel))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddTemplate(format.CardTemplate{Name: "Half a template", Front: "Where does {{Name}} live?"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	want := cardModel + "\n## Half a template\n\n### Front\n\nWhere does {{Name}} live?\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("template written wrong\n want %q\n  got %q", want, got)
	}

	read := format.ReadCardModel(markdown.Parse(domain.FileStat{Path: "Animal.md"}, f.Bytes()))
	var missing int
	for _, p := range read.Problems {
		if p.Fault == format.FaultTemplateSide {
			missing++
		}
	}
	if missing != 1 {
		t.Errorf("problems = %+v, want the template that lays out nothing reported", read.Problems)
	}
}

// The wikilink under a card's heading is an ordinary link, so a card is cut by
// the note that link lands on, whatever the brackets spell.
func TestACardIsSplitByWhatItsLinkPointsAt(t *testing.T) {
	content := "---\ntype: deck\n---\n\n## Llama\n\n[[Animal|the beast]]\n\n### Height\n\nabout 45\"\n"
	f, err := format.OpenDeck([]byte(content))
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
	if got := f.Deck(domain.FileStat{}).Cards[0]; got.CardModelLink != "Animal|the beast" {
		t.Errorf("the link was rewritten: %q", got.CardModelLink)
	}
}

// A machine must not choose between two cards of one card id: which of the two a
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
	if _, err := f.Deck(domain.FileStat{}).Card("k7m2xq9fzp"); !errors.Is(err, format.ErrTwoCards) {
		t.Errorf("card = %v, want ErrTwoCards", err)
	}
}

// A heading is one line. A heading composed with a break in it carries the
// lines under it into the file, where the next read takes a second `##` for a
// second card and gives it a card id.
func TestAHeadingAndASectionsNameAreSplitAtTheFirstBreak(t *testing.T) {
	f, err := format.OpenDeck([]byte("---\ntype: deck\n---\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{
		Heading:       "Question\n\n## Injected ^k7m2xq9fzp\n\n[[Term]]\n\n### Q\n\nsmuggled",
		CardModelLink: "Animal",
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := f.AddSection(format.Section{Name: "Roots\n\n## Also injected"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	read := f.Deck(domain.FileStat{})
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
func TestAFieldsNameIsSplitAtTheFirstBreak(t *testing.T) {
	f, err := format.OpenDeck([]byte("---\ntype: deck\n---\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.AddCard(format.Card{
		Heading:       "Question",
		CardModelLink: "Animal",
		Values: []format.Value{{
			Field: "Answer\n\n## Injected ^k7m2xq9fzp\n\n[[Term]]",
			Text:  "smuggled",
		}},
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	read := f.Deck(domain.FileStat{})
	if len(read.Cards) != 1 {
		t.Fatalf("cards = %+v, want the one card that was written", read.Cards)
	}
	if got := read.Cards[0].Values; len(got) != 1 || got[0].Field != "Answer" {
		t.Errorf("values = %+v, want the one field, cut at the first break", got)
	}
}

// Nothing follows the last section, so taking its heading away leaves the break
// the file ends with and no blank line above it.
func TestRemovingTheLastSectionPreservesNoBlankLine(t *testing.T) {
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

// The card id is created where the deck is made whole, so that is what knows it
// and what says which card was given which.
func TestWholeReturnsWhichCardsItGave(t *testing.T) {
	body := "\n## Llama\n\n[[Animal]]\n\n### Name\n\nLlama\n" +
		"\n## Alpaca ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nAlpaca\n" +
		"\n## Vicuña\n\n[[Animal]]\n\n### Name\n\nVicuña\n"
	ids := []cardid.ID{"zpqrstvwxy", "m9n8b7v6c5"}
	createID := func() cardid.ID {
		out := ids[0]
		ids = ids[1:]
		return out
	}

	_, provided := format.Whole(body, map[string]format.CardModel{
		"Animal": {Fields: []string{"Name"}},
	}, createID)
	want := []format.CardID{{Card: 0, CardID: "m9n8b7v6c5"}, {Card: 2, CardID: "zpqrstvwxy"}}
	if !slices.Equal(provided, want) {
		t.Errorf("given = %+v, want %+v", provided, want)
	}
}

// A deck's body is the preamble as it arrived, its sections and its cards in
// the order they stand, and the tail below the last value. A section no card
// is placed under is written at its current position.
func TestDeckBodyWritesTheSectionsNoCardExistsBelow(t *testing.T) {
	body, err := format.DeckBody(format.Deck{
		Preamble: "Cards I am learning.",
		Sections: []format.Section{{Name: "Empty"}, {Name: "The ones with fur"}, {Name: "Last"}},
		Cards: []format.Card{{
			Heading: "Llama", CardID: "k7m2xq9fzp", CardModelLink: "Animal", Section: 1,
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

// A card standing under a section the deck does not hold fails. Writing it
// under whichever section stands last moves somebody's card without saying so.
func TestDeckBodyRefusesACardBelowASectionTheDeckDoesNotHold(t *testing.T) {
	_, err := format.DeckBody(format.Deck{
		Sections: []format.Section{{Name: "One"}, {Name: "Two"}},
		Cards:    []format.Card{{Heading: "Llama", CardModelLink: "Animal", Section: 99}},
	})
	if !errors.Is(err, format.ErrNoSuchSection) {
		t.Errorf("body = %v, want ErrNoSuchSection", err)
	}
}

// rough is a deck body as a person leaves one: CRLF endings, blank lines
// doubled where they felt like it, and whitespace at the ends of their lines.
// None of it is what the format would write, and all of it is theirs.
var rough = strings.ReplaceAll(
	"Cards I am learning.\n\n\n"+
		"# The ones with fur\n\n"+
		"## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n\n"+
		"## Alpaca ^3n8vr4tqch\n\n[[Animal]]\n\n\nsomebody's prose   \n   \n\n"+
		"### Name\n\n\nAlpaca  \n\n\n### Height\n\nabout 36\"   \n\n\n"+
		"## Vicuna ^9wq2ktr5bd\n\n[[Animal]]\n\n### Name\n\nVicuna \n\n\n### Height\n\nabout 34\"\n\n"+
		"## Guanaco ^5xz1hbn7ty\n\n[[Animal]]\n\n### Name\n\nGuanaco\n\n"+
		"# The ones without\n",
	"\n", "\r\n")

// The card ids of the two cards no change below names.
const (
	alpaca = "3n8vr4tqch"
	vicuna = "9wq2ktr5bd"
)

// A deck is a file somebody writes by hand beside the tools, so a change that
// names one card leaves every other card the bytes it was — the blank lines
// they doubled, the whitespace at the ends of their lines, and the endings the
// file carries.
func TestChangingOneCardPreservesEveryOtherTheBytesItWas(t *testing.T) {
	for name, change := range map[string]func(*format.DeckFile) error{
		"a value written": func(f *format.DeckFile) error {
			return f.SetValue(llama, "Height", "about 46\"")
		},
		"a field the card did not carry": func(f *format.DeckFile) error {
			return f.SetValue(llama, "Life span", "about 20 years")
		},
		"a field taken off": func(f *format.DeckFile) error {
			return f.RemoveValue(llama, "Height")
		},
		"a card model written": func(f *format.DeckFile) error {
			return f.SetCardModel(llama, "Beast")
		},
		"a card removed": func(f *format.DeckFile) error {
			return f.RemoveCard(llama)
		},
		"a card added": func(f *format.DeckFile) error {
			return f.AddCard(format.Card{Heading: "Camel", CardModelLink: "Animal"})
		},
		"a card added under a section": func(f *format.DeckFile) error {
			return f.AddCardUnder(0, format.Card{Heading: "Camel", CardModelLink: "Animal"})
		},
		"a section added": func(f *format.DeckFile) error {
			return f.AddSection(format.Section{Name: "The ones without"})
		},
		"a section renamed": func(f *format.DeckFile) error {
			return f.RenameSection(0, "Furry")
		},
		"a section removed": func(f *format.DeckFile) error {
			return f.RemoveSection(0)
		},
	} {
		t.Run(name, func(t *testing.T) {
			file := format.OpenDeckBody(rough)
			if err := change(file); err != nil {
				t.Fatalf("change: %v", err)
			}
			content := file.Body()
			for _, cardID := range []cardid.ID{alpaca, vicuna} {
				was := cardRun(t, rough, cardID)
				if !strings.Contains(content, was) {
					t.Errorf("the card of %s was rewritten\n was %q\n now %q",
						cardID, was, cardRun(t, content, cardID))
				}
			}
		})
	}
}

// cardRun is the bytes one card of a body occupies, from its heading to the
// heading of the card below it. Neither card it is asked about stands last, so
// what follows one is always another card's heading.
func cardRun(t *testing.T, body string, card cardid.ID) string {
	t.Helper()
	from := strings.Index(body, "## "+getCardHeading(t, body, card))
	if from < 0 {
		t.Fatalf("the body holds no card of the card id %s:\n%s", card, body)
	}
	rest := body[from:]
	if pos := strings.Index(rest[1:], "\n## "); pos >= 0 {
		return rest[:pos+2]
	}
	return rest
}

// getCardHeading is the heading line of the card of a card id, without its hashes.
func getCardHeading(t *testing.T, body string, card cardid.ID) string {
	t.Helper()
	for _, line := range strings.Split(markdown.Normalise(body), "\n") {
		if strings.HasPrefix(line, "## ") && strings.HasSuffix(line, "^"+string(card)) {
			return strings.TrimPrefix(line, "## ")
		}
	}
	t.Fatalf("the body holds no card of the card id %s:\n%s", card, body)
	return ""
}

// A card of a deck that never held one is given the wikilink under its heading,
// and the values it already holds stay where they stand.
func TestSetCardModelOnACardThatNamedNone(t *testing.T) {
	file := format.OpenDeckBody("## Llama ^k7m2xq9fzp\n\nsomebody's prose\n\n### Name\n\nLlama\n")
	if err := file.SetCardModel(llama, "Animal"); err != nil {
		t.Fatalf("set the card model: %v", err)
	}
	want := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\nsomebody's prose\n\n### Name\n\nLlama\n"
	if got := file.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// Removing the last card of a deck leaves what is above it ending on the one
// break a file ends with.
func TestRemovingTheLastCardPreservesNoBlankLine(t *testing.T) {
	file := format.OpenDeckBody(
		"## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n" +
			"## Alpaca ^3n8vr4tqch\n\n[[Animal]]\n\n### Name\n\nAlpaca\n")
	if err := file.RemoveCard(alpaca); err != nil {
		t.Fatalf("remove: %v", err)
	}
	want := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n"
	if got := file.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// A card asked into a section the deck does not hold fails, the way a card
// written under one is.
func TestAddCardBelowASectionTheDeckDoesNotHold(t *testing.T) {
	file := format.OpenDeckBody(rough)
	if err := file.AddCardUnder(9, format.Card{Heading: "Camel"}); !errors.Is(err, format.ErrNoSuchSection) {
		t.Errorf("add = %v, want ErrNoSuchSection", err)
	}
}

// A deck whose frontmatter is never closed holds no prose to write to: every
// change fails with the markdown package's ErrUnterminated, and nothing is
// written.
func TestAChangeToADeckWithUnclosedFrontmatterIsRefused(t *testing.T) {
	raw := "---\ntype: deck\n\n# Fur\n\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n" +
		"## Bare\n\n[[Animal]]\n\n### Name\n\nBare\n"
	cardModels := map[string]format.CardModel{"Animal": {Fields: []string{"Name"}}}
	createID := func() cardid.ID { return "m9n8b7v6c5" }

	for name, change := range map[string]func(*format.DeckFile) error{
		"whole": func(f *format.DeckFile) error {
			provided, err := f.Whole(cardModels, createID)
			if provided != nil {
				t.Errorf("card ids given = %+v, want none", provided)
			}
			return err
		},
		"set a value": func(f *format.DeckFile) error { return f.SetValue(llama, "Name", "Lama") },
		"set a card model": func(f *format.DeckFile) error {
			return f.SetCardModel(llama, "Beast")
		},
		"remove a value": func(f *format.DeckFile) error { return f.RemoveValue(llama, "Name") },
		"remove a card":  func(f *format.DeckFile) error { return f.RemoveCard(llama) },
		"rename a section": func(f *format.DeckFile) error {
			return f.RenameSection(0, "Furred")
		},
		"remove a section": func(f *format.DeckFile) error { return f.RemoveSection(0) },
		"add a section":    func(f *format.DeckFile) error { return f.AddSection(format.Section{Name: "New"}) },
		"add a card":       func(f *format.DeckFile) error { return f.AddCard(format.Card{Heading: "Camel"}) },
		"add a card under": func(f *format.DeckFile) error {
			return f.AddCardUnder(0, format.Card{Heading: "Camel"})
		},
		"rename a field": func(f *format.DeckFile) error {
			n, err := f.RenameField(map[string]string{"Animal": "Animal"}, "Animal", "Name", "Title")
			if n != 0 {
				t.Errorf("renamed %d, want 0", n)
			}
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			f, err := format.OpenDeck([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := change(f); !errors.Is(err, markdown.ErrUnterminated) {
				t.Errorf("err = %v, want ErrUnterminated", err)
			}
			if string(f.Bytes()) != raw {
				t.Error("a rejected change was written anyway")
			}
		})
	}
}

// Making a deck whole through its file writes the card ids into the file and says
// which cards were given one.
func TestDeckFileWholeWritesTheMarksItProduces(t *testing.T) {
	f := format.OpenDeckBody("## Llama\n\n[[Animal]]\n\n### Name\n\nLlama\n")
	provided, err := f.Whole(map[string]format.CardModel{"Animal": {Fields: []string{"Name"}}},
		func() cardid.ID { return "m9n8b7v6c5" })
	if err != nil {
		t.Fatalf("whole: %v", err)
	}
	if want := []format.CardID{{Card: 0, CardID: "m9n8b7v6c5"}}; !slices.Equal(provided, want) {
		t.Errorf("given = %+v, want %+v", provided, want)
	}
	if want := "## Llama ^m9n8b7v6c5\n\n[[Animal]]\n\n### Name\n\nLlama\n"; f.Body() != want {
		t.Errorf("body\n want %q\n  got %q", want, f.Body())
	}
}

// A card that cannot be found fails before anything is written, whichever
// change was asked for.
func TestSetCardModelAndRemoveCardOfACardTheDeckDoesNotHold(t *testing.T) {
	body := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n"
	f := format.OpenDeckBody(body)
	if err := f.SetCardModel("wxyz01234t", "Animal"); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("set card model = %v, want ErrNoSuchCard", err)
	}
	if err := f.RemoveCard("wxyz01234t"); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("remove card = %v, want ErrNoSuchCard", err)
	}
	if f.Body() != body {
		t.Error("a card that is not there was written anyway")
	}
}

// The card ids of the cards the tests below write by hand.
const (
	cardIDA = "aaaaaaaaaa"
	cardIDB = "bbbbbbbbbb"
)

// Taking the only card of a deck away leaves an empty deck, and taking the last
// card away from under one character of prose leaves that character on the one
// break a file ends with.
func TestRemovingTheLastCardAtTheTopOfTheBody(t *testing.T) {
	for name, one := range map[string]struct{ raw, want string }{
		"the only card":    {"## Llama ^aaaaaaaaaa\n\n[[Animal]]\n", ""},
		"one character up": {"x\n\n## Llama ^aaaaaaaaaa\n", "x\n"},
	} {
		t.Run(name, func(t *testing.T) {
			f := format.OpenDeckBody(one.raw)
			if err := f.RemoveCard(cardIDA); err != nil {
				t.Fatalf("remove: %v", err)
			}
			if got := f.Body(); got != one.want {
				t.Errorf("body\n want %q\n  got %q", one.want, got)
			}
		})
	}
}

// The same holds for a section standing last.
func TestRemovingTheLastSectionAtTheTopOfTheBody(t *testing.T) {
	for name, one := range map[string]struct{ raw, want string }{
		"the only section": {"# Fur\n", ""},
		"one character up": {"x\n\n# Fur\n", "x\n"},
	} {
		t.Run(name, func(t *testing.T) {
			f := format.OpenDeckBody(one.raw)
			if err := f.RemoveSection(0); err != nil {
				t.Fatalf("remove: %v", err)
			}
			if got := f.Body(); got != one.want {
				t.Errorf("body\n want %q\n  got %q", one.want, got)
			}
		})
	}
}

// A section removed from a file written with carriage returns takes the blank
// line under its heading with it.
func TestRemoveSectionOfACrlfDeck(t *testing.T) {
	raw := strings.ReplaceAll(deck, "\n", "\r\n")
	f, err := format.OpenDeck([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.RemoveSection(0); err != nil {
		t.Fatalf("remove: %v", err)
	}
	want := strings.Replace(raw, "# The ones with fur\r\n\r\n", "", 1)
	if got := string(f.Bytes()); got != want {
		t.Errorf("section removed wrong\n want %q\n  got %q", want, got)
	}
}

// A card that already names a card model has the name replaced at its current position.
func TestSetCardModelOnACardThatNamedOne(t *testing.T) {
	f, err := format.OpenDeck([]byte(deck))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := f.SetCardModel(llama, "Beast"); err != nil {
		t.Fatalf("set the card model: %v", err)
	}
	want := strings.Replace(deck, "[[Animal]]", "[[Beast]]", 1)
	if got := string(f.Bytes()); got != want {
		t.Errorf("card model set wrong\n want %q\n  got %q", want, got)
	}
}

// A card with no card id is not the card of the empty card id.
func TestAnEmptyMarkAddressesNoCard(t *testing.T) {
	raw := "## Bare\n\n### Name\n\nx\n\n## Llama ^aaaaaaaaaa\n\n### Name\n\ny\n"
	f := format.OpenDeckBody(raw)
	if err := f.SetValue("", "Name", "z"); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("err = %v, want ErrNoSuchCard", err)
	}
	if f.Body() != raw {
		t.Error("a card without a card id was written to")
	}
}

// A place before the first section or after the last stands nowhere.
func TestASectionOutsideTheDeckIsRefused(t *testing.T) {
	for _, item := range []int{-1, 1} {
		f := format.OpenDeckBody(deck)
		if err := f.RenameSection(item, "Nowhere"); !errors.Is(err, format.ErrNoSuchSection) {
			t.Errorf("rename %d = %v, want ErrNoSuchSection", item, err)
		}
		if err := f.RemoveSection(item); !errors.Is(err, format.ErrNoSuchSection) {
			t.Errorf("remove %d = %v, want ErrNoSuchSection", item, err)
		}
		if err := f.AddCardUnder(item, format.Card{Heading: "Camel"}); !errors.Is(err, format.ErrNoSuchSection) {
			t.Errorf("add under %d = %v, want ErrNoSuchSection", item, err)
		}
		if f.Body() != deck {
			t.Errorf("a section at %d that is not there was written to", item)
		}
	}
}

// A section whose heading ends the file is renamed without a break the file did
// not end with.
func TestRenamingTheLastSectionWhoseHeadingEndsTheFile(t *testing.T) {
	f := format.OpenDeckBody("# One\n\n## Llama ^aaaaaaaaaa\n\n# Last")
	if err := f.RenameSection(1, "Final"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if want := "# One\n\n## Llama ^aaaaaaaaaa\n\n# Final"; f.Body() != want {
		t.Errorf("body\n want %q\n  got %q", want, f.Body())
	}
}

// A card asked into a section is written at the end of it: in front of the
// section after it, or at the end of the deck where none follows.
func TestAddCardBelowASectionWritesAtItsEnd(t *testing.T) {
	raw := "# One\n\n## Llama ^aaaaaaaaaa\n\n# Two\n"
	card := format.Card{Heading: "Camel", CardID: cardIDB}

	first := format.OpenDeckBody(raw)
	if err := first.AddCardUnder(0, card); err != nil {
		t.Fatalf("add: %v", err)
	}
	if want := "# One\n\n## Llama ^aaaaaaaaaa\n\n## Camel ^bbbbbbbbbb\n\n# Two\n"; first.Body() != want {
		t.Errorf("body\n want %q\n  got %q", want, first.Body())
	}

	last := format.OpenDeckBody(raw)
	if err := last.AddCardUnder(1, card); err != nil {
		t.Fatalf("add: %v", err)
	}
	if want := raw + "\n## Camel ^bbbbbbbbbb\n"; last.Body() != want {
		t.Errorf("body\n want %q\n  got %q", want, last.Body())
	}
}

// A card standing under a section before the first or past the last fails.
func TestDeckBodyRefusesACardBelowAPlaceOutsideTheSections(t *testing.T) {
	for _, item := range []int{-2, 2} {
		_, err := format.DeckBody(format.Deck{
			Sections: []format.Section{{Name: "One"}, {Name: "Two"}},
			Cards:    []format.Card{{Heading: "Llama", Section: item}},
		})
		if !errors.Is(err, format.ErrNoSuchSection) {
			t.Errorf("section %d: body = %v, want ErrNoSuchSection", item, err)
		}
	}
}

// What follows the last value is the deck's tail, and a deck of prose alone has
// none to take.
func TestDeckBodyEndsOnTheTailOfTheDeck(t *testing.T) {
	for name, one := range map[string]struct {
		deck format.Deck
		want string
	}{
		"prose alone": {format.Deck{Preamble: "Only prose"}, "Only prose\n"},
		"empty":       {format.Deck{}, ""},
		"cards alone": {
			format.Deck{
				Cards: []format.Card{{Heading: "Llama", CardID: cardIDA, Section: format.NoSection}},
				Tail:  "\n\nafter",
			},
			"## Llama ^aaaaaaaaaa\n\nafter",
		},
		"sections alone": {
			format.Deck{Sections: []format.Section{{Name: "One"}}, Tail: "\n\nafter"},
			"# One\n\nafter",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := format.DeckBody(one.deck)
			if err != nil {
				t.Fatalf("body: %v", err)
			}
			if got != one.want {
				t.Errorf("body\n want %q\n  got %q", one.want, got)
			}
		})
	}
}

// Only the cards cut by the card model are renamed, whichever order they stand in,
// and a card whose link lands on another card model keeps its field.
func TestRenameFieldSkipsTheCardsOfOtherCardModels(t *testing.T) {
	raw := "## Compost ^zpqrstvwxy\n\n[[Term]]\n\n### Name\n\nx\n\n" +
		"## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n"
	f := format.OpenDeckBody(raw)
	n, err := f.RenameField(
		map[string]string{"Term": "Term.md", "Animal": "Animal.md"}, "Animal.md", "Name", "Title")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if n != 1 {
		t.Errorf("renamed = %d, want 1", n)
	}
	want := strings.Replace(raw, "### Name\n\nLlama", "### Title\n\nLlama", 1)
	if f.Body() != want {
		t.Errorf("body\n want %q\n  got %q", want, f.Body())
	}
}

// A field heading that ends the file is renamed without a break the file did
// not end with.
func TestRenameFieldOfAHeadingThatEndsTheFile(t *testing.T) {
	f := format.OpenDeckBody("## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name")
	n, err := f.RenameField(map[string]string{"Animal": "Animal.md"}, "Animal.md", "Name", "Title")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if want := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Title"; n != 1 || f.Body() != want {
		t.Errorf("renamed %d\n want %q\n  got %q", n, want, f.Body())
	}
}

// The file keeps the newline it ends with when the section standing last is
// given another name.
func TestRenamingTheLastSectionKeepsTheFinalNewline(t *testing.T) {
	const front = "---\n" +
		"id: 01J8F3K2M9QRSTVWXYZ012\n" +
		"type: deck\n" +
		"---\n" +
		"\n"
	tests := []struct {
		name string
		body string
	}{
		{
			name: "cards under the heading",
			body: "# The ones with fur\n" +
				"\n" +
				"## Llama ^k7m2xq9fzp\n" +
				"\n" +
				"Q: Hair?\n" +
				"A: Wool.\n",
		},
		{
			name: "heading on the last line",
			body: "# The ones with fur\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := front + tt.body
			f, err := format.OpenDeck([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := f.RenameSection(0, "The furred ones"); err != nil {
				t.Fatalf("rename: %v", err)
			}
			want := strings.Replace(raw, "# The ones with fur\n", "# The furred ones\n", 1)
			if got := string(f.Bytes()); got != want {
				t.Errorf("rename changed more than the heading\n want %q\n  got %q", want, got)
			}
		})
	}
}

const twoFieldCard = "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\n1.8 m\n"

// A value set to nothing leaves its heading with the blank line a heading is
// followed by, and the fields after it keep their places.
func TestSetValueToNothingKeepsTheFieldsAfterIt(t *testing.T) {
	f := format.OpenDeckBody(twoFieldCard)
	if err := f.SetValue("k7m2xq9fzp", "Name", ""); err != nil {
		t.Fatalf("set: %v", err)
	}
	want := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\n### Height\n\n1.8 m\n"
	if got := f.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// The last field of the file set to nothing ends the file on its heading.
func TestSetTheLastValueToNothingEndsTheFileOnItsHeading(t *testing.T) {
	f := format.OpenDeckBody(twoFieldCard)
	if err := f.SetValue("k7m2xq9fzp", "Height", ""); err != nil {
		t.Fatalf("set: %v", err)
	}
	want := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n"
	if got := f.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// A card written after text that does not end its line stands two breaks below
// it.
func TestAddCardAfterTextWithoutALineBreak(t *testing.T) {
	f := format.OpenDeckBody("Notes on animals")
	if err := f.AddCard(format.Card{Heading: "Alpaca"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	want := "Notes on animals\n\n## Alpaca\n"
	if got := f.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// What a section's lead holds belongs to the section, so a deck that ends on
// that lead has only the break that ends the file.
func TestADeckEndingOnASectionLeadHasOnlyTheFinalBreakAsTail(t *testing.T) {
	deck := format.ReadDeck(note(t, "# Pets\n\nSome lead.\n"))
	if len(deck.Sections) != 1 || deck.Sections[0].Preamble != "Some lead." {
		t.Fatalf("sections = %+v", deck.Sections)
	}
	if deck.Tail != "\n" {
		t.Errorf("tail = %q, want the final break", deck.Tail)
	}
}

// A line that closes no pair of brackets after a link that does is not a link,
// and the link before it still holds the cut off.
func TestProjectKeepsALinkWhenALaterPairIsNeverClosed(t *testing.T) {
	head := strings.Repeat("a", format.HeadingRunes-3)
	got := format.Project(head + " [[xy]] [[never closed and more")
	if got != head {
		t.Errorf("project = %q, want %q", got, head)
	}
}

// A file that carries an identifier is left as it is, and one that carries none
// is given it.
func TestWriteIdentifier(t *testing.T) {
	t.Run("a file carrying none", func(t *testing.T) {
		f, err := format.OpenDeck([]byte("---\ntitle: Animals\n---\n" + twoFieldCard))
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		wrote, err := f.WriteIdentifier("abc123")
		if err != nil || !wrote {
			t.Fatalf("write = %v, %v, want true, nil", wrote, err)
		}
		doc, err := markdown.Open(f.Bytes())
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}
		if id, ok := doc.Identifier(); !ok || id != "abc123" {
			t.Errorf("identifier = %q, %v", id, ok)
		}
	})

	t.Run("a file carrying one", func(t *testing.T) {
		raw := "---\nid: first\n---\n" + twoFieldCard
		f, err := format.OpenDeck([]byte(raw))
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		wrote, err := f.WriteIdentifier("second")
		if err != nil || wrote {
			t.Fatalf("write = %v, %v, want false, nil", wrote, err)
		}
		if got := string(f.Bytes()); got != raw {
			t.Errorf("file changed: %q", got)
		}
	})

	t.Run("a frontmatter that cannot be written", func(t *testing.T) {
		f, err := format.OpenDeck([]byte("---\n{title: Animals}\n---\n" + twoFieldCard))
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if _, err := f.WriteIdentifier("abc123"); !errors.Is(err, markdown.ErrInline) {
			t.Errorf("err = %v, want ErrInline", err)
		}
	})
}

// proseCardModel is the awkward one the round trip is tested against: a key
// nobody owns, prose above the first template and prose under a template's heading.
const proseCardModel = "---\n" +
	"id: 01J8F3K2M9QRSTVWXYZ012\n" +
	"type: card-model\n" +
	"mine: keep me verbatim\n" +
	"fields:\n" +
	"  - Name\n" +
	"  - Height\n" +
	"---\n" +
	"How I show these.\n" +
	"\n" +
	"## Recognise\n" +
	"\n" +
	"The one to start with.\n" +
	"\n" +
	"### Front\n" +
	"\n" +
	"{{Name}}\n" +
	"\n" +
	"### Back\n" +
	"\n" +
	"{{Height}}\n"

// A card model read and written straight back comes out as the body it went in
// as. Anything less is a diff the person did not ask for, on every save,
// forever.
func TestReadingACardModelAndWritingItBackChangesNothing(t *testing.T) {
	for name, raw := range map[string]string{
		"the card model": proseCardModel,
		"crlf":           strings.ReplaceAll(proseCardModel, "\n", "\r\n"),
		"no frontmatter": "Above them all.\n\n## Recognise\n\nThe one to start with.\n\n" +
			"### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
		"no trailing break": "---\ntype: card-model\n---\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}",
		"no trailing break under a template's heading": "---\ntype: card-model\n---\n" +
			"## Recognise",
		"no templates": "---\ntype: card-model\n---\n",
		"bom":          "\xef\xbb\xbf---\ntype: card-model\n---\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
		"comments and order": "---\n# a note to myself\nzebra: 1\n\ntype: card-model\n---\n" +
			"## Recognise\n\n### Front\n\none\n\n\ntwo\n\n### Back\n\n{{Height}}\n",
		"a heading of another name": "---\ntype: card-model\n---\n" +
			"## Recognise\n\n### Front\n\n{{Name}}\n\n### Notes\n\nThe one to start with.\n\n" +
			"### Back\n\n{{Height}}\n\n### Afterwards\n\nAnd this.\n",
		"a side written twice": "---\ntype: card-model\n---\n" +
			"## Recognise\n\n### Front\n\n{{Name}}\n\n### Front\n\nthe second one\n\n### Back\n\n{{Height}}\n",
	} {
		t.Run(name, func(t *testing.T) {
			doc, err := markdown.Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			read := format.ReadCardModel(markdown.Parse(domain.FileStat{Path: "Animal.md"}, []byte(raw)))
			body, err := format.CardModelBody(read.Preamble, read.Templates, read.Tail)
			if err != nil {
				t.Fatalf("card model body: %v", err)
			}
			if want := markdown.Normalise(doc.Body()); body != want {
				t.Errorf("the round trip changed the card model\n want %q\n  got %q", want, body)
			}
		})
	}
}

// vault is where the fixtures a person can open in the application stand.
const vault = "testdata/cards"

// Every deck and card model of the fixture vault, read and written straight back,
// comes out as the body it went in as. These are the files a person is shown,
// awkward on purpose, and a save that touched nothing is a save that changed
// nothing.
func TestReadingTheFixtureVaultAndWritingItBackChangesNothing(t *testing.T) {
	entries, err := os.ReadDir(vault)
	if err != nil {
		t.Fatalf("read the vault: %v", err)
	}
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(vault, entry.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			doc, err := markdown.Open(raw)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			n := markdown.Parse(domain.FileStat{Path: path}, raw)

			var body string
			switch n.Type {
			case format.TypeDeck:
				body, err = format.DeckBody(format.ReadDeck(n))
			case format.TypeCardModel:
				read := format.ReadCardModel(n)
				body, err = format.CardModelBody(read.Preamble, read.Templates, read.Tail)
			default:
				t.Fatalf("type = %q, want a deck or a card model", n.Type)
			}
			if err != nil {
				t.Fatalf("body: %v", err)
			}
			if want := markdown.Normalise(doc.Body()); body != want {
				t.Errorf("the round trip changed the file\n want %q\n  got %q", want, body)
			}
		})
	}
}

// A write that reaches one template reaches nothing else: the prose above the first
// template and the prose under every other template's heading come out as the bytes
// they went in as.
func TestWritingOneTemplatePreservesTheRestOfTheCardModelAlone(t *testing.T) {
	raw := "---\ntype: card-model\n---\n" +
		"How I show these.\n\n" +
		"## Recognise\n\nThe one to start with.\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n\n" +
		"## Name it\n\nAnd this one the other way round.\n\n### Front\n\n{{Height}}\n\n### Back\n\n{{Name}}\n"

	read := format.ReadCardModel(markdown.Parse(domain.FileStat{Path: "Animal.md"}, []byte(raw)))
	target := append([]format.CardTemplate(nil), read.Templates...)
	target[0].Back = "**{{Height}}**"

	body, err := format.CardModelBody(read.Preamble, target, read.Tail)
	if err != nil {
		t.Fatalf("card model body: %v", err)
	}

	doc, err := markdown.Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	want := strings.Replace(markdown.Normalise(doc.Body()), "{{Height}}\n\n## Name it",
		"**{{Height}}**\n\n## Name it", 1)
	if body != want {
		t.Errorf("the write reached further than the template\n want %q\n  got %q", want, body)
	}
}

// A card model shows a card once through each template it carries, and a template is not
// addressed by its name here. Two templates of one name are two templates, and what
// each holds is in the file.
func TestACardModelKeepsEveryTemplateItIsGiven(t *testing.T) {
	body, err := format.CardModelBody("", []format.CardTemplate{
		{Name: "Recognise", Front: "{{Name}}", Back: "{{Height}}"},
		{Name: "Recognise", Front: "{{Height}}", Back: "{{Name}}"},
	}, "")
	if err != nil {
		t.Fatalf("card model body: %v", err)
	}

	read := format.ReadCardModel(domain.Note{Body: body})
	if len(read.Templates) != 2 {
		t.Fatalf("templates = %+v, want both of them", read.Templates)
	}
	if read.Templates[0].Front != "{{Name}}" || read.Templates[0].Back != "{{Height}}" {
		t.Errorf("the first template = %+v", read.Templates[0])
	}
	if read.Templates[1].Front != "{{Height}}" || read.Templates[1].Back != "{{Name}}" {
		t.Errorf("the second template = %+v", read.Templates[1])
	}
}

func wholeAnimals() map[string]format.CardModel {
	return map[string]format.CardModel{"Animal": {Fields: []string{"Name"}}}
}

func createWholeID() cardid.ID { return "zpqrstvwxy" }

// A heading that has fallen out of step with the card's first field is written
// again from the first line of that field, with the card id the card carries.
func TestWholeWritesTheHeadingFromTheFirstField(t *testing.T) {
	body := "\n## Old ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\nthe woolly one\n"

	got, provided := format.Whole(body, wholeAnimals(), createWholeID)

	want := "\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\nthe woolly one\n"
	if got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
	if len(provided) != 0 {
		t.Errorf("given = %+v, want no card id given", provided)
	}
}

// A card of a card model nobody holds keeps the heading it has, and is given its
// card id all the same.
func TestWholeKeepsTheHeadingOfACardWhoseCardModelIsUnknown(t *testing.T) {
	body := "\n## Old\n\n[[Plant]]\n\n### Name\n\nFern\n"

	got, _ := format.Whole(body, wholeAnimals(), createWholeID)

	want := "\n## Old ^zpqrstvwxy\n\n[[Plant]]\n\n### Name\n\nFern\n"
	if got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// A heading already in step is not written again: the spacing around it stays
// as the person typed it.
func TestWholePreservesAHeadingInStepAsItExists(t *testing.T) {
	body := "\n##   Llama ^k7m2xq9fzp  \n\n[[Animal]]\n\n### Name\n\nLlama\n"

	got, provided := format.Whole(body, wholeAnimals(), createWholeID)

	if got != body {
		t.Errorf("body\n want %q\n  got %q", body, got)
	}
	if len(provided) != 0 {
		t.Errorf("given = %+v", provided)
	}
}

// A heading that ends the body is written without a line break after it.
func TestWholeWritesTheLastHeadingWithoutAnEnding(t *testing.T) {
	got, provided := format.Whole("## Llama", nil, createWholeID)

	if want := "## Llama ^zpqrstvwxy"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if len(provided) != 1 {
		t.Errorf("given = %+v", provided)
	}
}

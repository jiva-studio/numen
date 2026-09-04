package cards_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// The two vaults every test here uses. They share no word: a heading renamed in
// one is recognisable in the other, and so is a stencil listed from the wrong
// place.
var (
	animals = map[string]string{
		"Animal.md": "---\n" +
			"type: stencil\n" +
			"mine: keep me verbatim\n" +
			"fields:\n" +
			"  - Name\n" +
			"  - Height\n" +
			"  - Life span\n" +
			"---\n" +
			"\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
		"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Height\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Height}}\n",
		"decks/Mammals.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: deck\nmine: keep me verbatim\n---\n" +
			"\nCards I am learning.\n" +
			"\n# The ones with fur\n" +
			"\nThe section says something of its own.\n" +
			"\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n" +
			"\n### Height\n\nabout 45\"\n" +
			"\n### Life span\n\nabout 20 years\n" +
			"\n## Gloss ^zpqrstvwxy\n\n[[Term]]\n\n### Height\n\nnot a length at all\n" +
			"\n### Word\n\nGloss\n",
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Wren ^3f4g5h6j7k\n\n[[Animal]]\n\n### Name\n\nWren\n\n### Height\n\nabout 4\"\n",
		"Weather.md": "# Weather\n\nNo card in here.\n",
	}
	minerals = map[string]string{
		"Mineral.md": "---\ntype: stencil\nfields:\n  - Sample\n  - Height\n---\n" +
			"\n## Spot it\n\n### Front\n\n{{Sample}}\n\n### Back\n\n{{Height}}\n",
		// A card of this vault's own stencil, and a card naming the other
		// vault's by the name it is filed under there.
		"decks/Quartz.md": "---\ntype: deck\n---\n" +
			"\n## Quartz ^m9n8b7v6c5\n\n[[Mineral]]\n\n### Sample\n\nQuartz\n" +
			"\n### Height\n\na crystal habit\n" +
			"\n## Llama ^q1w2e3r4t5\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n" +
			"\n### Name\n\nLlama\n",
	}
)

// vaulted is two vaults on disk, indexed, and the ports every scenario is built
// out of.
type vaulted struct {
	db     *index.DB
	first  domain.Vault
	second domain.Vault
}

func indexed(t *testing.T) vaulted {
	t.Helper()
	ctx := t.Context()

	db, err := index.Open(ctx, indexfile.Path(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := vaults.Scan{
		Readers: filesystem.VaultReaders{}, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.NoteQueries(), Maintenance: db.Maintenance(),
	}
	out := vaulted{db: db}
	for i, notes := range []map[string]string{animals, minerals} {
		v := testsupport.NewVault(t, notes)
		if _, err := scan.Execute(ctx, v); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			out.first = v
		} else {
			out.second = v
		}
	}
	return out
}

func (vs vaulted) index(t *testing.T) func(context.Context, domain.Vault, []string) error {
	t.Helper()
	scan := vaults.Scan{
		Readers: filesystem.VaultReaders{}, Vaults: vs.db.Vaults(), Notes: vs.db.Notes(),
		Known: vs.db.NoteQueries(), Maintenance: vs.db.Maintenance(),
	}
	return func(ctx context.Context, v domain.Vault, _ []string) error {
		_, err := scan.Execute(ctx, v)
		return err
	}
}

// unlevelled brings nothing level: these tests read the file back and not the
// index. The ones that ask the index level it with a whole scan.
func unlevelled(context.Context, domain.Vault, []string) error { return nil }

func read(t *testing.T, v domain.Vault, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// write puts a note in the vault, with the folders above it.
func write(t *testing.T, v domain.Vault, path, raw string) {
	t.Helper()
	at := filepath.Join(v.Path, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A deck opened and put back with the prose it came out of is the file it was.
// Anything less is a diff nobody asked for, on every save, forever.
func TestADeckReadAndWrittenBackIsTheFileItWas(t *testing.T) {
	vs := indexed(t)
	before := read(t, vs.first, "decks/Mammals.md")

	got, err := cards.Read{Readers: filesystem.VaultReaders{}}.Deck(t.Context(), vs.first, "decks/Mammals.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.Ok || got.Type != domain.TypeDeck {
		t.Fatalf("outcome = %q, type = %q", got.Outcome, got.Type)
	}
	if len(got.Body.Cards) != 2 || got.Body.Preamble != "\nCards I am learning.\n\n" {
		t.Fatalf("deck = %+v", got.Body)
	}

	w := cards.NewWrite(
		filesystem.VaultReaders{}, filesystem.VaultWriters{}, vs.db.NoteQueries(), unlevelled)
	if _, err := w.Deck(t.Context(), vs.first, "decks/Mammals.md", prose(t, before), got.Fingerprint); err != nil {
		t.Fatalf("write: %v", err)
	}
	if after := read(t, vs.first, "decks/Mammals.md"); after != before {
		t.Errorf("the file changed\n was %q\n now %q", before, after)
	}
}

// Every field stands under a heading of its own, the first included, so a card
// writing the first field twice is the one fault a card writing any field twice
// is, and it is reported once.
func TestACardWritingItsFirstFieldTwiceIsOneFieldWrittenTwice(t *testing.T) {
	vs := indexed(t)
	written := read(t, vs.first, "decks/Mammals.md")
	replaced := strings.Replace(written,
		"### Name\n\nLlama\n",
		"### Name\n\nLlama\n\n### Name\n\nLlama, a second time\n", 1)
	if replaced == written {
		t.Fatal("the fixture is not what this test writes into")
	}
	if err := os.WriteFile(
		filepath.Join(vs.first.Path, "decks", "Mammals.md"), []byte(replaced), 0o600,
	); err != nil {
		t.Fatal(err)
	}

	u := cards.Read{Readers: filesystem.VaultReaders{}, Links: vs.db.NoteQueries()}
	got, err := u.Deck(t.Context(), vs.first, "decks/Mammals.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var filed []format.Problem
	for _, p := range got.Body.Problems {
		if p.Fault == format.FaultTwoValues {
			filed = append(filed, p)
		}
	}
	if len(filed) != 1 {
		t.Fatalf("problems = %+v, want the one card that writes it twice", got.Body.Problems)
	}
	if filed[0].Card != 0 || filed[0].Field != "Name" {
		t.Errorf("problem = %+v, want it against the first card and Name", filed[0])
	}

	// Both are kept, and the first stands.
	if held, ok := got.Body.Cards[0].Value("Name"); !ok || held != "Llama" {
		t.Errorf("Name = %q, want the first", held)
	}
	if len(got.Body.Cards) != 2 {
		t.Errorf("cards = %+v", got.Body.Cards)
	}
}

// A card's wikilink may reach a note that is not a stencil, and then no face
// lays the card out and nothing says what it is asked for. That is a problem
// against the deck, on the card it stands against.
func TestACardNamingANoteThatIsNotAStencilIsReported(t *testing.T) {
	vs := indexed(t)
	write(t, vs.first, "decks/Loose.md", "---\ntype: deck\n---\n"+
		"\n## Llama\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n"+
		"\n## Rain\n\n[[Weather]]\n\n### Height\n\nno stencil says what this is\n")
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	u := cards.Read{Readers: filesystem.VaultReaders{}, Links: vs.db.NoteQueries()}
	got, err := u.Deck(t.Context(), vs.first, "decks/Loose.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var filed []format.Problem
	for _, p := range got.Body.Problems {
		if p.Fault == format.FaultNotAStencil {
			filed = append(filed, p)
		}
	}
	if len(filed) != 1 {
		t.Fatalf("problems = %+v, want the one card naming a note that is not a stencil", got.Body.Problems)
	}
	if filed[0].Card != 1 {
		t.Errorf("problem = %+v, want it against the second card", filed[0])
	}
	if !strings.Contains(filed[0].Detail, "Weather") {
		t.Errorf("the note was not named: %q", filed[0].Detail)
	}
	// The values are read either way.
	if held, ok := got.Body.Cards[1].Value("Height"); !ok || held == "" {
		t.Errorf("the card was not read: %+v", got.Body.Cards[1])
	}
}

// prose is what stands below the frontmatter, which is what a write is given.
func prose(t *testing.T, raw string) string {
	t.Helper()
	_, body, found := strings.Cut(raw, "---\n")
	if !found {
		return raw
	}
	_, body, found = strings.Cut(body, "---\n")
	if !found {
		t.Fatal("no frontmatter in the fixture")
	}
	return body
}

// The size is asked of the file before it is opened, so a deck over the bound
// is refused with none of its bytes read.
func TestADeckOverTheBoundIsNotRead(t *testing.T) {
	vs := indexed(t)
	path := filepath.Join(vs.first.Path, "decks", "Mammals.md")
	if err := os.Truncate(path, cards.MaxBytes+1); err != nil {
		t.Fatal(err)
	}

	counted := &counting{VaultReaders: filesystem.VaultReaders{}}
	got, err := cards.Read{Readers: counted}.Deck(t.Context(), vs.first, "decks/Mammals.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.TooLarge {
		t.Fatalf("outcome = %q, want it refused", got.Outcome)
	}
	if counted.reads != 0 {
		t.Errorf("the file was opened %d times", counted.reads)
	}
	if len(got.Body.Problems) != 1 || got.Body.Problems[0].Fault != format.FaultTooLarge {
		t.Errorf("problems = %+v", got.Body.Problems)
	}
	if !strings.Contains(got.Body.Problems[0].Detail, "8388608") {
		t.Errorf("the bound was not said: %q", got.Body.Problems[0].Detail)
	}
}

// A stencil is a note and is bounded as one, which is a different number.
func TestAStencilIsBoundedAsANote(t *testing.T) {
	vs := indexed(t)
	if cards.MaxBytes == note.MaxBytes {
		t.Fatal("a deck and a note are bounded the same")
	}
	if err := os.Truncate(filepath.Join(vs.first.Path, "Animal.md"), note.MaxBytes+1); err != nil {
		t.Fatal(err)
	}

	got, err := cards.Read{Readers: filesystem.VaultReaders{}}.Stencil(t.Context(), vs.first, "Animal.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.TooLarge {
		t.Errorf("outcome = %q, want it refused", got.Outcome)
	}
}

// A deck that changed since it was read is left alone: someone editing their own
// file outranks a caller that read it, thought about it, and arrived late.
func TestADeckThatChangedSinceItWasReadIsNotWrittenOver(t *testing.T) {
	vs := indexed(t)
	w := cards.NewWrite(
		filesystem.VaultReaders{}, filesystem.VaultWriters{}, vs.db.NoteQueries(), unlevelled)

	first, err := cards.Read{Readers: filesystem.VaultReaders{}}.Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Deck(t.Context(), vs.first, "decks/Birds.md",
		"## Wren\n\n[[Animal]]\n\n### Height\n\nabout 5\"\n", first.Fingerprint); err != nil {
		t.Fatalf("write: %v", err)
	}
	written := read(t, vs.first, "decks/Birds.md")

	// The same fingerprint again is a caller holding what the file no longer is.
	_, err = w.Deck(t.Context(), vs.first, "decks/Birds.md",
		"## Wren\n\n[[Animal]]\n\n### Height\n\nsomething else\n", first.Fingerprint)
	if !errors.Is(err, port.ErrChanged) {
		t.Fatalf("write = %v, want it refused", err)
	}
	if after := read(t, vs.first, "decks/Birds.md"); after != written {
		t.Errorf("the file was written over\n was %q\n now %q", written, after)
	}
}

// A deck and a stencil are made with the key that says what they are, so each is
// what it is to everything that reads the vault before a card is written into
// it.
func TestWhatIsMadeSaysWhatItIs(t *testing.T) {
	vs := indexed(t)
	u := cards.Create{Writers: filesystem.VaultWriters{}, Index: vs.index(t)}

	deck, err := u.Deck(t.Context(), vs.first, cards.New{Title: "Birds of prey", Folder: "decks"})
	if err != nil {
		t.Fatalf("make a deck: %v", err)
	}
	if deck.Path != "decks/Birds of prey.md" {
		t.Errorf("path = %q", deck.Path)
	}
	if got := read(t, vs.first, deck.Path); !strings.Contains(got, "type: deck\n") {
		t.Errorf("no stamp: %q", got)
	}

	stencil, err := u.Stencil(t.Context(), vs.first, cards.New{
		Title: "Bird", Fields: []string{"Wingspan", "Call"},
	})
	if err != nil {
		t.Fatalf("make a stencil: %v", err)
	}
	got := read(t, vs.first, stencil.Path)
	if !strings.Contains(got, "type: stencil\n") || !strings.Contains(got, "fields:\n  - Wingspan\n  - Call\n") {
		t.Errorf("stencil written wrong: %q", got)
	}

	types, err := vs.db.NoteQueries().Types(t.Context(), vs.first.ID, []string{deck.Path, stencil.Path})
	if err != nil {
		t.Fatal(err)
	}
	if types[deck.Path] != domain.TypeDeck || types[stencil.Path] != domain.TypeStencil {
		t.Errorf("the index does not hold them as what they are: %v", types)
	}
}

// A preset is made with the key that says what it is and with none of its
// settings, and every key it does not carry stands at the default.
func TestAPresetIsMadeNamingNoneOfItsSettings(t *testing.T) {
	vs := indexed(t)
	u := cards.Create{Writers: filesystem.VaultWriters{}, Index: vs.index(t)}

	made, err := u.Preset(t.Context(), vs.first, cards.New{Title: "Prosody", Folder: "presets"})
	if err != nil {
		t.Fatalf("make a preset: %v", err)
	}
	if made.Path != "presets/Prosody.md" {
		t.Errorf("path = %q", made.Path)
	}
	got := read(t, vs.first, made.Path)
	if !strings.Contains(got, "type: preset\n") {
		t.Errorf("no stamp: %q", got)
	}
	for _, key := range []string{"goal:", "minutes_a_day:", "fields:"} {
		if strings.Contains(got, key) {
			t.Errorf("the file names %s: %q", key, got)
		}
	}

	types, err := vs.db.NoteQueries().Types(t.Context(), vs.first.ID, []string{made.Path})
	if err != nil {
		t.Fatal(err)
	}
	if types[made.Path] != domain.TypePreset {
		t.Errorf("the index does not hold it as what it is: %v", types)
	}
}

// A stencil's first field is what its cards are named by, so a stencil is made
// with one and nothing is written where there is none.
func TestAStencilIsMadeWithAFirstField(t *testing.T) {
	vs := indexed(t)
	u := cards.Create{Writers: filesystem.VaultWriters{}, Index: vs.index(t)}

	if _, err := u.Stencil(t.Context(), vs.first, cards.New{Title: "Bird"}); !errors.Is(
		err, cards.ErrNoFields,
	) {
		t.Fatalf("make a stencil of no fields = %v, want it refused", err)
	}
	if _, err := os.Stat(filepath.Join(vs.first.Path, "Bird.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the file was written anyway")
	}

	// A deck declares none, and is made all the same.
	if _, err := u.Deck(t.Context(), vs.first, cards.New{Title: "Buntings", Folder: "decks"}); err != nil {
		t.Errorf("make a deck: %v", err)
	}
}

// The window asking which kind of card to make is drawn from the vault's own
// stencils, and from no other vault's.
func TestTheStencilsOfOneVaultAreListed(t *testing.T) {
	vs := indexed(t)
	u := cards.List{Readers: filesystem.VaultReaders{}, Notes: vs.db.NoteQueries()}

	got, held, err := u.Execute(t.Context(), vs.first, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var paths []string
	for _, s := range got {
		paths = append(paths, s.Path)
	}
	if !slices.Equal(paths, []string{"Animal.md", "Term.md"}) {
		t.Fatalf("stencils = %v", paths)
	}
	if held != 2 {
		t.Errorf("held = %d, want the two the vault holds", held)
	}
	if !slices.Equal(got[0].Fields, []string{"Name", "Height", "Life span"}) {
		t.Errorf("fields = %v", got[0].Fields)
	}
	if got[0].Title != "Animal" {
		t.Errorf("title = %q", got[0].Title)
	}

	other, _, err := u.Execute(t.Context(), vs.second, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0].Path != "Mineral.md" {
		t.Errorf("the other vault's stencils = %+v", other)
	}
}

// A list that stops short opens no file for the stencils it leaves out, and the
// count still says how many the vault holds.
func TestAListReadsNoMoreThanItAnswersWith(t *testing.T) {
	vs := indexed(t)
	counted := &counting{VaultReaders: filesystem.VaultReaders{}}

	got, held, err := (cards.List{Readers: counted, Notes: vs.db.NoteQueries()}).
		Execute(t.Context(), vs.first, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Path != "Animal.md" {
		t.Fatalf("stencils = %+v, want the one asked for", got)
	}
	if held != 2 {
		t.Errorf("held = %d, want the two the vault holds", held)
	}
	if counted.reads != 1 {
		t.Errorf("%d files were opened, want the one that was answered with", counted.reads)
	}
}

// counting is a reader that says how many times a file was opened, so a test
// about what was not read can say so.
type counting struct {
	port.VaultReaders
	reads int
}

func (c *counting) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := c.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return &countingReader{VaultReader: reader, on: c}, nil
}

type countingReader struct {
	port.VaultReader
	on *counting
}

func (r *countingReader) Read(ctx context.Context, path string) ([]byte, error) {
	r.on.reads++
	return r.VaultReader.Read(ctx, path)
}

// refusing is a writer that will not write one path, which is the deck a rename
// cannot reach.
type refusing struct {
	port.VaultWriters
	path string
}

var errRefused = errors.New("the filesystem refused this file")

func (w refusing) Open(v domain.Vault) (port.VaultWriter, error) {
	writer, err := w.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return refusingWriter{VaultWriter: writer, path: w.path}, nil
}

type refusingWriter struct {
	port.VaultWriter
	path string
}

func (w refusingWriter) Write(
	ctx context.Context, path string, content []byte, fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	if path == w.path {
		return domain.Fingerprint{}, errRefused
	}
	return w.VaultWriter.Write(ctx, path, content, fingerprint)
}

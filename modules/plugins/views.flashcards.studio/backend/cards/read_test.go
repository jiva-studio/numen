package cards_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/note"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
)

var errOutOfReach = errors.New("the disk did not answer")

// faulty is the vault on disk with part of it out of reach: the vault refusing
// to open, or one path that cannot be looked at or read. A path in saved is
// written over on disk, once, the moment it has been read.
type faulty struct {
	inner ports.VaultReaders
	open  error
	stat  map[string]error
	read  map[string]error
	saved map[string]string
	root  string
}

func (f faulty) Open(v domain.Vault) (ports.VaultReader, error) {
	if f.open != nil {
		return nil, f.open
	}
	reader, err := f.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return faultyReader{VaultReader: reader, on: f}, nil
}

type faultyReader struct {
	ports.VaultReader
	on faulty
}

func (r faultyReader) GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error) {
	if err := r.on.stat[path]; err != nil {
		return domain.Fingerprint{}, err
	}
	return r.VaultReader.GetFingerprint(ctx, path)
}

func (r faultyReader) Read(ctx context.Context, path string) ([]byte, error) {
	if err := r.on.read[path]; err != nil {
		return nil, err
	}
	raw, err := r.VaultReader.Read(ctx, path)
	if after, ok := r.on.saved[path]; ok {
		delete(r.on.saved, path)
		if werr := os.WriteFile(filepath.Join(r.on.root, filepath.FromSlash(path)), []byte(after), 0o600); werr != nil {
			return nil, werr
		}
	}
	return raw, err
}

// unanswered is an index that cannot say where a wikilink lands.
type unanswered struct {
	ports.LinkQueries
	t *testing.T
	// isQuiet marks a question asked of it as a failure of the test.
	isQuiet bool
}

func (u unanswered) ResolveLinks(context.Context, domain.VaultID, string, []string) (map[string]domain.ResolvedLink, error) {
	if u.isQuiet {
		u.t.Error("a link question was asked")
	}
	return nil, errOutOfReach
}

// landing is an index that answers where each wikilink lands from a table.
type landing struct {
	ports.LinkQueries
	at map[string]string
}

func (l landing) ResolveLinks(
	_ context.Context, _ domain.VaultID, from string, written []string,
) (map[string]domain.ResolvedLink, error) {
	out := make(map[string]domain.ResolvedLink, len(written))
	for _, raw := range written {
		if to, ok := l.at[raw]; ok {
			out[raw] = domain.ResolvedLink{From: from, To: to}
		}
	}
	return out, nil
}

// A vault that will not open is the caller's error, and no outcome of the file.
func TestAVaultThatWillNotOpenIsAnError(t *testing.T) {
	vs := newVaults(t)
	u := cards.NewRead(faulty{inner: vs.ports.Readers, open: errOutOfReach}, vs.queries)

	if _, err := u.Deck(t.Context(), vs.first, "decks/Birds.md"); !errors.Is(err, errOutOfReach) {
		t.Errorf("read a deck = %v, want the vault's error", err)
	}
	if _, err := u.Stencil(t.Context(), vs.first, "Animal.md"); !errors.Is(err, errOutOfReach) {
		t.Errorf("read a stencil = %v, want the vault's error", err)
	}
}

// A deck is read against its stencils, so a stencil the vault cannot look at
// is an error of the deck's read.
func TestADeckWhoseStencilCannotBeLookedAtIsAnError(t *testing.T) {
	vs := newVaults(t)
	reader := cards.NewRead(vs.ports.Readers, vs.queries)
	birds, err := reader.Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatal(err)
	}

	u := cards.NewRead(faulty{inner: vs.ports.Readers, stat: map[string]error{"Animal.md": errOutOfReach}}, vs.queries)
	if _, err := u.Deck(t.Context(), vs.first, "decks/Birds.md"); !errors.Is(err, errOutOfReach) {
		t.Errorf("read the deck = %v, want the stencil's error", err)
	}
	if _, err := u.GetCardStencils(t.Context(), vs.first, "decks/Birds.md", birds.Body); !errors.Is(err, errOutOfReach) {
		t.Errorf("get the card stencils = %v, want the stencil's error", err)
	}
}

// Where a card's wikilink lands is the index's answer, and an index that does
// not answer is an error of the read.
func TestADeckWhoseLinksCannotBeResolvedIsAnError(t *testing.T) {
	vs := newVaults(t)
	reader := cards.NewRead(vs.ports.Readers, vs.queries)
	birds, err := reader.Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatal(err)
	}

	u := cards.NewRead(vs.ports.Readers, unanswered{t: t})
	if _, err := u.Deck(t.Context(), vs.first, "decks/Birds.md"); !errors.Is(err, errOutOfReach) {
		t.Errorf("read the deck = %v, want the index's error", err)
	}
	if _, err := u.GetCardStencils(t.Context(), vs.first, "decks/Birds.md", birds.Body); !errors.Is(err, errOutOfReach) {
		t.Errorf("get the card stencils = %v, want the index's error", err)
	}
}

// A deck whose cards name no stencil asks the index nothing.
func TestADeckNamingNoStencilAsksNoLinkQuestion(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/Bare.md", "---\ntype: deck\n---\n\n## Wren\n\n### Name\n\nWren\n")

	u := cards.NewRead(vs.ports.Readers, unanswered{t: t, isQuiet: true})
	got, err := u.Deck(t.Context(), vs.first, "decks/Bare.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.Ok {
		t.Errorf("outcome = %q, want the deck read", got.Outcome)
	}
}

// A deck whose one card names a note that is not a stencil is reported, and that
// card is cut by a stencil of nothing.
func TestTheOneCardNamingANoteThatIsNotAStencilIsReported(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/Rain.md", "---\ntype: deck\n---\n"+
		"\n## Rain\n\n[[Weather]]\n\n### Height\n\nno stencil says what this is\n")
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	u := cards.NewRead(vs.ports.Readers, vs.queries)
	got, err := u.Deck(t.Context(), vs.first, "decks/Rain.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var filed []format.Problem
	for _, p := range got.Body.Problems {
		if p.Fault == format.FaultNotAStencil {
			filed = append(filed, p)
		}
	}
	if len(filed) != 1 || filed[0].Card != 0 {
		t.Errorf("problems = %+v, want the one card naming a note that is not a stencil", got.Body.Problems)
	}

	by, err := u.GetCardStencils(t.Context(), vs.first, "decks/Rain.md", got.Body)
	if err != nil {
		t.Fatalf("get the card stencils: %v", err)
	}
	if target, ok := by["Weather"]; !ok || !reflect.DeepEqual(target, format.Stencil{}) {
		t.Errorf("the card is cut by %+v, want a stencil of nothing", target)
	}
}

// A stencil too large to read is a stencil that did not read, and the card
// naming it is not reported as naming a note that is not a stencil.
func TestACardNamingAStencilTooLargeToReadIsNotCalledANote(t *testing.T) {
	vs := newVaults(t)
	if err := os.Truncate(filepath.Join(vs.first.Path, "Animal.md"), note.MaxBytes+1); err != nil {
		t.Fatal(err)
	}

	got, err := cards.NewRead(vs.ports.Readers, vs.queries).
		Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, p := range got.Body.Problems {
		if p.Fault == format.FaultNotAStencil {
			t.Errorf("problem = %+v, want none naming the stencil a note", p)
		}
	}
}

// Each stencil is opened once, however many of the names the cards write land
// on it.
func TestAStencilNamedTwoWaysIsOpenedOnce(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/Pair.md", "---\ntype: deck\n---\n"+
		"\n## Llama\n\n[[Animal]]\n\n### Name\n\nLlama\n"+
		"\n## Wren\n\n[[Beast]]\n\n### Name\n\nWren\n")
	links := landing{at: map[string]string{"Animal": "Animal.md", "Beast": "Animal.md"}}
	counted := &counting{VaultReaders: vs.ports.Readers}

	got, err := cards.NewRead(counted, links).Deck(t.Context(), vs.first, "decks/Pair.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got.Stencils) != 2 {
		t.Fatalf("stencils = %v, want both names landing", got.Stencils)
	}
	if counted.reads != 2 {
		t.Errorf("%d files were opened, want the deck and the one stencil", counted.reads)
	}
}

// A stencil exactly at a note's bound is read.
func TestAStencilAtTheBoundIsRead(t *testing.T) {
	vs := newVaults(t)
	raw := animals["Animal.md"] + "\n"
	raw += strings.Repeat("x", note.MaxBytes-len(raw)-1) + "\n"
	write(t, vs.first, "Animal.md", raw)

	got, err := cards.NewRead(vs.ports.Readers, nil).Stencil(t.Context(), vs.first, "Animal.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.Ok || got.Fingerprint.Size != note.MaxBytes {
		t.Errorf("outcome = %q, size = %d, want a stencil of %d bytes read", got.Outcome, got.Fingerprint.Size, note.MaxBytes)
	}
}

// A path with nothing at it is an outcome, and no error.
func TestADeckWhereNothingIsIsMissing(t *testing.T) {
	vs := newVaults(t)

	got, err := cards.NewRead(vs.ports.Readers, nil).Deck(t.Context(), vs.first, "decks/Gone.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.Missing {
		t.Errorf("outcome = %q, want missing", got.Outcome)
	}
}

// A file the vault holds as a source of another kind is not a note, whatever its
// bytes say.
func TestAFileOfAnotherKindIsNotADeck(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/Wrens.url", "---\ntype: deck\n---\n\n## Wren\n\n[[Animal]]\n")

	got, err := cards.NewRead(vs.ports.Readers, nil).Deck(t.Context(), vs.first, "decks/Wrens.url")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.NotANote || len(got.Body.Cards) != 0 {
		t.Errorf("outcome = %q, cards = %+v, want it not read as a note", got.Outcome, got.Body.Cards)
	}
	if got.Fingerprint.Kind != domain.KindURL {
		t.Errorf("fingerprint = %+v, want what the vault says is there", got.Fingerprint)
	}
}

// What the vault says is at a path outranks the bytes: a path it says holds no
// note is not read as one, and a read failing there is an error.
func TestAPathTheVaultSaysHoldsNothingIsNotReadAsANote(t *testing.T) {
	vs := newVaults(t)

	nothing := faulty{inner: vs.ports.Readers, stat: map[string]error{"decks/Birds.md": fs.ErrNotExist}}
	got, err := cards.NewRead(nothing, nil).Deck(t.Context(), vs.first, "decks/Birds.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.NotANote || len(got.Body.Cards) != 0 {
		t.Errorf("outcome = %q, cards = %+v, want it not read as a note", got.Outcome, got.Body.Cards)
	}

	nothing.read = map[string]error{"decks/Birds.md": errOutOfReach}
	if _, err := cards.NewRead(nothing, nil).Deck(t.Context(), vs.first, "decks/Birds.md"); !errors.Is(err, errOutOfReach) {
		t.Errorf("read = %v, want the read's error", err)
	}
}

// A deck goes out in string fields, so bytes that are not text are named as such.
func TestADeckThatIsNotTextIsSaidToBeNotText(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/Scan.md", "---\ntype: deck\n---\n\n## Wren \xff\xfe\n\n[[Animal]]\n")

	got, err := cards.NewRead(vs.ports.Readers, nil).Deck(t.Context(), vs.first, "decks/Scan.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.NotText || got.Raw != "" {
		t.Errorf("outcome = %q, raw = %q, want not text", got.Outcome, got.Raw)
	}
	if got.Fingerprint.Path != "decks/Scan.md" || got.Fingerprint.Size == 0 {
		t.Errorf("fingerprint = %+v, want the file as it was asked about", got.Fingerprint)
	}
}

// A deck whose frontmatter is not YAML is an outcome of the file, and no error.
func TestADeckWhoseFrontmatterCannotBeReadIsUnreadable(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/Broken.md", "---\ntype: [deck\n---\n\n## Wren\n\n[[Animal]]\n")

	got, err := cards.NewRead(vs.ports.Readers, nil).Deck(t.Context(), vs.first, "decks/Broken.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.Unreadable || got.Raw != "" {
		t.Errorf("outcome = %q, raw = %q, want unreadable", got.Outcome, got.Raw)
	}
	if got.Fingerprint.Path != "decks/Broken.md" || got.Fingerprint.Size == 0 {
		t.Errorf("fingerprint = %+v, want the file as it was asked about", got.Fingerprint)
	}
}

// A deck over the bound says how large it is.
func TestADeckOverTheBoundSaysItsSize(t *testing.T) {
	vs := newVaults(t)
	if err := os.Truncate(filepath.Join(vs.first.Path, "decks", "Mammals.md"), cards.MaxBytes+1); err != nil {
		t.Fatal(err)
	}

	got, err := cards.NewRead(vs.ports.Readers, nil).Deck(t.Context(), vs.first, "decks/Mammals.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Fingerprint.Size != cards.MaxBytes+1 {
		t.Errorf("fingerprint = %+v, want the deck's size", got.Fingerprint)
	}
	if len(got.Body.Problems) != 1 || !strings.Contains(got.Body.Problems[0].Detail, strconv.Itoa(cards.MaxBytes+1)) {
		t.Errorf("problems = %+v, want the size said", got.Body.Problems)
	}
}

// A file the vault's own rules leave alone is not a note, and its bytes are not
// handed over.
func TestAFileTheVaultLeavesAloneIsNotADeck(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/.Hidden.md", "---\ntype: deck\n---\n\n## Wren\n\n[[Animal]]\n")

	got, err := cards.NewRead(vs.ports.Readers, nil).Deck(t.Context(), vs.first, "decks/.Hidden.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != note.NotANote || got.Raw != "" {
		t.Errorf("outcome = %q, raw = %q, want it not read as a note", got.Outcome, got.Raw)
	}
}

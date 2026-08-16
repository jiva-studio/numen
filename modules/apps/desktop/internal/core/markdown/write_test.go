package markdown

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

// A note that is opened and not changed comes back byte for byte. Anything less
// is a diff the person did not ask for, on every save, forever.
func TestOpenAndCloseChangesNothing(t *testing.T) {
	for name, raw := range map[string]string{
		"no frontmatter":     "# Entropy\n\nA measure.\n",
		"empty file":         "",
		"plain":              "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n\n# Entropy\n",
		"comments and order": "---\n# mine\nzebra: 1\nid: 01J8\n\n# theirs\napple: 2\n---\nbody\n",
		"crlf":               "---\r\nid: 01J8\r\n---\r\nbody\r\n",
		"no trailing eol":    "---\nid: 01J8\n---",
		"bom":                "\xef\xbb\xbfhello\n",
		"rule in body":       "# One\n\n---\n\n# Two\n",
		"unterminated":       "---\nid: 01J8\nbody with no close\n",
		"empty frontmatter":  "---\n---\nbody\n",
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if got := string(d.Bytes()); got != raw {
				t.Errorf("round trip changed the note\n want %q\n  got %q", raw, got)
			}
		})
	}
}

// The keys around the one being written are what the person owns, and they are
// the whole reason this is a splice rather than a re-encode.
func TestSetLeavesEveryOtherKeyAlone(t *testing.T) {
	raw := "---\n" +
		"# a note to myself\n" +
		"zebra:   1\n" +
		"links:\n" +
		"  - to: Old\n" +
		"    role: parent\n" +
		"\n" +
		"# about apples\n" +
		"apple: 'two'\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.AddLink(domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: "New"},
		Role:   domain.RoleChild,
	}); err != nil {
		t.Fatalf("add link: %v", err)
	}

	got := string(d.Bytes())
	for _, kept := range []string{
		"# a note to myself\n", "zebra:   1\n", "# about apples\n", "apple: 'two'\n", "body\n",
	} {
		if !strings.Contains(got, kept) {
			t.Errorf("lost %q from\n%s", kept, got)
		}
	}
	if !strings.Contains(got, "to: Old") {
		t.Errorf("adding a link took the other one with it:\n%s", got)
	}
	if !strings.Contains(got, "- to: New") || !strings.Contains(got, "role: child") {
		t.Errorf("the new link is not there:\n%s", got)
	}
	// The comment above `apple` belongs to `apple`, not to the entry before it.
	if strings.Index(got, "# about apples") > strings.Index(got, "apple: 'two'") {
		t.Errorf("a comment moved:\n%s", got)
	}
}

func TestSetAddsAKeyThatWasNotThere(t *testing.T) {
	d, err := Open([]byte("---\napple: 2\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetIdentifier("01J8F3K2M9QRSTVWXYZ012"); err != nil {
		t.Fatalf("set identifier: %v", err)
	}
	got := string(d.Bytes())
	if !strings.Contains(got, "apple: 2\n") || !strings.Contains(got, "id: 01J8F3K2M9QRSTVWXYZ012\n") {
		t.Fatalf("expected both keys:\n%s", got)
	}
	if id, ok := reopen(t, got).Identifier(); !ok || id != "01J8F3K2M9QRSTVWXYZ012" {
		t.Errorf("the identifier did not survive a reopen: %q %v", id, ok)
	}
}

func TestANoteWithNoFrontmatterGrowsOne(t *testing.T) {
	d, err := Open([]byte("# Entropy\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetIdentifier("01J8"); err != nil {
		t.Fatalf("set identifier: %v", err)
	}
	if got, want := string(d.Bytes()), "---\nid: 01J8\n---\n# Entropy\n"; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestRemovingEveryLinkTakesTheKeyOut(t *testing.T) {
	d, err := Open([]byte("---\nid: 01J8\nlinks:\n  - to: Old\n    role: parent\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := d.RemoveLink(domain.Address{Scheme: domain.SchemeName, Value: "Old"}, ""); err != nil {
		t.Fatalf("remove link: %v", err)
	}
	if got, want := string(d.Bytes()), "---\nid: 01J8\n---\nbody\n"; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// A note nobody can read is a note nobody may write: the alternative is
// guessing at the block that failed to parse.
func TestFrontmatterThatDoesNotParseIsRefused(t *testing.T) {
	_, err := Open([]byte("---\nid: [unterminated\n---\nbody\n"))
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("want ErrUnreadable, got %v", err)
	}
}

func TestWhatIsWrittenIntoACRLFNoteIsCRLF(t *testing.T) {
	d, err := Open([]byte("---\r\nid: 01J8\r\n---\r\nbody\r\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.AddLink(domain.Link{
		Target: domain.Address{Scheme: domain.SchemeNote, Value: "01J9"},
		Role:   domain.RoleParent,
	}); err != nil {
		t.Fatalf("add link: %v", err)
	}
	got := string(d.Bytes())
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare newline was written into a CRLF note:\n%q", got)
	}
	if !strings.Contains(got, "to: note://01J9") {
		t.Errorf("an identifier address is written whole:\n%s", got)
	}
}

// `name://` is how the index holds an address and never how a file spells one.
func TestANameIsWrittenAsItself(t *testing.T) {
	d, err := Open([]byte("---\nid: 01J8\n---\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.AddLink(domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
		Role:   domain.RoleJump,
		Label:  "why it matters",
	}); err != nil {
		t.Fatalf("add link: %v", err)
	}
	got := string(d.Bytes())
	if strings.Contains(got, "name://") {
		t.Errorf("name:// reached a file:\n%s", got)
	}
	if !strings.Contains(got, "to: Entropy") || !strings.Contains(got, "label: why it matters") {
		t.Errorf("want the link written plainly:\n%s", got)
	}
}

// What Create writes has to be readable by the parser that reads every other
// note, or the application has invented a second format.
func TestCreateIsReadBackByTheParser(t *testing.T) {
	identifier, err := ulid.New(time.Now())
	if err != nil {
		t.Fatalf("new identifier: %v", err)
	}
	raw := Create(identifier, "# Entropy\n")
	n := Parse(domain.FileRef{Path: "entropy.md"}, raw)
	if n.ID != identifier {
		t.Errorf("identifier not read back: %q", n.ID)
	}
	if n.Title != "Entropy" {
		t.Errorf("title comes from the heading, got %q", n.Title)
	}
	if len(n.Problems) != 0 {
		t.Errorf("a note the application wrote has problems: %v", n.Problems)
	}
}

func reopen(t *testing.T, raw string) *Document {
	t.Helper()
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return d
}

// The finding that sent this back: writing one link must not touch the entries
// around it. An entry the parser could not act on is still somebody's writing,
// a key the application has never heard of is theirs, and a comment inside the
// block is a note to themselves.
func TestWritingOneLinkLeavesTheOthersAsBytes(t *testing.T) {
	raw := "---\n" +
		"links:\n" +
		"  # why this one exists\n" +
		"  - to: Entropy\n" +
		"    role: supersedes\n" +
		"    mine: keep me\n" +
		"  - to: Heat\n" +
		"    role: parent\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.AddLink(domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: "Work"},
		Role:   domain.RoleJump,
	}); err != nil {
		t.Fatalf("add link: %v", err)
	}

	got := string(d.Bytes())
	for _, kept := range []string{
		"# why this one exists\n", "- to: Entropy\n", "role: supersedes\n",
		"mine: keep me\n", "- to: Heat\n",
	} {
		if !strings.Contains(got, kept) {
			t.Errorf("lost %q from\n%s", kept, got)
		}
	}
	if !strings.Contains(got, "- to: Work") {
		t.Errorf("the new link is not there:\n%s", got)
	}
}

// A key the application does not own is the person's, and an entry carrying one
// is refused rather than rewritten without it.
func TestAnEntryCarryingSomebodyElsesKeyIsRefused(t *testing.T) {
	d, err := Open([]byte("---\nlinks:\n  - to: Entropy\n    role: parent\n    mine: keep me\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = d.UpdateLink(
		domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
		domain.Link{Role: domain.RoleChild},
	)
	if !errors.Is(err, ErrNotOurs) {
		t.Fatalf("want ErrNotOurs, got %v", err)
	}
	if !strings.Contains(string(d.Bytes()), "mine: keep me") {
		t.Error("the refused change was made anyway")
	}
}

// Repairing an address changes the address and nothing else on the line, nor
// anything around it.
func TestPointingALinkSomewhereElseChangesOnlyTheAddress(t *testing.T) {
	raw := "---\n" +
		"links:\n" +
		"  - to: notes/Entropy   # the old place\n" +
		"    role: parent\n" +
		"    label: follows from\n" +
		"---\n" +
		"see [[notes/Entropy|entropy]] and [[Heat]]\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	from := domain.ParseAddress("notes/Entropy")
	inBlock, err := d.PointLinksAt(from, "Entropy")
	if err != nil {
		t.Fatalf("point links: %v", err)
	}
	inProse := d.PointProseAt(from, "Entropy")
	if inBlock != 1 || inProse != 1 {
		t.Fatalf("want one of each, got %d in the block and %d in prose", inBlock, inProse)
	}

	got := string(d.Bytes())
	for _, kept := range []string{
		"# the old place", "role: parent\n", "label: follows from\n",
		"[[Entropy|entropy]]", "[[Heat]]",
	} {
		if !strings.Contains(got, kept) {
			t.Errorf("lost %q from\n%s", kept, got)
		}
	}
	if strings.Contains(got, "notes/Entropy") {
		t.Errorf("the old address is still written:\n%s", got)
	}
}

// A name is somebody's filename, and a filename may hold anything. Writing it
// into YAML raw is how renaming one note makes another unparseable.
func TestARenameCannotBreakTheNoteThatPointedAtIt(t *testing.T) {
	for _, name := range []string{
		"[Draft] Physics", "{x", "- dash", "%pct", "@at", "*ref", "&anchor",
		"yes", "true", "3.14", "null", "a: b",
	} {
		t.Run(name, func(t *testing.T) {
			raw := "---\nlinks:\n  - to: notes/Old\n    role: parent\n---\nbody\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.PointLinksAt(domain.ParseAddress("notes/Old"), name); err != nil {
				t.Fatalf("point links: %v", err)
			}

			after, err := Open(d.Bytes())
			if err != nil {
				t.Fatalf("the note stopped being readable: %v\n%s", err, d.Bytes())
			}
			links, err := after.Links()
			if err != nil {
				t.Fatalf("links: %v", err)
			}
			if len(links) != 1 || links[0].Target.Value != name {
				t.Errorf("want the link pointing at %q, got %+v\n%s", name, links, d.Bytes())
			}
		})
	}
}

// The address is a key, not a line that reads like one. A sentence carrying
// `to:` in it, or a key called `proto`, is neither.
func TestOnlyTheAddressKeyIsRewritten(t *testing.T) {
	raw := "---\n" +
		"links:\n" +
		"  - role: jump\n" +
		"    note: \"pass it to: Bob\"\n" +
		"    to: A\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	moved, err := d.PointLinksAt(domain.ParseAddress("A"), "B")
	if err != nil {
		t.Fatalf("point links: %v", err)
	}
	if moved != 1 {
		t.Fatalf("want one address moved, got %d", moved)
	}

	got := string(d.Bytes())
	if !strings.Contains(got, `note: "pass it to: Bob"`) {
		t.Errorf("somebody's own words were rewritten:\n%s", got)
	}
	if !strings.Contains(got, "to: B") {
		t.Errorf("the address was not moved:\n%s", got)
	}
	if _, err := Open([]byte(got)); err != nil {
		t.Errorf("the note stopped being readable: %v", err)
	}
}

// One line holding several keys means the span of any of them is the span of
// all of them, so a write meant for one would take the rest with it.
func TestFrontmatterOnOneLineIsRefusedRatherThanMangled(t *testing.T) {
	for name, raw := range map[string]string{
		"a flow mapping":  "---\n{title: T, id: b}\n---\nbody\n",
		"a flow sequence": "---\nlinks: [{to: A, role: jump}]\n---\nbody\n",
	} {
		t.Run(name, func(t *testing.T) {
			// Reading it is fine; only changing it is refused.
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := d.SetIdentifier("01J8"); !errors.Is(err, ErrInline) {
				t.Errorf("want ErrInline, got %v", err)
			}
			if err := d.AddLink(domain.Link{
				Target: domain.ParseAddress("B"), Role: domain.RoleJump,
			}); !errors.Is(err, ErrInline) {
				t.Errorf("want ErrInline from AddLink, got %v", err)
			}
			if got := string(d.Bytes()); got != raw {
				t.Errorf("a refused write changed the note\n want %q\n  got %q", raw, got)
			}
		})
	}
}

// A file may be called almost anything; an address may not. A note whose name
// no link could spell is left unreached rather than reached by a link that now
// says something else.
func TestANameNoAddressCanSpellIsNotWrittenIn(t *testing.T) {
	for _, name := range []string{"#tag", "multi\nline", "a|b", "note://x", "ends]]"} {
		t.Run(name, func(t *testing.T) {
			raw := "---\nlinks:\n  - to: notes/Old\n    role: parent\n---\nsee [[notes/Old]]\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			from := domain.ParseAddress("notes/Old")
			inBlock, err := d.PointLinksAt(from, name)
			if err != nil {
				t.Fatalf("point links: %v", err)
			}
			if inBlock != 0 || d.PointProseAt(from, name) != 0 {
				t.Errorf("a name no link can spell was written in anyway:\n%s", d.Bytes())
			}
			if got := string(d.Bytes()); got != raw {
				t.Errorf("the note changed\n want %q\n  got %q", raw, got)
			}
		})
	}
}

// A file that opens the block and never closes it is somebody's frontmatter
// with a line missing. Writing would put a second block above theirs and turn
// their keys into prose.
func TestANoteWithAnUnclosedBlockIsNotWrittenTo(t *testing.T) {
	raw := "---\ntitle: theirs\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("reading it is still fine: %v", err)
	}
	if err := d.SetIdentifier("01J8"); !errors.Is(err, ErrUnterminated) {
		t.Fatalf("want ErrUnterminated, got %v", err)
	}
	if got := string(d.Bytes()); got != raw {
		t.Errorf("the note changed\n want %q\n  got %q", raw, got)
	}
}

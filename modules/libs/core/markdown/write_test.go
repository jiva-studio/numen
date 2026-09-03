package markdown

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
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
	if n.Title != "entropy" {
		t.Errorf("title comes from the filename, got %q", n.Title)
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
	raw := "---\n{title: T, id: b}\n---\nbody\n"

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
}

// A value written on one line occupies its key's own lines, so a key beside it
// is written as usual. An entry of that value is what cannot be replaced on its
// own.
func TestAValueOnOneLineLeavesTheKeysAroundItWritable(t *testing.T) {
	raw := "---\nlinks: [{to: A, role: jump}]\nlight_days: [sat]\n---\nbody\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetIdentifier("01J8"); err != nil {
		t.Errorf("write: %v", err)
	}
	if err := d.AddLink(domain.Link{
		Target: domain.ParseAddress("B"), Role: domain.RoleJump,
	}); !errors.Is(err, ErrInline) {
		t.Errorf("want ErrInline from AddLink, got %v", err)
	}

	got := string(d.Bytes())
	for _, kept := range []string{"links: [{to: A, role: jump}]", "light_days: [sat]", "id: 01J8"} {
		if !strings.Contains(got, kept) {
			t.Errorf("%q is not in the note:\n%s", kept, got)
		}
	}
	if _, err := Open([]byte(got)); err != nil {
		t.Errorf("the note stopped being readable: %v", err)
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

// A rename reaches a link whether it is quoted or written plainly.
func TestARenameReachesALinkHoweverItIsQuoted(t *testing.T) {
	for _, one := range []struct {
		name    string
		written string
		target  string
	}{
		{"double quoted brackets", `"[[The far corner]]"`, "[[The far corner]]"},
		{"single quoted brackets", `'[[The far corner]]'`, "[[The far corner]]"},
		{"double quoted path", `"notes/The far corner"`, "notes/The far corner"},
		{"single quoted path", `'notes/The far corner'`, "notes/The far corner"},
		{"plainly written", `notes/The far corner`, "notes/The far corner"},
	} {
		t.Run(one.name, func(t *testing.T) {
			raw := "---\n" +
				"links:\n" +
				"  - to: " + one.written + "   # where it sat\n" +
				"    role: child\n" +
				"---\n" +
				"body\n"

			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			moved, err := d.PointLinksAt(domain.ParseAddress(one.target), "Marrowfield")
			if err != nil {
				t.Fatalf("point links: %v", err)
			}
			if moved != 1 {
				t.Fatalf("want the link moved, got %d\n%s", moved, d.Bytes())
			}

			got := string(d.Bytes())
			after, err := Open(d.Bytes())
			if err != nil {
				t.Fatalf("the note stopped being readable: %v\n%s", err, got)
			}
			links, err := after.Links()
			if err != nil {
				t.Fatalf("links: %v", err)
			}
			if len(links) != 1 || links[0].Target.Value != "Marrowfield" {
				t.Errorf("want the link at Marrowfield, got %+v\n%s", links, got)
			}
			for _, kept := range []string{"# where it sat", "role: child\n"} {
				if !strings.Contains(got, kept) {
					t.Errorf("lost %q from\n%s", kept, got)
				}
			}
			if strings.Contains(got, "corner") {
				t.Errorf("the old address is still written:\n%s", got)
			}
		})
	}
}

// A quote inside a quoted name is written the way that quoting writes one, and
// the span has to end at the quote that closes the scalar and not at that one.
func TestARenameReadsPastAQuoteInsideTheName(t *testing.T) {
	for _, one := range []struct {
		name    string
		written string
		target  string
	}{
		{"a doubled quote", `'Alice''s key'`, "Alice's key"},
		{"an escaped quote", `"Alice\"s key"`, `Alice"s key`},
		{"a hash that is not a comment", `"Shed #LXVI"`, "Shed #LXVI"},
	} {
		t.Run(one.name, func(t *testing.T) {
			raw := "---\nlinks:\n  - to: " + one.written + "\n    role: child\n---\nbody\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			moved, err := d.PointLinksAt(domain.ParseAddress(one.target), "Marrowfield")
			if err != nil {
				t.Fatalf("point links: %v", err)
			}
			if moved != 1 {
				t.Fatalf("want the link moved, got %d\n%s", moved, d.Bytes())
			}
			after, err := Open(d.Bytes())
			if err != nil {
				t.Fatalf("the note stopped being readable: %v\n%s", err, d.Bytes())
			}
			links, err := after.Links()
			if err != nil {
				t.Fatalf("links: %v", err)
			}
			if len(links) != 1 || links[0].Target.Value != "Marrowfield" {
				t.Errorf("want the link at Marrowfield, got %+v\n%s", links, d.Bytes())
			}
		})
	}
}

// A scalar written over lines of its own is not one token on one line. The link
// stays as it was written, and the note is left readable.
func TestARenameLeavesABlockScalarAlone(t *testing.T) {
	raw := "---\nlinks:\n  - to: >-\n      The far corner\n    role: child\n---\nbody\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	moved, err := d.PointLinksAt(domain.ParseAddress("The far corner"), "Marrowfield")
	if err != nil {
		t.Fatalf("point links: %v", err)
	}
	if moved != 0 {
		t.Fatalf("want nothing moved, got %d\n%s", moved, d.Bytes())
	}
	if got := string(d.Bytes()); got != raw {
		t.Errorf("the note changed\n want %q\n  got %q", raw, got)
	}
}

// How somebody writes their own links is theirs: the quotes around the name and
// the brackets inside them are how they wrote it, and a rename changes the name.
func TestARenameKeepsTheNotationItFound(t *testing.T) {
	for _, one := range []struct {
		name    string
		written string
		wants   string
	}{
		{"double stays double", `"[[The far corner]]"`, `to: "[[Marrowfield]]"`},
		{"single stays single", `'[[The far corner]]'`, `to: '[[Marrowfield]]'`},
		{"an alias is left standing", `"[[The far corner|him]]"`, `to: "[[Marrowfield|him]]"`},
		{"a place inside is left standing", `"[[The far corner#gate]]"`, `to: "[[Marrowfield#gate]]"`},
		{"a quoted path is repaired by name", `"notes/The far corner"`, `to: "Marrowfield"`},
		{"plainly written stays plain", `The far corner`, `to: Marrowfield`},
	} {
		t.Run(one.name, func(t *testing.T) {
			raw := "---\nlinks:\n  - to: " + one.written + "\n    role: child\n---\nbody\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			target := strings.Trim(one.written, `"'`)
			if _, err := d.PointLinksAt(domain.ParseAddress(target), "Marrowfield"); err != nil {
				t.Fatalf("point links: %v", err)
			}

			got := string(d.Bytes())
			if !strings.Contains(got, one.wants) {
				t.Errorf("want %q in\n%s", one.wants, got)
			}
		})
	}
}

// A note names another in one role, so a role written where one already
// stands is the role it stands in from then on.
func TestALinkWrittenAgainInAnotherRoleIsReseated(t *testing.T) {
	raw := "---\n" +
		"links:\n" +
		"  - to: \"[[Entropy]]\"\n" +
		"    role: child\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.AddLink(domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
		Role:   domain.RoleParent,
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	out := string(d.Bytes())
	if strings.Count(out, "to:") != 1 {
		t.Errorf("the note names it twice:\n%s", out)
	}
	if !strings.Contains(out, "role: parent") || strings.Contains(out, "role: child") {
		t.Errorf("it did not take the new role:\n%s", out)
	}
}

// A frontmatter written in from the margin is spliced where its own keys stand.
// A line written flush ends the mapping, and every key below it becomes text
// the YAML decoder drops without saying so — the person's own keys among them.
func TestAKeyIsWrittenWhereTheBlocksOwnKeysStand(t *testing.T) {
	raw := "---\n" +
		"  type: preset\n" +
		"  id: 01J8\n" +
		"  minutes_a_day: 20\n" +
		"  colour: green\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetValue("minutes_a_day", 35); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := d.SetValue("new_a_day", 8); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := string(d.Bytes())
	back := reopen(t, got)
	if id, ok := back.Identifier(); !ok || id != "01J8" {
		t.Errorf("the note lost its identity: %q %v\n%s", id, ok, got)
	}
	front := Parse(domain.FileRef{Path: "Sanskrit.md"}, []byte(got)).Frontmatter
	want := map[string]any{
		"type": "preset", "id": "01J8", "minutes_a_day": 35, "new_a_day": 8,
		"colour": "green",
	}
	if !reflect.DeepEqual(front, want) {
		t.Errorf("frontmatter = %v, want %v\n%s", front, want, got)
	}
}

// A frontmatter carrying an anchor is left alone. Replacing the value the anchor
// stands on leaves the alias pointing at nothing, and the note stops opening at
// all — not the one key, the whole of it.
func TestAnAnchoredFrontmatterIsRefused(t *testing.T) {
	raw := "---\n" +
		"id: 01J8\n" +
		"retention: &target 0.87\n" +
		"mine: *target\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	added := domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: "New"}, Role: domain.RoleChild,
	}
	for name, write := range map[string]func() error{
		"a value":         func() error { return d.SetValue("retention", 0.9) },
		"the identifier":  func() error { return d.SetIdentifier("01J9") },
		"the title":       func() error { return d.SetTitle("Sanskrit") },
		"a mapping":       func() error { return d.SetMapping("load", []Entry{{Key: "sat", Value: 50}}) },
		"a list":          func() error { return d.SetList("tags", []string{"study"}) },
		"a link":          func() error { return d.AddLink(added) },
		"a key not there": func() error { return d.SetValue("new_a_day", 8) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := write(); !errors.Is(err, ErrAnchored) {
				t.Fatalf("want ErrAnchored, got %v", err)
			}
		})
	}
	if got := string(d.Bytes()); got != raw {
		t.Errorf("the note was written\n want %q\n  got %q", raw, got)
	}
	if _, err := Open(d.Bytes()); err != nil {
		t.Errorf("the note no longer opens: %v", err)
	}
}

// A comment beside a key is the person's, on a key the application owns as much
// as on any other. Every writer here keeps it.
func TestAWriteKeepsTheCommentBesideTheKey(t *testing.T) {
	raw := "---\n" +
		"id: 01J8 # the one it was made with\n" +
		"title: Old # what I called it\n" +
		"minutes_a_day: 20 # twenty is plenty\n" +
		"load: # the week\n" +
		"  sat: 50 # half a Saturday\n" +
		"tags: # what it is about\n" +
		"  - study\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetIdentifier("01J9"); err != nil {
		t.Fatalf("set identifier: %v", err)
	}
	if err := d.SetTitle("New"); err != nil {
		t.Fatalf("set title: %v", err)
	}
	if err := d.SetValue("minutes_a_day", 35); err != nil {
		t.Fatalf("set value: %v", err)
	}
	if err := d.SetMapping("load", []Entry{{Key: "sat", Value: 20}}); err != nil {
		t.Fatalf("set mapping: %v", err)
	}
	if err := d.SetList("tags", []string{"study", "grammar"}); err != nil {
		t.Fatalf("set list: %v", err)
	}

	got := string(d.Bytes())
	for _, kept := range []string{
		"id: 01J9 # the one it was made with\n",
		"title: New # what I called it\n",
		"minutes_a_day: 35 # twenty is plenty\n",
		"load: # the week\n",
		"sat: 20 # half a Saturday\n",
		"tags: # what it is about\n",
	} {
		if !strings.Contains(got, kept) {
			t.Errorf("want %q in\n%s", kept, got)
		}
	}
}

// A note names one place under a link type. The entry carrying the type is the
// one that moves, and every other entry comes out as the bytes it went in as.
func TestOneTypeNamesOnePlace(t *testing.T) {
	raw := "---\n" +
		"links:\n" +
		"  - to: Thermodynamics\n" +
		"    role: parent\n" +
		"  - to: Sanskrit   # twenty minutes\n" +
		"    role: ref\n" +
		"    type: preset\n" +
		"    note: as much as I have\n" +
		"---\nbody\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := d.SetLinkOfType(
		"preset", domain.Address{Scheme: domain.SchemeName, Value: "Slow going"}, domain.RoleRef,
	); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := string(d.Bytes())
	if !strings.Contains(got, "  - to: Thermodynamics\n    role: parent\n") {
		t.Errorf("the other entry was rewritten in\n%s", got)
	}
	if strings.Contains(got, "to: Sanskrit") {
		t.Errorf("the entry still names where it went in\n%s", got)
	}
	if !strings.Contains(got, "to: Slow going") || !strings.Contains(got, "note: as much as I have") {
		t.Errorf("the entry did not move whole in\n%s", got)
	}
	if strings.Count(got, "type: preset") != 1 {
		t.Errorf("the note names more than one place under the type in\n%s", got)
	}
}

// A note naming no place under the type grows an entry for it, and one written
// with no block grows the block too.
func TestATypeNobodyNamedIsWrittenIn(t *testing.T) {
	for name, raw := range map[string]string{
		"no block":       "---\ntype: deck\n---\nbody\n",
		"a block":        "---\nlinks:\n  - to: Thermodynamics\n    role: parent\n---\nbody\n",
		"no frontmatter": "body\n",
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := d.SetLinkOfType(
				"preset", domain.Address{Scheme: domain.SchemeName, Value: "Sanskrit"}, domain.RoleRef,
			); err != nil {
				t.Fatalf("set: %v", err)
			}

			n := Parse(domain.FileRef{Path: "decks/Roots.md"}, d.Bytes())
			var named []domain.Link
			for _, link := range n.Links {
				if link.Type == "preset" {
					named = append(named, link)
				}
			}
			if len(named) != 1 {
				t.Fatalf("the note names %d presets:\n%s", len(named), d.Bytes())
			}
			if named[0].Target.Value != "Sanskrit" || named[0].Role != domain.RoleRef {
				t.Errorf("the entry says %+v", named[0])
			}
		})
	}
}

// An empty address takes the entry out, and the block goes with it when it held
// nothing else.
func TestNamingNoPlaceUnderATypeTakesTheEntryOut(t *testing.T) {
	d, err := Open([]byte(
		"---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetLinkOfType("preset", domain.Address{}, domain.RoleRef); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := string(d.Bytes()); got != "---\ntype: deck\n---\nbody\n" {
		t.Errorf("got %q", got)
	}
}

// An entry carrying a key the application does not own is left as the person
// wrote it, and nothing is written.
func TestAnEntryOfTheTypeCarryingSomebodyElsesKeyIsRefused(t *testing.T) {
	raw := "---\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n    mine: keep me\n---\nbody\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	err = d.SetLinkOfType(
		"preset", domain.Address{Scheme: domain.SchemeName, Value: "Slow going"}, domain.RoleRef)
	if !errors.Is(err, ErrNotOurs) {
		t.Fatalf("want ErrNotOurs, got %v", err)
	}
	if got := string(d.Bytes()); got != raw {
		t.Errorf("the refused change was made anyway\n%s", got)
	}
}

// An entry written with no role is not a link, so it names nothing under the
// type and is left exactly as it stands.
func TestAnEntryWithNoRoleIsLeftWhereItStands(t *testing.T) {
	d, err := Open([]byte(
		"---\nlinks:\n  - to: Sanskrit\n    type: preset\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetLinkOfType(
		"preset", domain.Address{Scheme: domain.SchemeName, Value: "Slow going"}, domain.RoleRef,
	); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := string(d.Bytes())
	if !strings.Contains(got, "  - to: Sanskrit\n    type: preset\n") {
		t.Errorf("the entry the parser could not read was rewritten in\n%s", got)
	}
	if !strings.Contains(got, "to: Slow going") {
		t.Errorf("the entry was not written in\n%s", got)
	}
}

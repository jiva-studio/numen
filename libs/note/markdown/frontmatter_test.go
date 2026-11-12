package markdown

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// The keys around the one being written belong to the user: the write is a
// splice of that key's span.
func TestSetPreservesEveryOtherKeyAlone(t *testing.T) {
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
	for _, want := range []string{
		"# a note to myself\n", "zebra:   1\n", "# about apples\n", "apple: 'two'\n", "body\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lost %q from\n%s", want, got)
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

func TestRemovingEveryLinkConsumesTheKeyOut(t *testing.T) {
	d, err := Open([]byte("---\nid: 01J8\nlinks:\n  - to: Old\n    role: parent\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := d.RemoveLink(domain.LinkKey{Target: domain.Address{Scheme: domain.SchemeName, Value: "Old"}}); err != nil {
		t.Fatalf("remove link: %v", err)
	}
	if got, want := string(d.Bytes()), "---\nid: 01J8\n---\nbody\n"; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// A note that cannot be read is not written, since the block that failed to
// parse cannot be located.
func TestFrontmatterThatDoesNotParseIsRefused(t *testing.T) {
	_, err := Open([]byte("---\nid: [unterminated\n---\nbody\n"))
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("want ErrUnreadable, got %v", err)
	}
}

func TestWhatIsSavedIntoACRLFNoteIsCRLF(t *testing.T) {
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
func TestANameIsSavedAsItself(t *testing.T) {
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

func reopen(t *testing.T, raw string) *Document {
	t.Helper()
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return d
}

// Writing one link must not touch the surrounding entries. An entry the parser
// could not act on is user content, a key the application does not recognize
// belongs to the user, and a comment inside the block is the user's note.
func TestWritingOneLinkPreservesTheOthersAsBytes(t *testing.T) {
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
	for _, want := range []string{
		"# why this one exists\n", "- to: Entropy\n", "role: supersedes\n",
		"mine: keep me\n", "- to: Heat\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lost %q from\n%s", want, got)
		}
	}
	if !strings.Contains(got, "- to: Work") {
		t.Errorf("the new link is not there:\n%s", got)
	}
}

// A key the application does not own belongs to the user, and an entry
// carrying one fails and is not rewritten without it.
func TestAnEntryCarryingSomebodyElsesKeyIsRefused(t *testing.T) {
	d, err := Open([]byte("---\nlinks:\n  - to: Entropy\n    role: parent\n    mine: keep me\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = d.UpdateLink(
		domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
		domain.Link{Role: domain.RoleChild},
	)
	if !errors.Is(err, ErrNotOwned) {
		t.Fatalf("want ErrNotOwned, got %v", err)
	}
	if !strings.Contains(string(d.Bytes()), "mine: keep me") {
		t.Error("the rejected change was made anyway")
	}
}

// application owns the second and not the first, and the note comes out of the
// call as every byte it went in.
func TestARefusedUpdateWritesNoEntryAtAll(t *testing.T) {
	raw := "---\nlinks:\n" +
		"  - to: Entropy\n    role: parent\n    mine: keep me\n" +
		"  - to: Entropy\n    role: ref\n" +
		"---\nbody\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	changed, err := d.UpdateLink(
		domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
		domain.Link{Role: domain.RoleChild},
	)
	if !errors.Is(err, ErrNotOwned) {
		t.Fatalf("want ErrNotOwned, got %v", err)
	}
	if changed != 0 {
		t.Errorf("changed = %d", changed)
	}
	if got := string(d.Bytes()); got != raw {
		t.Errorf("a rejected change was written\n want %q\n  got %q", raw, got)
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
	inBlock, err := d.RedirectLinks(from, "Entropy")
	if err != nil {
		t.Fatalf("point links: %v", err)
	}
	inProse, err := d.RedirectProse(from, "Entropy")
	if err != nil {
		t.Fatalf("point prose: %v", err)
	}
	if inBlock != 1 || inProse != 1 {
		t.Fatalf("want one of each, got %d in the block and %d in prose", inBlock, inProse)
	}

	got := string(d.Bytes())
	for _, want := range []string{
		"# the old place", "role: parent\n", "label: follows from\n",
		"[[Entropy|entropy]]", "[[Heat]]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lost %q from\n%s", want, got)
		}
	}
	if strings.Contains(got, "notes/Entropy") {
		t.Errorf("the old address is still written:\n%s", got)
	}
}

// A name is a filename, and a filename may hold any character. Writing it raw
// into YAML makes renaming one note render another unparseable.
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
			if _, err := d.RedirectLinks(domain.ParseAddress("notes/Old"), name); err != nil {
				t.Fatalf("point links: %v", err)
			}

			after, err := Open(d.Bytes())
			if err != nil {
				t.Fatalf("the note stopped being readable: %v\n%s", err, d.Bytes())
			}
			links, err := readLinks(after)
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
	moved, err := d.RedirectLinks(domain.ParseAddress("A"), "B")
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
func TestFrontmatterOnOneLineIsRefusedInsteadOfMangled(t *testing.T) {
	raw := "---\n{title: T, id: b}\n---\nbody\n"

	// Reading it is fine; only changing it fails.
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
		t.Errorf("a failed write changed the note\n want %q\n  got %q", raw, got)
	}
}

// A value written on one line occupies its key's own lines, so a key beside it
// is written as usual. An entry of that value is what cannot be replaced on its
// own.
func TestAValueOnOneLinePreservesTheKeysAroundItWritable(t *testing.T) {
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
	for _, want := range []string{"links: [{to: A, role: jump}]", "light_days: [sat]", "id: 01J8"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q is not in the note:\n%s", want, got)
		}
	}
	if _, err := Open([]byte(got)); err != nil {
		t.Errorf("the note stopped being readable: %v", err)
	}
}

// A file may have almost any name; a link address may not. A note whose name
// no link can express stays unreached by links, and no link is rewritten to
// point at it.
func TestANameNoAddressCanSpellIsNotSavedIn(t *testing.T) {
	for _, name := range []string{"#tag", "multi\nline", "a|b", "note://x", "ends]]"} {
		t.Run(name, func(t *testing.T) {
			raw := "---\nlinks:\n  - to: notes/Old\n    role: parent\n---\nsee [[notes/Old]]\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			from := domain.ParseAddress("notes/Old")
			inBlock, err := d.RedirectLinks(from, name)
			if err != nil {
				t.Fatalf("point links: %v", err)
			}
			inProse, err := d.RedirectProse(from, name)
			if err != nil {
				t.Fatalf("point prose: %v", err)
			}
			if inBlock != 0 || inProse != 0 {
				t.Errorf("a name no link can spell was written in anyway:\n%s", d.Bytes())
			}
			if got := string(d.Bytes()); got != raw {
				t.Errorf("the note changed\n want %q\n  got %q", raw, got)
			}
		})
	}
}

// A file that opens the block and never closes it is frontmatter with a
// missing line. Writing a key would add a second block above it and turn its
// keys into prose.
func TestANoteWithAnUnclosedBlockRejectsAFrontmatterWrite(t *testing.T) {
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

// the half-written block, and rewriting the prose's links to a new name
// changes the links inside it.
func TestANoteWithAnUnclosedBlockRejectsABodyWrite(t *testing.T) {
	raw := "---\ntitle: theirs\nlinks:\n  - to: notes/Old\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("reading it is still fine: %v", err)
	}
	if err := d.SetBody("mine\n"); !errors.Is(err, ErrUnterminated) {
		t.Fatalf("SetBody: want ErrUnterminated, got %v", err)
	}
	moved, err := d.RedirectProse(domain.ParseAddress("notes/Old"), "New")
	if !errors.Is(err, ErrUnterminated) {
		t.Fatalf("RedirectProse: want ErrUnterminated, got %v", err)
	}
	if moved != 0 {
		t.Errorf("it moved %d links", moved)
	}
	if got := string(d.Bytes()); got != raw {
		t.Errorf("the note changed\n want %q\n  got %q", raw, got)
	}
}

// A rename rewrites a link whether it is quoted or unquoted.
func TestARenameReachesALinkHoweverItIsQuoted(t *testing.T) {
	for _, one := range []struct {
		name   string
		input  string
		target string
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
				"  - to: " + one.input + "   # where it sat\n" +
				"    role: child\n" +
				"---\n" +
				"body\n"

			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			moved, err := d.RedirectLinks(domain.ParseAddress(one.target), "Marrowfield")
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
			links, err := readLinks(after)
			if err != nil {
				t.Fatalf("links: %v", err)
			}
			if len(links) != 1 || links[0].Target.Value != "Marrowfield" {
				t.Errorf("want the link at Marrowfield, got %+v\n%s", links, got)
			}
			for _, want := range []string{"# where it sat", "role: child\n"} {
				if !strings.Contains(got, want) {
					t.Errorf("lost %q from\n%s", want, got)
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
		content string
		target  string
	}{
		{"a doubled quote", `'Alice''s key'`, "Alice's key"},
		{"an escaped quote", `"Alice\"s key"`, `Alice"s key`},
		{"a hash that is not a comment", `"Shed #LXVI"`, "Shed #LXVI"},
	} {
		t.Run(one.name, func(t *testing.T) {
			raw := "---\nlinks:\n  - to: " + one.content + "\n    role: child\n---\nbody\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			moved, err := d.RedirectLinks(domain.ParseAddress(one.target), "Marrowfield")
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
			links, err := readLinks(after)
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
func TestARenamePreservesABlockScalarAlone(t *testing.T) {
	raw := "---\nlinks:\n  - to: >-\n      The far corner\n    role: child\n---\nbody\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	moved, err := d.RedirectLinks(domain.ParseAddress("The far corner"), "Marrowfield")
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

// How the user writes their own links is preserved: the quotes around the name
// and the brackets inside them stay as written, and a rename changes the name.
func TestARenamePreservesTheNotationItFound(t *testing.T) {
	for _, one := range []struct {
		name  string
		input string
		wants string
	}{
		{"double stays double", `"[[The far corner]]"`, `to: "[[Marrowfield]]"`},
		{"single stays single", `'[[The far corner]]'`, `to: '[[Marrowfield]]'`},
		{"an alias is left standing", `"[[The far corner|him]]"`, `to: "[[Marrowfield|him]]"`},
		{"a place inside is left standing", `"[[The far corner#gate]]"`, `to: "[[Marrowfield#gate]]"`},
		{"a quoted path is repaired by name", `"notes/The far corner"`, `to: "Marrowfield"`},
		{"plainly written stays plain", `The far corner`, `to: Marrowfield`},
	} {
		t.Run(one.name, func(t *testing.T) {
			raw := "---\nlinks:\n  - to: " + one.input + "\n    role: child\n---\nbody\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			target := strings.Trim(one.input, `"'`)
			if _, err := d.RedirectLinks(domain.ParseAddress(target), "Marrowfield"); err != nil {
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
func TestALinkSavedAgainInAnotherRoleIsRelatedAgain(t *testing.T) {
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

// A frontmatter indented from the margin is spliced where its own keys are.
// A line written flush ends the mapping, and every key below it becomes text
// the YAML decoder drops silently, the user's own keys among them.
func TestAKeyIsSavedWhereTheBlocksOwnKeysExist(t *testing.T) {
	raw := "---\n" +
		"  type: preset\n" +
		"  id: 01J8\n" +
		"  minutes_a_day: 20\n" +
		"  color: green\n" +
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
	front := Parse(domain.FileStat{Path: "Sanskrit.md"}, []byte(got)).Frontmatter
	want := map[string]any{
		"type": "preset", "id": "01J8", "minutes_a_day": 35, "new_a_day": 8,
		"color": "green",
	}
	if !reflect.DeepEqual(front, want) {
		t.Errorf("frontmatter = %v, want %v\n%s", front, want, got)
	}
}

// A frontmatter carrying an anchor is left alone. Replacing the value the anchor
// relies on leaves the alias pointing at nothing, and the note stops opening at
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

// A comment beside a key belongs to the user, on a key the application owns as
// much as on any other. Every writer preserves it.
func TestAWritePreservesTheCommentAdjacentTheKey(t *testing.T) {
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
	for _, want := range []string{
		"id: 01J9 # the one it was made with\n",
		"title: New # what I called it\n",
		"minutes_a_day: 35 # twenty is plenty\n",
		"load: # the week\n",
		"sat: 20 # half a Saturday\n",
		"tags: # what it is about\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("want %q in\n%s", want, got)
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
func TestAnUnnamedTypeIsSavedIn(t *testing.T) {
	for name, raw := range map[string]string{
		"no block":       "---\ntype: sample\n---\nbody\n",
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

			n := Parse(domain.FileStat{Path: "samples/Roots.md"}, d.Bytes())
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
func TestNamingNoPlaceBelowATypeConsumesTheEntryOut(t *testing.T) {
	d, err := Open([]byte(
		"---\ntype: sample\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetLinkOfType("preset", domain.Address{}, domain.RoleRef); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := string(d.Bytes()); got != "---\ntype: sample\n---\nbody\n" {
		t.Errorf("got %q", got)
	}
}

// An entry carrying a key the application does not own is left as the user
// wrote it, and nothing is written.
func TestAnEntryOfTheTypeCarryingSomebodyElsesKeyIsRefused(t *testing.T) {
	raw := "---\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n    mine: keep me\n---\nbody\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	err = d.SetLinkOfType(
		"preset", domain.Address{Scheme: domain.SchemeName, Value: "Slow going"}, domain.RoleRef)
	if !errors.Is(err, ErrNotOwned) {
		t.Fatalf("want ErrNotOwned, got %v", err)
	}
	if got := string(d.Bytes()); got != raw {
		t.Errorf("the rejected change was made anyway\n%s", got)
	}
}

// An entry written with no role is not a link, so it is empty under the
// type and is left exactly currently.
func TestAnEntryWithNoRoleIsRemainingWhereItExists(t *testing.T) {
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

type failingYAMLMarshaler struct{}

func (failingYAMLMarshaler) MarshalYAML() (any, error) {
	return nil, errors.New("marshal failed")
}

func TestFrontmatterInternalHelpersAndBoundaries(t *testing.T) {
	nested := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "outer"},
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "inner"},
					{Kind: yaml.ScalarNode, Value: "val"},
				},
			},
		},
	}
	rendered, err := render(nested)
	if err != nil || rendered != "outer:\n  inner: val\n" {
		t.Fatalf("render(nested) = (%q, %v), want %q", rendered, err, "outer:\n  inner: val\n")
	}

	if got := pair(&yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{{Value: "k"}, {Value: "v"}}}, "k"); got.name != nil || got.value != nil {
		t.Fatalf("pair on SequenceNode = %+v, want empty", got)
	}
	if got := pair(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Value: "k"}}}, "k"); got.name != nil {
		t.Fatalf("pair on odd Content = %+v, want empty", got)
	}
	pairMap := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "k1"},
			{Kind: yaml.ScalarNode, Value: "k2"},
			{Kind: yaml.ScalarNode, Value: "k2"},
			{Kind: yaml.ScalarNode, Value: "v2"},
		},
	}
	if got := pair(pairMap, "k2"); got.value == nil || got.value.Value != "v2" {
		t.Fatalf("pair(pairMap, k2) = %+v, want v2", got)
	}

	if got := valueOf(&yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{{Value: "k"}, {Value: "v"}}}, "k"); got != nil {
		t.Fatalf("valueOf on SequenceNode = %+v, want nil", got)
	}
	if got := valueOf(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Value: "k"}}}, "k"); got != nil {
		t.Fatalf("valueOf on odd Content = %+v, want nil", got)
	}
	if got := valueOf(pairMap, "k2"); got == nil || got.Value != "v2" {
		t.Fatalf("valueOf(pairMap, k2) = %+v, want v2", got)
	}

	anchored, err := Open([]byte("---\nid: &a 01J8\n---\n"))
	if err != nil {
		t.Fatalf("open anchored: %v", err)
	}
	if err := anchored.SetScalar("id", ""); !errors.Is(err, ErrAnchored) {
		t.Fatalf("SetScalar empty on anchored = %v, want ErrAnchored", err)
	}

	noFront, err := Open([]byte("# Body\n"))
	if err != nil {
		t.Fatalf("open noFront: %v", err)
	}
	if err := noFront.SetScalar("missing", ""); err != nil {
		t.Fatalf("SetScalar empty on noFront: %v", err)
	}
	if got := string(noFront.Bytes()); got != "# Body\n" {
		t.Fatalf("noFront changed to %q", got)
	}

	oneByteRenderDoc := &Document{eol: "\n"}
	if err := oneByteRenderDoc.appendKey(nil, []byte("\n")); err != nil || string(oneByteRenderDoc.open) != "---\n" {
		t.Fatalf("appendKey 1-byte = %v, open = %q", err, oneByteRenderDoc.open)
	}

	oneByteFrontDoc := &Document{open: []byte("---\n"), front: []byte("#"), closeDelimiter: []byte("---\n"), eol: "\n"}
	if err := oneByteFrontDoc.SetIdentifier("01J8"); err != nil {
		t.Fatalf("SetIdentifier on 1-byte comment front: %v", err)
	}
	if id, ok := oneByteFrontDoc.Identifier(); !ok || id != "01J8" {
		t.Fatalf("Identifier after 1-byte comment front = (%q, %v)", id, ok)
	}

	spacedOpenDoc := &Document{open: []byte("---\n"), closeDelimiter: []byte("---\n"), eol: "\r\n"}
	if err := spacedOpenDoc.SetIdentifier("01J8"); err != nil {
		t.Fatalf("SetIdentifier on spacedOpenDoc: %v", err)
	}
	if string(spacedOpenDoc.open) != "---\n" {
		t.Fatalf("spacedOpenDoc overwrote existing open delimiter: %q", spacedOpenDoc.open)
	}

	existingNoOpen := &Document{front: []byte("a: 1\n"), eol: "\n"}
	existingNode, err := existingNoOpen.readMapping()
	if err != nil || existingNode == nil {
		t.Fatalf("readMapping: %v", err)
	}
	if err := existingNoOpen.appendKey(existingNode, []byte("b: 2\n")); err != nil {
		t.Fatalf("appendKey existingNode: %v", err)
	}
	if len(existingNoOpen.open) != 0 {
		t.Fatalf("existingNoOpen grew open delimiter %q", existingNoOpen.open)
	}

	indentDoc := &Document{front: []byte("  a: 1\n")}
	if got := indentDoc.indent(&yaml.Node{Kind: yaml.MappingNode}); got != "" {
		t.Fatalf("indent empty Content = %q, want empty", got)
	}
	if got := indentDoc.indent(&yaml.Node{Content: []*yaml.Node{{Line: 1}}}); got != "  " {
		t.Fatalf("indent 1-item Content = %q, want 2 spaces", got)
	}
	if got := indentDoc.indent(&yaml.Node{Content: []*yaml.Node{{Line: 0}}}); got != "" {
		t.Fatalf("indent Line 0 = %q, want empty", got)
	}
	if got := indentDoc.indent(&yaml.Node{Content: []*yaml.Node{{Line: 2}}}); got != "" {
		t.Fatalf("indent Line==len(lines) = %q, want empty", got)
	}

	if got := string(indentLines([]byte("x"), "  ")); got != "  x" {
		t.Fatalf("indentLines 1-byte = %q, want %q", got, "  x")
	}
	if got := string(indentLines([]byte("\na\n \nb"), "  ")); got != "\n  a\n \n  b" {
		t.Fatalf("indentLines multi = %q, want %q", got, "\n  a\n \n  b")
	}
}

func TestSpanAndEndLineBoundaries(t *testing.T) {
	d := &Document{front: []byte("a: 1\nb: 2\n")}
	if s, e, ok := d.span(nil, "a"); s != 0 || e != 0 || ok {
		t.Fatalf("span(nil) = (%d, %d, %v)", s, e, ok)
	}
	if s, e, ok := d.span(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Value: "a"}}}, "a"); s != 0 || e != 0 || ok {
		t.Fatalf("span(odd Content) = (%d, %d, %v)", s, e, ok)
	}
	if s, e, ok := d.span(
		&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Value: "a", Line: 1, Column: 1}, {Value: "1", Line: 1}}},
		"missing",
	); s != 0 || e != 0 ||
		ok {
		t.Fatalf("span(missing) = (%d, %d, %v)", s, e, ok)
	}

	dupSpanNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Value: "x", Line: 1, Column: 1},
			{Value: "k", Line: 1, Column: 4},
			{Value: "k", Line: 1, Column: 1},
			{Value: "1", Line: 1, Column: 4},
			{Value: "k", Line: 2, Column: 1},
			{Value: "2", Line: 2, Column: 4},
		},
	}
	if s, e, ok := d.span(dupSpanNode, "k"); !ok || s != 0 || e != 5 {
		t.Fatalf("span(dupSpanNode, k) = (%d, %d, %v), want (0, 5, true)", s, e, ok)
	}

	if s, e, ok := d.span(&yaml.Node{Content: []*yaml.Node{{Value: "a", Line: 0}, {Value: "1", Line: 1}}}, "a"); s != 0 || e != 0 || ok {
		t.Fatalf("span(keyLine 0) = (%d, %d, %v)", s, e, ok)
	}
	if s, e, ok := d.span(&yaml.Node{Content: []*yaml.Node{{Value: "a", Line: 3}, {Value: "1", Line: 3}}}, "a"); s != 0 || e != 0 || ok {
		t.Fatalf("span(keyLine>=len(lines)) = (%d, %d, %v)", s, e, ok)
	}

	seqDoc, err := Open([]byte("---\ntags:\n- a\n- b\n---\n"))
	if err != nil {
		t.Fatalf("open seqDoc: %v", err)
	}
	seqMap, err := seqDoc.readMapping()
	if err != nil {
		t.Fatalf("readMapping seqDoc: %v", err)
	}
	if s, e, ok := seqDoc.span(seqMap, "tags"); !ok || s != 0 || e != len(seqDoc.front) {
		t.Fatalf("span(seqDoc, tags) = (%d, %d, %v), want (0, %d, true)", s, e, ok, len(seqDoc.front))
	}

	zeroValNode := &yaml.Node{
		Content: []*yaml.Node{
			{Value: "a", Line: 1, Column: 1},
			{Kind: yaml.ScalarNode, Line: 0},
		},
	}
	if s, e, ok := d.span(zeroValNode, "a"); !ok || s != 0 || e != 5 {
		t.Fatalf("span(zeroValNode) = (%d, %d, %v), want (0, 5, true)", s, e, ok)
	}

	clampNextNode := &yaml.Node{
		Content: []*yaml.Node{
			{Value: "a", Line: 1, Column: 1},
			{Kind: yaml.MappingNode, Line: 5},
			{Value: "b", Line: 2, Column: 1},
			{Kind: yaml.ScalarNode, Line: 2},
		},
	}
	if s, e, ok := d.span(clampNextNode, "a"); !ok || s != 0 || e != 5 {
		t.Fatalf("span(clampNextNode) = (%d, %d, %v), want (0, 5, true)", s, e, ok)
	}

	withCommentDoc := &Document{front: []byte("a: 1\n# comment\nb: 2\n")}
	withCommentNode := &yaml.Node{
		Content: []*yaml.Node{
			{Value: "a", Line: 1, Column: 1},
			{Kind: yaml.ScalarNode, Line: 1, Column: 4},
			{Value: "b", Line: 3, Column: 1},
			{Kind: yaml.ScalarNode, Line: 3, Column: 4},
		},
	}
	if s, e, ok := withCommentDoc.span(withCommentNode, "a"); !ok || s != 0 || e != 5 {
		t.Fatalf("span(withCommentNode) = (%d, %d, %v), want (0, 5, true)", s, e, ok)
	}

	clampLinesNode := &yaml.Node{
		Content: []*yaml.Node{
			{Value: "a", Line: 1, Column: 1},
			{Kind: yaml.MappingNode, Line: 10},
		},
	}
	if s, e, ok := d.span(clampLinesNode, "a"); !ok || s != 0 || e != 10 {
		t.Fatalf("span(clampLinesNode) = (%d, %d, %v), want (0, 10, true)", s, e, ok)
	}

	lines := lineOffsets(d.front)
	if got := d.endLine(lines, nil, 1); got != 0 {
		t.Fatalf("endLine(nil) = %d, want 0", got)
	}
	if got := d.endLine(lines, &yaml.Node{Line: 0}, 1); got != 0 {
		t.Fatalf("endLine(Line 0) = %d, want 0", got)
	}

	oddMap := &yaml.Node{
		Kind: yaml.MappingNode,
		Line: 1,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Line: 1, Column: 1},
		},
	}
	if got := d.endLineOfMapping(lines, oddMap); got != 1 {
		t.Fatalf("endLineOfMapping(oddMap) = %d, want 1", got)
	}
	evenMap := &yaml.Node{
		Kind: yaml.MappingNode,
		Line: 1,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Line: 1, Column: 1},
			{Kind: yaml.MappingNode, Line: 4},
			{Kind: yaml.ScalarNode, Line: 2, Column: 1},
			{Kind: yaml.MappingNode, Line: 3},
		},
	}
	if got := d.endLineOfMapping(lines, evenMap); got != 4 {
		t.Fatalf("endLineOfMapping(evenMap) = %d, want 4", got)
	}

	scalarDoc := &Document{front: []byte("a: |\n  x\n\n b\nc: 2\n  d: 3\n")}
	scalarLines := lineOffsets(scalarDoc.front)
	if got := scalarDoc.endLineOfScalar(scalarLines, &yaml.Node{Kind: yaml.ScalarNode, Line: 1}, 1); got != 4 {
		t.Fatalf("endLineOfScalar = %d, want 4", got)
	}
	twoLineDoc := &Document{front: []byte("a: |\n  x\n")}
	twoLines := lineOffsets(twoLineDoc.front)
	if got := twoLineDoc.endLineOfScalar(twoLines, &yaml.Node{Kind: yaml.ScalarNode, Line: 1}, 1); got != 2 {
		t.Fatalf("endLineOfScalar(twoLineDoc) = %d, want 2", got)
	}
}

func TestReadMappingAndNodePredicates(t *testing.T) {
	d := &Document{front: []byte("a: 1\n")}
	n1, err := d.readMapping()
	if err != nil || n1 == nil {
		t.Fatalf("readMapping 1: %v", err)
	}
	n2, err := d.readMapping()
	if err != nil || n1 != n2 {
		t.Fatalf("readMapping did not return cached pointer: %p vs %p", n1, n2)
	}

	oneCharInvalid := &Document{front: []byte("[")}
	if _, err := oneCharInvalid.readMapping(); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("readMapping([) = %v, want ErrUnreadable", err)
	}

	notMapping := &Document{front: []byte("- item\n")}
	if _, err := notMapping.readMapping(); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("readMapping(- item) = %v, want ErrUnreadable", err)
	}

	badWritable := &Document{front: []byte(": bad")}
	if n, err := badWritable.getWritableNode(); n != nil || !errors.Is(err, ErrUnreadable) {
		t.Fatalf("getWritableNode(bad) = (%v, %v), want (nil, ErrUnreadable)", n, err)
	}

	if hasAnchor(nil) {
		t.Fatal("hasAnchor(nil) = true")
	}
	if !hasAnchor(&yaml.Node{Kind: yaml.ScalarNode, Anchor: "a"}) {
		t.Fatal("hasAnchor(Anchor) = false")
	}
	if !hasAnchor(&yaml.Node{Kind: yaml.AliasNode}) {
		t.Fatal("hasAnchor(AliasNode) = false")
	}

	if isFlowCollection(nil) {
		t.Fatal("isFlowCollection(nil) = true")
	}
	if isFlowCollection(&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.FlowStyle}) {
		t.Fatal("isFlowCollection(ScalarNode) = true")
	}
	if !isFlowCollection(&yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}) {
		t.Fatal("isFlowCollection(Flow SequenceNode) = false")
	}

	if isNullValue(&yaml.Node{Kind: yaml.MappingNode, Tag: "!!null", Value: ""}) {
		t.Fatal("isNullValue(MappingNode) = true")
	}
	if isNullValue(&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ""}) {
		t.Fatal("isNullValue(!!str) = true")
	}

	flowChildTree := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode},
			{Kind: yaml.MappingNode, Style: yaml.FlowStyle},
		},
	}
	if !hasFlowStyle(flowChildTree) {
		t.Fatal("hasFlowStyle(flowChildTree) = false")
	}
}

func TestKeysMethodsAndBoundaries(t *testing.T) {
	emptyDoc := &Document{}
	if id, ok := emptyDoc.Identifier(); id != "" || ok {
		t.Fatalf("emptyDoc.Identifier() = (%q, %v)", id, ok)
	}
	if title, ok := emptyDoc.Title(); title != "" || ok {
		t.Fatalf("emptyDoc.Title() = (%q, %v)", title, ok)
	}
	if list, ok := emptyDoc.List("tags"); list != nil || ok {
		t.Fatalf("emptyDoc.List() = (%v, %v)", list, ok)
	}

	badDoc := &Document{front: []byte(": bad")}
	if id, ok := badDoc.Identifier(); id != "" || ok {
		t.Fatalf("badDoc.Identifier() = (%q, %v)", id, ok)
	}
	if title, ok := badDoc.Title(); title != "" || ok {
		t.Fatalf("badDoc.Title() = (%q, %v)", title, ok)
	}
	if list, ok := badDoc.List("tags"); list != nil || ok {
		t.Fatalf("badDoc.List() = (%v, %v)", list, ok)
	}
	if names, ok := badDoc.EntryNames("load"); names != nil || ok {
		t.Fatalf("badDoc.EntryNames() = (%v, %v)", names, ok)
	}

	oddMappingDoc := &Document{
		mapping: &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "id"}},
		},
	}
	if id, ok := oddMappingDoc.Identifier(); id != "" || ok {
		t.Fatalf("oddMappingDoc.Identifier() = (%q, %v)", id, ok)
	}
	if title, ok := oddMappingDoc.Title(); title != "" || ok {
		t.Fatalf("oddMappingDoc.Title() = (%q, %v)", title, ok)
	}

	valCollisionDoc, err := Open([]byte("---\nother: id\nid: real-id\nother2: title\ntitle: real-title\n---\n"))
	if err != nil {
		t.Fatalf("Open valCollisionDoc: %v", err)
	}
	if id, ok := valCollisionDoc.Identifier(); !ok || id != "real-id" {
		t.Fatalf("valCollisionDoc.Identifier() = (%q, %v), want real-id", id, ok)
	}
	if title, ok := valCollisionDoc.Title(); !ok || title != "real-title" {
		t.Fatalf("valCollisionDoc.Title() = (%q, %v), want real-title", title, ok)
	}

	blankTitleDoc, err := Open([]byte("---\ntitle: \"   \"\n---\n"))
	if err != nil {
		t.Fatalf("Open blankTitleDoc: %v", err)
	}
	if title, ok := blankTitleDoc.Title(); title != "" || ok {
		t.Fatalf("blankTitleDoc.Title() = (%q, %v), want empty/false", title, ok)
	}

	mixedListDoc := &Document{
		mapping: &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "tags"},
				{
					Kind: yaml.SequenceNode,
					Content: []*yaml.Node{
						{Kind: yaml.MappingNode, Tag: "!!str", Value: "skip"},
						{Kind: yaml.ScalarNode, Tag: "!!str", Value: "keep"},
					},
				},
			},
		},
	}
	if list, ok := mixedListDoc.List("tags"); !ok || !reflect.DeepEqual(list, []string{"keep"}) {
		t.Fatalf("mixedListDoc.List() = (%v, %v), want [keep]", list, ok)
	}

	if got := emptyDoc.entries(nil, "load"); got == nil || got.Kind != yaml.MappingNode {
		t.Fatalf("entries(nil) = %+v, want MappingNode", got)
	}
	nullEntryDoc, err := Open([]byte("---\nload:\nscalar: 1\n---\n"))
	if err != nil {
		t.Fatalf("Open nullEntryDoc: %v", err)
	}
	if names, ok := nullEntryDoc.EntryNames("load"); !ok || len(names) != 0 {
		t.Fatalf("EntryNames(null load) = (%v, %v), want empty/true", names, ok)
	}
	if names, ok := nullEntryDoc.EntryNames("missing"); !ok || len(names) != 0 {
		t.Fatalf("EntryNames(missing) = (%v, %v), want empty/true", names, ok)
	}
	if names, ok := nullEntryDoc.EntryNames("scalar"); ok || names != nil {
		t.Fatalf("EntryNames(scalar) = (%v, %v), want nil/false", names, ok)
	}

	badUnterminated := &Document{isUnterminated: true, front: []byte(": bad")}
	if err := badUnterminated.SetMapping("load", []Entry{{Key: "a", Value: 1}}); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("SetMapping on badUnterminated = %v, want ErrUnreadable", err)
	}
	if err := nullEntryDoc.SetMapping("load", []Entry{{Key: "a", Value: failingYAMLMarshaler{}}}); err == nil {
		t.Fatal("SetMapping with failingYAMLMarshaler succeeded, want error")
	}
	if err := nullEntryDoc.SetValue("load", failingYAMLMarshaler{}); err == nil {
		t.Fatal("SetValue with failingYAMLMarshaler succeeded, want error")
	}

	verbatimDoc, err := Open([]byte("---\nid: 01J8\nload:\n  sat: 50\n---\n"))
	if err != nil {
		t.Fatalf("Open verbatimDoc: %v", err)
	}
	if err := verbatimDoc.SetMapping("load", []Entry{{Key: "missing", IsVerbatim: true}, {Key: "sat", IsVerbatim: true}}); err != nil {
		t.Fatalf("SetMapping partial verbatim: %v", err)
	}
	if names, ok := verbatimDoc.EntryNames("load"); !ok || !reflect.DeepEqual(names, []string{"sat"}) {
		t.Fatalf("EntryNames after partial verbatim = (%v, %v)", names, ok)
	}
	if err := verbatimDoc.SetMapping("load", []Entry{{Key: "missing", IsVerbatim: true}}); err != nil {
		t.Fatalf("SetMapping all missing verbatim: %v", err)
	}
	if strings.Contains(string(verbatimDoc.Bytes()), "load:") {
		t.Fatalf("all-missing verbatim did not remove key:\n%s", verbatimDoc.Bytes())
	}

	if got := emptyDoc.getSpelling("tags"); got == nil || len(got) != 0 {
		t.Fatalf("emptyDoc.getSpelling = %v, want non-nil empty map", got)
	}
	if got := badDoc.getSpelling("tags"); got == nil || len(got) != 0 {
		t.Fatalf("badDoc.getSpelling = %v, want non-nil empty map", got)
	}
	if got := nullEntryDoc.getSpelling("scalar"); got == nil || len(got) != 0 {
		t.Fatalf("nullEntryDoc.getSpelling(scalar) = %v, want non-nil empty map", got)
	}
}

func TestLinkblockHelpersAndBoundaries(t *testing.T) {
	anchored, err := Open([]byte("---\nid: &a 01J8\n---\n"))
	if err != nil {
		t.Fatalf("Open anchored: %v", err)
	}
	if _, err := anchored.block(); !errors.Is(err, ErrAnchored) {
		t.Fatalf("anchored.block() = %v, want ErrAnchored", err)
	}
	if err := anchored.SetLink(domain.Link{Target: domain.ParseAddress("A"), Role: domain.RoleJump}); !errors.Is(err, ErrAnchored) {
		t.Fatalf("anchored.SetLink() = %v, want ErrAnchored", err)
	}
	if _, err := anchored.RemoveLink(domain.LinkKey{Target: domain.ParseAddress("A")}); !errors.Is(err, ErrAnchored) {
		t.Fatalf("anchored.RemoveLink() = %v, want ErrAnchored", err)
	}
	if err := anchored.SetLinkOfType("preset", domain.ParseAddress("A"), domain.RoleRef); !errors.Is(err, ErrAnchored) {
		t.Fatalf("anchored.SetLinkOfType() = %v, want ErrAnchored", err)
	}
	if _, err := anchored.UpdateLink(domain.ParseAddress("A"), domain.Link{Role: domain.RoleJump}); !errors.Is(err, ErrAnchored) {
		t.Fatalf("anchored.UpdateLink() = %v, want ErrAnchored", err)
	}
	if _, err := anchored.RedirectLinks(domain.ParseAddress("A"), "B"); !errors.Is(err, ErrAnchored) {
		t.Fatalf("anchored.RedirectLinks() = %v, want ErrAnchored", err)
	}

	noLinksDoc, err := Open([]byte("---\nid: 01J8\n---\n"))
	if err != nil {
		t.Fatalf("Open noLinksDoc: %v", err)
	}
	b, err := noLinksDoc.block()
	if err != nil || b.isFound || b.indent != "  " {
		t.Fatalf("noLinksDoc.block() = (%+v, %v)", b, err)
	}

	nullLinksDoc, err := Open([]byte("---\nlinks:\n---\n"))
	if err != nil {
		t.Fatalf("Open nullLinksDoc: %v", err)
	}
	b, err = nullLinksDoc.block()
	if err != nil || !b.isFound || b.indent != "  " {
		t.Fatalf("nullLinksDoc.block() = (%+v, %v)", b, err)
	}
	if n, err := nullLinksDoc.RemoveLink(
		domain.LinkKey{Target: domain.ParseAddress("A")},
	); err != nil || n != 0 ||
		!strings.Contains(string(nullLinksDoc.Bytes()), "links:") {
		t.Fatalf("nullLinksDoc.RemoveLink = (%d, %v), bytes = %q", n, err, nullLinksDoc.Bytes())
	}
	if err := nullLinksDoc.SetLinkOfType("preset", domain.Address{}, domain.RoleRef); err != nil || !strings.Contains(string(nullLinksDoc.Bytes()), "links:") {
		t.Fatalf("nullLinksDoc.SetLinkOfType empty = %v, bytes = %q", err, nullLinksDoc.Bytes())
	}

	nilItemsDoc := &Document{
		front: []byte("links:\n"),
		mapping: &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "links", Line: 1, Column: 1},
				nil,
			},
		},
	}
	if b, err := nilItemsDoc.block(); err != nil || !b.isFound || b.indent != "  " {
		t.Fatalf("nilItemsDoc.block() = (%+v, %v)", b, err)
	}

	emptySeqDoc := &Document{
		front: []byte("links:\n"),
		mapping: &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "links", Line: 1, Column: 1},
				{Kind: yaml.SequenceNode, Line: 1, Column: 1},
			},
		},
	}
	if b, err := emptySeqDoc.block(); err != nil || b.indent != "  " {
		t.Fatalf("emptySeqDoc.block() = (%+v, %v)", b, err)
	}

	entriesDoc := &Document{front: []byte("  - to: A\n    role: jump\n  - to: B\n    role: parent\n")}
	seqItems := &yaml.Node{
		Kind:   yaml.SequenceNode,
		Column: 3,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Line: 1,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "to", Line: 1, Column: 5},
					{Kind: yaml.ScalarNode, Value: "A", Line: 4, Column: 9},
				},
			},
			{
				Kind: yaml.MappingNode,
				Line: 3,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "to", Line: 3, Column: 5},
					{Kind: yaml.ScalarNode, Value: "B", Line: 1, Column: 9},
				},
			},
			{Kind: yaml.MappingNode, Line: 0},
			{Kind: yaml.MappingNode, Line: 6},
		},
	}
	gotEntries, gotIndent := entriesDoc.readBlockEntries(seqItems, 25)
	if len(gotEntries) != 2 || gotIndent != "  " {
		t.Fatalf("readBlockEntries = (%d, %q), want (2, \"  \")", len(gotEntries), gotIndent)
	}
	if gotEntries[0].start != 0 || gotEntries[0].end != 25 {
		t.Fatalf("gotEntries[0] span = (%d, %d), want (0, 25)", gotEntries[0].start, gotEntries[0].end)
	}
	if gotEntries[1].start != 25 || gotEntries[1].end != 35 {
		t.Fatalf("gotEntries[1] span = (%d, %d), want (25, 35)", gotEntries[1].start, gotEntries[1].end)
	}

	addDoc, err := Open(
		[]byte(
			"---\nlinks:\n  - to: A\n    role: ref\n    type: other\n  - to: A # keep comment\n    role: parent\n    type: target\n    note: initial\n---\n",
		),
	)
	if err != nil {
		t.Fatalf("Open addDoc: %v", err)
	}
	if err := addDoc.AddLink(domain.Link{
		Target: domain.ParseAddress("A"),
		Role:   domain.RoleParent,
		Type:   "target",
	}); err != nil {
		t.Fatalf("AddLink unchanged: %v", err)
	}
	if !strings.Contains(string(addDoc.Bytes()), "# keep comment") {
		t.Fatalf("AddLink unchanged rewrote entry:\n%s", addDoc.Bytes())
	}
	if err := addDoc.AddLink(domain.Link{
		Target: domain.ParseAddress("A"),
		Role:   domain.RoleChild,
		Type:   "target",
		Marker: domain.MarkerArrow,
		Why:    "updated",
	}); err != nil {
		t.Fatalf("AddLink updated: %v", err)
	}
	gotAdd := string(addDoc.Bytes())
	if !strings.Contains(gotAdd, "marker: arrow") || !strings.Contains(gotAdd, "note: updated") || !strings.Contains(gotAdd, "type: other") {
		t.Fatalf("AddLink updated output:\n%s", gotAdd)
	}

	setLinkDoc, err := Open([]byte("---\nlinks:\n  - to: Other\n    role: jump\n  - to: OldA\n    role: parent\n  - to: NewA\n    role: child\n---\n"))
	if err != nil {
		t.Fatalf("Open setLinkDoc: %v", err)
	}
	if err := setLinkDoc.SetLink(
		domain.Link{Target: domain.ParseAddress("NewA"), Role: domain.RoleRef},
		domain.ParseAddress("OldA"),
	); err != nil {
		t.Fatalf("SetLink: %v", err)
	}
	gotSetLink := string(setLinkDoc.Bytes())
	if !strings.Contains(gotSetLink, "to: OldA\n    role: ref") || strings.Contains(gotSetLink, "to: NewA") {
		t.Fatalf("SetLink did not deduplicate to first matching target:\n%s", gotSetLink)
	}

	unownedSetDoc, err := Open([]byte("---\nlinks:\n  - to: A\n    role: parent\n    custom: 1\n---\n"))
	if err != nil {
		t.Fatalf("Open unownedSetDoc: %v", err)
	}
	if err := unownedSetDoc.SetLink(domain.Link{Target: domain.ParseAddress("A"), Role: domain.RoleRef}); !errors.Is(err, ErrNotOwned) {
		t.Fatalf("SetLink on unowned = %v, want ErrNotOwned", err)
	}

	removeRoleDoc, err := Open([]byte("---\nlinks:\n  - to: A\n    role: parent\n  - to: B\n    role: child\n  - to: A\n    role: child\n---\n"))
	if err != nil {
		t.Fatalf("Open removeRoleDoc: %v", err)
	}
	removedCount, err := removeRoleDoc.RemoveLink(domain.LinkKey{Target: domain.ParseAddress("A"), Role: domain.RoleChild})
	if err != nil || removedCount != 1 {
		t.Fatalf("RemoveLink by role = (%d, %v), want 1", removedCount, err)
	}
	if !strings.Contains(string(removeRoleDoc.Bytes()), "role: parent") || !strings.Contains(string(removeRoleDoc.Bytes()), "to: B") {
		t.Fatalf("RemoveLink removed wrong entry:\n%s", removeRoleDoc.Bytes())
	}

	carrying, remCount, err := findTypedEntries([]entry{
		{link: domain.Link{Type: "other"}, isReadable: true, isOwned: true},
		{link: domain.Link{Type: "preset"}, isReadable: true, isOwned: true},
	}, "preset")
	if err != nil || !reflect.DeepEqual(carrying, []int{1}) || remCount != 1 {
		t.Fatalf("findTypedEntries = (%v, %d, %v), want ([1], 1, nil)", carrying, remCount, err)
	}

	multiPresetDoc, err := Open([]byte("---\nlinks:\n  - to: P1\n    role: ref\n    type: preset\n  - to: P2\n    role: ref\n    type: preset\n---\n"))
	if err != nil {
		t.Fatalf("Open multiPresetDoc: %v", err)
	}
	if err := multiPresetDoc.SetLinkOfType("preset", domain.ParseAddress("P3"), domain.RoleRef); err != nil {
		t.Fatalf("SetLinkOfType multiPresetDoc: %v", err)
	}
	if strings.Count(string(multiPresetDoc.Bytes()), "type: preset") != 1 || !strings.Contains(string(multiPresetDoc.Bytes()), "to: P3") {
		t.Fatalf("SetLinkOfType did not collapse multiple entries:\n%s", multiPresetDoc.Bytes())
	}

	origLink := domain.Link{
		Target: domain.ParseAddress("A"),
		Role:   domain.RoleParent,
		Type:   "t1",
		Why:    "w1",
		Label:  "l1",
		Marker: domain.MarkerArrow,
	}
	if got := applyLinkChange(origLink, domain.Link{}); got != origLink {
		t.Fatalf("applyLinkChange empty = %+v, want %+v", got, origLink)
	}
	changedLink := domain.Link{
		Role:   domain.RoleChild,
		Type:   "t2",
		Why:    "w2",
		Label:  "l2",
		Marker: domain.LinkMarker("custom"),
	}
	if got := applyLinkChange(
		origLink,
		changedLink,
	); got.Role != domain.RoleChild || got.Type != "t2" || got.Why != "w2" || got.Label != "l2" ||
		got.Marker != domain.LinkMarker("custom") {
		t.Fatalf("applyLinkChange full = %+v", got)
	}

	updateOrderDoc, err := Open([]byte("---\nlinks:\n  - to: Target\n    role: parent\n  - to: Other\n    role: jump\n---\n"))
	if err != nil {
		t.Fatalf("Open updateOrderDoc: %v", err)
	}
	if n, err := updateOrderDoc.UpdateLink(domain.ParseAddress("Target"), domain.Link{Role: domain.RoleChild}); err != nil || n != 1 {
		t.Fatalf("UpdateLink = (%d, %v), want 1", n, err)
	}

	redirectOrderDoc, err := Open(
		[]byte(
			"---\nlinks:\n  - to: Target\n    role: parent\n  - to: Other\n    role: jump\n  - to: Target\n    role: unknown_role\n  - to: Target\n    role: parent\n    extra: 1\n---\n",
		),
	)
	if err != nil {
		t.Fatalf("Open redirectOrderDoc: %v", err)
	}
	if moved, err := redirectOrderDoc.RedirectLinks(domain.ParseAddress("Target"), "Moved"); err != nil || moved != 1 {
		t.Fatalf("RedirectLinks = (%d, %v), want 1", moved, err)
	}

	if got := linkOf(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Value: "label"},
			{Value: "to"},
			{Value: "to"},
			{Value: "Dest"},
			{Value: "orphan"},
		},
	}); got.Label != "to" || got.Target.Value != "Dest" {
		t.Fatalf("linkOf = %+v", got)
	}

	if isReadable(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Value: "role"},
			{Value: "parent"},
		},
	}) {
		t.Fatal("isReadable without target = true")
	}

	if hasOnlyOwnedKeys(&yaml.Node{Kind: yaml.ScalarNode}) {
		t.Fatal("hasOnlyOwnedKeys(ScalarNode) = true")
	}
	if !hasOnlyOwnedKeys(&yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{{Value: "to"}},
	}) {
		t.Fatal("hasOnlyOwnedKeys(odd Content) = false")
	}

	if got := keepLinkNotation("[[a", "B"); got != "B" {
		t.Fatalf("keepLinkNotation([[a) = %q, want B", got)
	}
	if got := keepLinkNotation("a]]", "B"); got != "B" {
		t.Fatalf("keepLinkNotation(a]]) = %q, want B", got)
	}
	if got := keepLinkNotation("[[]]", "B"); got != "[[B]]" {
		t.Fatalf("keepLinkNotation([[]]) = %q, want [[B]]", got)
	}
	if got := keepLinkNotation("[[old|alias]]", "B"); got != "[[B|alias]]" {
		t.Fatalf("keepLinkNotation([[old|alias]]) = %q, want [[B|alias]]", got)
	}
}

func TestScalarHelpersAndBoundaries(t *testing.T) {
	d := &Document{front: []byte("\"actual\"\n")}
	if s, e, ok := d.scalarSpan(&yaml.Node{Kind: yaml.MappingNode}); s != 0 || e != 0 || ok {
		t.Fatalf("scalarSpan(MappingNode) = (%d, %d, %v)", s, e, ok)
	}
	if s, e, ok := d.scalarSpan(&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.LiteralStyle, Line: 1, Column: 1}); s != 0 || e != 0 || ok {
		t.Fatalf("scalarSpan(LiteralStyle) = (%d, %d, %v)", s, e, ok)
	}
	if s, e, ok := d.scalarSpan(
		&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Line: 1, Column: 1, Value: "mismatch"},
	); s != 0 || e != 0 ||
		ok {
		t.Fatalf("scalarSpan(mismatch) = (%d, %d, %v)", s, e, ok)
	}

	if pos, ok := d.readScalarStart(&yaml.Node{Kind: yaml.SequenceNode}); pos != 0 || ok {
		t.Fatalf("readScalarStart(SequenceNode) = (%d, %v)", pos, ok)
	}
	if pos, ok := d.readScalarStart(&yaml.Node{Kind: yaml.ScalarNode, Line: 0, Column: 1}); pos != 0 || ok {
		t.Fatalf("readScalarStart(Line 0) = (%d, %v)", pos, ok)
	}
	if pos, ok := d.readScalarStart(&yaml.Node{Kind: yaml.ScalarNode, Line: 2, Column: 1}); pos != 0 || ok {
		t.Fatalf("readScalarStart(Line>=len) = (%d, %v)", pos, ok)
	}
	if pos, ok := d.readScalarStart(&yaml.Node{Kind: yaml.ScalarNode, Line: 1, Column: 0}); pos != 0 || ok {
		t.Fatalf("readScalarStart(Column 0) = (%d, %v)", pos, ok)
	}
	if pos, ok := d.readScalarStart(&yaml.Node{Kind: yaml.ScalarNode, Line: 1, Column: 1}); !ok || pos != 0 {
		t.Fatalf("readScalarStart(1, 1) = (%d, %v), want (0, true)", pos, ok)
	}

	noBreakDoc := &Document{front: []byte("abc")}
	if pos, ok := noBreakDoc.readScalarStart(&yaml.Node{Kind: yaml.ScalarNode, Line: 1, Column: 10}); pos != 0 || ok {
		t.Fatalf("readScalarStart(past end) = (%d, %v), want (0, false)", pos, ok)
	}

	if end, ok := noBreakDoc.findScalarEnd(&yaml.Node{Style: 0, Value: "abcd"}, 0); end != 0 || ok {
		t.Fatalf("findScalarEnd(past end) = (%d, %v)", end, ok)
	}
	if end, ok := noBreakDoc.findScalarEnd(&yaml.Node{Style: 0, Value: "abc"}, 0); !ok || end != 3 {
		t.Fatalf("findScalarEnd(exact end) = (%d, %v), want (3, true)", end, ok)
	}
	if end, ok := noBreakDoc.findScalarEnd(&yaml.Node{Style: 0, Value: "abd"}, 0); end != 0 || ok {
		t.Fatalf("findScalarEnd(mismatch) = (%d, %v)", end, ok)
	}
	if end, ok := d.findScalarEnd(&yaml.Node{Style: yaml.TaggedStyle, Value: "actual"}, 0); end != 0 || ok {
		t.Fatalf("findScalarEnd(TaggedStyle) = (%d, %v), want (0, false)", end, ok)
	}

	if isValidScalarSpan([]byte("[unclosed"), 0, 9, "") {
		t.Fatal("isValidScalarSpan(invalid YAML, empty want) = true")
	}
	if isValidScalarSpan([]byte("actual"), 0, 6, "want") {
		t.Fatal("isValidScalarSpan(mismatch) = true")
	}

	if canEscapeBackslash([]byte(`'\a'`), 1, '\'') {
		t.Fatal("canEscapeBackslash(single quote) = true")
	}
	if canEscapeBackslash([]byte(`"\`), 1, '"') {
		t.Fatal("canEscapeBackslash(at end) = true")
	}
	if !canEscapeBackslash([]byte(`"\"`), 1, '"') {
		t.Fatal("canEscapeBackslash(valid) = false")
	}
	if canEscapeBackslash([]byte("\"\\\n\""), 1, '"') {
		t.Fatal("canEscapeBackslash(before newline) = true")
	}

	if isEscapedSingleQuote([]byte(`"''"`), 1, '"') {
		t.Fatal("isEscapedSingleQuote(double quote) = true")
	}
	if isEscapedSingleQuote([]byte(`'a'`), 2, '\'') {
		t.Fatal("isEscapedSingleQuote(at end) = true")
	}
	if !isEscapedSingleQuote([]byte(`'''`), 1, '\'') {
		t.Fatal("isEscapedSingleQuote(valid) = false")
	}

	if end, ok := findQuotedEnd([]byte("abc"), 0, '"'); end != 0 || ok {
		t.Fatalf("findQuotedEnd(unquoted) = (%d, %v)", end, ok)
	}
	if end, ok := findQuotedEnd([]byte(`""`), 0, '"'); !ok || end != 2 {
		t.Fatalf("findQuotedEnd(empty quotes) = (%d, %v), want (2, true)", end, ok)
	}
	if end, ok := findQuotedEnd([]byte("\"a\nb\""), 0, '"'); end != 0 || ok {
		t.Fatalf("findQuotedEnd(multiline) = (%d, %v)", end, ok)
	}

	if got, err := scalarLike("true", 0); err != nil || got != `"true"` {
		t.Fatalf("scalarLike(true, 0) = (%q, %v), want \"true\"", got, err)
	}
	if got, err := scalarLike("true", yaml.TaggedStyle); err != nil || got != "true" {
		t.Fatalf("scalarLike(true, TaggedStyle) = (%q, %v), want true", got, err)
	}
	if got, err := scalarLike("a\r\nb", yaml.SingleQuotedStyle); err != nil || got != `"a\r\nb"` {
		t.Fatalf("scalarLike(CRLF, SingleQuotedStyle) = (%q, %v), want \"a\\r\\nb\"", got, err)
	}
}

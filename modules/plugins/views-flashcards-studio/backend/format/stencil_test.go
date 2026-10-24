package format_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/format"
)

func faceNames(s format.Stencil) []string {
	var out []string
	for _, f := range s.Faces {
		out = append(out, f.Name)
	}
	return out
}

// A section is a deck's, and a stencil is not a deck: a first-level heading in
// a face is text the face lays out.
func TestAFirstLevelHeadingInAStencilIsText(t *testing.T) {
	s := format.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - Name\n---\n"+
		"\n# Animal\n\nHow I show these.\n"+
		"\n## Recognise\n\n### Front\n\n# {{Name}}\n\n### Back\n\n# Anything\n"))

	if got := faceNames(s); !slices.Equal(got, []string{"Recognise"}) {
		t.Fatalf("faces = %v", got)
	}
	if s.Preamble != "\n# Animal\n\nHow I show these.\n\n" {
		t.Errorf("preamble = %q, want the heading kept as it stands", s.Preamble)
	}
	if s.Faces[0].Front != "# {{Name}}" || s.Faces[0].Back != "# Anything" {
		t.Errorf("face = %+v, want both sides read whole", s.Faces[0])
	}
}

func TestAStencilDeclaresFieldsAndFaces(t *testing.T) {
	n := note(t, `---
type: stencil
fields:
  - Name
  - Height
  - Weight
  - Life span
---

## Recognise

### Front

{{Name}}

### Back

**Height:** {{Height}}
**Life span:** {{Life span}}

## Name it

### Front

Which animal lives {{Life span}}?

### Back

{{Name}}
`)
	if n.Type != format.TypeStencil {
		t.Fatalf("type = %q", n.Type)
	}

	s := format.ReadStencil(n)
	// The order is the order a person is asked for them, and the first of them
	// is what a card is named by.
	if got := s.Fields; !slices.Equal(got, []string{"Name", "Height", "Weight", "Life span"}) {
		t.Errorf("fields = %v", got)
	}
	if s.First() != "Name" {
		t.Errorf("first = %q", s.First())
	}
	if got := faceNames(s); !slices.Equal(got, []string{"Recognise", "Name it"}) {
		t.Errorf("faces = %v", got)
	}
	if len(s.Problems) != 0 {
		t.Errorf("problems = %v", s.Problems)
	}
	if s.Faces[0].Front != "{{Name}}" {
		t.Errorf("front = %q", s.Faces[0].Front)
	}
	if s.Faces[0].Back != "**Height:** {{Height}}\n**Life span:** {{Life span}}" {
		t.Errorf("back = %q", s.Faces[0].Back)
	}
}

// A stencil declaring no field has no name to put in a card's heading, so it
// cuts nothing and says so. The rest of the file is read.
func TestAStencilDeclaringNoField(t *testing.T) {
	for name, content := range map[string]string{
		"no key at all":     "",
		"an empty list":     "fields: []\n",
		"a name on its own": "fields: Height\n",
		"a number":          "fields:\n  - 12\n",
		"an empty name":     "fields:\n  - \"  \"\n",
	} {
		t.Run(name, func(t *testing.T) {
			s := format.ReadStencil(note(t, "---\ntype: stencil\n"+content+
				"---\n\n## Recognise\n\n### Front\n\n{{Height}}\n\n### Back\n\nnothing\n"))

			if s.Fields != nil {
				t.Errorf("fields = %v, want none", s.Fields)
			}
			if s.First() != "" {
				t.Errorf("first = %q, want no name at all", s.First())
			}
			if got := getProblem(t, s.Problems, format.FaultNoFields); got.Card != format.NoPosition {
				t.Errorf("problem = %+v, want it against the file", got)
			}
			// The rest of it is read, and a face placing anything at all places
			// a field nobody declared.
			if len(s.Faces) != 1 {
				t.Fatalf("faces = %v, the rest of the stencil is read", s.Faces)
			}
			if got := getProblem(t, s.Problems, format.FaultPlaceholder).Field; got != "Height" {
				t.Errorf("field = %q", got)
			}
		})
	}
}

func TestTwoFieldsOfOneNameInAStencil(t *testing.T) {
	s := format.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - Height\n  - Height\n---\n"))

	if got := s.Fields; !slices.Equal(got, []string{"Height"}) {
		t.Errorf("fields = %v", got)
	}
	if got := getProblem(t, s.Problems, format.FaultTwoFields).Field; got != "Height" {
		t.Errorf("field = %q", got)
	}
}

func TestAFacePlacesAFieldTheStencilDoesNotDeclare(t *testing.T) {
	s := format.ReadStencil(note(t, `---
type: stencil
fields:
  - Name
  - Height
---

## Recognise

### Front

{{Name}}

### Back

{{Height}} and {{Weight}}
`))

	if got := getProblem(t, s.Problems, format.FaultPlaceholder); got.Face != 0 || got.Field != "Weight" {
		t.Errorf("problem = %+v, want it against the first face and Weight", got)
	}
	// The rest of it is read as usual.
	if len(s.Faces) != 1 || s.Faces[0].Back != "{{Height}} and {{Weight}}" {
		t.Errorf("faces = %v", s.Faces)
	}
}

// A face missing either side is shown, and lays out nothing.
func TestAFaceMissingASideLaysOutNothing(t *testing.T) {
	for name, sides := range map[string]string{
		"no back":  "### Front\n\n{{Name}}\n",
		"no front": "### Back\n\n{{Name}}\n",
		"neither":  "Nothing under this one.\n",
	} {
		t.Run(name, func(t *testing.T) {
			s := format.ReadStencil(note(t,
				"---\ntype: stencil\nfields:\n  - Name\n  - Height\n---\n\n## Broken\n\n"+sides+
					"\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n"))

			if got := faceNames(s); !slices.Equal(got, []string{"Broken", "Recognise"}) {
				t.Fatalf("faces = %v, want both, so a problem can address one", got)
			}
			if got := getProblem(t, s.Problems, format.FaultFaceSide).Face; got != 0 {
				t.Errorf("face = %d, want the broken one", got)
			}
			front, back := format.Lay(s.Faces[0], format.Card{Heading: "Llama"})
			if front != "" || back != "" {
				t.Errorf("the face laid out %q and %q, want nothing", front, back)
			}
		})
	}
}

// A file ending on a heading ends where the heading's line ends, and what
// is placed under it is nothing.
func TestAStencilEndingOnAHeading(t *testing.T) {
	for name, content := range map[string]string{
		"a face":  "## Recognise",
		"a side":  "## Recognise\n\n### Front\n\n{{Name}}\n\n### Back",
		"a lead":  "## Recognise\n\nThe one to start with.\n\n### Front",
		"deeper":  "## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n#### Aside",
		"nothing": "##",
	} {
		t.Run(name, func(t *testing.T) {
			s := format.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - Name\n---\n\n"+content))

			if len(s.Faces) != 1 {
				t.Fatalf("faces = %+v, want the one the file opens", s.Faces)
			}
		})
	}
}

func TestAStencilWithNoFaces(t *testing.T) {
	s := format.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - Height\n---\n"))

	if len(s.Faces) != 0 {
		t.Errorf("faces = %v", s.Faces)
	}
	if len(s.Problems) != 0 {
		t.Errorf("problems = %v", s.Problems)
	}
	if got := s.Fields; !slices.Equal(got, []string{"Height"}) {
		t.Errorf("fields = %v", got)
	}
}

func TestAFieldThatIsNotLatin(t *testing.T) {
	s := format.ReadStencil(note(t, `---
type: stencil
fields:
  - Слово
  - Жизнь
  - Life span
---

## Вспомнить

### Front

{{Слово}}

### Back

{{Жизнь}}, {{Life span}}
`))

	if got := s.Fields; !slices.Equal(got, []string{"Слово", "Жизнь", "Life span"}) {
		t.Errorf("fields = %v", got)
	}
	if len(s.Problems) != 0 {
		t.Errorf("problems = %v", s.Problems)
	}
	if got := faceNames(s); !slices.Equal(got, []string{"Вспомнить"}) {
		t.Errorf("faces = %v", got)
	}
}

const stencilHead = "---\ntype: stencil\nfields:\n  - Name\n---\n"

func TestAStencilWhoseFrontmatterCannotBeReadIsNotOpened(t *testing.T) {
	f, err := format.OpenStencil([]byte("---\nfields: [Name\n---\n\n## Recognise\n"))

	if !errors.Is(err, markdown.ErrUnreadable) {
		t.Fatalf("open = %v, want the frontmatter rejected", err)
	}
	if f != nil {
		t.Errorf("a stencil was opened over frontmatter nobody can read")
	}
}

func TestRenamingAFieldOfAStencilSavedOnOneLineIsRefused(t *testing.T) {
	raw := "---\n{type: stencil, fields: [Name, Height]}\n---\n\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n"
	f, err := format.OpenStencil([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := f.RenameField("Height", "Shoulder height"); !errors.Is(err, markdown.ErrInline) {
		t.Fatalf("rename = %v, want it rejected", err)
	}
	if got := string(f.Bytes()); got != raw {
		t.Errorf("the file was written: %q", got)
	}
}

// A heading deeper than a side's is text the side lays out.
func TestADeeperHeadingInASideIsText(t *testing.T) {
	s := format.ReadStencil(note(t, stencilHead+
		"\n## Recognise\n\n### Front\n\n{{Name}}\n\n#### Detail\n\nmore\n\n### Back\n\nanswer\n"))

	if len(s.Faces) != 1 {
		t.Fatalf("faces = %+v", s.Faces)
	}
	if want := "{{Name}}\n\n#### Detail\n\nmore"; s.Faces[0].Front != want {
		t.Errorf("front = %q, want %q", s.Faces[0].Front, want)
	}
	if s.Faces[0].Back != "answer" {
		t.Errorf("back = %q", s.Faces[0].Back)
	}
	for _, p := range s.Problems {
		if p.Fault == format.FaultFaceSide {
			t.Errorf("a side went missing: %+v", p)
		}
	}
}

// A side is opened by the first heading of its name; a heading of another name
// and a second heading of the same name stand under the side above them.
func TestSideHeadingsThatOpenNothingStandBelowTheSideAboveThem(t *testing.T) {
	for name, tc := range map[string]struct{ body, front string }{
		"another name": {
			"### Front\n\n{{Name}}\n\n### Notes\n\nn\n\n### Back\n\nanswer\n",
			"{{Name}}\n\n### Notes\n\nn",
		},
		"a second front": {
			"### Front\n\na\n\n### Front\n\nb\n\n### Back\n\nanswer\n",
			"a\n\n### Front\n\nb",
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := format.ReadStencil(note(t, stencilHead+"\n## Recognise\n\n"+tc.body))

			if len(s.Faces) != 1 {
				t.Fatalf("faces = %+v", s.Faces)
			}
			if s.Faces[0].Front != tc.front {
				t.Errorf("front = %q, want %q", s.Faces[0].Front, tc.front)
			}
			if s.Faces[0].Back != "answer" {
				t.Errorf("back = %q, want the side after the stray heading", s.Faces[0].Back)
			}
		})
	}
}

// The tail starts where the last thing the stencil read ends.
func TestTheTailOfAStencilStartsWhereTheLastFaceEnds(t *testing.T) {
	for name, tc := range map[string]struct{ body, tail string }{
		"a lead and no sides": {"## Recognise\n\nLead text.\n\n\n", "\n\n\n"},
		"a face and nothing":  {"## Recognise\n\n\n", "\n\n\n"},
		"an empty back":       {"## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n\n", "\n\n\n"},
		"a filled back":       {"## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\nanswer\n\n\n", "\n\n\n"},
	} {
		t.Run(name, func(t *testing.T) {
			s := format.ReadStencil(note(t, stencilHead+"\n"+tc.body))

			if s.Tail != tc.tail {
				t.Errorf("tail = %q, want %q", s.Tail, tc.tail)
			}
		})
	}
}

func TestAFaceMissingASideNamesTheSideItMisses(t *testing.T) {
	for name, tc := range map[string]struct{ sides, detail string }{
		"no back":  {"### Front\n\n{{Name}}\n", "this face has no Back, and lays out nothing"},
		"no front": {"### Back\n\n{{Name}}\n", "this face has no Front, and lays out nothing"},
	} {
		t.Run(name, func(t *testing.T) {
			s := format.ReadStencil(note(t, stencilHead+"\n## Broken\n\n"+tc.sides))

			if got := getProblem(t, s.Problems, format.FaultFaceSide).Detail; got != tc.detail {
				t.Errorf("detail = %q, want %q", got, tc.detail)
			}
		})
	}
}

func TestAStencilWithNoFacesHasNoTail(t *testing.T) {
	s := format.ReadStencil(note(t, stencilHead+"\nJust prose.\n"))

	if s.Tail != "" {
		t.Errorf("tail = %q, want none", s.Tail)
	}
	if s.Preamble != "\nJust prose.\n" {
		t.Errorf("preamble = %q", s.Preamble)
	}
}

func TestAFieldThatIsNotAWordIsSkippedAndTheFieldsAfterItStand(t *testing.T) {
	s := format.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - 5\n  - \"\"\n  - Name\n  - Height\n---\n"))

	if !slices.Equal(s.Fields, []string{"Name", "Height"}) {
		t.Errorf("fields = %v", s.Fields)
	}
}

// A mark stands last, one space after the heading's text, and is a mark only at
// that length and in that alphabet. Anything else at the end of a heading is
// heading text.
func TestReadHeading(t *testing.T) {
	for name, one := range map[string]struct {
		heading, text string
		mark          cardid.ID
	}{
		"a text and a mark": {
			"Compost, what is it made of ^k7m2xq9fzp", "Compost, what is it made of", "k7m2xq9fzp",
		},
		"a mark and nothing else": {"^k7m2xq9fzp", "", "k7m2xq9fzp"},
		"nothing at all":          {"", "", ""},
		"no mark":                 {"Compost, what is it made of", "Compost, what is it made of", ""},
		"too few characters":      {"Compost ^k7m2xq9fz", "Compost ^k7m2xq9fz", ""},
		"too many characters":     {"Compost ^k7m2xq9fzpp", "Compost ^k7m2xq9fzpp", ""},
		"outside the alphabet":    {"Compost ^K7M2XQ9FZP", "Compost ^K7M2XQ9FZP", ""},
		"no caret":                {"Compost k7m2xq9fzp", "Compost k7m2xq9fzp", ""},
		"no space in front":       {"Compost^k7m2xq9fzp", "Compost^k7m2xq9fzp", ""},
		"not last on the line":    {"Compost ^k7m2xq9fzp is a heap", "Compost ^k7m2xq9fzp is a heap", ""},
		"a caret and nothing":     {"Compost ^", "Compost ^", ""},
		"a text of one space":     {"Compost  ^k7m2xq9fzp", "Compost ", "k7m2xq9fzp"},
		"a text that is a mark":   {"^k7m2xq9fzp ^zpqrstvwxy", "^k7m2xq9fzp", "zpqrstvwxy"},
	} {
		t.Run(name, func(t *testing.T) {
			text, mark := format.ReadHeading(one.heading)
			if text != one.text || mark != one.mark {
				t.Errorf("read %q = %q, %q, want %q, %q", one.heading, text, mark, one.text, one.mark)
			}
			if got := format.WriteHeading(text, mark); got != one.heading {
				t.Errorf("written back = %q, want %q", got, one.heading)
			}
		})
	}
}

// A card whose first field is empty is placed under a heading of its mark alone,
// and a heading is the text alone until the card is written.
func TestWriteHeading(t *testing.T) {
	for name, one := range map[string]struct {
		text string
		mark cardid.ID
		want string
	}{
		"both":         {"Compost", "k7m2xq9fzp", "Compost ^k7m2xq9fzp"},
		"no text":      {"", "k7m2xq9fzp", "^k7m2xq9fzp"},
		"no mark":      {"Compost", "", "Compost"},
		"neither":      {"", "", ""},
		"beyond latin": {"компост", "k7m2xq9fzp", "компост ^k7m2xq9fzp"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := format.WriteHeading(one.text, one.mark); got != one.want {
				t.Errorf("write = %q, want %q", got, one.want)
			}
		})
	}
}

// The heading is the first line of the first field, cut to fit one line: it
// stops at the first break and at format.HeadingRunes characters.
func TestProject(t *testing.T) {
	long := strings.Repeat("a", format.HeadingRunes+40)
	upTo := func(n int) string { return strings.Repeat("a", n) }

	for name, one := range map[string]struct{ value, want string }{
		"one line":            {"Compost, what is it made of", "Compost, what is it made of"},
		"beyond latin":        {"компост, из чего он", "компост, из чего он"},
		"two lines":           {"Compost, what is it made of\nand what it is for", "Compost, what is it made of"},
		"carriage returns":    {"Compost\r\nand more", "Compost"},
		"a lone carriage":     {"Compost\rand more", "Compost"},
		"nothing":             {"", ""},
		"spaces alone":        {"   ", ""},
		"spaces around":       {"  Compost  ", "Compost"},
		"a blank first line":  {"\nCompost", ""},
		"too long":            {long, upTo(format.HeadingRunes)},
		"a break before then": {upTo(10) + "\n" + long, upTo(10)},

		// The cut never falls inside one of these: it falls before the whole of
		// whichever it lands in.
		"inside a wikilink": {
			upTo(format.HeadingRunes-5) + " [[Compost heap]] and on", upTo(format.HeadingRunes - 5),
		},
		"inside an embed": {
			upTo(format.HeadingRunes-5) + " ![[llama.png]] and on", upTo(format.HeadingRunes - 5),
		},
		"inside a run of emphasis": {
			upTo(format.HeadingRunes-5) + " **very heavy** and on", upTo(format.HeadingRunes - 5),
		},
		"inside emphasis of one star": {
			upTo(format.HeadingRunes-5) + " *very heavy* and on", upTo(format.HeadingRunes - 5),
		},
		"inside emphasis of an underscore": {
			upTo(format.HeadingRunes-5) + " __very heavy__ and on", upTo(format.HeadingRunes - 5),
		},
		// A wikilink the cut falls after is text like any other.
		"after a wikilink": {
			upTo(format.HeadingRunes-20) + " [[Compost heap]] " + long,
			upTo(format.HeadingRunes-20) + " [[Compost heap]] " + upTo(2),
		},
		// A star with no partner opens no run of emphasis.
		"a lone star": {
			upTo(format.HeadingRunes-5) + " *very heavy and on", upTo(format.HeadingRunes-5) + " *ver",
		},

		// A link inside a run of emphasis is the link's, and the run around it
		// is one run.
		"inside emphasis holding a link": {
			upTo(100) + " *see [[a very long target]] now*", upTo(100),
		},
		"inside emphasis holding an embed": {
			upTo(100) + " **the [[compost]] one made of leaves**", upTo(100),
		},
		// A link's brackets hold whatever a person wrote, brackets included, and
		// the link closes at its own last one.
		"inside a wikilink holding an embed": {
			upTo(100) + " [[a ![[img.png]] b]] tail", upTo(100),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := format.Project(one.value)
			if got != one.want {
				t.Errorf("project %q\n want %q\n  got %q", one.value, one.want, got)
			}
			if n := len([]rune(got)); n > format.HeadingRunes {
				t.Errorf("the heading is %d characters, and %d is the most one is", n, format.HeadingRunes)
			}
		})
	}
}

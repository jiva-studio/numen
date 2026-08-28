package cards_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

func faceNames(s cards.Stencil) []string {
	var out []string
	for _, f := range s.Faces {
		out = append(out, f.Name)
	}
	return out
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
	if n.Type != domain.TypeStencil {
		t.Fatalf("type = %q", n.Type)
	}

	s := cards.ReadStencil(n)
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
	for name, written := range map[string]string{
		"no key at all":     "",
		"an empty list":     "fields: []\n",
		"a name on its own": "fields: Height\n",
		"a number":          "fields:\n  - 12\n",
		"an empty name":     "fields:\n  - \"  \"\n",
	} {
		t.Run(name, func(t *testing.T) {
			s := cards.ReadStencil(note(t, "---\ntype: stencil\n"+written+
				"---\n\n## Recognise\n\n### Front\n\n{{Height}}\n\n### Back\n\nnothing\n"))

			if s.Fields != nil {
				t.Errorf("fields = %v, want none", s.Fields)
			}
			if s.First() != "" {
				t.Errorf("first = %q, want no name at all", s.First())
			}
			if got := filed(t, s.Problems, cards.CheckNoFields); got.Card != cards.NoPosition {
				t.Errorf("problem = %+v, want it against the file", got)
			}
			// The rest of it is read, and a face placing anything at all places
			// a field nobody declared.
			if len(s.Faces) != 1 {
				t.Fatalf("faces = %v, the rest of the stencil is read", s.Faces)
			}
			if got := filed(t, s.Problems, cards.CheckPlaceholder).Field; got != "Height" {
				t.Errorf("field = %q", got)
			}
		})
	}
}

func TestTwoFieldsOfOneNameInAStencil(t *testing.T) {
	s := cards.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - Height\n  - Height\n---\n"))

	if got := s.Fields; !slices.Equal(got, []string{"Height"}) {
		t.Errorf("fields = %v", got)
	}
	if got := filed(t, s.Problems, cards.CheckTwoFields).Field; got != "Height" {
		t.Errorf("field = %q", got)
	}
}

func TestAFacePlacesAFieldTheStencilDoesNotDeclare(t *testing.T) {
	s := cards.ReadStencil(note(t, `---
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

	if got := filed(t, s.Problems, cards.CheckPlaceholder); got.Face != 0 || got.Field != "Weight" {
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
			s := cards.ReadStencil(note(t,
				"---\ntype: stencil\nfields:\n  - Name\n  - Height\n---\n\n## Broken\n\n"+sides+
					"\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n"))

			if got := faceNames(s); !slices.Equal(got, []string{"Broken", "Recognise"}) {
				t.Fatalf("faces = %v, want both, so a problem can address one", got)
			}
			if got := filed(t, s.Problems, cards.CheckFaceSide).Face; got != 0 {
				t.Errorf("face = %d, want the broken one", got)
			}
			front, back := cards.Lay(s, s.Faces[0], cards.Card{Name: "Llama"})
			if front != "" || back != "" {
				t.Errorf("the face laid out %q and %q, want nothing", front, back)
			}
		})
	}
}

// A file ending on a heading ends where the heading's line ends, and what
// stands under it is nothing.
func TestAStencilEndingOnAHeading(t *testing.T) {
	for name, written := range map[string]string{
		"a face":  "## Recognise",
		"a side":  "## Recognise\n\n### Front\n\n{{Name}}\n\n### Back",
		"a lead":  "## Recognise\n\nAsked of me by Anna.\n\n### Front",
		"deeper":  "## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n#### Aside",
		"nothing": "##",
	} {
		t.Run(name, func(t *testing.T) {
			s := cards.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - Name\n---\n\n"+written))

			if len(s.Faces) != 1 {
				t.Fatalf("faces = %+v, want the one the file opens", s.Faces)
			}
		})
	}
}

func TestAStencilWithNoFaces(t *testing.T) {
	s := cards.ReadStencil(note(t, "---\ntype: stencil\nfields:\n  - Height\n---\n"))

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
	s := cards.ReadStencil(note(t, `---
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

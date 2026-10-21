package format_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

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

func TestRenamingAFieldOfAStencilWrittenOnOneLineIsRefused(t *testing.T) {
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
func TestSideHeadingsThatOpenNothingStandUnderTheSideAboveThem(t *testing.T) {
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

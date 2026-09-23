package markdown_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// One key says what a note is, out of a closed list of five.
func TestOneKeySaysWhatANoteIs(t *testing.T) {
	for written, want := range map[string]domain.NoteType{
		"---\ntype: note\n---\n":    domain.TypeNote,
		"---\ntype: deck\n---\n":    domain.TypeDeck,
		"---\ntype: stencil\n---\n": domain.TypeStencil,
		"---\ntype: preset\n---\n":  domain.TypePreset,

		"---\ntitle: Entropy\n---\n": domain.TypeNote,
		"---\ntype:\n---\n":          domain.TypeNote,
		"---\ntype: \"\"\n---\n":     domain.TypeNote,
		"# Entropy\n":                domain.TypeNote,
	} {
		n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(written))
		if n.Type != want {
			t.Errorf("%q: type = %q, want %q", written, n.Type, want)
		}
		if len(n.Problems) != 0 {
			t.Errorf("%q: problems = %v", written, n.Problems)
		}
	}
}

// A value outside the list is a problem against the note, and the note is read
// as an ordinary note.
func TestATypeOutsideTheList(t *testing.T) {
	for _, written := range []string{
		"---\ntype: something-else\ntitle: Entropy\n---\n\n# Entropy\n",
		"---\ntype: Deck\n---\n",
		"---\ntype: 12\n---\n",
		"---\ntype:\n  - deck\n---\n",
	} {
		n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(written))
		if n.Type != domain.TypeNote {
			t.Errorf("%q: type = %q, want it read as an ordinary note", written, n.Type)
		}
		if len(n.Problems) != 1 || !strings.Contains(n.Problems[0], "type") {
			t.Errorf("%q: problems = %v", written, n.Problems)
		}
	}
}

// A note's type and a link's type are two keys of one word, told apart by an
// indent and by nothing else.
func TestALinksTypeIsNotTheNotesType(t *testing.T) {
	n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(
		"---\ntype: deck\nlinks:\n  - to: Thermodynamics\n    role: parent\n    type: source\n---\n"))

	if n.Type != domain.TypeDeck {
		t.Errorf("type = %q", n.Type)
	}
	if len(n.Links) != 1 || n.Links[0].Type != "source" {
		t.Errorf("links = %+v", n.Links)
	}
	if len(n.Problems) != 0 {
		t.Errorf("problems = %v", n.Problems)
	}
}

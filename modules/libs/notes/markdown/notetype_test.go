package markdown_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
)

// One key says what a note is, and any word will do: a plugin gives a note the type it works
// with.
func TestOneKeyReturnsWhatANoteIs(t *testing.T) {
	for content, want := range map[string]domain.NoteType{
		"---\ntype: note\n---\n":     domain.TypeNote,
		"---\ntype: deck\n---\n":     "deck",
		"---\ntype: Deck\n---\n":     "Deck",
		"---\ntype: sprocket\n---\n": "sprocket",

		"---\ntitle: Entropy\n---\n": domain.TypeNote,
		"---\ntype:\n---\n":          domain.TypeNote,
		"---\ntype: \"\"\n---\n":     domain.TypeNote,
		"# Entropy\n":                domain.TypeNote,
	} {
		n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(content))
		if n.Type != want {
			t.Errorf("%q: type = %q, want %q", content, n.Type, want)
		}
		if len(n.Problems) != 0 {
			t.Errorf("%q: problems = %v", content, n.Problems)
		}
	}
}

// A value that is not text is a problem against the note, and the note is read as an ordinary
// note.
func TestATypeThatIsNotText(t *testing.T) {
	for _, content := range []string{
		"---\ntype: 12\n---\n",
		"---\ntype:\n  - deck\n---\n",
	} {
		n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(content))
		if n.Type != domain.TypeNote {
			t.Errorf("%q: type = %q, want it read as an ordinary note", content, n.Type)
		}
		if len(n.Problems) != 1 || !strings.Contains(n.Problems[0], "type") {
			t.Errorf("%q: problems = %v", content, n.Problems)
		}
	}
}

// A note's type and a link's type are two keys of one word, told apart by an
// indent and by nothing else.
func TestALinksTypeIsNotTheNotesType(t *testing.T) {
	n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(
		"---\ntype: deck\nlinks:\n  - to: Thermodynamics\n    role: parent\n    type: source\n---\n"))

	if n.Type != "deck" {
		t.Errorf("type = %q", n.Type)
	}
	if len(n.Links) != 1 || n.Links[0].Type != "source" {
		t.Errorf("links = %+v", n.Links)
	}
	if len(n.Problems) != 0 {
		t.Errorf("problems = %v", n.Problems)
	}
}

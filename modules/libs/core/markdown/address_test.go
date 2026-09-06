package markdown_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// A link note carries where it points in the frontmatter, where every kind of
// note carries what is its own, and it is read from there.
func TestALinkNoteCarriesWhereItPoints(t *testing.T) {
	n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(
		"---\ntype: link\nurl: https://youtu.be/dQw4w9WgXcQ?si=Ab1Cd2Ef3\n---\n\nWhat I made of it.\n"))

	if n.Type != domain.TypeLink {
		t.Fatalf("type = %q", n.Type)
	}
	at, wrong := domain.ReadAddress(n.Frontmatter)
	if len(wrong) != 0 {
		t.Fatalf("problems = %v", wrong)
	}
	if at.URL != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("points at %q", at.URL)
	}
	if !at.IsVideo() {
		t.Errorf("points at no video")
	}
	if len(n.Problems) != 0 {
		t.Errorf("problems = %v", n.Problems)
	}
	if n.Body != "\nWhat I made of it.\n" {
		t.Errorf("body = %q", n.Body)
	}
}

// A link note with nowhere to point is missing its whole subject, and that is
// said rather than guessed at. The note is read as every other note is.
func TestALinkNoteWithNowhereToPoint(t *testing.T) {
	for _, written := range []string{
		"---\ntype: link\n---\n\nWhat I made of it.\n",
		"---\ntype: link\nurl:\n---\n",
		"---\ntype: link\nurl: 12\n---\n",
		"---\ntype: link\nurl: not an address\n---\n",
		"---\ntype: link\nurl: file:///etc/passwd\n---\n",
	} {
		n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(written))
		if n.Type != domain.TypeLink {
			t.Errorf("%q: type = %q", written, n.Type)
		}
		if len(n.Problems) != 1 || !strings.Contains(n.Problems[0], "url") {
			t.Errorf("%q: problems = %v", written, n.Problems)
		}
		if at, wrong := domain.ReadAddress(n.Frontmatter); at != (domain.WebAddress{}) || len(wrong) == 0 {
			t.Errorf("%q: points at %+v", written, at)
		}
	}
}

// The key is read on a link note and nowhere else: an address written on any
// other note is a key of the person's own, kept as they wrote it.
func TestAnAddressOnAnyOtherNoteIsTheirOwnKey(t *testing.T) {
	n := markdown.Parse(domain.Fingerprint{Path: "Entropy.md"}, []byte(
		"---\nurl: not an address\n---\n"))

	if n.Type != domain.TypeNote {
		t.Fatalf("type = %q", n.Type)
	}
	if len(n.Problems) != 0 {
		t.Errorf("problems = %v", n.Problems)
	}
	if n.Frontmatter["url"] != "not an address" {
		t.Errorf("url = %v", n.Frontmatter["url"])
	}
}

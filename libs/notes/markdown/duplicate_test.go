package markdown

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A key written twice names two values, and reading the note gives one of them.
// already gives.
func TestFrontmatterWithAKeySavedTwiceIsUnreadable(t *testing.T) {
	raw := "---\ntitle: One\nid: 01J8\ntitle: Two\n---\nbody\n"

	if _, err := Open([]byte(raw)); !errors.Is(err, ErrUnreadable) {
		t.Errorf("open: want ErrUnreadable, got %v", err)
	}
	if got := Parse(domain.Fingerprint{Path: "D.md"}, []byte(raw)); got.FrontmatterErr == "" {
		t.Errorf("parse read the note: %v", got.Frontmatter)
	}
}

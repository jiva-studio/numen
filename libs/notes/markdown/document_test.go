package markdown

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/ulid"
)

// A note that is opened and not changed is returned byte for byte. Anything less
// is an unrequested diff on every save.
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

// What Create writes has to be readable by the parser that reads every other
// note, or the application has invented a second format.
func TestCreateIsReadBackByTheParser(t *testing.T) {
	identifier := ulid.New(time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC))
	raw := Create(identifier, "# Entropy\n")
	n := Parse(domain.FileStat{Path: "entropy.md"}, raw)
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

package markdown

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/ulid"
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
	identifier, err := ulid.NewGenerator(rand.Reader).NewID(time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
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

	if got := string(Create("01ABC", "")); got != "---\nid: 01ABC\n---\n" {
		t.Errorf("Create empty body = %q", got)
	}
	if got := string(Create("01ABC", "Hello")); got != "---\nid: 01ABC\n---\nHello\n" {
		t.Errorf("Create body without newline = %q", got)
	}
	if got := string(Create("01ABC", "Hello\n")); got != "---\nid: 01ABC\n---\nHello\n" {
		t.Errorf("Create body with newline = %q", got)
	}
}

func TestSetBodyAndSpliceBodyBranches(t *testing.T) {
	unterm, err := Open([]byte("---\nid: 01J8\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := unterm.SetBody("x"); !errors.Is(err, ErrUnterminated) {
		t.Errorf("SetBody on unterminated = %v", err)
	}
	if err := unterm.SpliceBody(0, 0, "x"); !errors.Is(err, ErrUnterminated) {
		t.Errorf("SpliceBody on unterminated = %v", err)
	}

	d := OpenBody([]byte("hello world\n"))
	if err := d.SetBody(""); err != nil || d.Body() != "" {
		t.Errorf("SetBody empty = %q, %v", d.Body(), err)
	}
	if err := d.SetBody("no newline"); err != nil || d.Body() != "no newline\n" {
		t.Errorf("SetBody no newline = %q, %v", d.Body(), err)
	}

	if err := d.SpliceBody(-1, 0, "x"); err == nil {
		t.Error("expected error for start < 0")
	}
	if err := d.SpliceBody(2, 1, "x"); err == nil {
		t.Error("expected error for end < start")
	}
	if err := d.SpliceBody(0, len(d.Body())+1, "x"); err == nil {
		t.Error("expected error for end > len(body)")
	}
	if err := d.SpliceBody(0, 0, "prefix "); err != nil || d.Body() != "prefix no newline\n" {
		t.Errorf("SpliceBody at 0,0 = %q, %v", d.Body(), err)
	}
	if err := d.SpliceBody(7, len(d.Body()), "tail\n"); err != nil || d.Body() != "prefix tail\n" {
		t.Errorf("SpliceBody to end = %q, %v", d.Body(), err)
	}

	crlfDoc, err := Open([]byte("---\r\nid: 01J8\r\n---\r\nhello world\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := crlfDoc.SpliceBody(6, 11, "a\nb"); err != nil || crlfDoc.Body() != "hello a\r\nb\r\n" {
		t.Errorf("SpliceBody CRLF = %q, %v", crlfDoc.Body(), err)
	}
}

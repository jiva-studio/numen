package markdown

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A wikilink inside a code fence is an example of a link, not one, and a note
// moving somewhere else leaves it as it was written.
func TestProseInsideAFenceIsNotPointedAnywhere(t *testing.T) {
	raw := "---\nid: 01J8\n---\n" +
		"A mention of [[Old]].\n" +
		"\n" +
		"```\n" +
		"[[Old]]\n" +
		"```\n" +
		"\n" +
		"~~~\n" +
		"```\n" +
		"[[Old]]\n" +
		"~~~\n" +
		"\n" +
		"And [[Old|the old one]] again.\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	moved, err := d.PointProseAt(domain.ParseAddress("Old"), "New")
	if err != nil {
		t.Fatalf("point prose: %v", err)
	}
	if moved != 2 {
		t.Errorf("moved = %d, want 2", moved)
	}

	want := "---\nid: 01J8\n---\n" +
		"A mention of [[New]].\n" +
		"\n" +
		"```\n" +
		"[[Old]]\n" +
		"```\n" +
		"\n" +
		"~~~\n" +
		"```\n" +
		"[[Old]]\n" +
		"~~~\n" +
		"\n" +
		"And [[New|the old one]] again.\n"
	if got := string(d.Bytes()); got != want {
		t.Errorf("a move rewrote a link that is not one\n want %q\n  got %q", want, got)
	}
}

// A fence opened with tildes is closed by tildes. Backticks inside it are the
// text of the example.
func TestBackticksInsideATildeFenceAreNotAFence(t *testing.T) {
	body := []byte("~~~\n```\n[[Inside]]\n~~~\n\n[[Outside]]\n")
	links := bodyLinks(body)
	if len(links) != 1 || links[0].Target.Value != "Outside" {
		t.Errorf("links = %v", links)
	}
	if got := headings([]byte("~~~\n```\n# Inside\n~~~\n\n# Outside\n")); len(got) != 1 || got[0].Text != "Outside" {
		t.Errorf("headings = %v", got)
	}
}

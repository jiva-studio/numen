package markdown

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
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
	moved, err := d.RedirectProse(domain.ParseAddress("Old"), "New")
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

func TestRedirectProseAndParseHelpers(t *testing.T) {
	// Wikilink at byte 0, non-matching link before matching link on same line, and link at byte 0 of line 2.
	d := OpenBody([]byte("[[Old]] and [[Other]] then [[Old#sec|alias]]\n[[Old|lbl]]"))
	moved, err := d.RedirectProse(domain.ParseAddress("Old"), "New")
	if err != nil || moved != 3 {
		t.Fatalf("RedirectProse = %d, %v", moved, err)
	}
	if got := d.Body(); got != "[[New]] and [[Other]] then [[New#sec|alias]]\n[[New|lbl]]" {
		t.Errorf("got %q", got)
	}

	if got := getWikilinkSuffix("|alias"); got != "|alias" {
		t.Errorf("getWikilinkSuffix(|alias) = %q", got)
	}
	if got := getWikilinkSuffix("#frag"); got != "#frag" {
		t.Errorf("getWikilinkSuffix(#frag) = %q", got)
	}

	// Direct tests for splitFrontmatter and isFrontmatterDelimiter.
	if fm, body, ok := splitFrontmatter([]byte("no-newline")); ok || fm != nil || string(body) != "no-newline" {
		t.Errorf("splitFrontmatter(no-newline) = %q, %q, %v", fm, body, ok)
	}
	if fm, body, ok := splitFrontmatter([]byte("plain\nworld")); ok || fm != nil || string(body) != "plain\nworld" {
		t.Errorf("splitFrontmatter(plain) = %q, %q, %v", fm, body, ok)
	}
	if fm, body, ok := splitFrontmatter([]byte("---\n\n---\nbody")); !ok || string(fm) != "\n" || string(body) != "body" {
		t.Errorf("splitFrontmatter(blank line) = %q, %q, %v", fm, body, ok)
	}
	if fm, body, ok := splitFrontmatter([]byte("---\na: 1\nunterminated")); ok || fm != nil || string(body) != "---\na: 1\nunterminated" {
		t.Errorf("splitFrontmatter(unterminated) = %q, %q, %v", fm, body, ok)
	}
	for _, badDelim := range []string{"x--", "-x-", "--x", "--", "----"} {
		if isFrontmatterDelimiter([]byte(badDelim)) {
			t.Errorf("isFrontmatterDelimiter(%q) = true", badDelim)
		}
	}
	if !isFrontmatterDelimiter([]byte("---")) {
		t.Error("isFrontmatterDelimiter(---) = false")
	}

	// Direct tests for headings, parseHeading, trimClosingMarks, and Fence.
	hs := headings([]byte("# One\n# Two"))
	if len(hs) != 2 || hs[0].Line != 0 || hs[1].Line != 1 || hs[1].Offset != 6 {
		t.Errorf("headings(# One\\n# Two) = %+v", hs)
	}
	for _, invalid := range []string{"", "plain", "#", "###", "####### Seven", "#NoSpace", "#a ", "# ", "#   \t  "} {
		if lvl, txt, ok := parseHeading(invalid); ok {
			t.Errorf("parseHeading(%q) = %d, %q, true", invalid, lvl, txt)
		}
	}
	if lvl, txt, ok := parseHeading("###### Six"); !ok || lvl != 6 || txt != "Six" {
		t.Errorf("parseHeading(###### Six) = %d, %q, %v", lvl, txt, ok)
	}
	if lvl, txt, ok := parseHeading("#\tTab"); !ok || lvl != 1 || txt != "Tab" {
		t.Errorf("parseHeading(#\\tTab) = %d, %q, %v", lvl, txt, ok)
	}
	if lvl, txt, ok := parseHeading("# x"); !ok || lvl != 1 || txt != "x" {
		t.Errorf("parseHeading(# x) = %d, %q, %v", lvl, txt, ok)
	}
	if got := trimClosingMarks("Title ## "); got != "Title" {
		t.Errorf("trimClosingMarks(Title ## ) = %q", got)
	}
	if got := trimClosingMarks("###"); got != "#" {
		t.Errorf("trimClosingMarks(###) = %q", got)
	}

	var f1 Fence
	if !f1.IsCrossedBy("```") || f1.IsCrossedBy("~~~") || !f1.IsInside() || !f1.IsCrossedBy("```") || f1.IsInside() {
		t.Error("Fence.IsCrossedBy state transition failed")
	}
	var f2 Fence
	if !f2.IsCrossedByBytes([]byte("```")) || f2.IsCrossedByBytes([]byte("~~~")) || !f2.IsInside() || !f2.IsCrossedByBytes([]byte("```")) || f2.IsInside() {
		t.Error("Fence.IsCrossedByBytes state transition failed")
	}

	// Wikilink and code-span edge cases.
	w1 := WikilinksIn("[[A]] text [[B]] and [[]] and [[[Good]] and `unclosed [[AfterUnclosed]]")
	if len(w1) != 4 || w1[0].Inside != "A" || w1[1].Inside != "B" || w1[1].At != 11 || w1[2].Inside != "Good" || w1[3].Inside != "AfterUnclosed" {
		t.Errorf("WikilinksIn edge cases = %+v", w1)
	}
	w2 := WikilinksInText("\n```\n[[Ignored]]\n```\n[[Kept]]")
	if len(w2) != 1 || w2[0].Inside != "Kept" {
		t.Errorf("WikilinksInText leading newline = %+v", w2)
	}
	if spans := getCodeSpans("`unclosed"); len(spans) != 0 {
		t.Errorf("getCodeSpans(`unclosed) = %v", spans)
	}
	if spans := getCodeSpans("a`"); len(spans) != 0 {
		t.Errorf("getCodeSpans(a`) = %v", spans)
	}
	spans := getCodeSpans("a `b` c ``d ` e``")
	if len(spans) != 2 || spans[0] != [2]int{2, 5} || spans[1] != [2]int{8, 17} {
		t.Errorf("getCodeSpans multi = %v", spans)
	}
	if _, ok := findCodeSpan([][2]int{{2, 5}}, 1); ok {
		t.Error("findCodeSpan at 1 should be false")
	}
	if _, ok := findCodeSpan([][2]int{{2, 5}}, 2); !ok {
		t.Error("findCodeSpan at 2 should be true")
	}
	if _, ok := findCodeSpan([][2]int{{2, 5}}, 5); ok {
		t.Error("findCodeSpan at 5 should be false")
	}
	if hasCodeSpanIn([][2]int{{2, 5}}, 2, 5) || hasCodeSpanIn([][2]int{{2, 5}}, 1, 2) || hasCodeSpanIn([][2]int{{0, 2}}, 0, 3) || !hasCodeSpanIn([][2]int{{2, 5}}, 1, 3) {
		t.Error("hasCodeSpanIn boundary failure")
	}
}

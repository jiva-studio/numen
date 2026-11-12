package diff

import (
	"strings"
	"testing"
)

func TestCreateUnifiedIdentical(t *testing.T) {
	content := "line 1\nline 2\nline 3"
	diffText, additions, deletions := CreateUnified("notes/test.md", content, content)
	if diffText != "" {
		t.Errorf("diffText = %q, want empty", diffText)
	}
	if additions != 0 || deletions != 0 {
		t.Errorf("got +%d, -%d; want +0, -0", additions, deletions)
	}
}

func TestCreateUnifiedNewFile(t *testing.T) {
	proposed := "line 1\nline 2\nline 3"
	diffText, additions, deletions := CreateUnified("notes/test.md", "", proposed)
	if additions != 3 {
		t.Errorf("additions = %d, want 3", additions)
	}
	if deletions != 0 {
		t.Errorf("deletions = %d, want 0", deletions)
	}
	if !strings.Contains(diffText, "--- a/notes/test.md") {
		t.Errorf("missing diff header --- a/notes/test.md in %q", diffText)
	}
	if !strings.Contains(diffText, "+line 1") {
		t.Errorf("missing +line 1 in %q", diffText)
	}
}

func TestCreateUnifiedDeletedFile(t *testing.T) {
	original := "line 1\nline 2"
	diffText, additions, deletions := CreateUnified("notes/test.md", original, "")
	if additions != 0 {
		t.Errorf("additions = %d, want 0", additions)
	}
	if deletions != 2 {
		t.Errorf("deletions = %d, want 2", deletions)
	}
	if !strings.Contains(diffText, "-line 1") || !strings.Contains(diffText, "-line 2") {
		t.Errorf("missing -line 1 / -line 2 in %q", diffText)
	}
}

func TestCreateUnifiedModifications(t *testing.T) {
	original := "apple\nbanana\ncherry\ndate"
	proposed := "apple\nblueberry\ncherry\nfig\ngrape"
	diffText, additions, deletions := CreateUnified("notes/fruits.md", original, proposed)

	if additions != 3 { // blueberry, fig, grape
		t.Errorf("additions = %d, want 3", additions)
	}
	if deletions != 2 { // banana, date
		t.Errorf("deletions = %d, want 2", deletions)
	}
	if !strings.Contains(diffText, "-banana") {
		t.Errorf("expected -banana in diff: %s", diffText)
	}
	if !strings.Contains(diffText, "+blueberry") {
		t.Errorf("expected +blueberry in diff: %s", diffText)
	}
	want := "--- a/notes/fruits.md\n" +
		"+++ b/notes/fruits.md\n" +
		"@@ -1,4 +1,5 @@\n" +
		" apple\n" +
		"-banana\n" +
		"+blueberry\n" +
		" cherry\n" +
		"-date\n" +
		"+fig\n" +
		"+grape\n"
	if diffText != want {
		t.Errorf("diffText =\n%q\nwant:\n%q", diffText, want)
	}
}

func TestCreateUnifiedNormalizedEquivalent(t *testing.T) {
	diffText, additions, deletions := CreateUnified("notes/test.md", "line 1\r\nline 2\n", "line 1\nline 2")
	if diffText != "" || additions != 0 || deletions != 0 {
		t.Errorf("got (%q, %d, %d), want (\"\", 0, 0)", diffText, additions, deletions)
	}
}

func TestCreateUnifiedInsertAtStartAndDeleteAtStart(t *testing.T) {
	diffInsert, addIns, delIns := CreateUnified("notes/a.md", "a\nb\nc", "x\na\nb\nc")
	wantInsert := "--- a/notes/a.md\n+++ b/notes/a.md\n@@ -1,3 +1,4 @@\n+x\n a\n b\n c\n"
	if diffInsert != wantInsert || addIns != 1 || delIns != 0 {
		t.Errorf("insert at start = (%q, +%d, -%d), want (%q, +1, -0)", diffInsert, addIns, delIns, wantInsert)
	}

	diffDelete, addDel, delDel := CreateUnified("notes/a.md", "x\na\nb\nc", "a\nb\nc")
	wantDelete := "--- a/notes/a.md\n+++ b/notes/a.md\n@@ -1,4 +1,3 @@\n-x\n a\n b\n c\n"
	if diffDelete != wantDelete || addDel != 0 || delDel != 1 {
		t.Errorf("delete at start = (%q, +%d, -%d), want (%q, +0, -1)", diffDelete, addDel, delDel, wantDelete)
	}
}

func TestCreateUnifiedContextAndMultiHunkBoundaries(t *testing.T) {
	// Single change in the middle of 11 lines: 3 leading context lines and 3 trailing context lines.
	origSingle := "1\n2\n3\n4\n5\nold\n7\n8\n9\n10\n11"
	propSingle := "1\n2\n3\n4\n5\nnew\n7\n8\n9\n10\n11"
	diffSingle, addS, delS := CreateUnified("f.md", origSingle, propSingle)
	wantSingle := "--- a/f.md\n+++ b/f.md\n@@ -3,7 +3,7 @@\n 3\n 4\n 5\n-old\n+new\n 7\n 8\n 9\n"
	if diffSingle != wantSingle || addS != 1 || delS != 1 {
		t.Errorf("single hunk =\n%s\nwant:\n%s", diffSingle, wantSingle)
	}

	// 4 equal lines between two changes: merges into a single hunk.
	orig4 := "old1\n1\n2\n3\n4\nold2"
	prop4 := "new1\n1\n2\n3\n4\nnew2"
	diff4, _, _ := CreateUnified("f.md", orig4, prop4)
	want4 := "--- a/f.md\n+++ b/f.md\n@@ -1,6 +1,6 @@\n-old1\n+new1\n 1\n 2\n 3\n 4\n-old2\n+new2\n"
	if diff4 != want4 {
		t.Errorf("4 equal lines =\n%s\nwant:\n%s", diff4, want4)
	}

	// 7 equal lines between two changes: still merges into a single hunk under lookahead.
	orig7 := "old1\n1\n2\n3\n4\n5\n6\n7\nold2"
	prop7 := "new1\n1\n2\n3\n4\n5\n6\n7\nnew2"
	diff7, _, _ := CreateUnified("f.md", orig7, prop7)
	want7 := "--- a/f.md\n+++ b/f.md\n@@ -1,9 +1,9 @@\n-old1\n+new1\n 1\n 2\n 3\n 4\n 5\n 6\n 7\n-old2\n+new2\n"
	if diff7 != want7 {
		t.Errorf("7 equal lines =\n%s\nwant:\n%s", diff7, want7)
	}

	// 8 equal lines between two changes: splits into two hunks with 3 trailing and 3 leading context lines.
	orig8 := "old1\n1\n2\n3\n4\n5\n6\n7\n8\nold2\nt1\nt2\nt3\nt4"
	prop8 := "new1\n1\n2\n3\n4\n5\n6\n7\n8\nnew2\nt1\nt2\nt3\nt4"
	diff8, add8, del8 := CreateUnified("f.md", orig8, prop8)
	want8 := "--- a/f.md\n+++ b/f.md\n" +
		"@@ -1,4 +1,4 @@\n-old1\n+new1\n 1\n 2\n 3\n" +
		"@@ -7,7 +7,7 @@\n 6\n 7\n 8\n-old2\n+new2\n t1\n t2\n t3\n"
	if diff8 != want8 || add8 != 2 || del8 != 2 {
		t.Errorf("8 equal lines =\n%s\nwant:\n%s", diff8, want8)
	}
}

func TestBuildLcsTableAndHunkCollectorInvariants(t *testing.T) {
	table := buildLcsTable([]string{"a", "b"}, []string{"b", "c", "d"})
	if len(table) != 3 || len(table[0]) != 4 {
		t.Fatalf("table dimensions = %dx%d, want 3x4", len(table), len(table[0]))
	}
	if table[0][0] != 1 {
		t.Errorf("table[0][0] = %d, want 1", table[0][0])
	}
	if got := buildLcsTable([]string{"a", "b"}, []string{"a", "b"})[0][0]; got != 2 {
		t.Errorf("buildLcsTable identical 2-line = %d, want 2", got)
	}
	if got := buildLcsTable([]string{"a", "a"}, []string{"a"})[0][0]; got != 1 {
		t.Errorf("buildLcsTable duplicate = %d, want 1", got)
	}

	c := newHunkCollector(3)
	if c.contextLines != 3 || c.origStart != 0 || c.origLen != 0 || c.propStart != 0 || c.propLen != 0 || c.inHunk || c.trailingContext != 0 || c.origLine != 1 || c.propLine != 1 {
		t.Errorf("unexpected initial hunkCollector state: %+v", c)
	}

	// Flushing when not inHunk or when currentLines is empty is a no-op.
	c.currentLines = []string{" a"}
	c.flushHunk()
	if len(c.hunks) != 0 {
		t.Errorf("expected no hunks when !inHunk, got %d", len(c.hunks))
	}
	c.currentLines = nil
	c.inHunk = true
	c.flushHunk()
	if len(c.hunks) != 0 {
		t.Errorf("expected no hunks when currentLines is empty, got %d", len(c.hunks))
	}
}

func TestCountChanges(t *testing.T) {
	diff := `--- a/file.md
+++ b/file.md
@@ -1,3 +1,3 @@
 line 1
-old line
+new line
+another line
 line 3`
	additions, deletions := CountChanges(diff)
	if additions != 2 {
		t.Errorf("additions = %d, want 2", additions)
	}
	if deletions != 1 {
		t.Errorf("deletions = %d, want 1", deletions)
	}
}

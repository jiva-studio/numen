package service

import (
	"strings"
	"testing"
)

func TestCreateUnifiedDiffIdentical(t *testing.T) {
	content := "line 1\nline 2\nline 3"
	diffText, additions, deletions := CreateUnifiedDiff("notes/test.md", content, content)
	if diffText != "" {
		t.Errorf("diffText = %q, want empty", diffText)
	}
	if additions != 0 || deletions != 0 {
		t.Errorf("got +%d, -%d; want +0, -0", additions, deletions)
	}
}

func TestCreateUnifiedDiffNewFile(t *testing.T) {
	proposed := "line 1\nline 2\nline 3"
	diffText, additions, deletions := CreateUnifiedDiff("notes/test.md", "", proposed)
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

func TestCreateUnifiedDiffDeletedFile(t *testing.T) {
	original := "line 1\nline 2"
	diffText, additions, deletions := CreateUnifiedDiff("notes/test.md", original, "")
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

func TestCreateUnifiedDiffModifications(t *testing.T) {
	original := "apple\nbanana\ncherry\ndate"
	proposed := "apple\nblueberry\ncherry\nfig\ngrape"
	diffText, additions, deletions := CreateUnifiedDiff("notes/fruits.md", original, proposed)

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
}

func TestCountDiffChanges(t *testing.T) {
	diff := `--- a/file.md
+++ b/file.md
@@ -1,3 +1,3 @@
 line 1
-old line
+new line
+another line
 line 3`
	additions, deletions := CountDiffChanges(diff)
	if additions != 2 {
		t.Errorf("additions = %d, want 2", additions)
	}
	if deletions != 1 {
		t.Errorf("deletions = %d, want 1", deletions)
	}
}

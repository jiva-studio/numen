package markdown_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

func parseFile(t *testing.T, rel string) domain.Note {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(testsupport.VaultDir(t), rel))
	if err != nil {
		t.Fatalf("read testsupport.VaultDir(t): %v", err)
	}
	return markdown.Parse(domain.FileRef{Path: rel, Size: int64(len(raw))}, raw)
}

func TestFrontmatterIsKeptAsFound(t *testing.T) {
	n := parseFile(t, "notes/Entropy.md")

	if got := n.Frontmatter["status"]; got != "draft" {
		t.Errorf("status = %v, want draft", got)
	}
	// A key the application does not own must survive untouched: frontmatter is
	// shared with the user, not owned by us.
	if got := n.Frontmatter["custom-key-nobody-owns"]; got != "keep me verbatim" {
		t.Errorf("unknown key = %v, want it preserved verbatim", got)
	}
	if n.FrontmatterErr != "" {
		t.Errorf("unexpected frontmatter error: %s", n.FrontmatterErr)
	}
}

func TestTitlePreferenceOrder(t *testing.T) {
	if got := parseFile(t, "Thermodynamics.md").Title; got != "Thermodynamics" {
		t.Errorf("explicit title = %q", got)
	}
	// No title key: the first level-one heading stands in.
	if got := parseFile(t, "notes/Entropy.md").Title; got != "Entropy" {
		t.Errorf("title from heading = %q", got)
	}
	if got := parseFile(t, "daily/2026-08-15.md").Title; got != "Journal" {
		t.Errorf("title of a note with no frontmatter = %q", got)
	}
}

func TestTitleFallsBackToTheFilename(t *testing.T) {
	// What the user sees in a file manager, so it is never empty.
	n := markdown.Parse(domain.FileRef{Path: "notes/Some Note.md"}, []byte("no heading here\n"))
	if n.Title != "Some Note" {
		t.Errorf("title = %q, want the filename", n.Title)
	}
}

func TestHeadingsAreCollectedInOrder(t *testing.T) {
	n := parseFile(t, "Thermodynamics.md")
	var texts []string
	for _, h := range n.Headings {
		texts = append(texts, h.Text)
	}
	want := []string{"Thermodynamics", "First law", "Second law"}
	if !slices.Equal(texts, want) {
		t.Errorf("headings = %v, want %v", texts, want)
	}
	if n.Headings[0].Level != 1 || n.Headings[1].Level != 2 {
		t.Errorf("levels = %d, %d", n.Headings[0].Level, n.Headings[1].Level)
	}
	if !(n.Headings[0].Pos < n.Headings[1].Pos) {
		t.Error("positions are not in document order")
	}
}

func TestFencedCodeIsNotParsedAsContent(t *testing.T) {
	n := parseFile(t, "edge/code-fence.md")
	for _, h := range n.Headings {
		if h.Text == "Not a heading" {
			t.Error("a heading inside a code fence was collected")
		}
	}
}

func TestBrokenFrontmatterIsReportedNotFatal(t *testing.T) {
	n := parseFile(t, "edge/broken-frontmatter.md")
	if n.FrontmatterErr == "" {
		t.Fatal("invalid YAML was accepted silently")
	}
	// The body is still readable text, so it is still indexed. Refusing the file
	// would hide it from the user.
	if n.Body == "" {
		t.Error("body was dropped along with the broken frontmatter")
	}
	if n.Title != "Broken frontmatter" {
		t.Errorf("title = %q, want the heading", n.Title)
	}
}

func TestCarriageReturnsDoNotLeakIntoParsedValues(t *testing.T) {
	n := parseFile(t, "edge/crlf.md")
	if n.Title != "CRLF" {
		t.Errorf("title = %q, want CRLF with no trailing carriage return", n.Title)
	}
	if len(n.Headings) != 1 || n.Headings[0].Text != "CRLF" {
		t.Errorf("headings = %v", n.Headings)
	}
}

func TestNonLatinTextSurvivesParsing(t *testing.T) {
	// The product's own content is largely not Latin, so this is the one thing
	// the testsupport.VaultDir(t) keeps in another script deliberately.
	n := parseFile(t, "edge/unicode.md")
	for _, want := range []string{"энтропия", "熱力学"} {
		if !strings.Contains(n.Body, want) {
			t.Errorf("body lost %q", want)
		}
	}
}

func TestHorizontalRuleIsNotFrontmatter(t *testing.T) {
	raw := []byte("# Title\n\n---\n\nnot frontmatter\n")
	n := markdown.Parse(domain.FileRef{Path: "x.md"}, raw)
	if n.Frontmatter != nil {
		t.Errorf("a rule mid-document was read as frontmatter: %v", n.Frontmatter)
	}
	if n.Title != "Title" {
		t.Errorf("title = %q", n.Title)
	}
}

func TestUnterminatedFrontmatterLeavesTheFileAlone(t *testing.T) {
	raw := []byte("---\ntitle: x\n\nbody with no closing delimiter\n")
	n := markdown.Parse(domain.FileRef{Path: "x.md"}, raw)
	if n.Frontmatter != nil {
		t.Error("an unterminated block was treated as frontmatter")
	}
	if n.Body != string(raw) {
		t.Error("body was truncated")
	}
}

func TestEmptyFileIsANote(t *testing.T) {
	n := markdown.Parse(domain.FileRef{Path: "empty.md"}, nil)
	if n.Title != "empty" {
		t.Errorf("title = %q", n.Title)
	}
	if len(n.Headings) != 0 {
		t.Errorf("empty file produced %v", n.Headings)
	}
}

package searchquery_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

const entropyID = "01M02ACGM0FYMSXNDP29C90JNR"

func withLinks(body string, links ...string) string {
	if len(links) == 0 {
		return body
	}
	var b strings.Builder
	b.WriteString("---\nlinks:\n")
	for _, l := range links {
		role, to, _ := strings.Cut(l, " ")
		linkType := ""
		if before, after, found := strings.Cut(to, " type="); found {
			to, linkType = before, after
		}
		b.WriteString("  - to: \"" + to + "\"\n    role: " + role + "\n")
		if linkType != "" {
			b.WriteString("    type: " + linkType + "\n")
		}
	}
	b.WriteString("---\n")
	b.WriteString(body)
	return b.String()
}

func newLinkWorld(t *testing.T) world {
	t.Helper()
	w := newWorld(t, map[string]string{
		"Physics.md":        withLinks("Physics, the science of matter.\n", "child [[Mechanics]]"),
		"Mechanics.md":      "Mechanics, the motion of bodies.\n",
		"Thermo.md":         withLinks("Thermodynamics, heat and work.\n", "parent [[Physics]]"),
		"notes/Entropy.md":  "---\nid: " + entropyID + "\nlinks:\n  - to: \"[[Thermo]]\"\n    role: parent\n---\nEntropy, the measure of disorder.\n",
		"cycle/A.md":        withLinks("A.\n", "parent [[Physics]]", "parent [[B]]"),
		"cycle/B.md":        withLinks("B.\n", "parent [[A]]"),
		"by-name.md":        "See [[Entropy]].\n",
		"by-path.md":        "See [[notes/Entropy]].\n",
		"by-id.md":          "See [[note://" + entropyID + "]].\n",
		"by-block.md":       withLinks("Linked in the block.\n", "jump [[Entropy]] type=requires"),
		"unrelated.md":      "Nothing links here.\n",
		"Letters.md":        withLinks("Letters.\n", "attachment scans/1897-letter.pdf", "attachment https://example.org/letter"),
		"scans/index.md":    withLinks("Scans.\n", "attachment 1897-letter.pdf"),
		"reading/book.md":   "Read [[book.pdf]] tonight.\n",
		"Index.md":          withLinks("Index.\n", "attachment books/a.pdf"),
		"lonely/nothing.md": "Alone.\n",
	})
	for _, p := range []string{"scans/1897-letter.pdf", "reading/book.pdf", "books/a.pdf", "books/b.pdf"} {
		if err := w.db.Chunks().SaveSource(t.Context(), w.v.ID, chunk.Source{Path: p, Kind: "file", Size: 4, MTime: 1, ContentHash: "h-" + p}); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func (w world) newLinkFind() *searchquery.FindSearchResults {
	return w.newFind(nil)
}

func findPaths(t *testing.T, find *searchquery.FindSearchResults, w world, text string) []string {
	t.Helper()
	got, err := find.Execute(t.Context(), w.v, searchquery.Request{Text: text, Limit: 100})
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	paths := getPaths(got)
	slices.Sort(paths)
	return paths
}

func checkPaths(t *testing.T, w world, text string, want ...string) {
	t.Helper()
	got := findPaths(t, w.newLinkFind(), w, text)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s found %v, want %v", text, got, want)
	}
}

func TestHasLinkToFindsLinksByNamePathIdentifierAndBlock(t *testing.T) {
	w := newLinkWorld(t)
	checkPaths(t, w, `hasLinkTo("Entropy")`, "by-block.md", "by-id.md", "by-name.md", "by-path.md")
	checkPaths(t, w, `hasLinkTo("notes/Entropy.md")`, "by-block.md", "by-id.md", "by-name.md", "by-path.md")
	checkPaths(t, w, `hasLinkTo("[[Entropy|the measure]]")`, "by-block.md", "by-id.md", "by-name.md", "by-path.md")
	checkPaths(t, w, `hasLinkTo("note://`+entropyID+`")`, "by-block.md", "by-id.md", "by-name.md", "by-path.md")
}

func TestATypeArgumentKeepsOnlyLinksOfThatType(t *testing.T) {
	w := newLinkWorld(t)
	checkPaths(t, w, `hasLinkTo("Entropy", "requires")`, "by-block.md")
	checkPaths(t, w, `hasJumpTo("Entropy")`, "by-block.md")
	checkPaths(t, w, `hasReferenceTo("Entropy")`, "by-id.md", "by-name.md", "by-path.md")
	checkPaths(t, w, `hasJumpFrom("by-block")`, "notes/Entropy.md")
	checkPaths(t, w, `hasLinkFrom("by-name")`, "notes/Entropy.md")
}

func TestParentAndChildCountTheEdgeWrittenAtEitherEnd(t *testing.T) {
	w := newLinkWorld(t)
	checkPaths(t, w, `isChildOf("Physics")`, "Mechanics.md", "Thermo.md", "cycle/A.md")
	checkPaths(t, w, `isParentOf("Mechanics")`, "Physics.md")
	checkPaths(t, w, `isParentOf("Thermo")`, "Physics.md")
}

func TestDescendantsAndAncestorsFollowTheHierarchyAndEndOnACycle(t *testing.T) {
	w := newLinkWorld(t)
	checkPaths(t, w, `isDescendantOf("Physics")`, "Mechanics.md", "Thermo.md", "cycle/A.md", "cycle/B.md", "notes/Entropy.md")
	checkPaths(t, w, `isAncestorOf("Entropy")`, "Physics.md", "Thermo.md")
	checkPaths(t, w, `isAncestorOf("A")`, "Physics.md", "cycle/B.md")
}

func TestAHierarchyDeeperThanTheLimitIsRefused(t *testing.T) {
	w := newLinkWorld(t)
	find := w.newLinkFind()
	find.Limits.MaxLinkDepth = 2
	if paths := findPaths(t, find, w, `isAncestorOf("Entropy")`); len(paths) != 2 {
		t.Errorf("a chain of exactly 2 found %v, want 2 notes", paths)
	}
	find.Limits.MaxLinkDepth = 1
	_, err := find.Execute(t.Context(), w.v, searchquery.Request{Text: `isDescendantOf("Physics")`})
	if !errors.Is(err, searchquerydomain.ErrLimitExceeded) || !strings.Contains(err.Error(), "MaxLinkDepth") {
		t.Errorf("a hierarchy deeper than the limit returned %v, want ErrLimitExceeded naming MaxLinkDepth", err)
	}
}

func TestAttachmentsResolveByRootAndFolderPath(t *testing.T) {
	w := newLinkWorld(t)
	checkPaths(t, w, `hasAttachment("scans/1897-letter.pdf")`, "Letters.md", "scans/index.md")
	checkPaths(t, w, `hasAttachment("https://example.org/letter")`, "Letters.md")
	checkPaths(t, w, `isAttachedTo("Letters")`, "scans/1897-letter.pdf")
}

func TestHasLinkToAFileCountsAWikilinkToIt(t *testing.T) {
	w := newLinkWorld(t)
	checkPaths(t, w, `hasLinkTo("reading/book.pdf")`, "reading/book.md")
	checkPaths(t, w, `hasAttachment("reading/book.pdf")`)
	checkPaths(t, w, `hasLinkFrom("reading/book")`, "reading/book.pdf")
}

func TestANegatedLinkFunctionFiltersOtherDocuments(t *testing.T) {
	w := newLinkWorld(t)
	checkPaths(t, w, `!hasLinkFrom("Index") && kind == "pdf"`, "books/b.pdf", "reading/book.pdf", "scans/1897-letter.pdf")
	checkPaths(t, w, `!hasLinkTo("Entropy") && path.startsWith("by-")`)
}

func TestAnAmbiguousOrUnknownTargetIsAProblem(t *testing.T) {
	w := newWorld(t, map[string]string{
		"a/Entropy.md": "One.\n",
		"b/Entropy.md": "Two.\n",
		"c.md":         "See [[a/Entropy]].\n",
	})
	_, err := w.newLinkFind().Execute(t.Context(), w.v, searchquery.Request{Text: `hasLinkTo("Entropy")`})
	parseError, ok := errors.AsType[*searchquerydomain.ParseError](err)
	if !ok || len(parseError.Problems) != 1 || !strings.Contains(err.Error(), "a/Entropy.md, b/Entropy.md") {
		t.Fatalf("an ambiguous target returned %v, want a problem listing both notes", err)
	}
	if parseError.Problems[0].Position.Column != 1 {
		t.Errorf("the problem is at %+v, want the call", parseError.Problems[0].Position)
	}
	if paths := findPaths(t, w.newLinkFind(), w, `hasLinkTo("a/Entropy")`); !slices.Equal(paths, []string{"c.md"}) {
		t.Errorf("a path target found %v, want c.md", paths)
	}
	_, err = w.newLinkFind().Execute(t.Context(), w.v, searchquery.Request{Text: `hasLinkTo("Missing")`})
	if _, ok := errors.AsType[*searchquerydomain.ParseError](err); !ok || !strings.Contains(err.Error(), "Missing") {
		t.Errorf("an unknown target returned %v, want a problem naming it", err)
	}
}

func TestAMeaningAmongDescendantsFindsAFragmentThoughNearerOnesAreOutside(t *testing.T) {
	w := newWorld(t, map[string]string{
		"Physics.md":      withLinks("Physics.\n", "child [[Heat]]"),
		"Heat.md":         "The heat of the sun warms the stones. " + strings.Repeat("calm water ", 10),
		"outside/many.md": repeatSentences("near", 500),
	})
	w.cutBySize(t, "Heat.md", 40)
	w.cutBySize(t, "outside/many.md", 40)
	w.vectorize(t, "Heat.md", func(chunk.Record) []float32 { return newVector(dimensions / 4) })
	w.vectorize(t, "outside/many.md", func(chunk.Record) []float32 { return newVector(0) })
	find := w.newFind(texts{"heat": newVector(0)})
	find.Limits.CandidatesPerCondition = 5
	got, err := find.Execute(t.Context(), w.v, searchquery.Request{Text: `meaning("heat") && isDescendantOf("Physics")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 {
		t.Fatal("found nothing, want the passage of Heat.md")
	}
	for _, r := range got.Results {
		if r.Document.Path != "Heat.md" {
			t.Errorf("found %s outside the descendants", r.Document.Path)
		}
	}
}

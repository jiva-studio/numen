package searchquery_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func (w world) addFile(t *testing.T, filePath string, size int64) {
	t.Helper()
	if err := w.db.Chunks().SaveSource(t.Context(), w.v.ID, chunk.Source{Path: filePath, Kind: "file", Size: size, MTime: 1, ContentHash: "h-" + filePath}); err != nil {
		t.Fatal(err)
	}
}

func (w world) findPaths(t *testing.T, text string) []string {
	t.Helper()
	got, err := w.newFind(nil).Execute(t.Context(), w.v, searchquerydomain.Request{Text: text, Limit: 100})
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	paths := getPaths(got)
	slices.Sort(paths)
	return paths
}

func newPropertyWorld(t *testing.T) world {
	t.Helper()
	w := newWorld(t, map[string]string{
		"done.md":    "---\nstatus: done\nrating: 5\ndue: 2026-11-30\ndraft: true\ntopics: [physics, chemistry]\n---\nDone.",
		"open.md":    "---\nstatus: open\nrating: 3.5\ndue: 2026-12-02\ndraft: false\ntopics: physics\n---\nOpen.",
		"quoted.md":  "---\nstatus: Done\nrating: \"4\"\ndue: soon\ntopics: [biology]\nsource:\n---\nQuoted.",
		"plain.md":   "No frontmatter.",
		"empty.md":   "---\nsource: []\n---\nEmpty source.",
		"heading.md": "# Notes\n\n## summary\n\nText.",
		"weekly.md":  "## Summary of the week\n\nText.",
	})
	w.addFile(t, "books/a.pdf", 10)
	return w
}

func TestPropertyComparisonsSelectTheNotesTheStoredTypesPredict(t *testing.T) {
	w := newPropertyWorld(t)
	cases := map[string]string{
		`property("status") == "done"`:                  "done.md",
		`property("rating") >= 4`:                       "done.md",
		`property("due") < date("2026-12-01")`:          "done.md",
		`property("draft") == true`:                     "done.md",
		`"physics" in property("topics")`:               "done.md,open.md",
		`hasProperty("source")`:                         "empty.md,quoted.md",
		`property("status") in ["done", "open"]`:        "done.md,open.md",
		`hasHeading("Summary")`:                         "heading.md",
		`hasHeading("summary of the WEEK")`:             "weekly.md",
		`property("status") != "done" && kind == "pdf"`: "books/a.pdf",
	}
	for text, want := range cases {
		if got := strings.Join(w.findPaths(t, text), ","); got != want {
			t.Errorf("%s found %s, want %s", text, got, want)
		}
	}
	got := w.findPaths(t, `size >= 0 && property("status") != "done"`)
	if slices.Contains(got, "done.md") || !slices.Contains(got, "books/a.pdf") || !slices.Contains(got, "plain.md") {
		t.Errorf(`property("status") != "done" found %v, want every document but done.md`, got)
	}
}

func TestANoteTypeAndAModifiedRangeReadTheLocalZone(t *testing.T) {
	moscow := getMoscow()
	w := newWorldAt(t, map[string]string{
		"new.md":   "---\ntype: book\n---\nNew.",
		"late.md":  "---\ntype: book\n---\nLate.",
		"other.md": "An ordinary note.",
	}, map[string]time.Time{
		"new.md":   time.Date(2026, 1, 1, 0, 0, 0, 0, moscow),
		"late.md":  time.Date(2025, 12, 31, 23, 59, 0, 0, moscow),
		"other.md": time.Date(2026, 3, 1, 0, 0, 0, 0, moscow),
	})
	if got := strings.Join(w.findPaths(t, `noteType == "book" && modified >= date("2026-01-01")`), ","); got != "new.md" {
		t.Errorf("found %s, want new.md", got)
	}
	if got := strings.Join(w.findPaths(t, `noteType == "note"`), ","); got != "other.md" {
		t.Errorf("noteType == note found %s, want other.md", got)
	}
}

func TestATitlePrefixAmongManyDocumentsPassesTheDocumentLimit(t *testing.T) {
	w := newWorld(t, nil)
	for i := range 47 {
		w.addFile(t, fmt.Sprintf("books/Other %02d.pdf", i), 10)
	}
	for i := range 3 {
		w.addFile(t, fmt.Sprintf("books/Lecture %d.pdf", i), 10)
	}
	find := w.newFind(nil)
	find.Limits.MaxDocuments = 5
	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `title.startsWith("Lecture") && kind == "pdf"`})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(getPaths(got), ",") != "books/Lecture 0.pdf,books/Lecture 1.pdf,books/Lecture 2.pdf" {
		t.Errorf("found %v, want the three lectures", getPaths(got))
	}
}

func TestASizeComparisonSelectsDocumentsOfEveryKind(t *testing.T) {
	w := newWorld(t, map[string]string{"big.md": strings.Repeat("word ", 300), "small.md": "Small."})
	w.addFile(t, "books/big.pdf", 5000)
	w.addFile(t, "books/small.pdf", 10)
	if got := strings.Join(w.findPaths(t, `size > 1000`), ","); got != "big.md,books/big.pdf" {
		t.Errorf("found %s, want big.md and books/big.pdf", got)
	}
}

func TestAMeaningWithHasPropertyFindsItsChunkThoughNearerChunksLackTheKey(t *testing.T) {
	w := newWorld(t, map[string]string{
		"target.md": "---\nsource: lab\n---\nThe harbor at dusk, where the boats come home. " + strings.Repeat("calm water ", 10),
		"many.md":   repeatSentences("near", 500),
	})
	w.cutBySize(t, "target.md", 40)
	w.cutBySize(t, "many.md", 40)
	w.vectorize(t, "target.md", func(chunk.Record) []float32 { return newVector(dimensions / 4) })
	w.vectorize(t, "many.md", func(chunk.Record) []float32 { return newVector(0) })
	find := w.newFind(texts{"heat": newVector(0)})
	find.Limits.CandidatesPerCondition = 5
	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `meaning("heat") && hasProperty("source")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 {
		t.Fatal("found nothing, want the chunk of target.md")
	}
	for _, r := range got.Results {
		if r.Document.Path != "target.md" {
			t.Errorf("found %s, which has no source", r.Document.Path)
		}
	}
}

func TestUnreadableFrontmatterWarnsAPropertyQueryOnly(t *testing.T) {
	w := newWorld(t, map[string]string{
		"done.md":   "---\nstatus: done\n---\nDone.",
		"broken.md": "---\nstatus: [done\n---\nBroken.",
	})
	got, err := w.newFind(nil).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `property("status") == "done"`})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(getPaths(got), ",") != "done.md" || len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "1 notes have frontmatter that cannot be read") {
		t.Errorf("found %v with warnings %v, want done.md and one warning", getPaths(got), got.Warnings)
	}
	got, err = w.newFind(nil).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `title == "x"`})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 0 {
		t.Errorf("a title query warns %v", got.Warnings)
	}
}

func TestAPropertyKeyWithADotIsTheTopLevelKey(t *testing.T) {
	w := newWorld(t, map[string]string{
		"dot.md":    "---\na.b: 7\nit's: 7.5\n---\nDot.",
		"nested.md": "---\na:\n  b: 7\n---\nNested.",
	})
	if got := strings.Join(w.findPaths(t, `property("a.b") == 7 && property("it's") > 7`), ","); got != "dot.md" {
		t.Errorf("found %s, want dot.md", got)
	}
}

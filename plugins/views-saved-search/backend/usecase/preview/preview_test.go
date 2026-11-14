package preview_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
)

func TestPreviewReturnsNotesAboutTidesAndNoneAboutLighthouses(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Notes/Tide Tables.md", 0, 400, "The tide tables of the north quay."),
		chunk("c2", "Notes/Keeper.md", 0, 400, "The lighthouse keeper writes his diary."),
		chunk("c3", "Notes/Keeper Tides.md", 0, 400, "The lighthouse and the lighthouse lamp, and once a tide."),
		chunk("c4", "Notes/Spring Tides.md", 0, 400, "A spring tide and a neap tide pass the lighthouse."),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides") && !meaning("lighthouses")`})

	if got := getPaths(out); !slices.Equal(got, []string{"Notes/Tide Tables.md", "Notes/Spring Tides.md"}) {
		t.Errorf("results = %v, want the tide tables, then the note closer to tides than to lighthouses", got)
	}
}

func TestNegatedWordsDropsAResultWhoseChunkContainsTheWordPastItsExcerpt(t *testing.T) {
	filler := strings.Repeat("The quay is quiet at dawn. ", 110)
	long := "The tide turns at noon. " + filler + "The anchor chain rusts."
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/Anchor Log.md", 0, len(long), long),
		chunk("c2", "Logs/Tide Log.md", 0, 400, "The tide turns at dusk."),
	)
	found, err := w.search.With(testVault).SearchText(t.Context(), host.SearchResultQuery{Query: "tides", Mode: host.SearchModeMeaning})
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range found {
		if result.Path == "Logs/Anchor Log.md" && strings.Contains(result.Text, "anchor") {
			t.Fatalf("the fake search returned an excerpt of %d bytes that holds the word, want it cut to 1200 bytes", len(result.Text))
		}
	}

	out := w.run(t, api.PreviewInput{Query: `meaning("tides") && !words("anchor")`})

	if got := getPaths(out); !slices.Equal(got, []string{"Logs/Tide Log.md"}) {
		t.Errorf("results = %v, want only the tide log: the anchor log holds the word past its excerpt", got)
	}
}

func TestPreviewSearchesEachPositiveLeafInItsMode(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/Storm Log.md", 0, 400, "A storm and a tide at the harbor."))

	w.run(t, api.PreviewInput{Query: `meaning("tides") || (words("storm") && !words("anchor"))`})

	want := []host.SearchResultQuery{
		{Query: "tides", Mode: host.SearchModeMeaning, Limit: 100, ResultsPerSource: 100},
		{Query: "storm", Mode: host.SearchModeWords, IsRawWords: true, Limit: 100, ResultsPerSource: 100},
	}
	if got := w.host.getQueries(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("searches = %+v, want %+v", got, want)
	}
}

func TestPreviewScoresEveryCandidateAgainstEveryLeaf(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/Storm Log.md", 0, 400, "A storm at the harbor."),
		chunk("c2", "Logs/Tide Log.md", 0, 400, "The tide at the harbor."),
	)

	out := w.run(t, api.PreviewInput{Query: `words("harbor") && meaning("storms")`})

	if got := getPaths(out); !slices.Equal(got, []string{"Logs/Storm Log.md"}) {
		t.Errorf("results = %v, want the storm log: the tide log is far from storms", got)
	}
	if scored := w.host.getScoredChunks(); len(scored) != 1 || !slices.Contains(scored[0], "c2") {
		t.Errorf("scored chunks = %v, want both candidates in one call", scored)
	}
}

func indexKinds(w *world) {
	w.index(
		chunk("c1", "Notes/Tides.md", 0, 400, "A note on the tide."),
		chunk("c2", "Atlases/Tides.pdf", 0, 400, "A chart of the tide."),
		chunk("c3", "Books/Tides.epub", 0, 400, "A chapter on the tide."),
		chunk("c4", "Talks/Tides.mp3", 0, 400, "A talk about the tide."),
	)
}

func TestPreviewReturnsOnlyPdfsForKindPdf(t *testing.T) {
	w := newWorld(t)
	indexKinds(w)

	out := w.run(t, api.PreviewInput{Query: `kind == "pdf" && meaning("tides")`})

	if got := getPaths(out); !slices.Equal(got, []string{"Atlases/Tides.pdf"}) {
		t.Errorf("results = %v, want the PDF alone", got)
	}
	for _, query := range w.host.getQueries() {
		if !slices.Equal(query.Kinds, []domain.SourceKind{domain.KindFile}) {
			t.Errorf("search %+v, want the host asked for files only", query)
		}
	}
}

func TestPreviewReturnsOnlyNotesForKindNote(t *testing.T) {
	w := newWorld(t)
	indexKinds(w)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides") && kind == "note"`})

	if got := getPaths(out); !slices.Equal(got, []string{"Notes/Tides.md"}) {
		t.Errorf("results = %v, want the note alone", got)
	}
	for _, query := range w.host.getQueries() {
		if !slices.Equal(query.Kinds, []domain.SourceKind{domain.KindNote}) {
			t.Errorf("search %+v, want the host asked for notes only", query)
		}
	}
}

func TestPreviewReturnsTheKindsOfFilesNamedInAKindTermInsideAnOr(t *testing.T) {
	w := newWorld(t)
	indexKinds(w)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides") && (kind == "audio" || kind == "epub")`})

	got := getPaths(out)
	slices.Sort(got)
	if !slices.Equal(got, []string{"Books/Tides.epub", "Talks/Tides.mp3"}) {
		t.Errorf("results = %v, want the epub and the recording", got)
	}
}

func TestPreviewReturnsResultsOfBothFolders(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Atlases/North.pdf", 0, 400, "The tide in the north."),
		chunk("c2", "Logs/North.md", 0, 400, "The tide we logged."),
		chunk("c3", "Notes/North.md", 0, 400, "The tide in a note."),
	)

	out := w.run(t, api.PreviewInput{Query: `(path.startsWith("Atlases/") || path.startsWith("Logs/")) && meaning("tides")`})

	got := getPaths(out)
	slices.Sort(got)
	if !slices.Equal(got, []string{"Atlases/North.pdf", "Logs/North.md"}) {
		t.Errorf("results = %v, want the atlas and the log", got)
	}
	var prefixes []string
	for _, query := range w.host.getQueries() {
		prefixes = append(prefixes, query.PathPrefix)
	}
	if !slices.Equal(prefixes, []string{"Atlases/", "Logs/"}) {
		t.Errorf("searched prefixes = %v, want one search in each folder", prefixes)
	}
}

func TestPreviewWarnsWhenKindConditionsExcludeEveryKind(t *testing.T) {
	w := newWorld(t)
	indexKinds(w)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides") && kind == "note" && kind == "pdf"`})

	if len(out.Results) != 0 || !hasWarning(out, "kind conditions exclude every kind") {
		t.Errorf("output = %+v, want no result and the warning", out)
	}
}

func TestPreviewReturnsThePlaceTitleAndAddressOfAResult(t *testing.T) {
	w := newWorld(t)
	w.vault.File("Notes/Tides.md", "---\ntitle: Tide Tables\n---\nThe tide at the quay.")
	w.index(
		host.SearchResult{ChunkID: "c1", Path: "Atlases/Harbor Atlas.pdf", Text: "The tide of the harbor mouth.", Start: 1200, Length: 800, Selector: `{"page":4}`},
		host.SearchResult{ChunkID: "c2", Path: "Notes/Tides.md", Text: "The tide at the quay.", Start: 0, Length: 21, MatchLine: 2},
		host.SearchResult{ChunkID: "c3", Path: "Talks/Harbor Talk.mp3", Text: "Today the tide is low.", Start: 27, Length: 32, Selector: `{"t":[4.5,9]}`},
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	byPath := map[string]api.PreviewResult{}
	for _, result := range out.Results {
		byPath[result.Path] = result
	}
	pdf, note, talk := byPath["Atlases/Harbor Atlas.pdf"], byPath["Notes/Tides.md"], byPath["Talks/Harbor Talk.mp3"]
	if pdf.Title != "Harbor Atlas" || pdf.Place == nil || pdf.Place.Page != 4 || pdf.Start != 1200 || pdf.Length != 800 || pdf.Line != 1 || pdf.ChunkCount != 1 {
		t.Errorf("the PDF result is %+v, want the title of the file, page 4 and line 1", pdf)
	}
	if want := "numen:" + url.PathEscape("Atlases/Harbor Atlas.pdf") + "?start=1200&length=800"; pdf.URI != want {
		t.Errorf("the address is %q, want %q", pdf.URI, want)
	}
	if note.Title != "Tide Tables" || note.Place != nil || note.Line != 3 || note.Text != "The tide at the quay." {
		t.Errorf("the note result is %+v, want the title of the note, no place and line 3", note)
	}
	if talk.Title != "Harbor Talk" || talk.Place == nil || talk.Place.PositionSeconds == nil || *talk.Place.PositionSeconds != 4.5 {
		t.Errorf("the recording result is %+v, want 4.5 seconds", talk)
	}
}

func TestPreviewReturnsTheFloorAndTheNumberOfCandidates(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/One.md", 0, 400, "The tide at one."),
		chunk("c2", "Logs/Two.md", 0, 400, "The tide at two."),
		chunk("c3", "Logs/Three.md", 0, 400, "The lighthouse at three."),
	)

	out := w.run(t, api.PreviewInput{Query: `words("at") && meaning("tides")`})

	if out.Floor != 0.4 || out.CandidateCount != 3 || len(out.Results) != 2 || out.Query != `words("at") && meaning("tides")` {
		t.Errorf("output = %+v, want the floor 0.4, three candidates and two results", out)
	}
}

func TestPreviewWithMeaningFailsWithoutAModel(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/One.md", 0, 400, "The tide at one."))
	w.search.RemoveModel()

	_, err := w.execute(t.Context(), api.PreviewInput{Query: `words("tide") && !meaning("lighthouses")`})

	if err == nil || !strings.Contains(err.Error(), "the query uses meaning, and the installation has no embedding model") {
		t.Errorf("Execute = %v, want the error of a missing model", err)
	}
}

func TestPreviewWithWordsRunsWithoutAModel(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/One.md", 0, 400, "The tide at one."), chunk("c2", "Logs/Two.md", 0, 400, "A storm at two."))
	w.search.RemoveModel()

	out := w.run(t, api.PreviewInput{Query: `words("tide") && kind == "note"`})

	if got := getPaths(out); !slices.Equal(got, []string{"Logs/One.md"}) || out.Floor != 0 {
		t.Errorf("output = %+v, want the note with the word and no floor", out)
	}
}

func TestPreviewReportsAnInvalidWordsQueryWithItsPosition(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/One.md", 0, 400, "The tide at one."))
	w.host.refusedWords = "harbor NEAR("

	out, err := w.execute(t.Context(), api.PreviewInput{Query: "meaning(\"tides\") ||\n words(\"harbor NEAR(\")"})

	if err != nil || len(out.Errors) != 1 || len(out.Results) != 0 {
		t.Fatalf("Execute = %+v, %v, want one problem of the query and no result", out, err)
	}
	problem := out.Errors[0]
	if problem.Line != 2 || problem.Column != 7 || !strings.Contains(problem.Message, `is not a valid full-text query`) {
		t.Errorf("problem = %+v, want the words leaf at 2:7", problem)
	}
}

func TestPreviewNamesTheNegatedWordsLeafTheScoresRefuse(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/One.md", 0, 400, "The tide at one."))
	w.host.refusedWords = "NEAR("

	out, err := w.execute(t.Context(), api.PreviewInput{Query: `meaning("tides") && !words("anchor") && !words("NEAR(")`})

	if err != nil || len(out.Errors) != 1 {
		t.Fatalf("Execute = %+v, %v, want one problem of the query", out, err)
	}
	if problem := out.Errors[0]; problem.Line != 1 || problem.Column != 47 || !strings.Contains(problem.Message, `NEAR(`) {
		t.Errorf("problem = %+v, want the second words leaf at 1:47, not the first", problem)
	}
}

func TestPreviewReturnsTheProblemsOfAQueryThatDoesNotCompile(t *testing.T) {
	w := newWorld(t)

	out, err := w.execute(t.Context(), api.PreviewInput{Query: `meaning("tides") &&`})

	if err != nil || len(out.Errors) == 0 || len(out.Results) != 0 {
		t.Fatalf("Execute = %+v, %v, want the problems in the output", out, err)
	}
	if problem := out.Errors[0]; problem.Line != 1 || problem.Column != 20 || problem.Message == "" {
		t.Errorf("problem = %+v, want a message at 1:20", problem)
	}
	if len(w.host.getQueries()) != 0 {
		t.Errorf("searches = %+v, want none for a query that does not compile", w.host.getQueries())
	}
}

func TestPreviewDropsACandidateTheScoresLeaveOut(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/One.md", 0, 400, "The tide at one."), chunk("c2", "Logs/Two.md", 0, 400, "The tide at two."))
	w.host.forgotten["c2"] = true

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getPaths(out); !slices.Equal(got, []string{"Logs/One.md"}) {
		t.Errorf("results = %v, want the chunk the index still holds", got)
	}
}

func TestPreviewReturnsTheErrorOfACanceledContext(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/One.md", 0, 400, "The tide at one."))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := w.execute(ctx, api.PreviewInput{Query: `meaning("tides")`}); !errors.Is(err, context.Canceled) {
		t.Errorf("Execute = %v, want context.Canceled", err)
	}
}

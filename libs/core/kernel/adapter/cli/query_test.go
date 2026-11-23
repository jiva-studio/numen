package cli_test

import (
	"context"
	"path"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/operationresult"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery/searchquerytest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type fixedQueryFloor struct{}

func (fixedQueryFloor) GetFloor(context.Context, domain.Vault, string) float64 { return 0.3 }

type markdownTypes struct{}

func (markdownTypes) GetContentType(name string) domain.ContentType {
	if path.Ext(name) == ".md" {
		return "text/markdown"
	}
	return domain.UnknownContentType
}

func (markdownTypes) ListExtensions(domain.ContentType) []string { return nil }

func newQueryDeps(t *testing.T) stubSession {
	t.Helper()
	return newQuerySession(t, func(*index.DB, *searchquerytest.Deps) {})
}

func newQuerySession(t *testing.T, change func(db *index.DB, deps *searchquerytest.Deps)) stubSession {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.GetPath(t), index.DisableSync())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	notes := map[string]string{
		"notes/Harbor.md": "The tide comes into the harbor twice a day.",
		"notes/Moon.md":   "The moon pulls the tide, and the tide follows the moon.",
		"notes/Field.md":  "A field of wheat under a dry wind.",
		"notes/Forest.md": "Old oaks and the smell of rain in the forest.",
		"notes/Hill.md":   "Sheep graze on the hill above the village.",
		"notes/Sea.md":    "Salt water and gulls. See [[Harbor]].",
	}
	v := vaulttest.NewVault(t, notes)
	v.Name = "shelf"
	scan := indexing.NewScan(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), testsupport.EmptySourceBacklog{})
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	for p, text := range notes {
		whole := chunk.Chunk{Start: 0, Length: len(text), Text: text, Small: []chunk.Chunk{{Start: 0, Length: len(text), Text: text}}}
		if err := db.Chunks().ReplaceChunks(t.Context(), v.ID, "note", p, []chunk.Chunk{whole}); err != nil {
			t.Fatal(err)
		}
	}
	search := db.ChunkSearch()
	deps := searchquerytest.Deps{
		Parser: celquery.NewParser(searchquerydomain.DefaultLimits()), Clock: testsupport.NewClock(),
		Evaluate: searchquery.EvaluateQueryDeps{
			Documents: search, Vectors: search, Places: search, Matcher: wordmatch.NewMatcher(), Readers: vaultreader.Readers{},
			Floors:       fixedQueryFloor{},
			ContentTypes: markdownTypes{}, Contents: db.NoteQueryReads(), Links: db.NoteQueryReads(),
		},
	}
	change(db, &deps)
	find := searchquerytest.New(deps)
	return stubSession{
		Vaults: createVaultList(v),
		Indexed: func(context.Context, cli.IndexOptions) (cli.Indexed, error) {
			return cli.Indexed{Query: searchquery.NewFindSearchResultsByName(
				singleVaultList{v},
				func(context.Context, domain.Vault) (*searchquery.FindSearchResults, error) { return find, nil },
			), OperationResults: operationresult.NewClearOperationResults(db.GetOperationResults(100))}, nil
		},
	}
}

func TestQueryPrintsEachResultWithItsScoreKindPlaceAndExcerpt(t *testing.T) {
	out, stderr, err := runWithStderr(t, newQueryDeps(t), "query", "shelf", `words("tide")`)
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("printed %q, want two blocks of two lines", out)
	}
	if !strings.HasPrefix(lines[0], "0.") || !strings.Contains(lines[0], "  notes/Moon.md  note  line 1  meaning -  words 0.") {
		t.Errorf("the first result line is %q", lines[0])
	}
	if lines[1] != "    The moon pulls the tide, and the tide follows the moon." {
		t.Errorf("the first excerpt is %q", lines[1])
	}
	if !strings.Contains(lines[2], "  notes/Harbor.md  note  line 1  ") {
		t.Errorf("the second result line is %q", lines[2])
	}
}

func TestQueryReportsAMisspelledFunctionWithItsPlaceAndASuggestion(t *testing.T) {
	_, stderr, err := runWithStderr(t, newQueryDeps(t), "query", "shelf", `meanin("a")`)
	if err == nil {
		t.Fatal("a query that is not valid succeeded")
	}
	if !strings.Contains(stderr, "query:1:1: Unknown function meanin: did you mean meaning?") {
		t.Errorf("stderr is %q", stderr)
	}
}

func TestQueryPrintsAtMostTheLimitAndRefusesALimitThatIsNotACount(t *testing.T) {
	out, stderr, err := runWithStderr(t, newQueryDeps(t), "query", "--limit", "1", "shelf", `words("tide")`)
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	if lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n"); len(lines) != 2 {
		t.Errorf("printed %q, want one result", out)
	}
	_, _, err = runWithStderr(t, newQueryDeps(t), "query", "--limit", "x", "shelf", `words("tide")`)
	if err == nil || !strings.Contains(err.Error(), "usage: numen-cli query") {
		t.Errorf("--limit x returned %v, want the usage", err)
	}
}

func TestHelpListsTheQueryCommand(t *testing.T) {
	out, err := runWithDeps(stubSession{}, "help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "numen-cli query [--limit N] <vault> <query>  run a search query within one vault") {
		t.Errorf("help is %q", out)
	}
}

func TestQueryFindsTheNotesThatLinkToANote(t *testing.T) {
	out, stderr, err := runWithStderr(t, newQueryDeps(t), "query", "shelf", `hasLinkTo("Harbor")`)
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	if !strings.Contains(out, "notes/Sea.md  note") || strings.Contains(out, "notes/Moon.md") {
		t.Errorf("printed %q, want notes/Sea.md alone", out)
	}
}

func TestQueryReportsAnUnknownLinkTarget(t *testing.T) {
	_, stderr, err := runWithStderr(t, newQueryDeps(t), "query", "shelf", `hasLinkTo("Lighthouse")`)
	if err == nil || !strings.Contains(stderr, "query:1:1: no note or indexed file answers to the target: Lighthouse") {
		t.Errorf("returned %v, stderr %q", err, stderr)
	}
}

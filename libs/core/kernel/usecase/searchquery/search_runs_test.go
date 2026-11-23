package searchquery_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	searchrundomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

const (
	owner = "md.numen.agents.tools.vault"
	other = "md.numen.tools.saved.search"
)

func TestAStartReturnsBeforeTheSearchEndsAndAReadWaitsForTheEnd(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	q := w.hold(`words("tide")`)
	q.results = getResults("a.md", "b.md")
	id := w.start(t, owner, `words("tide")`)
	<-w.index.started

	read := make(chan searchquery.Page, 1)
	go func() {
		page, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id, Wait: time.Second})
		if err != nil {
			t.Error(err)
		}
		read <- page
	}()
	<-w.clock.asked
	w.clock.Advance(time.Second)
	if page := <-read; !page.Record.IsRunning {
		t.Fatalf("after one second the search is %+v, want running", page.Record)
	}
	close(q.release)
	page := w.readFinalPage(t, owner, id)
	if page.Record.IsRunning || !slices.Equal(getPagePaths(page), []string{"a.md", "b.md"}) || page.TotalResults != 2 {
		t.Fatalf("page %+v, want the two results", page)
	}
}

func TestAQueryThatFailsEndsWithItsProblemsAndNoResults(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	w.answer("mening(\"x\")", &runQuery{err: &searchquerydomain.ParseError{Problems: []searchquerydomain.Problem{
		{Position: searchquerydomain.Position{Line: 1, Column: 1}, Message: "unknown function mening", Suggestion: "meaning"},
	}}})
	w.answer(`words("x").rerank()`, &runQuery{err: fmt.Errorf("1:12: %w", errors.New("no reranker")), results: getResults("a.md")})

	parsed := w.readFinalPage(t, owner, w.start(t, owner, "mening(\"x\")"))
	reranked := w.readFinalPage(t, owner, w.start(t, owner, `words("x").rerank()`))

	want := []searchrundomain.Problem{{Line: 1, Column: 1, Message: "unknown function mening", Suggestion: "meaning"}}
	if !slices.Equal(parsed.Record.Errors, want) || parsed.TotalResults != 0 {
		t.Errorf("parse problems %+v with %d results, want %+v", parsed.Record.Errors, parsed.TotalResults, want)
	}
	if len(reranked.Record.Errors) != 1 || reranked.Record.Errors[0].Message != "1:12: no reranker" || reranked.TotalResults != 0 {
		t.Errorf("rerank problems %+v with %d results", reranked.Record.Errors, reranked.TotalResults)
	}
}

func TestAnEmptyQueryEndsAtOnceWithAProblem(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	page, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: w.start(t, owner, "  ")})
	if err != nil || page.Record.IsRunning || len(page.Record.Errors) != 1 {
		t.Fatalf("page %+v (%v), want a stopped search with one problem", page.Record, err)
	}
}

func TestAnotherPluginCannotReadOrRemoveASearch(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	id := w.start(t, owner, `words("tide")`)

	if _, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: other, ID: id}); !errors.Is(err, searchrundomain.ErrNotFound) {
		t.Errorf("read by another plugin: %v, want ErrNotFound", err)
	}
	if err := w.runs.Remove.Execute(t.Context(), other, id); !errors.Is(err, searchrundomain.ErrNotFound) {
		t.Errorf("remove by another plugin: %v, want ErrNotFound", err)
	}
	if _, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id}); err != nil {
		t.Errorf("read by its plugin: %v", err)
	}
}

func TestGetSearchReturnsNotFoundTenMinutesAfterTheLastRead(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	id := w.start(t, owner, `words("tide")`)
	w.readFinalPage(t, owner, id)

	w.clock.Advance(9 * time.Minute)
	if _, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id}); err != nil {
		t.Fatalf("read at minute 9: %v", err)
	}
	w.clock.Advance(9 * time.Minute)
	if _, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id}); err != nil {
		t.Fatalf("read at minute 18: %v", err)
	}
	w.clock.Advance(10 * time.Minute)
	if _, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id}); !errors.Is(err, searchrundomain.ErrNotFound) {
		t.Fatalf("read at minute 28: %v, want ErrNotFound", err)
	}
}

func TestTheTwentyFirstSearchDropsTheOneReadLeastRecently(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	var ids []searchrundomain.ID
	for i := range 20 {
		id := w.start(t, owner, fmt.Sprintf("words(\"tide%d\")", i))
		w.readFinalPage(t, owner, id)
		ids = append(ids, id)
		w.clock.Advance(time.Second)
	}
	w.start(t, owner, `words("sea")`)

	if _, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: ids[0]}); !errors.Is(err, searchrundomain.ErrNotFound) {
		t.Errorf("the oldest search: %v, want ErrNotFound", err)
	}
	if _, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: ids[1]}); err != nil {
		t.Errorf("the second search: %v", err)
	}
}

func TestTwentyRunningSearchesRefuseAnother(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	for i := range 20 {
		w.hold(fmt.Sprintf("words(\"tide%d\")", i))
		w.start(t, owner, fmt.Sprintf("words(\"tide%d\")", i))
	}
	_, err := w.runs.Start.Execute(t.Context(), searchrundomain.Start{Owner: owner, Vault: getTideVault(), Query: `words("sea")`})
	if !errors.Is(err, searchrundomain.ErrTooManySearches) {
		t.Fatalf("start: %v, want ErrTooManySearches", err)
	}
}

func TestAnEqualRunningSearchOfThePluginIsShared(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	q := w.hold(`words("tide")`)
	first := w.start(t, owner, `words("tide")`)
	if again := w.start(t, owner, ` words("tide") `); again != first {
		t.Errorf("an equal start while running is %s, want %s", again, first)
	}
	if another := w.start(t, other, `words("tide")`); another == first {
		t.Error("another plugin shares the search")
	}
	close(q.release)
	w.readFinalPage(t, owner, first)
	if after := w.start(t, owner, `words("tide")`); after == first {
		t.Error("an equal start after the end reuses the finished search")
	}
}

func TestAnInteractiveSearchRunsBeforeWaitingBackgroundOnes(t *testing.T) {
	policy := searchrundomain.DefaultPolicy()
	policy.MaxRunning = 1
	w := newRunWorld(t, policy)
	first := w.hold("words(\"first\")")
	w.start(t, owner, "words(\"first\")")
	<-w.index.started
	for _, text := range []string{"words(\"b1\")", "words(\"b2\")"} {
		if _, err := w.runs.Start.Execute(t.Context(), searchrundomain.Start{
			Owner: owner, Vault: getTideVault(), Query: text, Priority: modelcall.PriorityBackground, IsBackground: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	interactive := w.start(t, owner, "words(\"person\")")
	time.Sleep(10 * time.Millisecond)
	close(first.release)
	w.readFinalPage(t, owner, interactive)
	for range 3 {
		<-w.index.started
	}
	if ran := w.index.getRan(); !slices.Equal(ran[:2], []string{"words(\"first\")", "words(\"person\")"}) {
		t.Fatalf("order %v, want the interactive search second", ran)
	}
}

func TestASearchForAPersonShowsATaskThatCancelsIt(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	w.hold(`words("tide")`)
	id := w.start(t, owner, `words("tide")`)
	<-w.index.started
	shown := w.tasks.list()
	if len(shown) != 1 || shown[0].PluginName != "Plugin "+owner || shown[0].About != `words("tide")` {
		t.Fatalf("tasks %+v, want one of the plugin", shown)
	}

	shown[0].Cancel()

	page := w.readFinalPage(t, owner, id)
	if len(page.Record.Errors) != 1 || page.Record.Errors[0].Message != "the search was canceled" {
		t.Fatalf("problems %+v, want the cancellation", page.Record.Errors)
	}
	if left := w.tasks.list(); len(left) != 0 {
		t.Errorf("tasks %+v after the end", left)
	}
}

func TestABackgroundSearchShowsNoTask(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	w.hold(`words("tide")`)
	if _, err := w.runs.Start.Execute(t.Context(), searchrundomain.Start{
		Owner: owner, Vault: getTideVault(), Query: `words("tide")`, IsBackground: true,
	}); err != nil {
		t.Fatal(err)
	}
	<-w.index.started
	if shown := w.tasks.list(); len(shown) != 0 {
		t.Fatalf("tasks %+v, want none", shown)
	}
}

func TestAChangeOfTheIndexMakesAFinishedSearchStale(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	id := w.start(t, owner, `words("tide")`)
	before := w.readFinalPage(t, owner, id)
	w.generations.BumpGeneration(getTideVault().ID)
	after, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id})
	if err != nil || before.IsStale || !after.IsStale || after.Record.IndexGeneration != before.Record.IndexGeneration {
		t.Fatalf("stale before %v, after %v (%v), generations %q and %q", before.IsStale, after.IsStale, err,
			before.Record.IndexGeneration, after.Record.IndexGeneration)
	}
}

func TestAReadDuringTheRerankReturnsThePartialResults(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	q := w.hold(`words("tide").rerank()`)
	partial := getResults("a.md", "b.md")
	q.partial = &partial
	q.results = getResults("b.md", "a.md")
	id := w.start(t, owner, `words("tide").rerank()`)
	<-w.index.started

	during, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id})
	if err != nil || !during.Record.IsPartial || !slices.Equal(getPagePaths(during), []string{"a.md", "b.md"}) || during.NextPageToken != "" {
		t.Fatalf("during the rerank %+v (%v), want the partial results", during, err)
	}
	close(q.release)
	after := w.readFinalPage(t, owner, id)
	if after.Record.IsPartial || !slices.Equal(getPagePaths(after), []string{"b.md", "a.md"}) {
		t.Fatalf("after the rerank %+v, want the reranked results", after)
	}
}

func TestGetSearchReturnsPagesOfTwentyAndRefusesATokenOfAnotherSearch(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	var paths []string
	for i := range 45 {
		paths = append(paths, fmt.Sprintf("n%02d.md", i))
	}
	w.answer(`words("tide")`, &runQuery{results: getResults(paths...)})
	id := w.start(t, owner, `words("tide")`)
	w.readFinalPage(t, owner, id)

	var sizes []int
	token := ""
	for {
		page, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: id, PageSize: 20, PageToken: token})
		if err != nil {
			t.Fatal(err)
		}
		sizes = append(sizes, len(page.Record.Results.Results))
		if token = page.NextPageToken; token == "" {
			break
		}
	}
	if !slices.Equal(sizes, []int{20, 20, 5}) {
		t.Fatalf("pages %v, want [20 20 5]", sizes)
	}
	second := w.start(t, owner, `words("sea")`)
	w.readFinalPage(t, owner, second)
	_, err := w.runs.Get.Execute(t.Context(), searchquery.Read{Owner: owner, ID: second, PageToken: searchrundomain.EncodePageToken(id, 20)})
	if !errors.Is(err, searchrundomain.ErrInvalidPageToken) {
		t.Fatalf("a token of another search: %v, want ErrInvalidPageToken", err)
	}
}

func TestRemovingARunningSearchCancelsItsQuery(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	w.hold(`words("tide")`)
	id := w.start(t, owner, `words("tide")`)
	<-w.index.started

	if err := w.runs.Remove.Execute(t.Context(), owner, id); err != nil {
		t.Fatal(err)
	}

	select {
	case <-w.index.canceled:
	case <-time.After(10 * time.Second):
		t.Fatal("the query was not canceled")
	}
	if _, err := w.runs.Get.Execute(context.WithoutCancel(t.Context()), searchquery.Read{Owner: owner, ID: id}); !errors.Is(err, searchrundomain.ErrNotFound) {
		t.Fatalf("read after removal: %v, want ErrNotFound", err)
	}
}

func TestListSearchFunctionsReturnsTheCatalogAndItsRevision(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	catalog, revision := w.runs.Functions.Execute()
	if len(catalog.Functions) != len(searchquerydomain.GetBuiltinCatalog().Functions) || revision != searchquerydomain.GetCatalogRevision(catalog) {
		t.Fatalf("catalog of %d functions, revision %q", len(catalog.Functions), revision)
	}
}

func TestASearchRunsTheQueryOnTheGenerationItStartedOn(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	id := w.start(t, owner, `words("tide")`)
	page := w.readFinalPage(t, owner, id)

	w.index.mu.Lock()
	defer w.index.mu.Unlock()
	if len(w.index.generations) != 1 || w.index.generations[0] == "" || w.index.generations[0] != page.Record.IndexGeneration {
		t.Errorf("the query ran on generations %v, want %q", w.index.generations, page.Record.IndexGeneration)
	}
}

func TestRemovingASharedSearchKeepsItForTheOtherStarter(t *testing.T) {
	w := newRunWorld(t, searchrundomain.DefaultPolicy())
	q := w.hold(`words("tide")`)
	q.results = getResults("a.md")
	first := w.start(t, owner, `words("tide")`)
	second := w.start(t, owner, `words("tide")`)
	if second != first {
		t.Fatalf("the second start is %s, want %s", second, first)
	}

	if err := w.runs.Remove.Execute(t.Context(), owner, first); err != nil {
		t.Fatal(err)
	}
	close(q.release)

	page := w.readFinalPage(t, owner, second)
	if !slices.Equal(getPagePaths(page), []string{"a.md"}) {
		t.Errorf("the other starter reads %v, want a.md", getPagePaths(page))
	}
	if err := w.runs.Remove.Execute(t.Context(), owner, second); err != nil {
		t.Fatal(err)
	}
	if err := w.runs.Remove.Execute(t.Context(), owner, second); !errors.Is(err, searchrundomain.ErrNotFound) {
		t.Errorf("the last remove left the search: %v, want ErrNotFound", err)
	}
}

package hostendpoint

import (
	"context"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchslot"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchstore"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type noTasks struct{}

func (noTasks) Set(work.Task) {}
func (noTasks) Remove(string) {}

// queriedVaults is the vaults the queries of a test ran over.
type queriedVaults struct {
	vaults chan domain.VaultID
}

func (q queriedVaults) run(_ context.Context, v domain.Vault, _ searchquery.Request) (searchquery.SearchResults, error) {
	q.vaults <- v.ID
	score := 0.75
	return searchquery.SearchResults{
		Results: []searchquery.SearchResult{
			{
				Document: searchquery.Document{Path: "books/Tides.pdf", Title: "Tides", Kind: "pdf"},
				Chunks:   []searchquery.ChunkRange{{ID: "c1", Start: 10, Length: 20}}, Start: 10, Length: 20,
				Text: "the tide turns", PlaceLabel: "p. 3", MatchLine: 4, Selector: `{"page":3}`, Score: &score,
			},
			{Document: searchquery.Document{Path: "notes/Sea.md", Title: "Sea", Kind: "note"}, Text: "# Sea"},
		},
		ScoreKind: searchquery.ScoreRerank, Floor: 0.3, Warnings: []string{"a warning"},
		Reranks: []searchquery.RerankReport{{ConnectionName: "Local", SentCount: 2, ReusedCount: 1}},
	}, nil
}

func newRunCallers(t *testing.T) (*callers, queriedVaults) {
	t.Helper()
	q := queriedVaults{vaults: make(chan domain.VaultID, 8)}
	return newCallersRunning(q.run), q
}

func newCallersRunning(run searchqueryusecase.RunQuery) *callers {
	clock := testsupport.NewClock()
	runs := searchqueryusecase.NewSearchRuns(searchqueryusecase.Deps{
		Store: searchstore.New(), Cancels: searchcancel.NewSearches(clock), Tasks: noTasks{}, Clock: clock, CreateTimer: clock.StartTimer,
		Policy: searchrun.DefaultPolicy(), Slots: searchslot.New(searchrun.DefaultPolicy().MaxRunning),
		Generations: indexgeneration.New("epoch"), Run: run, Names: func(id string) string { return id },
		GetCatalog: searchquery.GetBuiltinCatalog, MaxResults: 100,
	})
	fakeCallers := newCallers()
	fakeCallers.vault = vaultOf(newMemoryVault(nil))
	fakeCallers.search = &SearchAccess{Runs: &runs}
	return fakeCallers
}

func TestTheSearchCallsNeedIndexRead(t *testing.T) {
	fakeCallers, _ := newRunCallers(t)
	fakeCallers.confined = true
	client := newSearchClient(t, fakeCallers, "token-a")
	ctx := t.Context()
	_, startErr := client.StartSearch(ctx, connect.NewRequest(&v1.StartSearchRequest{Query: `words("tide")`}))
	_, getErr := client.GetSearch(ctx, connect.NewRequest(&v1.GetSearchRequest{Id: "x"}))
	_, removeErr := client.DeleteSearch(ctx, connect.NewRequest(&v1.DeleteSearchRequest{Id: "x"}))
	_, listErr := client.ListSearchFunctions(ctx, connect.NewRequest(&v1.ListSearchFunctionsRequest{}))
	for name, err := range map[string]error{"start": startErr, "get": getErr, "remove": removeErr, "list": listErr} {
		if readCode(t, err) != connect.CodePermissionDenied {
			t.Errorf("%s: %v, want PermissionDenied", name, err)
		}
	}
	if !slices.Contains(fakeCallers.checked, "md.numen.a "+manifest.PermissionIndexRead) {
		t.Errorf("checked %v, want index:read", fakeCallers.checked)
	}
}

func TestASearchRoundTripsItsStateAndAnotherPluginCannotReadIt(t *testing.T) {
	fakeCallers, q := newRunCallers(t)
	client := newSearchClient(t, fakeCallers, "token-a")
	ctx := t.Context()
	started, err := client.StartSearch(ctx, connect.NewRequest(&v1.StartSearchRequest{Query: `words("tide").rerank()`, Priority: v1.Priority_PRIORITY_INTERACTIVE}))
	if err != nil || started.Msg.GetError() != 0 || started.Msg.GetId() == "" {
		t.Fatalf("start %v (%v)", started.Msg, err)
	}
	if v := <-q.vaults; v != "v1" {
		t.Errorf("the query ran over %s, want v1", v)
	}

	got, err := client.GetSearch(ctx, connect.NewRequest(&v1.GetSearchRequest{Id: started.Msg.GetId(), WaitMs: 30000}))

	state := got.Msg.GetSearch()
	if err != nil || state.GetIsRunning() || state.GetTotalResults() != 2 || state.GetScoreKind() != "rerank" || state.GetFloor() != 0.3 ||
		state.GetIndexGeneration() != "epoch.0" || state.GetProgress().GetStep() != "done" || len(state.GetReranks()) != 1 ||
		state.GetReranks()[0].GetReusedCount() != 1 || !slices.Equal(state.GetWarnings(), []string{"a warning"}) {
		t.Fatalf("state %v (%v)", state, err)
	}
	passage, whole := state.GetResults()[0], state.GetResults()[1]
	if passage.GetByteRange().GetEnd() != 30 || passage.GetKind() != "pdf" || passage.GetTitle() != "Tides" || passage.GetScore() != 0.75 ||
		passage.GetSelector() != `{"page":3}` || passage.GetIsWholeFile() {
		t.Errorf("passage %v", passage)
	}
	if !whole.GetIsWholeFile() || whole.GetByteRange() != nil || whole.Score != nil || whole.GetText() != "# Sea" {
		t.Errorf("whole file %v", whole)
	}
	other := newSearchClient(t, fakeCallers, "token-c")
	read, err := other.GetSearch(ctx, connect.NewRequest(&v1.GetSearchRequest{Id: started.Msg.GetId()}))
	if err != nil || read.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_SEARCH_NOT_FOUND {
		t.Errorf("read by another plugin %v (%v), want SEARCH_NOT_FOUND", read.Msg, err)
	}
	removed, err := client.DeleteSearch(ctx, connect.NewRequest(&v1.DeleteSearchRequest{Id: started.Msg.GetId()}))
	if err != nil || removed.Msg.GetError() != 0 {
		t.Errorf("remove %v (%v)", removed.Msg, err)
	}
}

func TestAProblemOverARangeArrivesWithItsEnd(t *testing.T) {
	problem := searchquery.Problem{
		Position: searchquery.Position{Line: 1, Column: 17}, End: searchquery.Position{Line: 1, Column: 22},
		Message: "Put this text in quotes: Кришна",
	}
	fakeCallers := newCallersRunning(func(context.Context, domain.Vault, searchquery.Request) (searchquery.SearchResults, error) {
		return searchquery.SearchResults{}, &searchquery.ParseError{Problems: []searchquery.Problem{problem}}
	})
	client := newSearchClient(t, fakeCallers, "token-a")
	started, err := client.StartSearch(t.Context(), connect.NewRequest(&v1.StartSearchRequest{Query: `meaning("a") && Кришна`}))
	if err != nil {
		t.Fatal(err)
	}

	got, err := client.GetSearch(t.Context(), connect.NewRequest(&v1.GetSearchRequest{Id: started.Msg.GetId(), WaitMs: 30000}))

	errs := got.Msg.GetSearch().GetErrors()
	if err != nil || len(errs) != 1 || errs[0].GetLine() != 1 || errs[0].GetColumn() != 17 || errs[0].GetEndLine() != 1 ||
		errs[0].GetEndColumn() != 22 || errs[0].GetMessage() != problem.Message {
		t.Errorf("errors %v (%v), want the problem from 1:17 to 1:22", errs, err)
	}
}

func TestListSearchFunctionsListsTheLanguage(t *testing.T) {
	fakeCallers, _ := newRunCallers(t)
	client := newSearchClient(t, fakeCallers, "token-a")
	got, err := client.ListSearchFunctions(t.Context(), connect.NewRequest(&v1.ListSearchFunctionsRequest{}))
	if err != nil || got.Msg.GetRevision() != searchquery.GetCatalogRevision(searchquery.GetBuiltinCatalog()) {
		t.Fatalf("listing %v (%v)", got.Msg, err)
	}
	var names []string
	for _, f := range got.Msg.GetFunctions() {
		names = append(names, f.GetName())
		if f.GetSummary() == "" || f.GetDescription() == "" || f.GetExample() == "" || len(f.GetSignatures()) == 0 {
			t.Errorf("function %v lacks a summary, a description, an example or a signature", f)
		}
	}
	if !slices.Equal(names, searchquery.ListFunctionNames(searchquery.GetBuiltinCatalog())) {
		t.Errorf("names %v", names)
	}
}

func TestAHostWithoutSearchesAnswersNoModel(t *testing.T) {
	fakeCallers := newSearchCallers(&recordingSearch{})
	client := newSearchClient(t, fakeCallers, "token-a")
	got, err := client.StartSearch(t.Context(), connect.NewRequest(&v1.StartSearchRequest{Query: `words("tide")`}))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL {
		t.Fatalf("start %v (%v), want NO_MODEL", got.Msg, err)
	}
}

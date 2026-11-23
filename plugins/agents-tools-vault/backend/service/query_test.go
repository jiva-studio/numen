package service_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest/searchtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

const tideQuery = `words("tide") && kind == "pdf"`

func getTideFunctions() host.SearchFunctions {
	return host.SearchFunctions{Revision: "r1", Functions: []host.SearchFunction{
		{Name: "words", Kind: "function", Level: "chunk", Signatures: []string{"words(string) -> SearchResults"}, Description: "Finds passages that contain the words you give.", Example: `words("tide")`},
		{Name: "kind", Kind: "variable", Level: "document", Signatures: []string{"kind: string"}, Description: "The type of the file.", Example: `kind == "pdf"`},
		{Name: "top", Kind: "stage", Signatures: []string{"SearchResults.top(int) -> SearchResults"}, Description: "Keeps only the first results; give how many.", Example: `words("tide").top(10)`},
	}}
}

func newQueryRegistry(t *testing.T) (*searchtest.Searches, *tools.Registry) {
	t.Helper()
	searches := searchtest.NewSearches()
	searches.SetFunctions(getTideFunctions(), nil)
	score := 0.8
	searches.SetResults(tideQuery, []host.SearchResult{
		{Path: "books/Tides.pdf", Kind: "pdf", Title: "Tides", Text: "the tide turns", PlaceLabel: "page 3", Start: 40, Length: 20, Score: &score},
		{Path: "books/Moon.pdf", Kind: "pdf", Title: "Moon", Text: "%PDF", IsWholeFile: true},
	})
	registry, err := service.New(vaultsOf(hosttest.New(testVault))).WithLibrary(func(domain.VaultID) service.Library {
		return service.Library{Versions: nil, Texts: nil, Search: nil, Queries: searches}
	}, getTideFunctions()).BuildRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return searches, registry
}

func TestLibraryQueryReturnsTheResultsOfAFastQuery(t *testing.T) {
	_, registry := newQueryRegistry(t)
	var out api.QueryOutput
	if failure := callFailure(t, registry, testVault, "library_query", `{"query":`+fmt.Sprintf("%q", tideQuery)+`}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.IsRunning || out.TotalResults != 2 || len(out.Results) != 2 {
		t.Fatalf("output %+v, want two results", out)
	}
	passage, whole := out.Results[0], out.Results[1]
	if passage.URI != "numen:books%2FTides.pdf?start=40&length=20" || passage.Place != "page 3" || *passage.Score != 0.8 {
		t.Errorf("passage %+v", passage)
	}
	if whole.URI != "numen:books%2FMoon.pdf" || !whole.IsWholeFile || whole.Score != nil {
		t.Errorf("whole file %+v", whole)
	}
}

func TestLibraryQueryReturnsTheIDOfASlowQueryAndItsResultsLater(t *testing.T) {
	searches, registry := newQueryRegistry(t)
	release := searches.HoldSearch(tideQuery)
	var started api.QueryOutput
	if failure := callFailure(t, registry, testVault, "library_query", `{"query":`+fmt.Sprintf("%q", tideQuery)+`,"wait":0.05}`, &started); failure != "" {
		t.Fatal(failure)
	}
	if !started.IsRunning || started.ID == "" || started.Message == "" {
		t.Fatalf("output %+v, want a running query with its id", started)
	}
	release()
	var done api.QueryOutput
	if failure := callFailure(t, registry, testVault, "library_query_results", `{"id":"`+started.ID+`"}`, &done); failure != "" {
		t.Fatal(failure)
	}
	if done.IsRunning || len(done.Results) != 2 {
		t.Fatalf("output %+v, want the results", done)
	}
}

func TestLibraryQueryResultsReadsTheNextPage(t *testing.T) {
	searches, registry := newQueryRegistry(t)
	var results []host.SearchResult
	for i := range 25 {
		results = append(results, host.SearchResult{Path: fmt.Sprintf("notes/n%02d.md", i), Kind: "note", Start: 0, Length: 5})
	}
	searches.SetResults(`words("sea")`, results)
	var first api.QueryOutput
	if failure := callFailure(t, registry, testVault, "library_query", `{"query":"words(\"sea\")"}`, &first); failure != "" {
		t.Fatal(failure)
	}
	var second api.QueryOutput
	arguments := `{"id":"` + first.ID + `","pageToken":"` + first.NextPageToken + `"}`
	if failure := callFailure(t, registry, testVault, "library_query_results", arguments, &second); failure != "" {
		t.Fatal(failure)
	}
	if len(first.Results) != 20 || first.NextPageToken == "" || len(second.Results) != 5 || second.NextPageToken != "" ||
		second.Results[0].Path != "notes/n20.md" {
		t.Fatalf("pages of %d and %d results, tokens %q and %q", len(first.Results), len(second.Results), first.NextPageToken, second.NextPageToken)
	}
}

func TestACanceledLibraryQueryRemovesItsQuery(t *testing.T) {
	searches, registry := newQueryRegistry(t)
	searches.HoldSearch(tideQuery)
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		for len(searches.ListStarted()) == 0 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()

	res, err := registry.InvokeTool(ctx, tools.Call{VaultID: testVault, Name: "library_query", Arguments: []byte(`{"query":` + fmt.Sprintf("%q", tideQuery) + `}`)})
	t.Logf("the canceled call answered %v, %v", res, err)

	if removed := searches.ListRemoved(); !slices.Equal(removed, []string{"search-1"}) {
		t.Fatalf("removed %v, want the query of the call", removed)
	}
}

func TestLibraryQueryResultsOfAnUnknownQueryIsAToolError(t *testing.T) {
	_, registry := newQueryRegistry(t)
	failure := callFailure(t, registry, testVault, "library_query_results", `{"id":"search-9"}`, nil)
	if !strings.Contains(failure, "search-9") || !strings.Contains(failure, "10 minutes") {
		t.Fatalf("failure %q, want the id and the expiry", failure)
	}
}

func TestTheDescriptionOfLibraryQueryNamesEveryFunction(t *testing.T) {
	_, registry := newQueryRegistry(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	index := slices.IndexFunc(listing.Tools, func(one *tools.ToolInfo) bool { return one.GetName() == "library_query" })
	if index < 0 {
		t.Fatal("library_query is not listed")
	}
	description := listing.Tools[index].GetDescription()
	for _, f := range getTideFunctions().Functions {
		if !strings.Contains(description, f.Signatures[0]+": "+f.Description+" Example: "+f.Example) {
			t.Errorf("the description lacks %s or its example: %s", f.Signatures[0], description)
		}
	}
	if strings.Index(description, "top(int)") < strings.Index(description, "kind: string") {
		t.Error("the stages are not listed last")
	}
}

func TestTheDescriptionOfLibraryQueryNamesTheFunctionsOfEachListing(t *testing.T) {
	searches := searchtest.NewSearches()
	searches.SetFunctions(getTideFunctions(), nil)
	registry, err := service.New(vaultsOf(hosttest.New(testVault))).WithLibrary(func(domain.VaultID) service.Library {
		return service.Library{Versions: nil, Texts: nil, Search: nil, Queries: searches}
	}, getTideFunctions()).WithFunctionLister(searches.ListSearchFunctions).BuildRegistry()
	if err != nil {
		t.Fatal(err)
	}
	shelved := getTideFunctions()
	shelved.Revision = "r2"
	shelved.Functions = append(shelved.Functions, host.SearchFunction{
		Name: "shelved", Kind: "function", Level: "document", Signatures: []string{"shelved(string) -> bool"},
		Description: "notes on a shelf.", Example: `shelved("east")`,
	})
	searches.SetFunctions(shelved, nil)

	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	index := slices.IndexFunc(listing.Tools, func(tool *tools.ToolInfo) bool { return tool.Name == "library_query" })
	if index < 0 || !strings.Contains(listing.Tools[index].Description, `shelved(string) -> bool: notes on a shelf. Example: shelved("east")`) {
		t.Errorf("the listing does not name shelved: %+v", listing.Tools)
	}
}

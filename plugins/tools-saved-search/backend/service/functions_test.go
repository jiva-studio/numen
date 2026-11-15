package service_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
)

func getGardenFunctions() host.SearchFunctions {
	return host.SearchFunctions{Revision: "r1", Functions: []host.SearchFunction{
		{Name: "words", Kind: "function", Level: "chunk", Summary: "passages with these words", Signatures: []string{"words(text)"}, Description: "Finds passages that contain the words you give.", Example: `words("spade")`},
		{Name: "tags", Kind: "function", Level: "document", Summary: "notes with this tag", Signatures: []string{"tags(name)"}, Description: "Finds notes that carry the tag you give.", Example: `tags("garden")`},
		{Name: "top", Kind: "stage", Summary: "keep the first N", Signatures: []string{".top(count)"}, Description: "Keeps only the first results; give how many.", Example: `words("seed").top(5)`},
	}}
}

func TestListSearchFunctionsReturnsTheCatalogOfTheCore(t *testing.T) {
	f := newFixture(t, 10)
	f.searches.SetFunctions(getGardenFunctions(), nil)

	var res api.ListSearchFunctionsResponse
	if failure := f.call(t, api.MethodListSearchFunctions, api.ListSearchFunctionsRequest{}, &res); failure.Code != "" {
		t.Fatal(failure)
	}

	names := make([]string, len(res.Functions))
	for i, function := range res.Functions {
		names[i] = function.Name
	}
	if !slices.Equal(names, []string{"words", "tags", "top"}) {
		t.Fatalf("names = %v, want words, tags, top", names)
	}
	want := api.SearchFunction{Name: "tags", Kind: "function", Summary: "notes with this tag", Description: "Finds notes that carry the tag you give.", Example: `tags("garden")`}
	if got := res.Functions[1]; got != want {
		t.Errorf("tags = %+v, want %+v", got, want)
	}
}

func TestListSearchFunctionsOfAnEmptyCatalogIsAnEmptyList(t *testing.T) {
	f := newFixture(t, 10)

	var res api.ListSearchFunctionsResponse
	if failure := f.call(t, api.MethodListSearchFunctions, api.ListSearchFunctionsRequest{}, &res); failure.Code != "" {
		t.Fatal(failure)
	}

	if res.Functions == nil || len(res.Functions) != 0 {
		t.Errorf("functions = %#v, want an empty list", res.Functions)
	}
}

func TestListSearchFunctionsFailsWhenTheCoreFails(t *testing.T) {
	f := newFixture(t, 10)
	f.searches.SetFunctions(host.SearchFunctions{}, errors.New("the catalog is not ready"))

	failure := f.call(t, api.MethodListSearchFunctions, api.ListSearchFunctionsRequest{}, &api.ListSearchFunctionsResponse{})

	if failure.Code == "" {
		t.Error("the call succeeded, want the failure of the core")
	}
}

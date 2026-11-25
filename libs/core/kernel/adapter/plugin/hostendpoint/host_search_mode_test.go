package hostendpoint

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func TestHostSearchTextPassesTheLimitAndTheResultsPerSource(t *testing.T) {
	search := &recordingSearch{}
	client := newTextSearchClient(t, newSearchCallers(search), "token-a")

	_, err := client.SearchText(t.Context(), connect.NewRequest(&v1.SearchTextRequest{
		Query: "heron reed", Limit: 300, ResultsPerSource: 10,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(search.textQueries) != 1 {
		t.Fatalf("the search ran %d times", len(search.textQueries))
	}
	got := search.textQueries[0]
	if got.Query != "heron reed" || got.Limit != 300 || got.ResultsPerSource != 10 {
		t.Errorf("the search received %+v, want the query, 300 results and 10 per file", got)
	}
}

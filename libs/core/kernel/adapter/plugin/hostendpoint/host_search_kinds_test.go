package hostendpoint

import (
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func TestSearchTextOfUnknownKindReturnsInvalidArgument(t *testing.T) {
	search := &recordingSearch{}
	client := newSearchClient(t, newSearchCallers(search), "token-a")

	_, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{
		Query: "heron", Kinds: []string{"note", "book"},
	}))

	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeInvalidArgument {
		t.Fatalf("the kind book returned %v, want invalid argument", err)
	}
	want := `invalid source kind "book": use "note" or "file"`
	if failure.Message() != want {
		t.Errorf("the kind book returned %q, want %q", failure.Message(), want)
	}
	if len(search.textQueries) != 0 {
		t.Errorf("the search ran for a rejected kind: %+v", search.textQueries)
	}
}

func TestSearchTextPassesTheLimitToTheSearchUnchanged(t *testing.T) {
	search := &recordingSearch{}
	client := newSearchClient(t, newSearchCallers(search), "token-a")

	for _, limit := range []int32{500, 0} {
		if _, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{
			Query: "heron", Limit: limit,
		})); err != nil {
			t.Fatal(err)
		}
	}

	if len(search.textQueries) != 2 || search.textQueries[0].Limit != 500 || search.textQueries[1].Limit != 0 {
		t.Errorf("the search received %+v, want the limits 500 and 0 as given", search.textQueries)
	}
}

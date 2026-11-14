package host_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

func TestASearchClientSendsTheLimitUnchanged(t *testing.T) {
	h := &textSearchHost{}
	c := dialTextSearch(t, h)

	for _, limit := range []int{500, 0, 7} {
		if _, err := c.SearchText(t.Context(), host.SearchResultQuery{Query: "heron", Limit: limit}); err != nil {
			t.Fatal(err)
		}
		if got := h.request.GetLimit(); got != int32(limit) {
			t.Errorf("a limit of %d was sent as %d, want it unchanged", limit, got)
		}
	}
}

func TestASearchClientSendsAnUnknownKindAndReturnsTheHostError(t *testing.T) {
	const rejection = `invalid source kind "recording": use "note" or "file"`
	h := &textSearchHost{status: connect.NewError(connect.CodeInvalidArgument, errors.New(rejection))}

	_, err := dialTextSearch(t, h).SearchText(t.Context(), host.SearchResultQuery{
		Query: "heron", Kinds: []domain.SourceKind{"recording", domain.KindNote},
	})

	if got := h.request.GetKinds(); !slices.Equal(got, []string{"recording", "note"}) {
		t.Errorf("the host received the kinds %v, want [recording note]", got)
	}
	if err == nil || !strings.Contains(err.Error(), rejection) {
		t.Errorf("the search returned %v, want the host's rejection %q", err, rejection)
	}
}

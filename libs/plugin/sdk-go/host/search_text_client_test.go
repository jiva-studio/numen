package host_test

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// textSearchHost returns SearchText with fixed search results and records the requests.
type textSearchHost struct {
	numenv1connect.UnimplementedHostSearchServiceHandler
	code    v1.HostSearchErrorCode
	status  *connect.Error
	vault   string
	request *v1.HostSearchServiceSearchTextRequest
}

func (h *textSearchHost) SearchText(_ context.Context, req *connect.Request[v1.HostSearchServiceSearchTextRequest]) (*connect.Response[v1.HostSearchServiceSearchTextResponse], error) {
	h.vault, h.request = req.Header().Get("Numen-Vault"), req.Msg
	if h.status != nil {
		return nil, h.status
	}
	return connect.NewResponse(&v1.HostSearchServiceSearchTextResponse{
		Error: h.code,
		Results: []*v1.SearchResult{{
			Path: "library/Birds.pdf", Text: "the heron stands in the reeds", PlaceLabel: "page 12",
			Range: &v1.Range{Start: 400, End: 1100}, Line: 0, Selector: `{"page":12}`,
		}, {
			Path: "notes/Heron.md", Text: "herons fish at dusk", ByteRange: &v1.Range{Start: 20, End: 60}, MatchLine: 4,
		}},
	}), nil
}

func dialTextSearch(t *testing.T, h *textSearchHost) *host.SearchClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostSearchServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewSearchClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestASearchClientReadsTheResultsTheHostFinds(t *testing.T) {
	h := &textSearchHost{}
	c := dialTextSearch(t, h)

	got, err := c.With("v2").SearchText(t.Context(), host.SearchResultQuery{
		Query: "heron", Limit: 5, Kinds: []domain.SourceKind{domain.KindFile}, PathPrefix: "library/",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []host.SearchResult{
		{Path: "library/Birds.pdf", Text: "the heron stands in the reeds", PlaceLabel: "page 12", Start: 400, Length: 700, Selector: `{"page":12}`},
		{Path: "notes/Heron.md", Text: "herons fish at dusk", Start: 20, Length: 40, MatchLine: 4},
	}
	if !slices.Equal(got, want) {
		t.Errorf("the passages are %+v, want %+v", got, want)
	}
	r := h.request
	if r.GetQuery() != "heron" || r.GetLimit() != 5 || !slices.Equal(r.GetKinds(), []string{"file"}) || r.GetPathPrefix() != "library/" || h.vault != "v2" {
		t.Errorf("the host received %v for vault %q", r, h.vault)
	}
}

func TestSearchClientReturnsErrorsOfTextSearch(t *testing.T) {
	for name, tc := range map[string]struct {
		h    *textSearchHost
		want error
	}{
		"no vault":          {&textSearchHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT}, host.ErrNoVault},
		"no search index":   {&textSearchHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL}, host.ErrNoModel},
		"permission denied": {&textSearchHost{status: connect.NewError(connect.CodePermissionDenied, errors.New("no"))}, host.ErrPermissionDenied},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := dialTextSearch(t, tc.h).SearchText(t.Context(), host.SearchResultQuery{Query: "heron"}); !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestASearchClientSendsALimitBeyondInt32AsTheNearestInt32(t *testing.T) {
	h := &textSearchHost{}
	c := dialTextSearch(t, h)
	for limit, want := range map[int]int32{1 << 40: math.MaxInt32, -(1 << 40): math.MinInt32, 7: 7} {
		if _, err := c.SearchText(t.Context(), host.SearchResultQuery{Query: "heron", Limit: limit}); err != nil {
			t.Fatal(err)
		}
		if got := h.request.GetLimit(); got != want {
			t.Errorf("a limit of %d was sent as %d, want %d", limit, got, want)
		}
	}
}

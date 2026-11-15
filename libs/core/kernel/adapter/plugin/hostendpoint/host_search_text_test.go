package hostendpoint

import (
	"context"
	"math"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
)

func (s *recordingSearch) SearchText(_ context.Context, v domain.Vault, q plugin.SearchResultQuery) ([]domain.SearchResult, error) {
	s.vaults = append(s.vaults, v.ID)
	s.textQueries = append(s.textQueries, q)
	return []domain.SearchResult{
		{Path: "library/Birds.pdf", Text: "the gray heron", PlaceLabel: "page 12", Start: 400, Length: 700, Selector: domain.Selector(`{"page":12}`)},
		{Path: "notes/Heron.md", Text: "herons fish", Start: 20, Length: 40, MatchLine: 4},
	}, s.err
}

func (s *recordingSearch) FindNearestChunks(_ context.Context, v domain.Vault, _ sourcedomain.NearestQuery) ([]searchsource.Match, error) {
	s.vaults = append(s.vaults, v.ID)
	return nil, s.err
}

func TestHostSearchTextReturnPlacesAndCheckPermission(t *testing.T) {
	search := &recordingSearch{}
	fakeCallers := newSearchCallers(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	got, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{
		Query: "heron", Limit: 5, Kinds: []string{"file"}, PathPrefix: "library/",
	}))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED || len(got.Msg.GetResults()) != 2 {
		t.Fatalf("returned %v (%v)", got.Msg, err)
	}
	first, second := got.Msg.GetResults()[0], got.Msg.GetResults()[1]
	if first.GetPath() != "library/Birds.pdf" || first.GetText() != "the gray heron" || first.GetPlaceLabel() != "page 12" ||
		first.GetByteRange().GetStart() != 400 || first.GetByteRange().GetEnd() != 1100 || first.GetSelector() != `{"page":12}` {
		t.Errorf("the first passage is %v", first)
	}
	if second.GetPath() != "notes/Heron.md" || second.GetMatchLine() != 4 || second.GetByteRange().GetEnd() != 60 {
		t.Errorf("the second passage is %v", second)
	}
	received := search.textQueries
	if len(received) != 1 || received[0].Query != "heron" || received[0].Limit != 5 || received[0].PathPrefix != "library/" ||
		!slices.Equal(received[0].Kinds, []domain.SourceKind{domain.KindFile}) {
		t.Errorf("the search was received %+v", received)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a index:read"}) {
		t.Errorf("checked %v, want index:read received once", fakeCallers.checked)
	}
}

func TestHostSearchTextDenyPluginWithoutPermission(t *testing.T) {
	fakeCallers := newSearchCallers(&recordingSearch{})
	denied := newSearchClient(t, fakeCallers, "token-b")
	if _, err := denied.SearchText(
		t.Context(),
		connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron"}),
	); readCode(
		t,
		err,
	) != connect.CodePermissionDenied {
		t.Errorf("a plugin without the permission got %v", err)
	}
}

func TestHostSearchTextRejectUnknownVaultAndKind(t *testing.T) {
	search := &recordingSearch{}
	fakeCallers := newSearchCallers(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	unknown := connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron"})
	unknown.Header().Set("Numen-Vault", "v9")
	if _, err := client.SearchText(t.Context(), unknown); readCode(t, err) != connect.CodeNotFound {
		t.Errorf("an unknown vault got %v, want not found", err)
	}
	if _, err := client.SearchText(
		t.Context(),
		connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron", Kinds: []string{"book"}}),
	); readCode(
		t,
		err,
	) != connect.CodeInvalidArgument {
		t.Errorf("a kind the index does not have got %v, want invalid argument", err)
	}
	if len(search.textQueries) != 0 {
		t.Errorf("the search ran for a call that was rejected: %+v", search.textQueries)
	}
}

func TestHostSearchWithoutIndexReportsNoIndex(t *testing.T) {
	fakeCallers := newSearchCallers(nil)
	client := newSearchClient(t, fakeCallers, "token-a")
	got, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron"}))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL {
		t.Errorf("returned %v (%v), want the no-index error", got.Msg, err)
	}
}

// farSearch returns a passage of a text longer than a range on the wire can address.
type farSearch struct {
	recordingSearch
}

func (s *farSearch) SearchText(context.Context, domain.Vault, plugin.SearchResultQuery) ([]domain.SearchResult, error) {
	return []domain.SearchResult{{Path: "library/Atlas.pdf", Text: "heron", Start: math.MaxInt32 - 5, Length: 10}}, nil
}

func TestHostSearchTextRejectsResultBeyondWireRange(t *testing.T) {
	client := newSearchClient(t, newSearchCallers(&farSearch{}), "token-a")
	got, err := client.SearchText(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchTextRequest{Query: "heron"}))
	if readCode(t, err) != connect.CodeOutOfRange {
		t.Errorf("a passage ending past the largest range returned %v (%v), want out of range", got, err)
	}
}

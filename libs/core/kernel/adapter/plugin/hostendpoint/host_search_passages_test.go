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
)

func (s *recordingSearch) SearchPassages(_ context.Context, v domain.Vault, q plugin.PassageQuery) ([]domain.Passage, error) {
	s.vaults = append(s.vaults, v.ID)
	s.passageQueries = append(s.passageQueries, q)
	return []domain.Passage{
		{Source: "library/Birds.pdf", Text: "the gray heron", Location: "page 12", Start: 400, Length: 700, Selector: domain.Selector(`{"page":12}`)},
		{Source: "notes/Heron.md", Text: "herons fish", Start: 20, Length: 40, Line: 4},
	}, s.err
}

func TestHostSearchPassagesReturnPlacesAndCheckPermission(t *testing.T) {
	search := &recordingSearch{}
	fakeCallers := newSearchCallers(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	got, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchPassagesRequest{
		Query: "heron", Limit: 5, Kinds: []string{"file"}, PathPrefix: "library/",
	}))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED || len(got.Msg.GetPassages()) != 2 {
		t.Fatalf("returned %v (%v)", got.Msg, err)
	}
	first, second := got.Msg.GetPassages()[0], got.Msg.GetPassages()[1]
	if first.GetPath() != "library/Birds.pdf" || first.GetText() != "the gray heron" || first.GetLocation() != "page 12" ||
		first.GetRange().GetStart() != 400 || first.GetRange().GetEnd() != 1100 || first.GetSelector() != `{"page":12}` {
		t.Errorf("the first passage is %v", first)
	}
	if second.GetPath() != "notes/Heron.md" || second.GetLine() != 4 || second.GetRange().GetEnd() != 60 {
		t.Errorf("the second passage is %v", second)
	}
	received := search.passageQueries
	if len(received) != 1 || received[0].Query != "heron" || received[0].Limit != 5 || received[0].PathPrefix != "library/" ||
		!slices.Equal(received[0].Kinds, []domain.SourceKind{domain.KindFile}) {
		t.Errorf("the search was received %+v", received)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a index:read"}) {
		t.Errorf("checked %v, want index:read received once", fakeCallers.checked)
	}
}

func TestHostSearchPassagesDenyPluginWithoutPermission(t *testing.T) {
	fakeCallers := newSearchCallers(&recordingSearch{})
	denied := newSearchClient(t, fakeCallers, "token-b")
	if _, err := denied.SearchPassages(
		t.Context(),
		connect.NewRequest(&v1.HostSearchServiceSearchPassagesRequest{Query: "heron"}),
	); readCode(
		t,
		err,
	) != connect.CodePermissionDenied {
		t.Errorf("a plugin without the permission got %v", err)
	}
}

func TestHostSearchPassagesRejectUnknownVaultAndKind(t *testing.T) {
	search := &recordingSearch{}
	fakeCallers := newSearchCallers(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	unknown := connect.NewRequest(&v1.HostSearchServiceSearchPassagesRequest{Query: "heron"})
	unknown.Header().Set("Numen-Vault", "v9")
	if _, err := client.SearchPassages(t.Context(), unknown); readCode(t, err) != connect.CodeNotFound {
		t.Errorf("an unknown vault got %v, want not found", err)
	}
	if _, err := client.SearchPassages(
		t.Context(),
		connect.NewRequest(&v1.HostSearchServiceSearchPassagesRequest{Query: "heron", Kinds: []string{"book"}}),
	); readCode(
		t,
		err,
	) != connect.CodeInvalidArgument {
		t.Errorf("a kind the index does not have got %v, want invalid argument", err)
	}
	if len(search.passageQueries) != 0 {
		t.Errorf("the search ran for a call that was rejected: %+v", search.passageQueries)
	}
}

func TestHostSearchWithoutIndexReportsNoIndex(t *testing.T) {
	fakeCallers := newSearchCallers(nil)
	client := newSearchClient(t, fakeCallers, "token-a")
	got, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchPassagesRequest{Query: "heron"}))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL {
		t.Errorf("returned %v (%v), want the no-index error", got.Msg, err)
	}
}

// farSearch returns a passage of a text longer than a range on the wire can address.
type farSearch struct {
	recordingSearch
}

func (s *farSearch) SearchPassages(context.Context, domain.Vault, plugin.PassageQuery) ([]domain.Passage, error) {
	return []domain.Passage{{Source: "library/Atlas.pdf", Text: "heron", Start: math.MaxInt32 - 5, Length: 10}}, nil
}

func TestHostSearchPassagesRejectsPassageBeyondWireRange(t *testing.T) {
	client := newSearchClient(t, newSearchCallers(&farSearch{}), "token-a")
	got, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.HostSearchServiceSearchPassagesRequest{Query: "heron"}))
	if readCode(t, err) != connect.CodeOutOfRange {
		t.Errorf("a passage ending past the largest range returned %v (%v), want out of range", got, err)
	}
}

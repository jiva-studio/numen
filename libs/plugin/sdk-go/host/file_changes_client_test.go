package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// feedHost answers ListFileChanges with one change and records the request.
type feedHost struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	asked   *v1.ListFileChangesRequest
	vault   string
	refused bool
}

func (h *feedHost) ListFileChanges(
	_ context.Context, req *connect.Request[v1.ListFileChangesRequest],
) (*connect.Response[v1.ListFileChangesResponse], error) {
	h.asked, h.vault = req.Msg, req.Header().Get("Numen-Vault")
	if h.refused {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no events"))
	}
	return connect.NewResponse(&v1.ListFileChangesResponse{
		Changes: []*v1.FileChange{{SourceId: 7, Path: "notes/river.md", Event: host.EventFileMoved, Seq: 12}},
		Next:    &v1.FileChangeCursor{FeedId: "feed-1", Seq: 12},
		HasMore: true,
	}), nil
}

func feedClientTo(t *testing.T, h *feedHost) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(h))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestListFileChangesSendsTheCursorAndReadsTheChanges(t *testing.T) {
	h := &feedHost{}

	page, err := feedClientTo(t, h).With("v2").ListFileChanges(t.Context(), host.FileChangeQuery{
		Since: host.FileChangeCursor{FeedID: "feed-1", Seq: 3}, Events: []string{host.EventFileMoved}, Limit: 5,
	})

	if err != nil {
		t.Fatal(err)
	}
	if h.asked.GetSince().GetFeedId() != "feed-1" || h.asked.GetSince().GetSeq() != 3 || h.asked.GetLimit() != 5 ||
		!slices.Equal(h.asked.GetEvents(), []string{host.EventFileMoved}) || h.vault != "v2" {
		t.Errorf("asked %v of %q", h.asked, h.vault)
	}
	want := []host.FileChange{{SourceID: 7, Path: "notes/river.md", Event: host.EventFileMoved, Seq: 12}}
	if !slices.Equal(page.Changes, want) || page.Next != (host.FileChangeCursor{FeedID: "feed-1", Seq: 12}) || !page.HasMore || page.IsReset {
		t.Errorf("read %+v", page)
	}
}

func TestListFileChangesOfAnInvalidQueryIsItsOwnError(t *testing.T) {
	_, err := feedClientTo(t, &feedHost{refused: true}).ListFileChanges(t.Context(), host.FileChangeQuery{})

	if !errors.Is(err, host.ErrInvalidFileChangeQuery) {
		t.Errorf("got %v, want ErrInvalidFileChangeQuery", err)
	}
}

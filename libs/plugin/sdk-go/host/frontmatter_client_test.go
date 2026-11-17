package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// frontmatterHost answers ReadFrontmatter with the notes it holds and records the request.
type frontmatterHost struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	asked   *v1.ReadFrontmatterRequest
	vault   string
	refused bool
}

func (h *frontmatterHost) ReadFrontmatter(
	_ context.Context, req *connect.Request[v1.ReadFrontmatterRequest],
) (*connect.Response[v1.ReadFrontmatterResponse], error) {
	h.asked, h.vault = req.Msg, req.Header().Get("Numen-Vault")
	if h.refused {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("1001 paths, at most 1000"))
	}
	return connect.NewResponse(&v1.ReadFrontmatterResponse{
		Notes:         []*v1.NoteFrontmatter{{Path: "a.md", Values: []byte(`{"tags":["river","lake"]}`)}, {Path: "b.md", Values: []byte(`{}`)}},
		NextPageToken: "b.md",
	}), nil
}

func frontmatterClientTo(t *testing.T, h *frontmatterHost) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(h))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestReadFrontmatterSendsTheQueryAndReadsTheValues(t *testing.T) {
	h := &frontmatterHost{}

	page, err := frontmatterClientTo(t, h).With("v2").ReadFrontmatter(t.Context(), host.FrontmatterQuery{
		Paths: []string{"a.md", "b.md"}, Keys: []string{"tags"}, PageSize: 2, PageToken: "0.md",
	})

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.asked.GetPaths(), []string{"a.md", "b.md"}) || !slices.Equal(h.asked.GetKeys(), []string{"tags"}) ||
		h.asked.GetPageSize() != 2 || h.asked.GetPageToken() != "0.md" || h.vault != "v2" {
		t.Errorf("asked %v of %q", h.asked, h.vault)
	}
	if len(page.Notes) != 2 || page.NextPageToken != "b.md" || page.Notes[0].Path != "a.md" ||
		!reflect.DeepEqual(page.Notes[0].Values["tags"], []any{"river", "lake"}) || len(page.Notes[1].Values) != 0 {
		t.Errorf("read %+v", page)
	}
}

func TestReadFrontmatterOverTheLimitsIsItsOwnError(t *testing.T) {
	_, err := frontmatterClientTo(t, &frontmatterHost{refused: true}).ReadFrontmatter(t.Context(), host.FrontmatterQuery{})

	if !errors.Is(err, host.ErrInvalidFrontmatterQuery) {
		t.Errorf("got %v, want ErrInvalidFrontmatterQuery", err)
	}
}

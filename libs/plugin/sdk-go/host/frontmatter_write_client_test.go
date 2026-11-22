package host_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// frontmatterWriter answers WriteFrontmatter: the first note is written, the second is stale and the
// third fails with no code. It records the request.
type frontmatterWriter struct {
	unimplementedHost
	asked     *v1.WriteFrontmatterRequest
	vault     string
	callError v1.ErrorCode
	denied    bool
}

func (h *frontmatterWriter) WriteFrontmatter(
	_ context.Context, req *connect.Request[v1.WriteFrontmatterRequest],
) (*connect.Response[v1.WriteFrontmatterResponse], error) {
	h.asked, h.vault = req.Msg, req.Header().Get("Numen-Vault")
	if h.denied {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("needs vault:write"))
	}
	if h.callError != 0 {
		return connect.NewResponse(&v1.WriteFrontmatterResponse{Error: h.callError}), nil
	}
	return connect.NewResponse(&v1.WriteFrontmatterResponse{Results: []*v1.NoteFrontmatterWritten{
		{Stat: &v1.FileStat{Path: "a.md", Size: 64, ModTime: 2000, IsNote: true}},
		{Error: v1.ErrorCode_ERROR_CODE_CONFLICT, Detail: "b.md changed since it was read"},
		{Error: v1.ErrorCode_ERROR_CODE_INTERNAL, Detail: "disk full"},
	}}), nil
}

func frontmatterWriterClient(t *testing.T, h *frontmatterWriter) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(h))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestWriteFrontmatterSendsThePathsTheValuesAsJSONAndTheBaselines(t *testing.T) {
	h := &frontmatterWriter{}
	baseline := domain.FileStat{Path: "a.md", Kind: domain.KindNote, Size: 40, ModTime: time.Unix(0, 1000)}

	_, err := frontmatterWriterClient(t, h).With("v2").WriteFrontmatter(t.Context(), []host.FrontmatterWrite{
		{Path: "a.md", Values: map[string]any{"tags": []string{"river", "lake"}, "status": nil, "rating": 3}, Baseline: baseline},
		{Path: "b.md", Values: map[string]any{"tags": nil}},
		{Path: "c.md", Values: map[string]any{}},
	})

	if err != nil {
		t.Fatal(err)
	}
	if h.vault != "v2" || len(h.asked.GetNotes()) != 3 {
		t.Fatalf("asked %v of %q, want three notes of v2", h.asked, h.vault)
	}
	first := h.asked.GetNotes()[0]
	var values map[string]any
	if err := json.Unmarshal(first.GetValues(), &values); err != nil {
		t.Fatalf("the values of a.md are not a JSON object: %v", err)
	}
	if _, hasStatus := values["status"]; first.GetPath() != "a.md" || !hasStatus || values["status"] != nil ||
		!reflect.DeepEqual(values["tags"], []any{"river", "lake"}) || values["rating"] != float64(3) {
		t.Errorf("the first note is %q with the values %s, want tags, rating and status as null", first.GetPath(), first.GetValues())
	}
	if first.GetBaseline().GetSize() != 40 || first.GetBaseline().GetMtime() != 1000 || !first.GetBaseline().GetIsNote() {
		t.Errorf("the baseline of a.md is %v, want the stat given", first.GetBaseline())
	}
	if h.asked.GetNotes()[1].GetBaseline() != nil {
		t.Errorf("a note without a baseline was sent the baseline %v", h.asked.GetNotes()[1].GetBaseline())
	}
}

func TestWriteFrontmatterAnswersOnePerNoteAndTellsAStaleNoteFromAFailedOne(t *testing.T) {
	results, err := frontmatterWriterClient(t, &frontmatterWriter{}).WriteFrontmatter(t.Context(), []host.FrontmatterWrite{
		{Path: "a.md", Values: map[string]any{"tags": "river"}},
		{Path: "b.md", Values: map[string]any{"tags": "river"}},
		{Path: "c.md", Values: map[string]any{"tags": "river"}},
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want one for each note", len(results))
	}
	if results[0].Err != nil || results[0].Stat.Size != 64 || results[0].Stat.Kind != domain.KindNote {
		t.Errorf("the written note is %+v, want its stat", results[0])
	}
	if !errors.Is(results[1].Err, host.ErrStale) {
		t.Errorf("the stale note returned %v, want ErrStale", results[1].Err)
	}
	if results[2].Err == nil || errors.Is(results[2].Err, host.ErrStale) || !strings.Contains(results[2].Err.Error(), "disk full") {
		t.Errorf("the failed note returned %v, want an error naming disk full", results[2].Err)
	}
}

func TestWriteFrontmatterWithoutAVaultIsNoVault(t *testing.T) {
	h := &frontmatterWriter{callError: v1.ErrorCode_ERROR_CODE_NO_VAULT}

	results, err := frontmatterWriterClient(t, h).WriteFrontmatter(t.Context(), []host.FrontmatterWrite{{Path: "a.md", Values: map[string]any{"tags": "river"}}})

	if !errors.Is(err, host.ErrNoVault) || len(results) != 0 {
		t.Errorf("got %v and %v, want ErrNoVault and no results", results, err)
	}
}

func TestWriteFrontmatterWithoutVaultWriteIsDenied(t *testing.T) {
	_, err := frontmatterWriterClient(t, &frontmatterWriter{denied: true}).WriteFrontmatter(t.Context(), []host.FrontmatterWrite{{Path: "a.md", Values: map[string]any{"tags": "river"}}})

	if !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("got %v, want a permission denied error", err)
	}
}

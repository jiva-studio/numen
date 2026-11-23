package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// absentHost serves the calls of absent files and records what they carried.
type absentHost struct {
	unimplementedHost
	code    v1.ErrorCode
	put     []*v1.HostAbsentFile
	filled  *v1.FillFileRequest
	evicted *v1.EvictFileRequest
	vault   string
}

func (a *absentHost) PutAbsentFiles(_ context.Context, req *connect.Request[v1.PutAbsentFilesRequest]) (*connect.Response[v1.PutAbsentFilesResponse], error) {
	a.put, a.vault = req.Msg.GetFiles(), req.Header().Get("Numen-Vault")
	return connect.NewResponse(&v1.PutAbsentFilesResponse{Error: a.code}), nil
}

func (a *absentHost) ListAbsentFiles(context.Context, *connect.Request[v1.ListAbsentFilesRequest]) (*connect.Response[v1.ListAbsentFilesResponse], error) {
	return connect.NewResponse(&v1.ListAbsentFilesResponse{Files: a.put}), nil
}

func (a *absentHost) FillFile(_ context.Context, req *connect.Request[v1.FillFileRequest]) (*connect.Response[v1.FillFileResponse], error) {
	a.filled = req.Msg
	return connect.NewResponse(&v1.FillFileResponse{Error: a.code}), nil
}

func (a *absentHost) EvictFile(_ context.Context, req *connect.Request[v1.EvictFileRequest]) (*connect.Response[v1.EvictFileResponse], error) {
	a.evicted = req.Msg
	return connect.NewResponse(&v1.EvictFileResponse{Error: a.code}), nil
}

func clientToAbsent(t *testing.T, a *absentHost) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(a))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestPutAbsentFilesSendsEachFileAndTheVaultOfTheClient(t *testing.T) {
	a := &absentHost{}
	modified := time.UnixMilli(1_700_000_000_123)
	files := []host.AbsentFile{{Path: "Books/Guide.pdf", Size: 5 << 20, ModTime: modified, Hash: "ab12"}}
	client := clientToAbsent(t, a).With("v1")
	if err := client.PutAbsentFiles(t.Context(), files); err != nil {
		t.Fatal(err)
	}
	if len(a.put) != 1 || a.put[0].GetPath() != "Books/Guide.pdf" || a.put[0].GetSize() != 5<<20 ||
		a.put[0].GetModifiedAtMs() != 1_700_000_000_123 || a.put[0].GetHash() != "ab12" || a.vault != "v1" {
		t.Fatalf("got %+v for the vault %q", a.put, a.vault)
	}
}

func TestListAbsentFilesReadsEachFile(t *testing.T) {
	a := &absentHost{put: []*v1.HostAbsentFile{{Path: "a.pdf", Size: 3, ModifiedAtMs: 1_700_000_000_123, Hash: "ff"}}}
	got, err := clientToAbsent(t, a).ListAbsentFiles(t.Context())
	want := host.AbsentFile{Path: "a.pdf", Size: 3, ModTime: time.UnixMilli(1_700_000_000_123), Hash: "ff"}
	if err != nil || len(got) != 1 || got[0].Path != want.Path || got[0].Size != want.Size || !got[0].ModTime.Equal(want.ModTime) || got[0].Hash != want.Hash {
		t.Fatalf("got %+v, %v, want %+v", got, err, want)
	}
}

func TestFillFileAndEvictFileSendTheirArguments(t *testing.T) {
	a := &absentHost{}
	client := clientToAbsent(t, a)
	if err := client.FillFile(t.Context(), "a.pdf", "a.part"); err != nil {
		t.Fatal(err)
	}
	if err := client.EvictFile(t.Context(), "a.pdf", "ff"); err != nil {
		t.Fatal(err)
	}
	if a.filled.GetPath() != "a.pdf" || a.filled.GetStoreName() != "a.part" || a.evicted.GetPath() != "a.pdf" || a.evicted.GetExpectedHash() != "ff" {
		t.Fatalf("got %+v and %+v", a.filled, a.evicted)
	}
}

func TestEachCodeOfAbsentFilesIsItsOwnError(t *testing.T) {
	cases := map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_FILE_ABSENT:   host.ErrFileAbsent,
		v1.ErrorCode_ERROR_CODE_FILE_PRESENT:  host.ErrFilePresent,
		v1.ErrorCode_ERROR_CODE_HASH_MISMATCH: host.ErrHashMismatch,
		v1.ErrorCode_ERROR_CODE_CHANGED:       host.ErrChanged,
	}
	for code, want := range cases {
		client := clientToAbsent(t, &absentHost{code: code})
		for name, err := range map[string]error{
			"put":   client.PutAbsentFiles(t.Context(), nil),
			"fill":  client.FillFile(t.Context(), "a.pdf", "a.part"),
			"evict": client.EvictFile(t.Context(), "a.pdf", "ff"),
		} {
			if !errors.Is(err, want) {
				t.Errorf("%s with %s returned %v, want %v", name, code, err, want)
			}
		}
	}
}

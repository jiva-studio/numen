package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// fileOpsHost is a host that answers the four file calls with code when it is set, and records the requests.
type fileOpsHost struct {
	unimplementedHost
	code   v1.ErrorCode
	moved  *v1.HostFileServiceMoveFileRequest
	gone   *v1.HostFileServiceRemoveFileRequest
	ranged *v1.ReadFileRangeRequest
	placed *v1.PlaceFileRequest
	named  string
}

func (h *fileOpsHost) MoveFile(_ context.Context, req *connect.Request[v1.HostFileServiceMoveFileRequest]) (*connect.Response[v1.HostFileServiceMoveFileResponse], error) {
	h.moved, h.named = req.Msg, req.Header().Get("Numen-Vault")
	return connect.NewResponse(&v1.HostFileServiceMoveFileResponse{Error: h.code, MovedCount: 7}), nil
}

func (h *fileOpsHost) RemoveFile(_ context.Context, req *connect.Request[v1.HostFileServiceRemoveFileRequest]) (*connect.Response[v1.HostFileServiceRemoveFileResponse], error) {
	h.gone = req.Msg
	return connect.NewResponse(&v1.HostFileServiceRemoveFileResponse{Error: h.code, RemovedCount: 3}), nil
}

func (h *fileOpsHost) ReadFileRange(_ context.Context, req *connect.Request[v1.ReadFileRangeRequest]) (*connect.Response[v1.ReadFileRangeResponse], error) {
	h.ranged = req.Msg
	return connect.NewResponse(&v1.ReadFileRangeResponse{
		Error: h.code, Content: []byte{0, 255, 7},
		Stat: &v1.FileStat{Path: "Scans/Delta.pdf", Size: 3, Mtime: 9000, Hash: "abc"},
	}), nil
}

func (h *fileOpsHost) PlaceFile(_ context.Context, req *connect.Request[v1.PlaceFileRequest]) (*connect.Response[v1.PlaceFileResponse], error) {
	h.placed = req.Msg
	return connect.NewResponse(&v1.PlaceFileResponse{
		Error: h.code, Stat: &v1.FileStat{Path: req.Msg.GetPath(), Size: 3, Mtime: 9000, Hash: req.Msg.GetExpectedHash()},
	}), nil
}

func fileOpsClient(t *testing.T, h *fileOpsHost) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(h))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func getBaselineStat() *domain.FileStat {
	return &domain.FileStat{Path: "Scans/Delta.pdf", Kind: domain.KindFile, Size: 5, ModTime: time.Unix(0, 4000), Hash: "h1"}
}

func TestTheClientMovesAFileWithTheBaselineOfTheFileAndTheVaultItNames(t *testing.T) {
	h := &fileOpsHost{}

	got, err := fileOpsClient(t, h).With("v2").MoveFile(t.Context(), "Scans/Delta.pdf", "Atlas/Delta.pdf", getBaselineStat())

	if err != nil || got.Count != 7 {
		t.Fatalf("the move returned %+v (%v), want a count of 7", got, err)
	}
	if h.moved.GetFrom() != "Scans/Delta.pdf" || h.moved.GetTo() != "Atlas/Delta.pdf" || h.named != "v2" {
		t.Errorf("the host received %v for the vault %q", h.moved, h.named)
	}
	if b := h.moved.GetBaseline(); b.GetHash() != "h1" || b.GetSize() != 5 || b.GetMtime() != 4000 || b.GetPath() != "Scans/Delta.pdf" {
		t.Errorf("the baseline was %v", b)
	}
}

func TestTheClientMovesAFolderWithNoBaseline(t *testing.T) {
	h := &fileOpsHost{}

	if _, err := fileOpsClient(t, h).MoveFile(t.Context(), "Scans", "Atlas", nil); err != nil {
		t.Fatal(err)
	}

	if h.moved.GetBaseline() != nil {
		t.Errorf("the host received the baseline %v, want none", h.moved.GetBaseline())
	}
}

func TestTheClientRemovesAFileAndReturnsTheCount(t *testing.T) {
	h := &fileOpsHost{}

	got, err := fileOpsClient(t, h).RemoveFile(t.Context(), "Scans", nil)

	if err != nil || got.Count != 3 || h.gone.GetPath() != "Scans" || h.gone.GetBaseline() != nil {
		t.Errorf("the removal returned %+v (%v) after %v, want 3 and no baseline", got, err, h.gone)
	}
	if _, err := fileOpsClient(t, h).RemoveFile(t.Context(), "Delta.pdf", getBaselineStat()); err != nil || h.gone.GetBaseline().GetHash() != "h1" {
		t.Errorf("the removal of a file sent %v (%v), want its baseline", h.gone, err)
	}
}

func TestTheClientReadsBytesAsTheyAreWithTheStatOfTheFile(t *testing.T) {
	h := &fileOpsHost{}

	got, err := fileOpsClient(t, h).ReadBytes(t.Context(), "Scans/Delta.pdf", 1<<33, 4<<20)

	if err != nil {
		t.Fatal(err)
	}
	if h.ranged.GetPath() != "Scans/Delta.pdf" || h.ranged.GetStart() != 1<<33 || h.ranged.GetLength() != 4<<20 {
		t.Errorf("the host received %v", h.ranged)
	}
	if string(got.Content) != "\x00\xff\x07" || got.Stat.Hash != "abc" || got.Stat.Size != 3 || !got.Stat.ModTime.Equal(time.Unix(0, 9000)) {
		t.Errorf("the range came back as %+v", got)
	}
}

func TestTheClientPlacesAStagedFileAndReturnsTheStatOfTheFile(t *testing.T) {
	h := &fileOpsHost{}

	got, err := fileOpsClient(t, h).PlaceFile(t.Context(), "ab.part", "Scans/Delta.pdf", "feed", getBaselineStat())

	if err != nil || got.Path != "Scans/Delta.pdf" || got.Hash != "feed" {
		t.Fatalf("the placement returned %+v (%v)", got, err)
	}
	if h.placed.GetName() != "ab.part" || h.placed.GetPath() != "Scans/Delta.pdf" || h.placed.GetExpectedHash() != "feed" || h.placed.GetBaseline().GetHash() != "h1" {
		t.Errorf("the host received %v", h.placed)
	}
	if _, err := fileOpsClient(t, h).PlaceFile(t.Context(), "ab.part", "New.pdf", "feed", nil); err != nil || h.placed.GetBaseline() != nil {
		t.Errorf("a placement with no baseline sent %v (%v)", h.placed.GetBaseline(), err)
	}
}

func TestTheClientReturnsTheErrorOfEachFileCall(t *testing.T) {
	for code, want := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_NOT_FOUND:      host.ErrNotExist,
		v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS: host.ErrAlreadyExists,
		v1.ErrorCode_ERROR_CODE_OUTSIDE_VAULT:  host.ErrOutside,
		v1.ErrorCode_ERROR_CODE_CONFLICT:       host.ErrStale,
		v1.ErrorCode_ERROR_CODE_LOCKED:         host.ErrLockedByAnother,
		v1.ErrorCode_ERROR_CODE_HASH_MISMATCH:  host.ErrHashMismatch,
		v1.ErrorCode_ERROR_CODE_INVALID_RANGE:  host.ErrInvalidRange,
		v1.ErrorCode_ERROR_CODE_NO_VAULT:       host.ErrNoVault,
	} {
		client := fileOpsClient(t, &fileOpsHost{code: code})
		ctx := t.Context()
		_, moveErr := client.MoveFile(ctx, "a", "b", nil)
		_, removeErr := client.RemoveFile(ctx, "a", nil)
		_, readErr := client.ReadBytes(ctx, "a", 0, 1)
		_, placeErr := client.PlaceFile(ctx, "a.part", "a", "h", nil)
		for call, err := range map[string]error{"move": moveErr, "remove": removeErr, "read": readErr, "place": placeErr} {
			if !errors.Is(err, want) {
				t.Errorf("%s with %s: got %v, want %v", call, code, err, want)
			}
		}
	}
}

func TestTheClientReturnsUnknownVaultWhenTheHostDoesNotKnowTheVaultItNames(t *testing.T) {
	failing := &notFoundHost{}
	server := httptest.NewServer(handler(failing))
	t.Cleanup(server.Close)
	client := host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"}).With("nobody")

	if _, err := client.ReadBytes(t.Context(), "a", 0, 1); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("a ranged read of an unknown vault was %v, want ErrUnknownVault", err)
	}
}

// notFoundHost fails every ranged read with the not found of a vault that is not known.
type notFoundHost struct{ unimplementedHost }

func (notFoundHost) ReadFileRange(context.Context, *connect.Request[v1.ReadFileRangeRequest]) (*connect.Response[v1.ReadFileRangeResponse], error) {
	return nil, connect.NewError(connect.CodeNotFound, errors.New("no such vault"))
}

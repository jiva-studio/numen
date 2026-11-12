package hostendpoint

import (
	"context"
	"errors"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

const (
	brokenPath = "Broken.md"
	diskFull   = "write Broken.md: the disk is full"
)

// brokenVault is the vault of rivers whose change of Broken.md fails with an error that has no wire
// code.
type brokenVault struct{ *fakeNoteVault }

func (b brokenVault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	if from == brokenPath {
		return domain.MoveResult{From: from, To: to}, errors.New(diskFull)
	}
	return b.fakeNoteVault.MoveNote(ctx, from, to)
}

func (b brokenVault) RemoveNote(ctx context.Context, target string) (string, []string, error) {
	if target == brokenPath {
		return "", nil, errors.New(diskFull)
	}
	return b.fakeNoteVault.RemoveNote(ctx, target)
}

func (b brokenVault) RenameNote(ctx context.Context, from, title string) (string, bool, *domain.MoveResult, error) {
	if from == brokenPath {
		return "", false, nil, errors.New(diskFull)
	}
	return b.fakeNoteVault.RenameNote(ctx, from, title)
}

func newBrokenClient(t *testing.T) (numenv1connect.HostVaultServiceClient, *fakeNoteVault) {
	t.Helper()
	vault := newFakeNoteVault()
	hostCallers := newCallers()
	hostCallers.vault = vaultOf(brokenVault{vault})
	endpoint := &Endpoint{callers: hostCallers.over(nil, manifest.PermissionVaultWrite)}
	mux := http.NewServeMux()
	mux.Handle(vaultapi.NewHandler(endpoint, endpoint.callers.Lent))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	return client, vault
}

func TestMoveReturnsUncodedErrorAsInternalAndMovesTheOthers(t *testing.T) {
	client, vault := newBrokenClient(t)

	res, err := client.MoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: []string{"Danube.md", brokenPath, "Nowhere.md", "Rivers/Rhine.md"}, Folder: "Archive",
	}))
	if err != nil {
		t.Fatalf("the move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 4 {
		t.Fatalf("the move returned error %v with %d entries, want 4 entries", res.Msg.GetError(), len(moves))
	}
	if m := moves[1]; m.GetMove().GetFrom() != brokenPath || m.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL ||
		!strings.Contains(m.GetDetail(), diskFull) {
		t.Errorf("the entry of %s is %v, want error INTERNAL with the detail %q", brokenPath, m, diskFull)
	}
	if m := moves[2]; m.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND || !strings.Contains(m.GetDetail(), "Nowhere.md") {
		t.Errorf("the entry of Nowhere.md is %v, want error NOT_FOUND with the core's text", m)
	}
	for _, i := range []int{0, 3} {
		if m := moves[i]; m.GetError() != 0 || m.GetDetail() != "" || m.GetMove().GetTo() == "" {
			t.Errorf("entry %d is %v, want it moved with no error and no detail", i, m)
		}
	}
	for _, target := range []string{"Archive/Danube.md", "Archive/Rhine.md"} {
		if _, found := vault.files[target]; !found {
			t.Errorf("%s is not in the vault", target)
		}
	}
}

func TestRemovalReturnsUncodedErrorAsInternalAndRemovesOthers(t *testing.T) {
	client, vault := newBrokenClient(t)

	res, err := client.RemoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: []string{"Danube.md", brokenPath, "Rivers"},
	}))
	if err != nil {
		t.Fatalf("the removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != 0 || len(removals) != 3 {
		t.Fatalf("the removal returned error %v with %d entries, want 3 entries", res.Msg.GetError(), len(removals))
	}
	if r := removals[1]; r.GetPath() != brokenPath || r.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL ||
		!strings.Contains(r.GetDetail(), diskFull) {
		t.Errorf("the entry of %s is %v, want error INTERNAL with the detail %q", brokenPath, r, diskFull)
	}
	if r := removals[2]; r.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE || r.GetDetail() == "" {
		t.Errorf("the entry of the folder is %v, want error NOT_A_NOTE with the core's text", r)
	}
	if r := removals[0]; r.GetError() != 0 || r.GetDetail() != "" || r.GetTrashPath() != ".trash/Danube.md" {
		t.Errorf("the first entry is %v, want Danube.md in .trash with no detail", r)
	}
	if _, found := vault.files["Danube.md"]; found {
		t.Error("Danube.md is still in the vault")
	}
}

func TestRenameReturnsUncodedErrorAsInternalWithTheDetail(t *testing.T) {
	client, _ := newBrokenClient(t)

	res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: brokenPath, Title: "Mended"}))
	if err != nil {
		t.Fatalf("the rename failed as a whole: %v", err)
	}

	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL || !strings.Contains(res.Msg.GetDetail(), diskFull) {
		t.Errorf("the rename returned %v, want error INTERNAL with the detail %q", res.Msg, diskFull)
	}
}

func TestRejectedRenameReturnsTheCoreTextInDetail(t *testing.T) {
	client, _ := newBrokenClient(t)

	res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Nowhere.md", Title: "Elsewhere"}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND || !strings.Contains(res.Msg.GetDetail(), "Nowhere.md") {
		t.Errorf("the rename of Nowhere.md returned %v, want error NOT_FOUND with the core's text", res.Msg)
	}
}

func TestRenameThatSucceedsReturnsNoDetail(t *testing.T) {
	client, _ := newBrokenClient(t)

	res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != 0 || res.Msg.GetDetail() != "" {
		t.Errorf("the rename returned %v, want no error and no detail", res.Msg)
	}
}

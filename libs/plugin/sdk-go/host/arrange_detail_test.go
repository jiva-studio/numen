package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

const diskFull = "write Broken.md: the disk is full"

// noCodeErrorHost is a host whose change of Broken.md fails with an error that has no code of its
// own, and whose other errors have the core's text.
type noCodeErrorHost struct {
	fakeArrangeHost
}

func (a *noCodeErrorHost) RenameNote(
	context.Context, *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{
		Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, Path: "Broken.md", Detail: diskFull,
	}), nil
}

func (a *noCodeErrorHost) MoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Moves: []*v1.HostNoteMove{
		{Move: &v1.MoveResult{From: "Danube.md", To: "Archive/Danube.md"}},
		{Move: &v1.MoveResult{From: "Broken.md"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, Detail: diskFull},
		{Move: &v1.MoveResult{From: "Nowhere.md"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND, Detail: "move Nowhere.md: no such note"},
	}}), nil
}

func (a *noCodeErrorHost) RemoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Removals: []*v1.HostNoteRemoval{
		{Path: "Danube.md", TrashPath: ".trash/Danube.md"},
		{Path: "Broken.md", Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, Detail: diskFull},
	}}), nil
}

func newNoCodeErrorClient(t *testing.T) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(&noCodeErrorHost{}))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestTheClientKeepsTheDetailOfARenameThatFailedWithNoCode(t *testing.T) {
	_, err := newNoCodeErrorClient(t).RenameNote(t.Context(), "Broken.md", "Mended")

	if err == nil || !strings.Contains(err.Error(), diskFull) {
		t.Errorf("the rename returned %v, want an error containing %q", err, diskFull)
	}
}

func TestTheClientKeepsTheDetailOfEachMoveEntryInItsError(t *testing.T) {
	moves, err := newNoCodeErrorClient(t).MoveNotes(t.Context(), []string{"Danube.md", "Broken.md", "Nowhere.md"}, "Archive")
	if err != nil {
		t.Fatalf("the move failed as a whole: %v", err)
	}

	if len(moves) != 3 || moves[0].Err != nil || !moves[0].Move.IsMoved {
		t.Fatalf("the moves came back as %+v", moves)
	}
	if m := moves[1]; m.Err == nil || !strings.Contains(m.Err.Error(), diskFull) {
		t.Errorf("the move of Broken.md came back as %+v, want an error containing %q", m, diskFull)
	}
	if m := moves[2]; !errors.Is(m.Err, host.ErrNotExist) || !strings.Contains(m.Err.Error(), "move Nowhere.md: no such note") {
		t.Errorf("the move of Nowhere.md came back as %+v, want ErrNotExist containing the core's text", m)
	}
}

func TestTheClientKeepsTheDetailOfARemovalEntryInItsError(t *testing.T) {
	removals, err := newNoCodeErrorClient(t).RemoveNotes(t.Context(), []string{"Danube.md", "Broken.md"})
	if err != nil {
		t.Fatalf("the removal failed as a whole: %v", err)
	}

	if len(removals) != 2 || removals[0].Err != nil {
		t.Fatalf("the removals came back as %+v", removals)
	}
	if r := removals[1]; r.Err == nil || !strings.Contains(r.Err.Error(), diskFull) {
		t.Errorf("the removal of Broken.md came back as %+v, want an error containing %q", r, diskFull)
	}
}

func getVaultsErrors() map[v1.VaultsErrorCode]error {
	return map[v1.VaultsErrorCode]error{
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:   host.ErrVaultOverlaps,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN: host.ErrVaultNameTaken,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT:    host.ErrVaultActive,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT: host.ErrLastVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND:  host.ErrUnknownVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE: host.ErrVaultUnreadable,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:       host.ErrVaultDuplicate,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED:    host.ErrUnsavedChanges,
	}
}

func changeVaults(t *testing.T, client *host.Client) map[string]error {
	t.Helper()
	ctx := t.Context()
	_, addErr := client.AddVault(ctx, "/vaults/Seas", "")
	_, renameErr := client.RenameVault(ctx, "v2", "Ponds")
	_, removeErr := client.RemoveVault(ctx, "v2")
	openErr := client.OpenVault(ctx, "v2")
	return map[string]error{"AddVault": addErr, "RenameVault": renameErr, "RemoveVault": removeErr, "OpenVault": openErr}
}

func TestAVaultsErrorWithADetailHasTheDetailAsItsText(t *testing.T) {
	const detail = "/vaults/Seas contains the vault Rivers; a vault cannot contain another vault"
	for code, want := range getVaultsErrors() {
		client := newFakeArrangeClient(t, &fakeArrangeHost{vaultsError: code, vaultsDetail: detail})
		for call, err := range changeVaults(t, client) {
			if !errors.Is(err, want) {
				t.Errorf("%s rejected with %s returned %v, want %v", call, code, err, want)
			}
			if err == nil || err.Error() != detail {
				t.Errorf("%s rejected with %s returned the text %q, want the detail %q alone", call, code, err, detail)
			}
		}
	}
}

func TestAVaultsErrorWithNoDetailHasThePlainSentinelText(t *testing.T) {
	for code, want := range getVaultsErrors() {
		client := newFakeArrangeClient(t, &fakeArrangeHost{vaultsError: code})
		for call, err := range changeVaults(t, client) {
			if !errors.Is(err, want) {
				t.Fatalf("%s rejected with %s returned %v, want %v", call, code, err, want)
			}
			text := err.Error()
			if text != want.Error() {
				t.Errorf("%s rejected with %s returned the text %q, want the text of the sentinel %q", call, code, text, want.Error())
			}
			for _, phrase := range []string{"host:", "add vault", "rename vault", "remove vault", "open vault"} {
				if strings.Contains(text, phrase) {
					t.Errorf("%s rejected with %s returned the text %q, which contains %q", call, code, text, phrase)
				}
			}
		}
	}
}

// detailedRenaming is a host whose rename of a note is rejected with a code and the core's text.
type detailedRenaming struct {
	fakeArrangeHost
	code   v1.HostVaultErrorCode
	detail string
}

func (a *detailedRenaming) RenameNote(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Error: a.code, Path: req.Msg.GetPath(), Detail: a.detail}), nil
}

func TestANoteErrorWithADetailHasTheDetailAsItsText(t *testing.T) {
	const detail = "Rivers/Donau.md already exists; Rivers/Danube.md is unchanged"
	server := httptest.NewServer(handler(&detailedRenaming{code: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS, detail: detail}))
	t.Cleanup(server.Close)
	client := host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})

	_, err := client.RenameNote(t.Context(), "Rivers/Danube.md", "Donau")

	if !errors.Is(err, host.ErrAlreadyExists) {
		t.Errorf("the rename returned %v, want ErrAlreadyExists", err)
	}
	if err == nil || err.Error() != detail {
		t.Errorf("the rename returned the text %q, want the detail %q alone", err, detail)
	}
}

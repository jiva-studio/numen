package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// fakeArrangeHost is a host of a vault of rivers that serves the calls renaming, moving and removing
// notes, running checks and changing the list of vaults, and records the requests.
type fakeArrangeHost struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	mu           sync.Mutex
	noteError    v1.HostVaultErrorCode
	vaultsError  v1.VaultsErrorCode
	vaultsDetail string
	requests     []string
}

func (a *fakeArrangeHost) record(request string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requests = append(a.requests, request)
}

func (a *fakeArrangeHost) getRequests() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.requests)
}

func (a *fakeArrangeHost) RenameNote(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	a.record("rename " + req.Msg.GetPath() + " " + req.Msg.GetTitle())
	if a.noteError != 0 {
		return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Error: a.noteError, Path: req.Msg.GetPath()}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{
		Path: "Donau.md", Title: "Donau", NameSource: v1.NameSource_NAME_SOURCE_FILENAME,
		Move: &v1.MoveResult{From: "Danube.md", To: "Donau.md", Repaired: []string{"Deltas.md"}, Dangling: []string{"Estuaries.md"}},
	}), nil
}

func (a *fakeArrangeHost) MoveNotes(
	_ context.Context, req *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	a.record("move " + req.Msg.GetFolder())
	return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Moves: []*v1.HostNoteMove{
		{Move: &v1.MoveResult{From: "Danube.md", To: "Rivers/Danube.md", Repaired: []string{"Deltas.md"}}},
		{Move: &v1.MoveResult{From: "Nowhere.md"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND},
		{Move: &v1.MoveResult{From: "Rivers"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE},
		{Move: &v1.MoveResult{From: "Deltas.md"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS},
	}}), nil
}

func (a *fakeArrangeHost) RemoveNotes(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	a.record("remove " + req.Msg.GetPaths()[0])
	return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Removals: []*v1.HostNoteRemoval{
		{Path: "Danube.md", TrashPath: ".trash/Danube.md", BrokenLinks: []string{"Deltas.md"}},
		{Path: "Rivers", Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE},
		{Path: "Nowhere.md", Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND},
	}}), nil
}

func (a *fakeArrangeHost) GetVaultProblems(
	_ context.Context, req *connect.Request[v1.GetVaultProblemsRequest],
) (*connect.Response[v1.GetVaultProblemsResponse], error) {
	return connect.NewResponse(&v1.GetVaultProblemsResponse{
		Checks: req.Msg.GetChecks(),
		Problems: []*v1.HostVaultProblem{{
			Path: "Deltas.md", Check: "ambiguous", Detail: "Danube matches two notes",
			TargetScheme: "name", TargetValue: "Danube", Candidates: []string{"Rivers/Danube.md", "Ships/Danube.md"},
		}},
	}), nil
}

func (a *fakeArrangeHost) GetNoteRefs(
	_ context.Context, req *connect.Request[v1.GetNoteRefsRequest],
) (*connect.Response[v1.GetNoteRefsResponse], error) {
	notes := make([]*v1.HostNoteRef, 0, len(req.Msg.GetPaths()))
	for _, one := range req.Msg.GetPaths() {
		notes = append(notes, &v1.HostNoteRef{Path: one, Title: "Danube", Id: "01JRIVERDANUBE"})
	}
	return connect.NewResponse(&v1.GetNoteRefsResponse{Notes: notes}), nil
}

func (a *fakeArrangeHost) ListHostVaults(context.Context, *connect.Request[v1.ListHostVaultsRequest]) (*connect.Response[v1.ListHostVaultsResponse], error) {
	return connect.NewResponse(&v1.ListHostVaultsResponse{Vaults: []*v1.HostVaultInfo{
		{Id: "v1", Name: "Rivers", Path: "/vaults/Rivers"},
		{Id: "v2", Name: "Lakes", Path: "/vaults/Lakes", IsMissing: true},
	}}), nil
}

func (a *fakeArrangeHost) AddVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceAddVaultRequest],
) (*connect.Response[v1.HostVaultServiceAddVaultResponse], error) {
	a.record("add " + req.Msg.GetRoot() + " " + req.Msg.GetName())
	if a.vaultsError != 0 {
		return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Error: a.vaultsError, Detail: a.vaultsDetail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Vault: &v1.Vault{Id: "v3", Name: req.Msg.GetName(), Path: req.Msg.GetRoot()}}), nil
}

func (a *fakeArrangeHost) RenameVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRenameVaultRequest],
) (*connect.Response[v1.HostVaultServiceRenameVaultResponse], error) {
	a.record("rename vault " + req.Msg.GetId() + " " + req.Msg.GetName())
	if a.vaultsError != 0 {
		return connect.NewResponse(&v1.HostVaultServiceRenameVaultResponse{Error: a.vaultsError, Detail: a.vaultsDetail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRenameVaultResponse{Vault: &v1.Vault{Id: req.Msg.GetId(), Name: req.Msg.GetName(), Path: "/vaults/Lakes"}}), nil
}

func (a *fakeArrangeHost) RemoveVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRemoveVaultRequest],
) (*connect.Response[v1.HostVaultServiceRemoveVaultResponse], error) {
	a.record("remove vault " + req.Msg.GetId())
	if a.vaultsError != 0 {
		return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Error: a.vaultsError, Detail: a.vaultsDetail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Path: "/vaults/Lakes"}), nil
}

func (a *fakeArrangeHost) OpenVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceOpenVaultRequest],
) (*connect.Response[v1.HostVaultServiceOpenVaultResponse], error) {
	a.record("open vault " + req.Msg.GetId())
	return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{Error: a.vaultsError, Detail: a.vaultsDetail}), nil
}

func newFakeArrangeClient(t *testing.T, a *fakeArrangeHost) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(a))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestTheClientRenamesANoteAndReadsTheMove(t *testing.T) {
	a := &fakeArrangeHost{}

	renamed, err := newFakeArrangeClient(t, a).RenameNote(t.Context(), "Danube.md", "Donau")
	if err != nil {
		t.Fatal(err)
	}

	if renamed.Path != "Donau.md" || renamed.Title != "Donau" || !renamed.IsFilenamed || renamed.Move == nil {
		t.Fatalf("the rename came back as %+v", renamed)
	}
	if m := *renamed.Move; m.From != "Danube.md" || m.To != "Donau.md" || !m.IsMoved ||
		!slices.Equal(m.Repaired, []string{"Deltas.md"}) || !slices.Equal(m.Dangling, []string{"Estuaries.md"}) {
		t.Errorf("the move came back as %+v", m)
	}
	if got := a.getRequests(); !slices.Equal(got, []string{"rename Danube.md Donau"}) {
		t.Errorf("the host received %v", got)
	}
}

func TestClientReturnsTheErrorCodeOfARejectedRename(t *testing.T) {
	for code, want := range map[v1.HostVaultErrorCode]error{
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND:      host.ErrNotExist,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE:     host.ErrNotANote,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS: host.ErrAlreadyExists,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT:       host.ErrStale,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_NAME:   domain.ErrInvalidName,
	} {
		_, err := newFakeArrangeClient(t, &fakeArrangeHost{noteError: code}).RenameNote(t.Context(), "Danube.md", "Donau")
		if !errors.Is(err, want) {
			t.Errorf("error %s: got %v, want %v", code, err, want)
		}
	}
}

func TestTheClientMovesNotesAndReturnsEachFailure(t *testing.T) {
	a := &fakeArrangeHost{}

	moves, err := newFakeArrangeClient(t, a).MoveNotes(t.Context(), []string{"Danube.md", "Nowhere.md", "Rivers", "Deltas.md"}, "Rivers")
	if err != nil {
		t.Fatal(err)
	}

	if len(moves) != 4 {
		t.Fatalf("the moves came back as %+v, want 4", moves)
	}
	if m := moves[0]; m.Err != nil || m.Move.From != "Danube.md" || m.Move.To != "Rivers/Danube.md" || !m.Move.IsMoved ||
		!slices.Equal(m.Move.Repaired, []string{"Deltas.md"}) {
		t.Errorf("the first move came back as %+v", m)
	}
	for i, want := range map[int]error{1: host.ErrNotExist, 2: host.ErrNotANote, 3: host.ErrAlreadyExists} {
		if m := moves[i]; !errors.Is(m.Err, want) || m.Move.IsMoved || m.Move.From == "" {
			t.Errorf("move %d came back as %+v, want the error %v", i, m, want)
		}
	}
	if got := a.getRequests(); !slices.Equal(got, []string{"move Rivers"}) {
		t.Errorf("the host received %v", got)
	}
}

func TestTheClientRemovesNotesAndReturnsAFolderAsNotANote(t *testing.T) {
	removals, err := newFakeArrangeClient(t, &fakeArrangeHost{}).RemoveNotes(t.Context(), []string{"Danube.md", "Rivers", "Nowhere.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(removals) != 3 {
		t.Fatalf("the removals came back as %+v, want 3", removals)
	}
	if r := removals[0]; r.Err != nil || r.Path != "Danube.md" || r.TrashPath != ".trash/Danube.md" || !slices.Equal(r.BrokenLinks, []string{"Deltas.md"}) {
		t.Errorf("the first removal came back as %+v", r)
	}
	if r := removals[1]; r.Path != "Rivers" || !errors.Is(r.Err, host.ErrNotANote) {
		t.Errorf("the folder came back as %+v, want ErrNotANote", r)
	}
	if r := removals[2]; r.Path != "Nowhere.md" || !errors.Is(r.Err, host.ErrNotExist) {
		t.Errorf("the missing note came back as %+v, want ErrNotExist", r)
	}
}

func TestTheClientRunsTheNamedChecks(t *testing.T) {
	a := &fakeArrangeHost{}

	problems, err := newFakeArrangeClient(t, a).GetProblems(t.Context(), []string{"ambiguous", "dangling"})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(problems.Checks, []string{"ambiguous", "dangling"}) {
		t.Errorf("checks came back as %v", problems.Checks)
	}
	if len(problems.Found) != 1 {
		t.Fatalf("the problems came back as %+v, want one", problems.Found)
	}
	p := problems.Found[0]
	if p.Path != "Deltas.md" || string(p.Kind) != "ambiguous" || p.Detail != "Danube matches two notes" ||
		string(p.Target.Scheme) != "name" || p.Target.Value != "Danube" ||
		!slices.Equal(p.Candidates, []string{"Rivers/Danube.md", "Ships/Danube.md"}) {
		t.Errorf("the problem came back as %+v", p)
	}
}

func TestTheClientReadsTheIdentifierOfEachNote(t *testing.T) {
	notes, err := newFakeArrangeClient(t, &fakeArrangeHost{}).GetNotes(t.Context(), []string{"Danube.md"})

	if err != nil || notes["Danube.md"].ID != "01JRIVERDANUBE" {
		t.Errorf("the notes came back as %+v (%v), want Danube.md with id 01JRIVERDANUBE", notes, err)
	}
}

func TestTheClientListsTheVaultsAndMarksAMissingFolder(t *testing.T) {
	known, err := newFakeArrangeClient(t, &fakeArrangeHost{}).GetVaultList(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []host.VaultEntry{
		{Vault: domain.Vault{ID: "v1", Name: "Rivers", Path: "/vaults/Rivers"}},
		{Vault: domain.Vault{ID: "v2", Name: "Lakes", Path: "/vaults/Lakes"}, IsMissing: true},
	}
	if !slices.Equal(known, want) {
		t.Errorf("the vaults came back as %+v, want %+v", known, want)
	}
}

func TestTheClientChangesTheListOfVaults(t *testing.T) {
	a := &fakeArrangeHost{}
	client := newFakeArrangeClient(t, a)
	ctx := t.Context()

	added, addErr := client.AddVault(ctx, "/vaults/Seas", "Seas")
	renamed, renameErr := client.RenameVault(ctx, "v2", "Ponds")
	folder, removeErr := client.RemoveVault(ctx, "v2")
	openErr := client.OpenVault(ctx, "v1")

	if addErr != nil || added != (domain.Vault{ID: "v3", Name: "Seas", Path: "/vaults/Seas"}) {
		t.Errorf("the added vault came back as %+v (%v)", added, addErr)
	}
	if renameErr != nil || renamed != (domain.Vault{ID: "v2", Name: "Ponds", Path: "/vaults/Lakes"}) {
		t.Errorf("the renamed vault came back as %+v (%v)", renamed, renameErr)
	}
	if removeErr != nil || folder != "/vaults/Lakes" {
		t.Errorf("the removal came back as %q (%v), want the folder /vaults/Lakes", folder, removeErr)
	}
	if openErr != nil {
		t.Errorf("the open returned %v", openErr)
	}
	want := []string{"add /vaults/Seas Seas", "rename vault v2 Ponds", "remove vault v2", "open vault v1"}
	if got := a.getRequests(); !slices.Equal(got, want) {
		t.Errorf("the host received %v, want %v", got, want)
	}
}

func TestClientReturnsEachVaultErrorCodeWithTheDetail(t *testing.T) {
	for code, want := range map[v1.VaultsErrorCode]error{
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN: host.ErrVaultNameTaken,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT: host.ErrLastVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT:    host.ErrVaultActive,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND:  host.ErrUnknownVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE: host.ErrVaultUnreadable,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:       host.ErrVaultDuplicate,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:   host.ErrVaultOverlaps,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED:    host.ErrUnsavedChanges,
	} {
		client := newFakeArrangeClient(t, &fakeArrangeHost{vaultsError: code, vaultsDetail: "vault v2: refused by the core"})
		ctx := t.Context()

		_, addErr := client.AddVault(ctx, "/vaults/Seas", "")
		_, renameErr := client.RenameVault(ctx, "v2", "Ponds")
		_, removeErr := client.RemoveVault(ctx, "v2")
		openErr := client.OpenVault(ctx, "v2")

		for call, err := range map[string]error{"AddVault": addErr, "RenameVault": renameErr, "RemoveVault": removeErr, "OpenVault": openErr} {
			if !errors.Is(err, want) {
				t.Errorf("%s with error %s: got %v, want %v", call, code, err, want)
			}
			if err == nil || !strings.Contains(err.Error(), "vault v2: refused by the core") {
				t.Errorf("%s with error %s: got %v, want the detail of the core", call, code, err)
			}
		}
	}
}

// addVaultErrorHost is a host whose AddVault rejects the folder with the core's message.
type addVaultErrorHost struct {
	fakeArrangeHost
	detail string
}

func (a *addVaultErrorHost) AddVault(
	context.Context, *connect.Request[v1.HostVaultServiceAddVaultRequest],
) (*connect.Response[v1.HostVaultServiceAddVaultResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Error: v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE, Detail: a.detail}), nil
}

func TestTheClientReturnsTheCoreMessageOfARejectedFolder(t *testing.T) {
	const detail = "/Users/river is the user's home directory, and a vault is a folder inside it"
	server := httptest.NewServer(handler(&addVaultErrorHost{detail: detail}))
	t.Cleanup(server.Close)
	client := host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})

	_, err := client.AddVault(t.Context(), "/Users/river", "")

	if !errors.Is(err, host.ErrVaultUnreadable) {
		t.Errorf("AddVault of the home folder returned %v, want ErrVaultUnreadable", err)
	}
	if err == nil || !strings.Contains(err.Error(), detail) {
		t.Errorf("the error is %v, want it to have %q", err, detail)
	}
}

// openVaultErrorHost is a host whose OpenVault rejects the open because the window has unsaved changes,
// with the core's message.
type openVaultErrorHost struct {
	fakeArrangeHost
	detail string
}

func (a *openVaultErrorHost) OpenVault(
	context.Context, *connect.Request[v1.HostVaultServiceOpenVaultRequest],
) (*connect.Response[v1.HostVaultServiceOpenVaultResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{Error: v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED, Detail: a.detail}), nil
}

func TestClientReturnsUnsavedChangesWithCoreMessageOfOpen(t *testing.T) {
	const detail = "open vault v2: the window has unsaved changes and is prompting the user; retry after the prompt closes"
	server := httptest.NewServer(handler(&openVaultErrorHost{detail: detail}))
	t.Cleanup(server.Close)
	client := host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})

	err := client.OpenVault(t.Context(), "v2")

	if !errors.Is(err, host.ErrUnsavedChanges) {
		t.Errorf("OpenVault of a window with unsaved changes returned %v, want ErrUnsavedChanges", err)
	}
	if err == nil || !strings.Contains(err.Error(), detail) {
		t.Errorf("the error is %v, want it to contain %q", err, detail)
	}
}

func TestErrUnsavedChangesContainsThePromptText(t *testing.T) {
	const want = "the window has unsaved changes and is prompting the user; retry after the prompt closes"
	if got := host.ErrUnsavedChanges.Error(); !strings.Contains(got, want) {
		t.Errorf("ErrUnsavedChanges is %q, want it to contain %q", got, want)
	}
}

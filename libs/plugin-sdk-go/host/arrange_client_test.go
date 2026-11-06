package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// arranging is a host of a vault of rivers that serves the calls renaming, moving and removing
// notes, running checks and changing the list of vaults, and records what it was asked.
type arranging struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	mu sync.Mutex
	// noteFailure, when set, is the per-note failure of every note answer.
	noteFailure *v1.ErrorCode
	// vaultsFailure, when set, is the failure of every change of the list of vaults.
	vaultsFailure *v1.VaultsErrorCode
	requests      []string
}

func (a *arranging) record(request string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requests = append(a.requests, request)
}

func (a *arranging) getRequests() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.requests)
}

func (a *arranging) RenameNote(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	a.record("rename " + req.Msg.GetPath() + " " + req.Msg.GetTitle())
	if a.noteFailure != nil {
		return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Path: req.Msg.GetPath(), Failure: a.noteFailure}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{
		Path: "Donau.md", Title: "Donau", NameSource: v1.NameSource_NAME_SOURCE_FILENAME,
		Moved: &v1.MoveResult{From: "Danube.md", To: "Donau.md", Repaired: []string{"Deltas.md"}, Dangling: []string{"Estuaries.md"}},
	}), nil
}

func (a *arranging) MoveNotes(
	_ context.Context, req *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	a.record("move " + req.Msg.GetFolder())
	missing := v1.ErrorCode_ERROR_CODE_MISSING
	notANote := v1.ErrorCode_ERROR_CODE_NOT_A_NOTE
	exists := v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS
	return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Moves: []*v1.HostNoteMove{
		{Moved: &v1.MoveResult{From: "Danube.md", To: "Rivers/Danube.md", Repaired: []string{"Deltas.md"}}},
		{Moved: &v1.MoveResult{From: "Nowhere.md"}, Failure: &missing},
		{Moved: &v1.MoveResult{From: "Rivers"}, Failure: &notANote},
		{Moved: &v1.MoveResult{From: "Deltas.md"}, Failure: &exists},
	}}), nil
}

func (a *arranging) RemoveNotes(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	a.record("remove " + req.Msg.GetPaths()[0])
	notANote := v1.ErrorCode_ERROR_CODE_NOT_A_NOTE
	missing := v1.ErrorCode_ERROR_CODE_MISSING
	return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Removals: []*v1.HostNoteRemoval{
		{Path: "Danube.md", Trashed: ".trash/Danube.md", Dangling: []string{"Deltas.md"}},
		{Path: "Rivers", Failure: &notANote},
		{Path: "Nowhere.md", Failure: &missing},
	}}), nil
}

func (a *arranging) GetVaultProblems(
	_ context.Context, req *connect.Request[v1.GetVaultProblemsRequest],
) (*connect.Response[v1.GetVaultProblemsResponse], error) {
	return connect.NewResponse(&v1.GetVaultProblemsResponse{
		Ran: req.Msg.GetChecks(),
		Problems: []*v1.HostVaultProblem{{
			Path: "Deltas.md", Check: "ambiguous", Detail: "Danube matches two notes",
			TargetScheme: "name", TargetValue: "Danube", Candidates: []string{"Rivers/Danube.md", "Ships/Danube.md"},
		}},
	}), nil
}

func (a *arranging) GetNoteRefs(
	_ context.Context, req *connect.Request[v1.GetNoteRefsRequest],
) (*connect.Response[v1.GetNoteRefsResponse], error) {
	notes := make([]*v1.HostNoteRef, 0, len(req.Msg.GetPaths()))
	for _, one := range req.Msg.GetPaths() {
		notes = append(notes, &v1.HostNoteRef{Path: one, Title: "Danube", Id: "01JRIVERDANUBE"})
	}
	return connect.NewResponse(&v1.GetNoteRefsResponse{Notes: notes}), nil
}

func (a *arranging) ListHostVaults(context.Context, *connect.Request[v1.ListHostVaultsRequest]) (*connect.Response[v1.ListHostVaultsResponse], error) {
	return connect.NewResponse(&v1.ListHostVaultsResponse{Vaults: []*v1.HostVaultInfo{
		{Id: "v1", Name: "Rivers", Path: "/vaults/Rivers"},
		{Id: "v2", Name: "Lakes", Path: "/vaults/Lakes", IsMissing: true},
	}}), nil
}

func (a *arranging) AddVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceAddVaultRequest],
) (*connect.Response[v1.HostVaultServiceAddVaultResponse], error) {
	a.record("add " + req.Msg.GetRoot() + " " + req.Msg.GetName())
	if a.vaultsFailure != nil {
		return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Failure: a.vaultsFailure}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Vault: &v1.Vault{Id: "v3", Name: req.Msg.GetName(), Path: req.Msg.GetRoot()}}), nil
}

func (a *arranging) RenameVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRenameVaultRequest],
) (*connect.Response[v1.HostVaultServiceRenameVaultResponse], error) {
	a.record("rename vault " + req.Msg.GetId() + " " + req.Msg.GetName())
	if a.vaultsFailure != nil {
		return connect.NewResponse(&v1.HostVaultServiceRenameVaultResponse{Failure: a.vaultsFailure}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRenameVaultResponse{Vault: &v1.Vault{Id: req.Msg.GetId(), Name: req.Msg.GetName(), Path: "/vaults/Lakes"}}), nil
}

func (a *arranging) RemoveVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRemoveVaultRequest],
) (*connect.Response[v1.HostVaultServiceRemoveVaultResponse], error) {
	a.record("remove vault " + req.Msg.GetId())
	if a.vaultsFailure != nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Failure: a.vaultsFailure}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Path: "/vaults/Lakes"}), nil
}

func (a *arranging) OpenVault(
	_ context.Context, req *connect.Request[v1.HostVaultServiceOpenVaultRequest],
) (*connect.Response[v1.HostVaultServiceOpenVaultResponse], error) {
	a.record("open vault " + req.Msg.GetId())
	return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{Failure: a.vaultsFailure}), nil
}

func arrangingClient(t *testing.T, a *arranging) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(a))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestTheClientRenamesANoteAndReadsTheMove(t *testing.T) {
	a := &arranging{}

	renamed, err := arrangingClient(t, a).RenameNote(t.Context(), "Danube.md", "Donau")
	if err != nil {
		t.Fatal(err)
	}

	if renamed.Path != "Donau.md" || renamed.Title != "Donau" || !renamed.IsFilenamed || renamed.Moved == nil {
		t.Fatalf("the rename came back as %+v", renamed)
	}
	if m := *renamed.Moved; m.From != "Danube.md" || m.To != "Donau.md" || !m.IsMoved ||
		!slices.Equal(m.Repaired, []string{"Deltas.md"}) || !slices.Equal(m.Dangling, []string{"Estuaries.md"}) {
		t.Errorf("the move came back as %+v", m)
	}
	if got := a.getRequests(); !slices.Equal(got, []string{"rename Danube.md Donau"}) {
		t.Errorf("the host was asked %v", got)
	}
}

func TestTheClientReturnsTheFailureOfARenameAsItsError(t *testing.T) {
	for code, want := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_MISSING:        host.ErrNotExist,
		v1.ErrorCode_ERROR_CODE_NOT_A_NOTE:     host.ErrNotANote,
		v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS: host.ErrAlreadyExists,
		v1.ErrorCode_ERROR_CODE_STALE:          host.ErrStale,
		v1.ErrorCode_ERROR_CODE_INVALID_NAME:   domain.ErrInvalidName,
	} {
		_, err := arrangingClient(t, &arranging{noteFailure: &code}).RenameNote(t.Context(), "Danube.md", "Donau")
		if !errors.Is(err, want) {
			t.Errorf("failure %s: got %v, want %v", code, err, want)
		}
	}
}

func TestTheClientMovesNotesAndReturnsEachFailure(t *testing.T) {
	a := &arranging{}

	moves, err := arrangingClient(t, a).MoveNotes(t.Context(), []string{"Danube.md", "Nowhere.md", "Rivers", "Deltas.md"}, "Rivers")
	if err != nil {
		t.Fatal(err)
	}

	if len(moves) != 4 {
		t.Fatalf("the moves came back as %+v, want 4", moves)
	}
	if m := moves[0]; m.Err != nil || m.Moved.From != "Danube.md" || m.Moved.To != "Rivers/Danube.md" || !m.Moved.IsMoved ||
		!slices.Equal(m.Moved.Repaired, []string{"Deltas.md"}) {
		t.Errorf("the first move came back as %+v", m)
	}
	for i, want := range map[int]error{1: host.ErrNotExist, 2: host.ErrNotANote, 3: host.ErrAlreadyExists} {
		if m := moves[i]; !errors.Is(m.Err, want) || m.Moved.IsMoved || m.Moved.From == "" {
			t.Errorf("move %d came back as %+v, want the error %v", i, m, want)
		}
	}
	if got := a.getRequests(); !slices.Equal(got, []string{"move Rivers"}) {
		t.Errorf("the host was asked %v", got)
	}
}

func TestTheClientRemovesNotesAndReturnsAFolderAsNotANote(t *testing.T) {
	removals, err := arrangingClient(t, &arranging{}).RemoveNotes(t.Context(), []string{"Danube.md", "Rivers", "Nowhere.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(removals) != 3 {
		t.Fatalf("the removals came back as %+v, want 3", removals)
	}
	if r := removals[0]; r.Err != nil || r.Path != "Danube.md" || r.Trashed != ".trash/Danube.md" || !slices.Equal(r.Dangling, []string{"Deltas.md"}) {
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
	a := &arranging{}

	problems, err := arrangingClient(t, a).GetProblems(t.Context(), []string{"ambiguous", "dangling"})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(problems.Ran, []string{"ambiguous", "dangling"}) {
		t.Errorf("ran came back as %v", problems.Ran)
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
	notes, err := arrangingClient(t, &arranging{}).GetNotes(t.Context(), []string{"Danube.md"})

	if err != nil || notes["Danube.md"].ID != "01JRIVERDANUBE" {
		t.Errorf("the notes came back as %+v (%v), want Danube.md with id 01JRIVERDANUBE", notes, err)
	}
}

func TestTheClientListsTheVaultsAndMarksAMissingFolder(t *testing.T) {
	known, err := arrangingClient(t, &arranging{}).GetVaultList(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []host.KnownVault{
		{Vault: domain.Vault{ID: "v1", Name: "Rivers", Path: "/vaults/Rivers"}},
		{Vault: domain.Vault{ID: "v2", Name: "Lakes", Path: "/vaults/Lakes"}, IsMissing: true},
	}
	if !slices.Equal(known, want) {
		t.Errorf("the vaults came back as %+v, want %+v", known, want)
	}
}

func TestTheClientChangesTheListOfVaults(t *testing.T) {
	a := &arranging{}
	client := arrangingClient(t, a)
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
		t.Errorf("the host was asked %v, want %v", got, want)
	}
}

func TestTheClientReturnsEachFailureOfTheListOfVaultsAsItsError(t *testing.T) {
	for code, want := range map[v1.VaultsErrorCode]error{
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN: host.ErrVaultNameTaken,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT: host.ErrLastVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING:    host.ErrVaultShowing,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN:    host.ErrUnknownVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE: host.ErrVaultUnreadable,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:       host.ErrVaultCopy,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:   host.ErrVaultOverlaps,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_ASKING:     host.ErrVaultAsking,
	} {
		client := arrangingClient(t, &arranging{vaultsFailure: &code})
		ctx := t.Context()

		_, addErr := client.AddVault(ctx, "/vaults/Seas", "")
		_, renameErr := client.RenameVault(ctx, "v2", "Ponds")
		_, removeErr := client.RemoveVault(ctx, "v2")
		openErr := client.OpenVault(ctx, "v2")

		for call, err := range map[string]error{"AddVault": addErr, "RenameVault": renameErr, "RemoveVault": removeErr, "OpenVault": openErr} {
			if !errors.Is(err, want) {
				t.Errorf("%s with failure %s: got %v, want %v", call, code, err, want)
			}
		}
	}
}

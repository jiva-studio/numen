package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// GetPluginVaultList is no list of vaults: the fake host of the other tests lends none.
func (c *callers) GetPluginVaultList() pluginport.PluginVaultList { return nil }

// listCallers is the fake host that lends its plugins a list of vaults and grants md.numen.a only
// the permissions named.
type listCallers struct {
	*callers
	list    pluginport.PluginVaultList
	granted []string
}

func (c listCallers) GetPluginVaultList() pluginport.PluginVaultList { return c.list }

func (c listCallers) CheckPluginPermission(pluginID, permission string) error {
	if err := c.callers.CheckPluginPermission(pluginID, permission); err != nil {
		return err
	}
	if !slices.Contains(c.granted, permission) {
		return pluginport.ErrPermissionDenied
	}
	return nil
}

// arrangingVault is the vault of rivers that renames, moves and removes its notes and runs its
// checks as the host does, and records what it was asked.
type arrangingVault struct {
	*memoryVault
	folders map[string]bool
	moved   []domain.Move
	removed []string
	checks  [][]domain.Check
}

func newArrangingVault() *arrangingVault {
	return &arrangingVault{
		memoryVault: newMemoryVault(map[string]string{
			"Danube.md":        "# Danube\n",
			"Deltas.md":        "---\nlinks:\n  - to: Danube\n    role: parent\n---\n# Deltas\n",
			"Rivers/Rhine.md":  "# Rhine\n",
			"Rivers/Deltas.md": "# Deltas\n",
		}),
		folders: map[string]bool{"Rivers": true},
	}
}

func (a *arrangingVault) GetVaults() ([]domain.Vault, error) {
	return []domain.Vault{
		{ID: "v1", Name: "Rivers", Path: "/vaults/Rivers"},
		{ID: "v2", Name: "Lakes", Path: "/vaults/Lakes"},
	}, nil
}

func (a *arrangingVault) GetNotes(_ context.Context, paths []string) (map[string]domain.NoteRef, error) {
	out := map[string]domain.NoteRef{}
	for _, one := range paths {
		if _, found := a.files[one]; found {
			out[one] = domain.NoteRef{Path: one, Title: domain.Basename(one), ID: "01JRIVER" + domain.Basename(one)}
		}
	}
	return out, nil
}

func (a *arrangingVault) RenameNote(_ context.Context, from, title string) (string, bool, *domain.MoveResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, found := a.files[from]; !found {
		return "", false, nil, fmt.Errorf("rename %s: %w", from, vaultport.ErrNotExist)
	}
	to := path.Join(path.Dir(from), title+".md")
	a.files[to] = a.files[from]
	delete(a.files, from)
	return to, true, &domain.MoveResult{From: from, To: to, IsMoved: true, Repaired: []string{"Deltas.md"}}, nil
}

func (a *arrangingVault) MoveNote(_ context.Context, from, to string) (domain.MoveResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.moved = append(a.moved, domain.Move{From: from, To: to})
	if _, found := a.files[from]; !found {
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, vaultport.ErrNotExist)
	}
	if _, found := a.files[to]; found {
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, vaultport.ErrAlreadyExists)
	}
	a.files[to] = a.files[from]
	delete(a.files, from)
	return domain.MoveResult{From: from, To: to, IsMoved: true}, nil
}

func (a *arrangingVault) RemoveNote(_ context.Context, target string) (string, []string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.removed = append(a.removed, target)
	if a.folders[target] {
		return "", nil, fmt.Errorf("remove %s: %w", target, vaultport.ErrNotANote)
	}
	if _, found := a.files[target]; !found {
		return "", nil, fmt.Errorf("remove %s: %w", target, vaultport.ErrNotExist)
	}
	delete(a.files, target)
	return ".trash/" + target, []string{"Deltas.md"}, nil
}

func (a *arrangingVault) GetProblems(_ context.Context, checks []domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checks = append(a.checks, checks)
	ran := checks
	if len(ran) == 0 {
		ran = []domain.Check{domain.CheckParse, domain.CheckFrontmatter, domain.CheckAmbiguous}
	}
	return ran, []domain.VaultProblem{{
		Path: "Deltas.md", Kind: domain.CheckAmbiguous, Detail: "Danube matches two notes",
		Target:     domain.Address{Scheme: domain.SchemeName, Value: "Danube"},
		Candidates: []string{"Rivers/Danube.md", "Ships/Danube.md"},
	}}, nil
}

// recordingList is a list of vaults that records the changes it is asked for and returns err for each.
// Its OpenVault returns err at once, else a swap that waits for release or the end of its context
// when release is set, and then sends the error of its context.
type recordingList struct {
	mu      sync.Mutex
	err     error
	missing map[domain.VaultID]bool
	calls   []string
	opened  chan domain.VaultID
	release chan struct{}
	ended   chan error
}

func newRecordingList() *recordingList {
	return &recordingList{
		missing: map[domain.VaultID]bool{"v2": true},
		opened:  make(chan domain.VaultID, 4),
		ended:   make(chan error, 4),
	}
}

func (l *recordingList) record(call string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call)
	return l.err
}

func (l *recordingList) getCalls() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.calls)
}

func (l *recordingList) AddVault(_ context.Context, root, name string) (domain.Vault, error) {
	if err := l.record("add " + root + " " + name); err != nil {
		return domain.Vault{}, err
	}
	return domain.Vault{ID: "v3", Name: "Seas", Path: root}, nil
}

func (l *recordingList) RenameVault(_ context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	if err := l.record("rename " + string(id) + " " + name); err != nil {
		return domain.Vault{}, err
	}
	return domain.Vault{ID: id, Name: name, Path: "/vaults/Lakes"}, nil
}

func (l *recordingList) RemoveVault(_ context.Context, id domain.VaultID) error {
	return l.record("remove " + string(id))
}

func (l *recordingList) OpenVault(_ context.Context, id domain.VaultID) (func(context.Context) error, error) {
	l.mu.Lock()
	openErr := l.err
	l.mu.Unlock()
	if openErr != nil {
		return nil, openErr
	}
	return func(ctx context.Context) error {
		err := l.record("open " + string(id))
		l.opened <- id
		if l.release != nil {
			select {
			case <-l.release:
			case <-ctx.Done():
			}
		}
		l.ended <- ctx.Err()
		return err
	}, nil
}

func (l *recordingList) IsMissing(v domain.Vault) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.missing[v.ID]
}

// newArrangeClient serves the vault endpoint over a host lending the vault of rivers and the list,
// nil for none, with md.numen.a granted the permissions named.
func newArrangeClient(
	t *testing.T, list pluginport.PluginVaultList, granted ...string,
) (numenv1connect.HostVaultServiceClient, *arrangingVault, *callers) {
	t.Helper()
	vault := newArrangingVault()
	hosted := newCallers(nil)
	hosted.vault = vault
	endpoint := &Endpoint{callers: listCallers{callers: hosted, list: list, granted: granted}}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostVaultServiceHandler(newVaultEndpoint(endpoint)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	return client, vault, hosted
}

func isDenied(err error) bool {
	var failure *connect.Error
	return errors.As(err, &failure) && failure.Code() == connect.CodePermissionDenied
}

// callNoteChanges makes the rename, the move and the removal of a note and returns the error of each.
func callNoteChanges(t *testing.T, client numenv1connect.HostVaultServiceClient) map[string]error {
	t.Helper()
	ctx := t.Context()
	_, renameErr := client.RenameNote(ctx, connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
	_, moveErr := client.MoveNotes(ctx, connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{Paths: []string{"Rivers/Rhine.md"}}))
	_, removeErr := client.RemoveNotes(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{Paths: []string{"Deltas.md"}}))
	return map[string]error{"RenameNote": renameErr, "MoveNotes": moveErr, "RemoveNotes": removeErr}
}

// callVaultChanges makes the four changes of the list of vaults and returns the error of each.
func callVaultChanges(t *testing.T, client numenv1connect.HostVaultServiceClient) map[string]error {
	t.Helper()
	ctx := t.Context()
	_, addErr := client.AddVault(ctx, connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: "/vaults/Seas"}))
	_, renameErr := client.RenameVault(ctx, connect.NewRequest(&v1.HostVaultServiceRenameVaultRequest{Id: "v2", Name: "Ponds"}))
	_, removeErr := client.RemoveVault(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveVaultRequest{Id: "v2"}))
	_, openErr := client.OpenVault(ctx, connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))
	return map[string]error{"AddVault": addErr, "RenameVault": renameErr, "RemoveVault": removeErr, "OpenVault": openErr}
}

func TestNoteChangesAreDeniedWithoutVaultWrite(t *testing.T) {
	client, vault, _ := newArrangeClient(t, newRecordingList(), pluginport.PermissionVaultRead)

	for call, err := range callNoteChanges(t, client) {
		if !isDenied(err) {
			t.Errorf("%s with only vault:read returned %v, want permission denied", call, err)
		}
	}
	if _, found := vault.files["Danube.md"]; !found {
		t.Error("Danube.md was renamed by a plugin without vault:write")
	}
}

func TestNoteChangesAreServedWithVaultWrite(t *testing.T) {
	client, _, hosted := newArrangeClient(t, newRecordingList(), pluginport.PermissionVaultWrite)

	for call, err := range callNoteChanges(t, client) {
		if err != nil {
			t.Errorf("%s with vault:write returned %v", call, err)
		}
	}
	hosted.mu.Lock()
	defer hosted.mu.Unlock()
	for _, one := range hosted.checked {
		if one != "md.numen.a "+pluginport.PermissionVaultWrite {
			t.Errorf("a note change checked %q, want vault:write only", one)
		}
	}
}

func TestVaultListChangesAreDeniedWithOnlyVaultWrite(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, pluginport.PermissionVaultRead, pluginport.PermissionVaultWrite)

	for call, err := range callVaultChanges(t, client) {
		if !isDenied(err) {
			t.Errorf("%s without vault:manage returned %v, want permission denied", call, err)
		}
	}
	if calls := list.getCalls(); len(calls) != 0 {
		t.Errorf("the list was changed without vault:manage: %v", calls)
	}
}

func TestVaultListChangesAreServedWithVaultManage(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, pluginport.PermissionVaultManage)

	errs := callVaultChanges(t, client)
	for call, err := range errs {
		if err != nil {
			t.Errorf("%s with vault:manage returned %v", call, err)
		}
	}
	if errs["OpenVault"] != nil {
		t.FailNow()
	}
	select {
	case <-list.opened:
	case <-time.After(10 * time.Second):
		t.Fatal("the list was not asked to open the vault")
	}
	want := []string{"add /vaults/Seas ", "rename v2 Ponds", "remove v2", "open v2"}
	if calls := list.getCalls(); !slices.Equal(calls, want) {
		t.Errorf("the list was asked %v, want %v", calls, want)
	}
}

func TestVaultChecksNeedVaultRead(t *testing.T) {
	denied, _, _ := newArrangeClient(t, nil)
	_, err := denied.GetVaultProblems(t.Context(), connect.NewRequest(&v1.GetVaultProblemsRequest{}))
	if !isDenied(err) {
		t.Errorf("the checks without vault:read returned %v, want permission denied", err)
	}

	client, vault, _ := newArrangeClient(t, nil, pluginport.PermissionVaultRead)
	res, err := client.GetVaultProblems(t.Context(), connect.NewRequest(&v1.GetVaultProblemsRequest{Checks: []string{"ambiguous", "dangling"}}))
	if err != nil {
		t.Fatalf("the checks with vault:read returned %v", err)
	}

	if !slices.Equal(res.Msg.GetRan(), []string{"ambiguous", "dangling"}) {
		t.Errorf("ran is %v, want ambiguous and dangling", res.Msg.GetRan())
	}
	problems := res.Msg.GetProblems()
	if len(problems) != 1 {
		t.Fatalf("the problems were %v, want one", problems)
	}
	problem := problems[0]
	if problem.GetPath() != "Deltas.md" || problem.GetCheck() != "ambiguous" || problem.GetDetail() != "Danube matches two notes" ||
		problem.GetTargetScheme() != "name" || problem.GetTargetValue() != "Danube" ||
		!slices.Equal(problem.GetCandidates(), []string{"Rivers/Danube.md", "Ships/Danube.md"}) {
		t.Errorf("the problem came out as %v", problem)
	}
	if len(vault.checks) != 1 || !slices.Equal(vault.checks[0], []domain.Check{domain.CheckAmbiguous, domain.CheckDangling}) {
		t.Errorf("the vault was asked for the checks %v", vault.checks)
	}
}

func TestARenameReturnsTheNewPathAndTheMove(t *testing.T) {
	client, _, _ := newArrangeClient(t, nil, pluginport.PermissionVaultWrite)

	res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.Failure != nil || res.Msg.GetPath() != "Donau.md" || res.Msg.GetTitle() != "Donau" ||
		res.Msg.GetNameSource() != v1.NameSource_NAME_SOURCE_FILENAME {
		t.Errorf("the rename returned %v", res.Msg)
	}
	moved := res.Msg.GetMoved()
	if moved.GetFrom() != "Danube.md" || moved.GetTo() != "Donau.md" || !slices.Equal(moved.GetRepaired(), []string{"Deltas.md"}) {
		t.Errorf("the move came out as %v", moved)
	}

	missing, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Nowhere.md", Title: "Elsewhere"}))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Msg.Failure == nil || missing.Msg.GetFailure() != v1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("a rename of a missing note returned %v, want failure MISSING", missing.Msg)
	}
}

func TestAMoveReturnsOneEntryPerPathInOrder(t *testing.T) {
	client, vault, _ := newArrangeClient(t, nil, pluginport.PermissionVaultWrite)

	res, err := client.MoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: []string{"Danube.md", "Nowhere.md", "Deltas.md"}, Folder: "Rivers",
	}))
	if err != nil {
		t.Fatal(err)
	}

	moves := res.Msg.GetMoves()
	if len(moves) != 3 {
		t.Fatalf("the move returned %d entries, want 3: %v", len(moves), moves)
	}
	if moves[0].Failure != nil || moves[0].GetMoved().GetFrom() != "Danube.md" || moves[0].GetMoved().GetTo() != "Rivers/Danube.md" {
		t.Errorf("the first entry is %v, want Danube.md moved to Rivers/Danube.md", moves[0])
	}
	if moves[1].GetMoved().GetFrom() != "Nowhere.md" || moves[1].GetFailure() != v1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("the second entry is %v, want Nowhere.md with failure MISSING", moves[1])
	}
	if moves[2].GetMoved().GetFrom() != "Deltas.md" || moves[2].GetFailure() != v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS {
		t.Errorf("the third entry is %v, want Deltas.md with failure ALREADY_EXISTS", moves[2])
	}

	root, err := client.MoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{Paths: []string{"Rivers/Rhine.md"}}))
	if err != nil {
		t.Fatal(err)
	}
	if got := root.Msg.GetMoves(); len(got) != 1 || got[0].GetMoved().GetTo() != "Rhine.md" {
		t.Errorf("a move to the empty folder returned %v, want Rhine.md at the vault root", got)
	}
	want := []domain.Move{
		{From: "Danube.md", To: "Rivers/Danube.md"},
		{From: "Nowhere.md", To: "Rivers/Nowhere.md"},
		{From: "Deltas.md", To: "Rivers/Deltas.md"},
		{From: "Rivers/Rhine.md", To: "Rhine.md"},
	}
	if !slices.Equal(vault.moved, want) {
		t.Errorf("the vault was asked to move %v, want %v", vault.moved, want)
	}
}

func TestARemovalReturnsTheTrashPathAndAFolderAsNotANote(t *testing.T) {
	client, vault, _ := newArrangeClient(t, nil, pluginport.PermissionVaultWrite)

	res, err := client.RemoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: []string{"Danube.md", "Rivers", "Nowhere.md"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	removals := res.Msg.GetRemovals()
	if len(removals) != 3 {
		t.Fatalf("the removal returned %d entries, want 3: %v", len(removals), removals)
	}
	if removals[0].Failure != nil || removals[0].GetPath() != "Danube.md" || removals[0].GetTrashed() != ".trash/Danube.md" ||
		!slices.Equal(removals[0].GetDangling(), []string{"Deltas.md"}) {
		t.Errorf("the first entry is %v, want Danube.md in .trash with Deltas.md dangling", removals[0])
	}
	if removals[1].GetPath() != "Rivers" || removals[1].Failure == nil || removals[1].GetFailure() != v1.ErrorCode_ERROR_CODE_NOT_A_NOTE {
		t.Errorf("the folder entry is %v, want failure NOT_A_NOTE", removals[1])
	}
	if removals[2].GetPath() != "Nowhere.md" || removals[2].GetFailure() != v1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("the third entry is %v, want failure MISSING", removals[2])
	}
	if !slices.Equal(vault.removed, []string{"Danube.md", "Rivers", "Nowhere.md"}) {
		t.Errorf("the vault was asked to remove %v", vault.removed)
	}
}

func TestTheListOfVaultsMarksAMissingFolder(t *testing.T) {
	client, _, _ := newArrangeClient(t, newRecordingList(), pluginport.PermissionVaultRead)

	res, err := client.ListHostVaults(t.Context(), connect.NewRequest(&v1.ListHostVaultsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	vaultInfos := res.Msg.GetVaults()
	if len(vaultInfos) != 2 || vaultInfos[0].GetId() != "v1" || vaultInfos[0].GetIsMissing() || vaultInfos[1].GetId() != "v2" || !vaultInfos[1].GetIsMissing() {
		t.Errorf("the vaults were %v, want v1 present and v2 missing", vaultInfos)
	}
}

func TestNoteRefsCarryTheIdentifier(t *testing.T) {
	client, _, _ := newArrangeClient(t, nil, pluginport.PermissionVaultRead)

	res, err := client.GetNoteRefs(t.Context(), connect.NewRequest(&v1.GetNoteRefsRequest{Paths: []string{"Danube.md"}}))
	if err != nil {
		t.Fatal(err)
	}

	if notes := res.Msg.GetNotes(); len(notes) != 1 || notes[0].GetId() != "01JRIVERDanube" {
		t.Errorf("the notes were %v, want Danube.md with id 01JRIVERDanube", notes)
	}
}

func TestVaultListErrorsReturnTheirVaultsErrorCode(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want v1.VaultsErrorCode
	}{
		"name taken": {err: vaults.ErrNameTaken, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN},
		"last vault": {err: vaults.ErrLastVault, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT},
		"showing":    {err: vaults.ErrShowing, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING},
		"unknown":    {err: vaults.ErrUnknown, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN},
	} {
		t.Run(name, func(t *testing.T) {
			list := newRecordingList()
			list.err = fmt.Errorf("vault v2: %w", c.err)
			client, _, _ := newArrangeClient(t, list, pluginport.PermissionVaultManage)
			ctx := t.Context()

			added, addErr := client.AddVault(ctx, connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: "/vaults/Seas"}))
			renamed, renameErr := client.RenameVault(ctx, connect.NewRequest(&v1.HostVaultServiceRenameVaultRequest{Id: "v2", Name: "Ponds"}))
			removed, removeErr := client.RemoveVault(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveVaultRequest{Id: "v2"}))

			if addErr != nil || added.Msg.Failure == nil || added.Msg.GetFailure() != c.want {
				t.Errorf("AddVault returned %v (%v), want failure %s", added, addErr, c.want)
			}
			if renameErr != nil || renamed.Msg.Failure == nil || renamed.Msg.GetFailure() != c.want {
				t.Errorf("RenameVault returned %v (%v), want failure %s", renamed, renameErr, c.want)
			}
			if removeErr != nil || removed.Msg.Failure == nil || removed.Msg.GetFailure() != c.want {
				t.Errorf("RemoveVault returned %v (%v), want failure %s", removed, removeErr, c.want)
			}
		})
	}
}

func TestVaultListChangesReturnTheVaultAndTheFolder(t *testing.T) {
	client, _, _ := newArrangeClient(t, newRecordingList(), pluginport.PermissionVaultManage)
	ctx := t.Context()

	added, err := client.AddVault(ctx, connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: "/vaults/Seas", Name: "Seas"}))
	if err != nil {
		t.Fatal(err)
	}
	if v := added.Msg.GetVault(); added.Msg.Failure != nil || v.GetId() != "v3" || v.GetName() != "Seas" || v.GetPath() != "/vaults/Seas" {
		t.Errorf("AddVault returned %v", added.Msg)
	}
	renamed, err := client.RenameVault(ctx, connect.NewRequest(&v1.HostVaultServiceRenameVaultRequest{Id: "v2", Name: "Ponds"}))
	if err != nil {
		t.Fatal(err)
	}
	if v := renamed.Msg.GetVault(); renamed.Msg.Failure != nil || v.GetId() != "v2" || v.GetName() != "Ponds" {
		t.Errorf("RenameVault returned %v", renamed.Msg)
	}
	removed, err := client.RemoveVault(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveVaultRequest{Id: "v2"}))
	if err != nil {
		t.Fatal(err)
	}
	if removed.Msg.Failure != nil || removed.Msg.GetPath() != "/vaults/Lakes" {
		t.Errorf("RemoveVault returned %v, want the folder /vaults/Lakes", removed.Msg)
	}
}

func TestVaultListChangesAreUnavailableWithoutAList(t *testing.T) {
	client, _, _ := newArrangeClient(t, nil, pluginport.PermissionVaultManage)

	for call, err := range callVaultChanges(t, client) {
		var failure *connect.Error
		if !errors.As(err, &failure) || failure.Code() != connect.CodeUnavailable {
			t.Errorf("%s on a host with no list of vaults returned %v, want unavailable", call, err)
		}
	}
}

func TestOpenVaultReturnsBeforeTheWindowChangesVault(t *testing.T) {
	list := newRecordingList()
	list.release = make(chan struct{})
	client, _, _ := newArrangeClient(t, list, pluginport.PermissionVaultManage)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	res, err := client.OpenVault(ctx, connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))
	if err != nil || res.Msg.Failure != nil {
		close(list.release)
		t.Fatalf("OpenVault returned %v (%v) while the window was changing vault", res, err)
	}
	select {
	case id := <-list.opened:
		if id != "v2" {
			t.Errorf("the window was asked to open %q, want v2", id)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the window was not asked to open the vault")
	}
	close(list.release)

	select {
	case ctxErr := <-list.ended:
		if ctxErr != nil {
			t.Errorf("the change of vault ran on a context that ended with the call: %v", ctxErr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the change of vault did not end after it was released")
	}
}

// outsideVault is the vault of rivers that refuses a path leaving the vault with ErrOutside, as the
// vault writer does, and whose rename moves the file and then fails to update the index.
type outsideVault struct{ *arrangingVault }

func (o outsideVault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	if strings.HasPrefix(from, "../") {
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, vaultport.ErrOutside)
	}
	return o.arrangingVault.MoveNote(ctx, from, to)
}

func (o outsideVault) RemoveNote(ctx context.Context, target string) (string, []string, error) {
	if strings.HasPrefix(target, "../") {
		return "", nil, fmt.Errorf("remove %s: %w", target, vaultport.ErrOutside)
	}
	return o.arrangingVault.RemoveNote(ctx, target)
}

func (o outsideVault) RenameNote(ctx context.Context, from, title string) (string, bool, *domain.MoveResult, error) {
	to, isFilenamed, moved, err := o.arrangingVault.RenameNote(ctx, from, title)
	if err != nil {
		return to, isFilenamed, moved, err
	}
	return to, isFilenamed, moved, errors.New("the index could not be updated for the repaired notes")
}

func newOutsideClient(t *testing.T) (numenv1connect.HostVaultServiceClient, *arrangingVault) {
	t.Helper()
	vault := newArrangingVault()
	hosted := newCallers(nil)
	hosted.vault = outsideVault{vault}
	endpoint := &Endpoint{callers: listCallers{callers: hosted, granted: []string{pluginport.PermissionVaultWrite}}}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostVaultServiceHandler(newVaultEndpoint(endpoint)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	return client, vault
}

func TestAMoveReportsAPathOutsideTheVaultAndMovesTheOthers(t *testing.T) {
	client, vault := newOutsideClient(t)

	res, err := client.MoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: []string{"Danube.md", "../Baikal.md", "Rivers/Rhine.md"}, Folder: "Archive",
	}))
	if err != nil {
		t.Fatal(err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move returned error %v with %d entries, want 3 entries", res.Msg.GetError(), len(moves))
	}
	if moves[0].Failure != nil || moves[0].GetMoved().GetTo() != "Archive/Danube.md" {
		t.Errorf("the first entry is %v, want Danube.md moved to Archive/Danube.md", moves[0])
	}
	if moves[1].GetMoved().GetFrom() != "../Baikal.md" || moves[1].Failure == nil {
		t.Errorf("the second entry is %v, want ../Baikal.md with a failure", moves[1])
	}
	if moves[2].Failure != nil || moves[2].GetMoved().GetTo() != "Archive/Rhine.md" {
		t.Errorf("the third entry is %v, want Rivers/Rhine.md moved to Archive/Rhine.md", moves[2])
	}
	for _, target := range []string{"Archive/Danube.md", "Archive/Rhine.md"} {
		if _, found := vault.files[target]; !found {
			t.Errorf("%s is not in the vault", target)
		}
	}
}

func TestARemovalReportsAPathOutsideTheVaultAndRemovesTheOthers(t *testing.T) {
	client, vault := newOutsideClient(t)

	res, err := client.RemoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: []string{"Danube.md", "../Baikal.md", "Rivers/Rhine.md"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != 0 || len(removals) != 3 {
		t.Fatalf("the removal returned error %v with %d entries, want 3 entries", res.Msg.GetError(), len(removals))
	}
	if removals[0].Failure != nil || removals[0].GetTrashed() != ".trash/Danube.md" {
		t.Errorf("the first entry is %v, want Danube.md in .trash", removals[0])
	}
	if removals[1].GetPath() != "../Baikal.md" || removals[1].Failure == nil {
		t.Errorf("the second entry is %v, want ../Baikal.md with a failure", removals[1])
	}
	if removals[2].Failure != nil || removals[2].GetTrashed() != ".trash/Rivers/Rhine.md" {
		t.Errorf("the third entry is %v, want Rivers/Rhine.md in .trash", removals[2])
	}
	for _, target := range []string{"Danube.md", "Rivers/Rhine.md"} {
		if _, found := vault.files[target]; found {
			t.Errorf("%s is still in the vault", target)
		}
	}
}

func TestTheHostRejectsABatchOfMoreThanFiftyNotes(t *testing.T) {
	client, vault := newOutsideClient(t)
	paths := make([]string, 51)
	for i := range paths {
		paths[i] = fmt.Sprintf("Lake %d.md", i)
	}

	_, moveErr := client.MoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{Paths: paths}))
	_, removeErr := client.RemoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{Paths: paths}))

	for call, err := range map[string]error{"MoveNotes": moveErr, "RemoveNotes": removeErr} {
		var failure *connect.Error
		if !errors.As(err, &failure) || failure.Code() != connect.CodeInvalidArgument {
			t.Errorf("%s of 51 paths returned %v, want invalid argument", call, err)
		}
	}
	if len(vault.moved) != 0 || len(vault.removed) != 0 {
		t.Errorf("the vault was asked to move %d and remove %d notes of a batch of 51", len(vault.moved), len(vault.removed))
	}
}

func TestARenameWhoseFileMovedReturnsTheMoveWhenTheFollowUpFails(t *testing.T) {
	client, _ := newOutsideClient(t)

	res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
	if err != nil {
		t.Fatalf("the rename returned %v, want the move", err)
	}

	if res.Msg.Failure != nil || res.Msg.GetError() != 0 || res.Msg.GetPath() != "Donau.md" {
		t.Errorf("the rename returned %v, want Donau.md with no failure", res.Msg)
	}
	if moved := res.Msg.GetMoved(); moved.GetFrom() != "Danube.md" || moved.GetTo() != "Donau.md" {
		t.Errorf("the move came out as %v, want Danube.md to Donau.md", moved)
	}
}

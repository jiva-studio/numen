package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"net/http"
	"net/http/httptest"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// fakeNoteVault is the vault of rivers that renames, moves and removes its notes and runs its
// checks as the host does, and records the requests.
type fakeNoteVault struct {
	*memoryVault
	folders map[string]bool
	moved   []domain.Move
	removed []string
	checks  [][]domain.Check
}

func newFakeNoteVault() *fakeNoteVault {
	return &fakeNoteVault{
		memoryVault: newMemoryVault(map[string]string{
			"Danube.md":        "# Danube\n",
			"Deltas.md":        "---\nlinks:\n  - to: Danube\n    role: parent\n---\n# Deltas\n",
			"Rivers/Rhine.md":  "# Rhine\n",
			"Rivers/Deltas.md": "# Deltas\n",
		}),
		folders: map[string]bool{"Rivers": true},
	}
}

func (a *fakeNoteVault) GetVaults() ([]domain.Vault, error) {
	return []domain.Vault{
		{ID: "v1", Name: "Rivers", Path: "/vaults/Rivers"},
		{ID: "v2", Name: "Lakes", Path: "/vaults/Lakes"},
	}, nil
}

func (a *fakeNoteVault) GetNotes(_ context.Context, paths []string) (map[string]domain.NoteRef, error) {
	out := map[string]domain.NoteRef{}
	for _, one := range paths {
		if _, found := a.files[one]; found {
			out[one] = domain.NoteRef{Path: one, Title: domain.Basename(one), ID: "01JRIVER" + domain.Basename(one)}
		}
	}
	return out, nil
}

func (a *fakeNoteVault) RenameNote(_ context.Context, from, title string) (string, bool, *domain.MoveResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, found := a.files[from]; !found {
		return "", false, nil, fmt.Errorf("rename %s: %w", from, edit.ErrNotExist)
	}
	to := path.Join(path.Dir(from), title+".md")
	a.files[to] = a.files[from]
	delete(a.files, from)
	return to, true, &domain.MoveResult{From: from, To: to, IsMoved: true, Repaired: []string{"Deltas.md"}}, nil
}

func (a *fakeNoteVault) MoveNote(_ context.Context, from, to string) (domain.MoveResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.moved = append(a.moved, domain.Move{From: from, To: to})
	if _, found := a.files[from]; !found {
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, edit.ErrNotExist)
	}
	if _, found := a.files[to]; found {
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, vaultdomain.ErrAlreadyExists)
	}
	a.files[to] = a.files[from]
	delete(a.files, from)
	return domain.MoveResult{From: from, To: to, IsMoved: true}, nil
}

func (a *fakeNoteVault) RemoveNote(_ context.Context, target string) (string, []string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.removed = append(a.removed, target)
	if a.folders[target] {
		return "", nil, fmt.Errorf("remove %s: %w", target, edit.ErrNotANote)
	}
	if _, found := a.files[target]; !found {
		return "", nil, fmt.Errorf("remove %s: %w", target, edit.ErrNotExist)
	}
	delete(a.files, target)
	return ".trash/" + target, []string{"Deltas.md"}, nil
}

func (a *fakeNoteVault) GetProblems(_ context.Context, checks []domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
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

// recordingList is a list of vaults that records the changes it receives and returns err for each.
// Its OpenVault returns err at once. Otherwise it records the open, sends the vault on openCh, waits
// for release or the end of its context when release is set, sends the error of its context on endCh
// and returns.
type recordingList struct {
	mu      sync.Mutex
	err     error
	missing map[domain.VaultID]bool
	calls   []string
	openCh  chan domain.VaultID
	release chan struct{}
	endCh   chan error
}

func newRecordingList() *recordingList {
	return &recordingList{
		missing: map[domain.VaultID]bool{"v2": true},
		openCh:  make(chan domain.VaultID, 4),
		endCh:   make(chan error, 4),
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

func (l *recordingList) OpenVault(ctx context.Context, id domain.VaultID) error {
	l.mu.Lock()
	openErr := l.err
	l.mu.Unlock()
	if openErr != nil {
		return openErr
	}
	err := l.record("open " + string(id))
	l.openCh <- id
	if l.release != nil {
		select {
		case <-l.release:
		case <-ctx.Done():
		}
	}
	l.endCh <- ctx.Err()
	return err
}

func (l *recordingList) IsMissing(v domain.Vault) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.missing[v.ID]
}

func newArrangeClient(
	t *testing.T, list vaultapi.List, granted ...string,
) (numenv1connect.HostVaultServiceClient, *fakeNoteVault, *callers) {
	t.Helper()
	vault := newFakeNoteVault()
	hostCallers := newCallers(nil)
	hostCallers.vault = vaultOf(vault)
	endpoint := &Endpoint{callers: hostCallers.over(list, granted...)}
	mux := http.NewServeMux()
	mux.Handle(vaultapi.NewHandler(endpoint, vaultSource{callers: endpoint.callers}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	return client, vault, hostCallers
}

func isDenied(err error) bool {
	var failure *connect.Error
	return errors.As(err, &failure) && failure.Code() == connect.CodePermissionDenied
}

func callNoteChanges(t *testing.T, client numenv1connect.HostVaultServiceClient) map[string]error {
	t.Helper()
	ctx := t.Context()
	_, renameErr := client.RenameNote(ctx, connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
	_, moveErr := client.MoveNotes(ctx, connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{Paths: []string{"Rivers/Rhine.md"}}))
	_, removeErr := client.RemoveNotes(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{Paths: []string{"Deltas.md"}}))
	return map[string]error{"RenameNote": renameErr, "MoveNotes": moveErr, "RemoveNotes": removeErr}
}

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
	client, vault, _ := newArrangeClient(t, newRecordingList(), manifest.PermissionVaultRead)

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
	client, _, hostCallers := newArrangeClient(t, newRecordingList(), manifest.PermissionVaultWrite)

	for call, err := range callNoteChanges(t, client) {
		if err != nil {
			t.Errorf("%s with vault:write returned %v", call, err)
		}
	}
	hostCallers.mu.Lock()
	defer hostCallers.mu.Unlock()
	for _, one := range hostCallers.checked {
		if one != "md.numen.a "+manifest.PermissionVaultWrite {
			t.Errorf("a note change checked %q, want vault:write only", one)
		}
	}
}

func TestVaultListChangesAreDeniedWithOnlyVaultWrite(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultRead, manifest.PermissionVaultWrite)

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
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

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
	case <-list.openCh:
	case <-time.After(10 * time.Second):
		t.Fatal("the list received no call to open the vault")
	}
	want := []string{"add /vaults/Seas ", "rename v2 Ponds", "remove v2", "open v2"}
	if calls := list.getCalls(); !slices.Equal(calls, want) {
		t.Errorf("the list received %v, want %v", calls, want)
	}
}

func TestVaultChecksNeedVaultRead(t *testing.T) {
	denied, _, _ := newArrangeClient(t, nil)
	_, err := denied.GetVaultProblems(t.Context(), connect.NewRequest(&v1.GetVaultProblemsRequest{}))
	if !isDenied(err) {
		t.Errorf("the checks without vault:read returned %v, want permission denied", err)
	}

	client, vault, _ := newArrangeClient(t, nil, manifest.PermissionVaultRead)
	res, err := client.GetVaultProblems(t.Context(), connect.NewRequest(&v1.GetVaultProblemsRequest{Checks: []string{"ambiguous", "dangling"}}))
	if err != nil {
		t.Fatalf("the checks with vault:read returned %v", err)
	}

	if !slices.Equal(res.Msg.GetChecks(), []string{"ambiguous", "dangling"}) {
		t.Errorf("checks are %v, want ambiguous and dangling", res.Msg.GetChecks())
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
		t.Errorf("the vault received a request for the checks %v", vault.checks)
	}
}

func TestARenameReturnsTheNewPathAndTheMove(t *testing.T) {
	client, _, _ := newArrangeClient(t, nil, manifest.PermissionVaultWrite)

	res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != 0 || res.Msg.GetPath() != "Donau.md" || res.Msg.GetTitle() != "Donau" ||
		res.Msg.GetNameSource() != v1.NameSource_NAME_SOURCE_FILENAME {
		t.Errorf("the rename returned %v", res.Msg)
	}
	moved := res.Msg.GetMove()
	if moved.GetFrom() != "Danube.md" || moved.GetTo() != "Donau.md" || !slices.Equal(moved.GetRepaired(), []string{"Deltas.md"}) {
		t.Errorf("the move came out as %v", moved)
	}

	missing, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Nowhere.md", Title: "Elsewhere"}))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND {
		t.Errorf("a rename of a missing note returned %v, want error NOT_FOUND", missing.Msg)
	}
}

func TestAMoveReturnsOneEntryPerPathInOrder(t *testing.T) {
	client, vault, _ := newArrangeClient(t, nil, manifest.PermissionVaultWrite)

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
	if moves[0].GetError() != 0 || moves[0].GetMove().GetFrom() != "Danube.md" || moves[0].GetMove().GetTo() != "Rivers/Danube.md" {
		t.Errorf("the first entry is %v, want Danube.md moved to Rivers/Danube.md", moves[0])
	}
	if moves[1].GetMove().GetFrom() != "Nowhere.md" || moves[1].GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND {
		t.Errorf("the second entry is %v, want Nowhere.md with error NOT_FOUND", moves[1])
	}
	if moves[2].GetMove().GetFrom() != "Deltas.md" || moves[2].GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS {
		t.Errorf("the third entry is %v, want Deltas.md with error ALREADY_EXISTS", moves[2])
	}

	root, err := client.MoveNotes(t.Context(), connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{Paths: []string{"Rivers/Rhine.md"}}))
	if err != nil {
		t.Fatal(err)
	}
	if got := root.Msg.GetMoves(); len(got) != 1 || got[0].GetMove().GetTo() != "Rhine.md" {
		t.Errorf("a move to the empty folder returned %v, want Rhine.md at the vault root", got)
	}
	want := []domain.Move{
		{From: "Danube.md", To: "Rivers/Danube.md"},
		{From: "Nowhere.md", To: "Rivers/Nowhere.md"},
		{From: "Deltas.md", To: "Rivers/Deltas.md"},
		{From: "Rivers/Rhine.md", To: "Rhine.md"},
	}
	if !slices.Equal(vault.moved, want) {
		t.Errorf("the vault received a call to move %v, want %v", vault.moved, want)
	}
}

func TestARemovalReturnsTheTrashPathAndAFolderAsNotANote(t *testing.T) {
	client, vault, _ := newArrangeClient(t, nil, manifest.PermissionVaultWrite)

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
	if removals[0].GetError() != 0 || removals[0].GetPath() != "Danube.md" || removals[0].GetTrashPath() != ".trash/Danube.md" ||
		!slices.Equal(removals[0].GetBrokenLinks(), []string{"Deltas.md"}) {
		t.Errorf("the first entry is %v, want Danube.md in .trash with Deltas.md dangling", removals[0])
	}
	if removals[1].GetPath() != "Rivers" || removals[1].GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE {
		t.Errorf("the folder entry is %v, want error NOT_A_NOTE", removals[1])
	}
	if removals[2].GetPath() != "Nowhere.md" || removals[2].GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND {
		t.Errorf("the third entry is %v, want error NOT_FOUND", removals[2])
	}
	if !slices.Equal(vault.removed, []string{"Danube.md", "Rivers", "Nowhere.md"}) {
		t.Errorf("the vault received a call to remove %v", vault.removed)
	}
}

func TestTheListOfVaultsMarksAMissingFolder(t *testing.T) {
	client, _, _ := newArrangeClient(t, newRecordingList(), manifest.PermissionVaultRead)

	res, err := client.ListHostVaults(t.Context(), connect.NewRequest(&v1.ListHostVaultsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	vaultInfos := res.Msg.GetVaults()
	if len(vaultInfos) != 2 || vaultInfos[0].GetId() != "v1" || vaultInfos[0].GetIsMissing() || vaultInfos[1].GetId() != "v2" || !vaultInfos[1].GetIsMissing() {
		t.Errorf("the vaults were %v, want v1 present and v2 missing", vaultInfos)
	}
}

func TestNoteRefsHaveTheIdentifier(t *testing.T) {
	client, _, _ := newArrangeClient(t, nil, manifest.PermissionVaultRead)

	res, err := client.GetNoteRefs(t.Context(), connect.NewRequest(&v1.GetNoteRefsRequest{Paths: []string{"Danube.md"}}))
	if err != nil {
		t.Fatal(err)
	}

	if notes := res.Msg.GetNotes(); len(notes) != 1 || notes[0].GetId() != "01JRIVERDanube" {
		t.Errorf("the notes were %v, want Danube.md with id 01JRIVERDanube", notes)
	}
}

func TestRejectedVaultListChangesReturnTheErrorCodeAndTheCoreText(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want v1.VaultsErrorCode
	}{
		"name taken":      {err: vaultdomain.ErrNameTaken, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN},
		"last vault":      {err: vaultdomain.ErrLastVault, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT},
		"current vault":   {err: vaultdomain.ErrShowing, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT},
		"no such vault":   {err: registry.ErrUnknown, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND},
		"unsaved changes": {err: plugin.ErrUnsavedChanges, want: v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED},
	} {
		t.Run(name, func(t *testing.T) {
			list := newRecordingList()
			list.err = fmt.Errorf("vault v2: %w", c.err)
			client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)
			ctx := t.Context()

			added, addErr := client.AddVault(ctx, connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: "/vaults/Seas"}))
			renamed, renameErr := client.RenameVault(ctx, connect.NewRequest(&v1.HostVaultServiceRenameVaultRequest{Id: "v2", Name: "Ponds"}))
			removed, removeErr := client.RemoveVault(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveVaultRequest{Id: "v2"}))
			opened, openErr := client.OpenVault(ctx, connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))

			for call, answer := range map[string]struct {
				err    error
				code   v1.VaultsErrorCode
				detail string
			}{
				"AddVault":    {err: addErr, code: added.Msg.GetError(), detail: added.Msg.GetDetail()},
				"RenameVault": {err: renameErr, code: renamed.Msg.GetError(), detail: renamed.Msg.GetDetail()},
				"RemoveVault": {err: removeErr, code: removed.Msg.GetError(), detail: removed.Msg.GetDetail()},
				"OpenVault":   {err: openErr, code: opened.Msg.GetError(), detail: opened.Msg.GetDetail()},
			} {
				if answer.err != nil || answer.code != c.want {
					t.Errorf("%s returned %s (%v), want error %s", call, answer.code, answer.err, c.want)
				}
				if !strings.Contains(answer.detail, list.err.Error()) {
					t.Errorf("%s returned the detail %q, want the core's text %q", call, answer.detail, list.err.Error())
				}
			}
		})
	}
}

func TestOpenVaultOfAnUnknownIdReturnsNotFoundWithTheDetail(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

	res, err := client.OpenVault(t.Context(), connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v9"}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND {
		t.Errorf("OpenVault of v9 returned %v, want error NOT_FOUND", res.Msg)
	}
	if !strings.Contains(res.Msg.GetDetail(), "v9") {
		t.Errorf("OpenVault of v9 returned the detail %q, want the core's text naming v9", res.Msg.GetDetail())
	}
	if got := list.getCalls(); len(got) != 0 {
		t.Errorf("the list received a request for %v, want nothing", got)
	}
}

func TestRemoveVaultOfAnUnknownIdReturnsNotFoundWithTheDetail(t *testing.T) {
	client, _, _ := newArrangeClient(t, newRecordingList(), manifest.PermissionVaultManage)

	res, err := client.RemoveVault(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveVaultRequest{Id: "v9"}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND || !strings.Contains(res.Msg.GetDetail(), "v9") {
		t.Errorf("RemoveVault of v9 returned %v, want error NOT_FOUND with the core's text naming v9", res.Msg)
	}
}

func TestVaultListChangesThatSucceedReturnNoDetail(t *testing.T) {
	client, _, _ := newArrangeClient(t, newRecordingList(), manifest.PermissionVaultManage)
	ctx := t.Context()

	added, addErr := client.AddVault(ctx, connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: "/vaults/Seas"}))
	renamed, renameErr := client.RenameVault(ctx, connect.NewRequest(&v1.HostVaultServiceRenameVaultRequest{Id: "v2", Name: "Ponds"}))
	removed, removeErr := client.RemoveVault(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveVaultRequest{Id: "v2"}))

	if err := errors.Join(addErr, renameErr, removeErr); err != nil {
		t.Fatal(err)
	}
	for call, detail := range map[string]string{
		"AddVault": added.Msg.GetDetail(), "RenameVault": renamed.Msg.GetDetail(), "RemoveVault": removed.Msg.GetDetail(),
	} {
		if detail != "" {
			t.Errorf("%s succeeded with the detail %q", call, detail)
		}
	}
}

// failingRenameVault is the vault of rivers whose rename of a note fails with err before any file moves.
type failingRenameVault struct {
	*fakeNoteVault
	err error
}

func (r failingRenameVault) RenameNote(context.Context, string, string) (string, bool, *domain.MoveResult, error) {
	return "", false, nil, r.err
}

func TestRejectedRenameReturnsItsCodeInError(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want v1.HostVaultErrorCode
	}{
		"invalid name":   {err: domain.ErrInvalidName, want: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_NAME},
		"already exists": {err: vaultdomain.ErrAlreadyExists, want: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS},
		"not found":      {err: edit.ErrNotExist, want: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND},
		"conflict":       {err: edit.ErrStale, want: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT},
	} {
		t.Run(name, func(t *testing.T) {
			hostCallers := newCallers(nil)
			hostCallers.vault = vaultOf(failingRenameVault{fakeNoteVault: newFakeNoteVault(), err: fmt.Errorf("rename Danube.md: %w", c.err)})
			endpoint := &Endpoint{callers: hostCallers.over(nil, manifest.PermissionVaultWrite)}
			mux := http.NewServeMux()
			mux.Handle(vaultapi.NewHandler(endpoint, vaultSource{callers: endpoint.callers}))
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)
			client := numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))

			res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
			if err != nil {
				t.Fatal(err)
			}

			if res.Msg.GetError() != c.want {
				t.Errorf("the rejected rename returned %v, want error %s", res.Msg, c.want)
			}
			if res.Msg.GetMove() != nil {
				t.Errorf("the rejected rename returned the move %v", res.Msg.GetMove())
			}
		})
	}
}

func TestVaultListChangesReturnTheVaultAndTheFolder(t *testing.T) {
	client, _, _ := newArrangeClient(t, newRecordingList(), manifest.PermissionVaultManage)
	ctx := t.Context()

	added, err := client.AddVault(ctx, connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: "/vaults/Seas", Name: "Seas"}))
	if err != nil {
		t.Fatal(err)
	}
	if v := added.Msg.GetVault(); added.Msg.GetError() != 0 || v.GetId() != "v3" || v.GetName() != "Seas" || v.GetPath() != "/vaults/Seas" {
		t.Errorf("AddVault returned %v", added.Msg)
	}
	renamed, err := client.RenameVault(ctx, connect.NewRequest(&v1.HostVaultServiceRenameVaultRequest{Id: "v2", Name: "Ponds"}))
	if err != nil {
		t.Fatal(err)
	}
	if v := renamed.Msg.GetVault(); renamed.Msg.GetError() != 0 || v.GetId() != "v2" || v.GetName() != "Ponds" {
		t.Errorf("RenameVault returned %v", renamed.Msg)
	}
	removed, err := client.RemoveVault(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveVaultRequest{Id: "v2"}))
	if err != nil {
		t.Fatal(err)
	}
	if removed.Msg.GetError() != 0 || removed.Msg.GetPath() != "/vaults/Lakes" {
		t.Errorf("RemoveVault returned %v, want the folder /vaults/Lakes", removed.Msg)
	}
}

func TestVaultListChangesAreUnavailableWithoutAList(t *testing.T) {
	client, _, _ := newArrangeClient(t, nil, manifest.PermissionVaultManage)

	for call, err := range callVaultChanges(t, client) {
		var failure *connect.Error
		if !errors.As(err, &failure) || failure.Code() != connect.CodeUnavailable {
			t.Errorf("%s on a host with no list of vaults returned %v, want unavailable", call, err)
		}
	}
}

func TestOpenVaultReturnsAfterTheWindowShowsTheVault(t *testing.T) {
	list := newRecordingList()
	list.release = make(chan struct{})
	release := sync.OnceFunc(func() { close(list.release) })
	t.Cleanup(release)
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	type openAnswer struct {
		res *connect.Response[v1.HostVaultServiceOpenVaultResponse]
		err error
	}
	answerResult := make(chan openAnswer, 1)
	go func() {
		res, err := client.OpenVault(ctx, connect.NewRequest(&v1.HostVaultServiceOpenVaultRequest{Id: "v2"}))
		answerResult <- openAnswer{res: res, err: err}
	}()
	select {
	case id := <-list.openCh:
		if id != "v2" {
			t.Errorf("the window received a call to open %q, want v2", id)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the window received no call to open the vault")
	}
	select {
	case a := <-answerResult:
		t.Fatalf("OpenVault returned %v (%v) before the window showed the vault", a.res, a.err)
	case <-time.After(200 * time.Millisecond):
	}
	release()

	select {
	case a := <-answerResult:
		if a.err != nil || a.res.Msg.GetError() != 0 {
			t.Errorf("OpenVault after the window showed the vault returned %v (%v)", a.res, a.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("OpenVault did not return after the window showed the vault")
	}
	select {
	case ctxErr := <-list.endCh:
		if ctxErr != nil {
			t.Errorf("the change of vault ended with %v, want it to end on its own", ctxErr)
		}
	default:
		t.Error("OpenVault returned while the change of vault was still running")
	}
}

// escapingPathVault is the vault of rivers that rejects a path leaving the vault with ErrOutside, as the
// vault writer does, and whose rename moves the file and then fails to update the index.
type escapingPathVault struct{ *fakeNoteVault }

func (o escapingPathVault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	if strings.HasPrefix(from, "../") {
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, vaultdomain.ErrOutside)
	}
	return o.fakeNoteVault.MoveNote(ctx, from, to)
}

func (o escapingPathVault) RemoveNote(ctx context.Context, target string) (string, []string, error) {
	if strings.HasPrefix(target, "../") {
		return "", nil, fmt.Errorf("remove %s: %w", target, vaultdomain.ErrOutside)
	}
	return o.fakeNoteVault.RemoveNote(ctx, target)
}

func (o escapingPathVault) RenameNote(ctx context.Context, from, title string) (string, bool, *domain.MoveResult, error) {
	to, isFilenamed, moved, err := o.fakeNoteVault.RenameNote(ctx, from, title)
	if err != nil {
		return to, isFilenamed, moved, err
	}
	return to, isFilenamed, moved, errors.New("the index could not be updated for the repaired notes")
}

func newEscapingPathClient(t *testing.T) (numenv1connect.HostVaultServiceClient, *fakeNoteVault) {
	t.Helper()
	vault := newFakeNoteVault()
	hostCallers := newCallers(nil)
	hostCallers.vault = vaultOf(escapingPathVault{vault})
	endpoint := &Endpoint{callers: hostCallers.over(nil, manifest.PermissionVaultWrite)}
	mux := http.NewServeMux()
	mux.Handle(vaultapi.NewHandler(endpoint, vaultSource{callers: endpoint.callers}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	return client, vault
}

func TestAMoveReportsAPathOutsideTheVaultAndMovesTheOthers(t *testing.T) {
	client, vault := newEscapingPathClient(t)

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
	if moves[0].GetError() != 0 || moves[0].GetMove().GetTo() != "Archive/Danube.md" {
		t.Errorf("the first entry is %v, want Danube.md moved to Archive/Danube.md", moves[0])
	}
	if moves[1].GetMove().GetFrom() != "../Baikal.md" || moves[1].GetError() == 0 {
		t.Errorf("the second entry is %v, want ../Baikal.md with an error", moves[1])
	}
	if moves[2].GetError() != 0 || moves[2].GetMove().GetTo() != "Archive/Rhine.md" {
		t.Errorf("the third entry is %v, want Rivers/Rhine.md moved to Archive/Rhine.md", moves[2])
	}
	for _, target := range []string{"Archive/Danube.md", "Archive/Rhine.md"} {
		if _, found := vault.files[target]; !found {
			t.Errorf("%s is not in the vault", target)
		}
	}
}

func TestARemovalReportsAPathOutsideTheVaultAndRemovesTheOthers(t *testing.T) {
	client, vault := newEscapingPathClient(t)

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
	if removals[0].GetError() != 0 || removals[0].GetTrashPath() != ".trash/Danube.md" {
		t.Errorf("the first entry is %v, want Danube.md in .trash", removals[0])
	}
	if removals[1].GetPath() != "../Baikal.md" || removals[1].GetError() == 0 {
		t.Errorf("the second entry is %v, want ../Baikal.md with an error", removals[1])
	}
	if removals[2].GetError() != 0 || removals[2].GetTrashPath() != ".trash/Rivers/Rhine.md" {
		t.Errorf("the third entry is %v, want Rivers/Rhine.md in .trash", removals[2])
	}
	for _, target := range []string{"Danube.md", "Rivers/Rhine.md"} {
		if _, found := vault.files[target]; found {
			t.Errorf("%s is still in the vault", target)
		}
	}
}

func TestTheHostRejectsABatchOfMoreThanFiftyNotes(t *testing.T) {
	client, vault := newEscapingPathClient(t)
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
		t.Errorf("the vault received a call to move %d and remove %d notes of a batch of 51", len(vault.moved), len(vault.removed))
	}
}

func TestARenameWhoseFileMovedReturnsTheMoveWhenTheFollowUpFails(t *testing.T) {
	client, _ := newEscapingPathClient(t)

	res, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.HostVaultServiceRenameNoteRequest{Path: "Danube.md", Title: "Donau"}))
	if err != nil {
		t.Fatalf("the rename returned %v, want the move", err)
	}

	if res.Msg.GetError() != 0 || res.Msg.GetPath() != "Donau.md" {
		t.Errorf("the rename returned %v, want Donau.md with no error", res.Msg)
	}
	if moved := res.Msg.GetMove(); moved.GetFrom() != "Danube.md" || moved.GetTo() != "Donau.md" {
		t.Errorf("the move came out as %v, want Danube.md to Donau.md", moved)
	}
}

func TestAddVaultOfARejectedFolderReturnsTheCoreMessage(t *testing.T) {
	for name, c := range map[string]struct {
		err     error
		want    v1.VaultsErrorCode
		message string
	}{
		"unreadable folder": {
			err:     domain.NewDetailError("/vaults/Seas cannot be read as a vault: the folder is not there", vaultdomain.ErrUnreadable),
			want:    v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE,
			message: "/vaults/Seas cannot be read as a vault: the folder is not there",
		},
		"folder inside a vault": {
			err:     domain.Vaults{{ID: "v1", Name: "Rivers", Path: "/vaults"}}.CheckOverlap("/vaults/Seas"),
			want:    v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS,
			message: "/vaults/Seas is inside the vault Rivers; a vault cannot be inside another vault",
		},
	} {
		t.Run(name, func(t *testing.T) {
			list := newRecordingList()
			list.err = c.err
			client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

			res, err := client.AddVault(t.Context(), connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: "/vaults/Seas"}))
			if err != nil {
				t.Fatal(err)
			}

			if res.Msg.GetError() != c.want {
				t.Errorf("AddVault returned %v, want error %s", res.Msg, c.want)
			}
			if res.Msg.GetDetail() != c.message {
				t.Errorf("the detail is %q, want %q", res.Msg.GetDetail(), c.message)
			}
		})
	}
}

func TestAddVaultOverlapReturnsOverlapsWithCoreTextAsDetail(t *testing.T) {
	vaultList := domain.Vaults{{ID: "v1", Name: "Rivers", Path: "/vaults/Rivers"}}
	for root, want := range map[string]string{
		"/vaults/Rivers/Deltas": "/vaults/Rivers/Deltas is inside the vault Rivers; a vault cannot be inside another vault",
		"/vaults":               "/vaults contains the vault Rivers; a vault cannot contain another vault",
	} {
		t.Run(root, func(t *testing.T) {
			list := newRecordingList()
			list.err = vaultList.CheckOverlap(root)
			client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

			res, err := client.AddVault(t.Context(), connect.NewRequest(&v1.HostVaultServiceAddVaultRequest{Root: root}))
			if err != nil {
				t.Fatal(err)
			}

			if res.Msg.GetError() != v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS {
				t.Errorf("AddVault of %s returned %v, want error OVERLAPS", root, res.Msg)
			}
			if detail := res.Msg.GetDetail(); detail != want {
				t.Errorf("AddVault of %s returned the detail %q, want %q", root, detail, want)
			}
		})
	}
}

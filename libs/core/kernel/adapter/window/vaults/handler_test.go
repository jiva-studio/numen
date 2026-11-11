package vaults

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vaultlist"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

// vaultRows is the index as a vault is written to and taken out of it.
type vaultRows struct {
	mu     sync.Mutex
	saved  []domain.Vault
	forgot []domain.VaultID
}

func (r *vaultRows) Save(_ context.Context, v domain.Vault) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, v)
	return nil
}

func (r *vaultRows) Register(context.Context, domain.VaultID) error { return nil }

func (r *vaultRows) Unregister(_ context.Context, vaultID domain.VaultID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.forgot = append(r.forgot, vaultID)
	return nil
}

func (r *vaultRows) getForgotten() []domain.VaultID {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.VaultID(nil), r.forgot...)
}

// bin collects what a test deletes, and models a machine with no trash
// location when refuse is set.
type bin struct {
	mu    sync.Mutex
	fails error
	moved []string
}

func (b *bin) Trash(path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fails != nil {
		return b.fails
	}
	b.moved = append(b.moved, path)
	return nil
}

func (b *bin) refuse(why error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.fails = why
}

func (b *bin) getTrashed() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.moved...)
}

// folders answers the folder picker in a test. The real dialog is
// machine-specific and needs a window.
type folders struct {
	mu       sync.Mutex
	pick     string
	isChosen bool
	fails    error
	title    string
	from     string
}

func (f *folders) ChooseFolder(_ context.Context, title, startingAt string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.title, f.from = title, startingAt
	return f.pick, f.isChosen, f.fails
}

func (f *folders) getRequested() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.title, f.from
}

// onTheList is two vaults this installation holds, a window showing the first,
// and a client asking about them the way the window does.
type onTheList struct {
	client   numenv1connect.VaultsServiceClient
	registry *appstate.VaultRegistry
	rows     *vaultRows
	bin      *bin
	dialog   *folders
	first    domain.Vault
	second   domain.Vault

	mu       sync.Mutex
	current  domain.Vault
	expected []domain.Vault
	failure  error
}

func (f *onTheList) GetCurrentVault() domain.Vault {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current
}

func (f *onTheList) ShowVault(_ context.Context, v domain.Vault) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.expected = append(f.expected, v)
	return f.failure
}

func (f *onTheList) standOnNothing() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = domain.Vault{}
}

func onAList(t *testing.T) *onTheList {
	t.Helper()

	registry := appstate.OpenAt(filepath.Join(t.TempDir(), "state", "vaults.json"))
	identity := vaultreader.Identity{}
	adding := vaults.NewAdd(identity, registry, registry, registry, testsupport.NewClock())

	rows := &vaultRows{}
	forget := vaults.NewUnregister(registry, registry, rows)
	f := &onTheList{
		registry: registry,
		rows:     rows,
		bin:      &bin{},
		dialog:   &folders{},
		first:    addVault(t, adding, "one"),
		second:   addVault(t, adding, "two"),
	}
	f.current = f.first
	rename := vaults.NewRename(registry, registry, rows)
	erase := vaults.NewErase(identity, f.bin, forget)
	list := vaultregistry.NewList(registry)
	find := vaultregistry.NewFind(registry)
	chooseFolder := picker.NewChooseFolder(f.dialog)
	folderCheck := vaultregistry.NewFolderCheck(vaultreader.Readers{})
	remove := vaults.NewRemove(find.Execute, forget, erase)
	handler := NewHandler(
		f,
		vaultlist.NewListVaults(list, folderCheck), vaultlist.NewAddVault(adding, folderCheck),
		vaultlist.NewRenameVault(find, rename, folderCheck), vaultlist.NewOpenVault(find, f), remove,
	)
	handler.SetFolderChooser(chooseFolder)
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewVaultsServiceHandler(handler))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	f.client = numenv1connect.NewVaultsServiceClient(server.Client(), server.URL)
	return f
}

func (f *onTheList) opening(why error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failure = why
}

func (f *onTheList) getShown() []domain.Vault {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.Vault(nil), f.expected...)
}

func addVault(t *testing.T, add vaults.Add, name string) domain.Vault {
	t.Helper()

	v, err := add.Execute(folderNamed(t, name), name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func folderNamed(t *testing.T, name string) string {
	t.Helper()

	pos := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(pos, 0o755); err != nil {
		t.Fatal(err)
	}
	return pos
}

func (f *onTheList) getList(t *testing.T) *v1.ListVaultsResponse {
	t.Helper()

	out, err := f.client.ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg
}

func onList(t *testing.T, list *v1.ListVaultsResponse, id string) *v1.Vault {
	t.Helper()

	for _, one := range list.GetVaults() {
		if one.GetId() == id {
			return one
		}
	}
	t.Fatalf("the list holds %v, and %s is not on it", list.GetVaults(), id)
	return nil
}

func TestTheListMarksAFolderThatIsMissing(t *testing.T) {
	f := onAList(t)
	if err := os.RemoveAll(f.second.Path); err != nil {
		t.Fatal(err)
	}

	list := f.getList(t)
	if here := onList(t, list, string(f.first.ID)); here.GetIsMissing() {
		t.Errorf("%s is marked missing, and its folder is at %s",
			here.GetName(), here.GetPath())
	}
	removed := onList(t, list, string(f.second.ID))
	if !removed.GetIsMissing() {
		t.Errorf("%s is not marked missing, and there is nothing at %s",
			removed.GetName(), removed.GetPath())
	}
	if removed.GetName() != f.second.Name || removed.GetPath() != f.second.Path {
		t.Errorf("the vault that is gone is answered as %v", removed)
	}
}

func TestAFolderInsideAVaultOnTheListIsRefused(t *testing.T) {
	f := onAList(t)
	inner := filepath.Join(f.first.Path, "projects", "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := f.client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: inner}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS {
		t.Errorf("a folder inside a vault was answered %v", got)
	}
	if out.Msg.GetVault() != nil {
		t.Errorf("it joined the list as %v", out.Msg.GetVault())
	}
}

// TestACopyOfAVaultOnTheListIsRefused. Two folders carrying one identity cannot
// both be indexed, and the check distinguishes them from a folder that moved.
func TestACopyOfAVaultOnTheListIsRefused(t *testing.T) {
	f := onAList(t)
	copied := folderNamed(t, "copy")
	service := filepath.Join(copied, vaultconfig.DefaultServiceDir)
	if err := os.MkdirAll(service, 0o755); err != nil {
		t.Fatal(err)
	}
	carried, err := os.ReadFile(filepath.Join(f.second.Path, vaultconfig.DefaultServiceDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(service, "config.json"), carried, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := f.client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: copied}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY {
		t.Errorf("a copy of a vault was answered %v", got)
	}
}

// TestAFolderJoinsTheListUnderANameAnotherVaultHas, numbered. Add does not
// refuse a taken name.
func TestAFolderJoinsTheListBelowANameAnotherVaultHas(t *testing.T) {
	f := onAList(t)

	out, err := f.client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{
		Path: folderNamed(t, "three"),
		Name: "one",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Error != nil {
		t.Fatalf("the folder failed: %v", out.Msg.GetError())
	}
	if got := out.Msg.GetVault().GetName(); got != "one 2" {
		t.Errorf("the vault that joined the list is called %q", got)
	}
	if len(f.getList(t).GetVaults()) != 3 {
		t.Errorf("the list holds %v", f.getList(t).GetVaults())
	}
}

func TestANameAnotherVaultHasIsNotGivenToASecond(t *testing.T) {
	f := onAList(t)

	out, err := f.client.RenameVault(t.Context(), connect.NewRequest(&v1.RenameVaultRequest{
		Id:   string(f.second.ID),
		Name: "one",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN {
		t.Errorf("a name another vault has was answered %v", got)
	}
	if onList(t, f.getList(t), string(f.second.ID)).GetName() != f.second.Name {
		t.Error("the vault was renamed all the same")
	}
}

func TestAVaultIsCalledWhatThePersonCallsIt(t *testing.T) {
	f := onAList(t)

	out, err := f.client.RenameVault(t.Context(), connect.NewRequest(&v1.RenameVaultRequest{
		Id:   string(f.second.ID),
		Name: "journal",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Error != nil {
		t.Fatalf("the vault was not renamed: %v", out.Msg.GetError())
	}
	if got := out.Msg.GetVault().GetName(); got != "journal" {
		t.Errorf("the vault is called %q", got)
	}
	if got := onList(t, f.getList(t), string(f.second.ID)).GetName(); got != "journal" {
		t.Errorf("the list calls it %q", got)
	}
}

// TestTheVaultTheWindowIsShowingStaysOnTheList. The use cases do not know which
// vault is current; the handler does.
func TestTheVaultTheWindowIsShowingStaysOnTheList(t *testing.T) {
	f := onAList(t)

	forgot, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.first.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := forgot.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT {
		t.Errorf("removing the vault being current was answered %v", got)
	}

	erased, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.first.ID), Trash: true}))
	if err != nil {
		t.Fatal(err)
	}
	if got := erased.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT {
		t.Errorf("trashing the vault being current was answered %v", got)
	}

	onList(t, f.getList(t), string(f.first.ID))
	if got := f.rows.getForgotten(); len(got) != 0 {
		t.Errorf("forgot %v", got)
	}
	if got := f.bin.getTrashed(); len(got) != 0 {
		t.Errorf("trashed %v", got)
	}
	if _, err := os.Stat(f.first.Path); err != nil {
		t.Errorf("the folder is gone: %v", err)
	}
}

func TestTheLastVaultAnInstallationHasStaysOnTheList(t *testing.T) {
	f := onAList(t)
	// A window standing on nothing is not what keeps this vault: the list is.
	f.standOnNothing()

	removed, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if removed.Msg.Error != nil {
		t.Fatalf("the second vault was not removed: %v", removed.Msg.GetError())
	}

	only, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.first.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := only.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT {
		t.Errorf("the only vault this installation has was answered %v", got)
	}
	onList(t, f.getList(t), string(f.first.ID))
}

// TestAnIdentityOnNoListIsUnknown, regardless of how it is queried.
func TestAnIdentityOnNoListIsUnknown(t *testing.T) {
	f := onAList(t)
	const nobody = "01JZZZZZZZZZZZZZZZZZZZZZZZ"

	rename, err := f.client.RenameVault(t.Context(),
		connect.NewRequest(&v1.RenameVaultRequest{Id: nobody, Name: "journal"}))
	if err != nil {
		t.Fatal(err)
	}
	forget, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: nobody}))
	if err != nil {
		t.Fatal(err)
	}
	erase, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: nobody, Trash: true}))
	if err != nil {
		t.Fatal(err)
	}
	open, err := f.client.OpenVault(t.Context(),
		connect.NewRequest(&v1.OpenVaultRequest{Id: nobody}))
	if err != nil {
		t.Fatal(err)
	}

	for what, got := range map[string]v1.VaultsErrorCode{
		"rename": rename.Msg.GetError(),
		"remove": forget.Msg.GetError(),
		"trash":  erase.Msg.GetError(),
		"open":   open.Msg.GetError(),
	} {
		if got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND {
			t.Errorf("%s of an identity on no list was answered %v", what, got)
		}
	}
	if visibleVaults := f.getShown(); len(visibleVaults) != 0 {
		t.Errorf("the window was asked to show %v", visibleVaults)
	}
}

func TestAVaultGoesToTheTrashAndOffTheList(t *testing.T) {
	f := onAList(t)

	out, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.second.ID), Trash: true}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Error != nil {
		t.Fatalf("the vault was not removed: %v", out.Msg.GetError())
	}
	if got := f.bin.getTrashed(); len(got) != 1 || got[0] != f.second.Path {
		t.Errorf("trashed %v, want %s", got, f.second.Path)
	}
	if got := f.rows.getForgotten(); len(got) != 1 || got[0] != f.second.ID {
		t.Errorf("forgot %v, want %s", got, f.second.ID)
	}
	if _, found, err := f.registry.FindVault(string(f.second.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
}

func TestAMachineWithNowhereToInsertWhatIsDeletedErasesNothing(t *testing.T) {
	f := onAList(t)
	f.bin.refuse(machine.ErrNoTrash)

	out, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.second.ID), Trash: true}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH {
		t.Errorf("a machine with no trash was answered %v", got)
	}
	onList(t, f.getList(t), string(f.second.ID))
	if got := f.rows.getForgotten(); len(got) != 0 {
		t.Errorf("forgot %v", got)
	}
	if _, err := os.Stat(f.second.Path); err != nil {
		t.Errorf("the folder is gone: %v", err)
	}
}

func TestAnotherVaultIsInsertInTheWindow(t *testing.T) {
	f := onAList(t)

	out, err := f.client.OpenVault(t.Context(),
		connect.NewRequest(&v1.OpenVaultRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Error != nil {
		t.Fatalf("the vault was not current: %v", out.Msg.GetError())
	}
	if visibleVaults := f.getShown(); len(visibleVaults) != 1 || visibleVaults[0].ID != f.second.ID {
		t.Errorf("the window was asked to show %v", visibleVaults)
	}
}

// TestAPageHoldingAnUnansweredQuestionKeepsTheVaultItWasTypedIn.
func TestOpenVault_ErrorsWhilePageHasPendingQuestion(t *testing.T) {
	f := onAList(t)
	f.opening(window.ErrAnswerPending)

	out, err := f.client.OpenVault(t.Context(),
		connect.NewRequest(&v1.OpenVaultRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED {
		t.Errorf("a page holding a question was answered %v", got)
	}
}

// TestAWindowThatIsGoingIsNotACodeAboutTheVault. The window stopped the swap,
// so it is reported as a status and not as a code about the requested vault.
func TestAWindowThatIsGoingIsNotACodeAboutTheVault(t *testing.T) {
	for _, one := range []struct {
		name string
		why  error
	}{
		{"the window is closing", window.ErrClosing},
		{"the window is settling", window.ErrFlushing},
	} {
		t.Run(one.name, func(t *testing.T) {
			f := onAList(t)
			f.opening(one.why)

			_, err := f.client.OpenVault(t.Context(),
				connect.NewRequest(&v1.OpenVaultRequest{Id: string(f.second.ID)}))
			if got := connect.CodeOf(err); got != connect.CodeUnavailable {
				t.Errorf("a window that is going answered %v, want %v", got, connect.CodeUnavailable)
			}
		})
	}
}

func TestAPersonWhoClosedTheFolderDialogChoseNothing(t *testing.T) {
	f := onAList(t)

	out, err := f.client.ChooseFolder(t.Context(), connect.NewRequest(&v1.ChooseFolderRequest{
		Title:      "Where are your notes?",
		StartingAt: f.first.Path,
	}))
	if err != nil {
		t.Fatalf("a dialog that was closed answered %v", err)
	}
	if out.Msg.GetIsChosen() || out.Msg.GetPath() != "" {
		t.Errorf("the answer is %v", out.Msg)
	}
	if title, from := f.dialog.getRequested(); title != "Where are your notes?" || from != f.first.Path {
		t.Errorf("the dialog was titled %q and opened at %q", title, from)
	}
}

func TestChooseFolder_ReturnsChosenPath(t *testing.T) {
	f := onAList(t)
	f.dialog.pick, f.dialog.isChosen = f.second.Path, true

	out, err := f.client.ChooseFolder(t.Context(), connect.NewRequest(&v1.ChooseFolderRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Msg.GetIsChosen() || out.Msg.GetPath() != f.second.Path {
		t.Errorf("the answer is %v", out.Msg)
	}
}

// TestASecondFolderDialogIsRefusedWhileOneIsUp. Only one dialog is answered at
// a time.
func TestASecondFolderDialogIsRefusedWhileOneIsUp(t *testing.T) {
	f := onAList(t)
	f.dialog.fails = machine.ErrChoosing

	_, err := f.client.ChooseFolder(t.Context(), connect.NewRequest(&v1.ChooseFolderRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Errorf("a second dialog answered %v, want %v", got, connect.CodeUnavailable)
	}
}

package editor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
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

func (r *vaultRows) Forget(_ context.Context, vaultID domain.VaultID) error {
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

// bin is where a test puts what a person deleted, and a machine with nowhere to
// put it where refuse is set.
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

// folders is the person choosing a folder, as a test answers for them. The real
// dialog is this machine's own and needs a window.
type folders struct {
	mu    sync.Mutex
	pick  string
	chose bool
	fails error
	title string
	from  string
}

func (f *folders) Choose(_ context.Context, title, startingAt string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.title, f.from = title, startingAt
	return f.pick, f.chose, f.fails
}

// getAsked is what the dialog was told to say and where to open.
func (f *folders) getAsked() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.title, f.from
}

// onTheList is two vaults this installation holds, a window showing the first,
// and a client asking about them the way the window does.
type onTheList struct {
	client   numenv1connect.VaultsServiceClient
	api      *API
	registry port.VaultRegistry
	rows     *vaultRows
	bin      *bin
	dialog   *folders
	first    domain.Vault
	second   domain.Vault

	mu      sync.Mutex
	shown   []domain.Vault
	refuses error
}

// onAList puts two vaults on a list of its own and serves the questions about
// them.
func onAList(t *testing.T) *onTheList {
	t.Helper()

	registry := appstate.OpenAt(filepath.Join(t.TempDir(), "state", "vaults.json"))
	identity := filesystem.VaultIdentity{}
	adding := vaults.Add{Identity: identity, Registry: registry, Now: time.Now}

	rows := &vaultRows{}
	forget := vaults.Forget{Registry: registry, Index: rows}
	f := &onTheList{
		registry: registry,
		rows:     rows,
		bin:      &bin{},
		dialog:   &folders{},
		first:    addVault(t, adding, "one"),
		second:   addVault(t, adding, "two"),
	}
	f.api = &API{
		Readers: filesystem.VaultReaders{},
		Vaults: Vaults{
			Registry:     registry,
			FolderDialog: f.dialog,
			Add:          &adding,
			Rename:       &vaults.Rename{Registry: registry, Index: rows},
			Forget:       &forget,
			Erase:        &vaults.Erase{Identity: identity, Trash: f.bin, Forget: forget},
		},
	}
	f.api.Opens = func(_ context.Context, v domain.Vault) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.shown = append(f.shown, v)
		return f.refuses
	}
	f.api.show(f.first)

	server := httptest.NewUnstartedServer(f.api.NewHandler(http.NotFoundHandler()))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	f.client = numenv1connect.NewVaultsServiceClient(server.Client(), server.URL)
	return f
}

// opening is what the window answers when it is asked to show a vault.
func (f *onTheList) opening(why error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refuses = why
}

// getShown is every vault the window was asked to show.
func (f *onTheList) getShown() []domain.Vault {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.Vault(nil), f.shown...)
}

// addVault makes a folder under a parent of its own and puts it on the list.
func addVault(t *testing.T, add vaults.Add, name string) domain.Vault {
	t.Helper()

	v, err := add.Execute(folderNamed(t, name), name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// folderNamed makes a directory named name, under a parent of its own.
func folderNamed(t *testing.T, name string) string {
	t.Helper()

	at := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(at, 0o755); err != nil {
		t.Fatal(err)
	}
	return at
}

// getList is the list as the client is answered with it.
func (f *onTheList) getList(t *testing.T) *v1.ListVaultsResponse {
	t.Helper()

	out, err := f.client.ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg
}

// onList is the vault of that identity on the list the client was answered with.
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

// TestTheListMarksAFolderThatIsGone.
func TestTheListMarksAFolderThatIsGone(t *testing.T) {
	f := onAList(t)
	if err := os.RemoveAll(f.second.Path); err != nil {
		t.Fatal(err)
	}

	list := f.getList(t)
	if here := onList(t, list, string(f.first.ID)); here.GetMissing() {
		t.Errorf("%s is marked missing, and its folder is at %s",
			here.GetName(), here.GetPath())
	}
	gone := onList(t, list, string(f.second.ID))
	if !gone.GetMissing() {
		t.Errorf("%s is not marked missing, and there is nothing at %s",
			gone.GetName(), gone.GetPath())
	}
	if gone.GetName() != f.second.Name || gone.GetPath() != f.second.Path {
		t.Errorf("the vault that is gone is answered as %v", gone)
	}
}

// TestAFolderInsideAVaultOnTheListIsRefused, and one holding a vault with it.
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
// both be indexed, and this is how they are told from a folder that moved.
func TestACopyOfAVaultOnTheListIsRefused(t *testing.T) {
	f := onAList(t)
	copied := folderNamed(t, "copy")
	service := filepath.Join(copied, filesystem.DefaultServiceDir)
	if err := os.MkdirAll(service, 0o755); err != nil {
		t.Fatal(err)
	}
	carried, err := os.ReadFile(filepath.Join(f.second.Path, filesystem.DefaultServiceDir, "config.json"))
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
func TestAFolderJoinsTheListUnderANameAnotherVaultHas(t *testing.T) {
	f := onAList(t)

	out, err := f.client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{
		Path: folderNamed(t, "three"),
		Name: "one",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Error != nil {
		t.Fatalf("the folder was refused: %v", out.Msg.GetError())
	}
	if got := out.Msg.GetVault().GetName(); got != "one 2" {
		t.Errorf("the vault that joined the list is called %q", got)
	}
	if len(f.getList(t).GetVaults()) != 3 {
		t.Errorf("the list holds %v", f.getList(t).GetVaults())
	}
}

// TestANameAnotherVaultHasIsNotGivenToASecond.
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

// TestAVaultIsCalledWhatThePersonCallsIt.
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

// TestTheVaultTheWindowIsShowingStaysOnTheList. The use cases are not told
// which vault is in front of the person; the handler is.
func TestTheVaultTheWindowIsShowingStaysOnTheList(t *testing.T) {
	f := onAList(t)

	forgot, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.first.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := forgot.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING {
		t.Errorf("removing the vault being shown was answered %v", got)
	}

	erased, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.first.ID), Trash: true}))
	if err != nil {
		t.Fatal(err)
	}
	if got := erased.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING {
		t.Errorf("trashing the vault being shown was answered %v", got)
	}

	onList(t, f.getList(t), string(f.first.ID))
	if got := f.rows.getForgotten(); len(got) != 0 {
		t.Errorf("the index was told to forget %v", got)
	}
	if got := f.bin.getTrashed(); len(got) != 0 {
		t.Errorf("trashed %v", got)
	}
	if _, err := os.Stat(f.first.Path); err != nil {
		t.Errorf("the folder is gone: %v", err)
	}
}

// TestTheLastVaultAnInstallationHasStaysOnTheList.
func TestTheLastVaultAnInstallationHasStaysOnTheList(t *testing.T) {
	f := onAList(t)
	// A window standing on nothing is not what keeps this vault: the list is.
	f.api.show(domain.Vault{})

	gone, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if gone.Msg.Error != nil {
		t.Fatalf("the second vault was not removed: %v", gone.Msg.GetError())
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

// TestAnIdentityOnNoListIsUnknown, whichever way it is asked about.
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
		if got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN {
			t.Errorf("%s of an identity on no list was answered %v", what, got)
		}
	}
	if shown := f.getShown(); len(shown) != 0 {
		t.Errorf("the window was asked to show %v", shown)
	}
}

// TestAVaultGoesToTheTrashAndOffTheList.
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
		t.Errorf("the index was told to forget %v, want %s", got, f.second.ID)
	}
	if _, found, err := f.registry.Find(string(f.second.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
}

// TestAMachineWithNowhereToPutWhatIsDeletedErasesNothing.
func TestAMachineWithNowhereToPutWhatIsDeletedErasesNothing(t *testing.T) {
	f := onAList(t)
	f.bin.refuse(port.ErrNoTrash)

	out, err := f.client.RemoveVault(t.Context(),
		connect.NewRequest(&v1.RemoveVaultRequest{Id: string(f.second.ID), Trash: true}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH {
		t.Errorf("a machine with nowhere to put it was answered %v", got)
	}
	onList(t, f.getList(t), string(f.second.ID))
	if got := f.rows.getForgotten(); len(got) != 0 {
		t.Errorf("the index was told to forget %v", got)
	}
	if _, err := os.Stat(f.second.Path); err != nil {
		t.Errorf("the folder is gone: %v", err)
	}
}

// TestAnotherVaultIsPutInTheWindow.
func TestAnotherVaultIsPutInTheWindow(t *testing.T) {
	f := onAList(t)

	out, err := f.client.OpenVault(t.Context(),
		connect.NewRequest(&v1.OpenVaultRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Error != nil {
		t.Fatalf("the vault was not shown: %v", out.Msg.GetError())
	}
	if shown := f.getShown(); len(shown) != 1 || shown[0].ID != f.second.ID {
		t.Errorf("the window was asked to show %v", shown)
	}
}

// TestAPageHoldingAnUnansweredQuestionKeepsTheVaultItWasTypedIn.
func TestAPageHoldingAnUnansweredQuestionKeepsTheVaultItWasTypedIn(t *testing.T) {
	f := onAList(t)
	f.opening(errAsking)

	out, err := f.client.OpenVault(t.Context(),
		connect.NewRequest(&v1.OpenVaultRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.VaultsErrorCode_VAULTS_ERROR_CODE_ASKING {
		t.Errorf("a page holding a question was answered %v", got)
	}
}

// TestAWindowThatIsGoingIsNotACodeAboutTheVault. What stopped the swap is the
// window, so it is said as a status and not as a code about the vault asked
// for.
func TestAWindowThatIsGoingIsNotACodeAboutTheVault(t *testing.T) {
	for _, one := range []struct {
		name string
		why  error
	}{
		{"the window is going", errGoing},
		{"the window is settling", errSettling},
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

// TestAPersonWhoClosedTheFolderDialogChoseNothing, which is an ordinary answer.
func TestAPersonWhoClosedTheFolderDialogChoseNothing(t *testing.T) {
	f := onAList(t)

	out, err := f.client.ChooseFolder(t.Context(), connect.NewRequest(&v1.ChooseFolderRequest{
		Title:      "Where are your notes?",
		StartingAt: f.first.Path,
	}))
	if err != nil {
		t.Fatalf("a dialog that was closed answered %v", err)
	}
	if out.Msg.GetChose() || out.Msg.GetPath() != "" {
		t.Errorf("the answer is %v", out.Msg)
	}
	if title, from := f.dialog.getAsked(); title != "Where are your notes?" || from != f.first.Path {
		t.Errorf("the dialog was titled %q and opened at %q", title, from)
	}
}

// TestTheFolderThePersonChoseIsAnswered.
func TestTheFolderThePersonChoseIsAnswered(t *testing.T) {
	f := onAList(t)
	f.dialog.pick, f.dialog.chose = f.second.Path, true

	out, err := f.client.ChooseFolder(t.Context(), connect.NewRequest(&v1.ChooseFolderRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Msg.GetChose() || out.Msg.GetPath() != f.second.Path {
		t.Errorf("the answer is %v", out.Msg)
	}
}

// TestASecondFolderDialogIsRefusedWhileOneIsUp. A person answers one at a time.
func TestASecondFolderDialogIsRefusedWhileOneIsUp(t *testing.T) {
	f := onAList(t)
	f.dialog.fails = port.ErrChoosing

	_, err := f.client.ChooseFolder(t.Context(), connect.NewRequest(&v1.ChooseFolderRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Errorf("a second dialog answered %v, want %v", got, connect.CodeUnavailable)
	}
}

// A build that does not serve the list answers no question about it. The list
// is a service and is mounted whole, so what is unanswered is unanswered
// because nothing serves it and not because a handler standing there has
// nothing behind it.
func TestABuildThatDoesNotServeTheListAnswersNothingAboutIt(t *testing.T) {
	api := &API{}
	server := httptest.NewUnstartedServer(
		api.NewHandler(http.NotFoundHandler(), numenv1connect.NoteServiceName))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)
	client := numenv1connect.NewVaultsServiceClient(server.Client(), server.URL)

	asked := map[string]func() error{
		"list": func() error {
			_, err := client.ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
			return err
		},
		"choose": func() error {
			_, err := client.ChooseFolder(t.Context(), connect.NewRequest(&v1.ChooseFolderRequest{}))
			return err
		},
		"add": func() error {
			_, err := client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: "/notes"}))
			return err
		},
		"rename": func() error {
			_, err := client.RenameVault(t.Context(), connect.NewRequest(&v1.RenameVaultRequest{Name: "journal"}))
			return err
		},
		"remove": func() error {
			_, err := client.RemoveVault(t.Context(), connect.NewRequest(&v1.RemoveVaultRequest{}))
			return err
		},
		"open": func() error {
			_, err := client.OpenVault(t.Context(), connect.NewRequest(&v1.OpenVaultRequest{}))
			return err
		},
	}
	for what, ask := range asked {
		if got := connect.CodeOf(ask()); got != connect.CodeUnimplemented {
			t.Errorf("%s answered %v, want %v", what, got, connect.CodeUnimplemented)
		}
	}
}

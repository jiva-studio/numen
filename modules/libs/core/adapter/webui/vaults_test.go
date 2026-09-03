package webui

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

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// vaultRows is the index as a vault is written to and taken out of it.
type vaultRows struct {
	mu     sync.Mutex
	saved  []domain.Vault
	forgot []string
}

func (r *vaultRows) Save(_ context.Context, v domain.Vault) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, v)
	return nil
}

func (r *vaultRows) Register(context.Context, domain.Vault) error { return nil }

func (r *vaultRows) Forget(_ context.Context, vaultID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.forgot = append(r.forgot, vaultID)
	return nil
}

func (r *vaultRows) forgotten() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.forgot...)
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

func (b *bin) trashed() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.moved...)
}

// folders is the person picking a folder, as a test answers for them. The real
// picker is this machine's own and needs a window.
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

// asked is what the picker was told to say and where to open.
func (f *folders) asked() (string, string) {
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
	picker   *folders
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

	registry := appstate.At(filepath.Join(t.TempDir(), "state", "vaults.json"))
	identity := filesystem.VaultIdentity{}
	adding := usecase.Add{Identity: identity, Registry: registry, Now: time.Now}

	rows := &vaultRows{}
	forget := usecase.Forget{Registry: registry, Index: rows}
	f := &onTheList{
		registry: registry,
		rows:     rows,
		bin:      &bin{},
		picker:   &folders{},
		first:    added(t, adding, "one"),
		second:   added(t, adding, "two"),
	}
	f.api = &API{
		Vaults:     registry,
		Picker:     f.picker,
		Adding:     &adding,
		Renaming:   &usecase.Rename{Registry: registry, Index: rows},
		Forgetting: &forget,
		Erasing:    &usecase.Erase{Identity: identity, Trash: f.bin, Forget: forget},
	}
	f.api.Opens = func(_ context.Context, v domain.Vault) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.shown = append(f.shown, v)
		return f.refuses
	}
	f.api.show(f.first)

	server := httptest.NewUnstartedServer(f.api.Serving(http.NotFoundHandler()))
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

// asked is every vault the window was asked to show.
func (f *onTheList) asked() []domain.Vault {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.Vault(nil), f.shown...)
}

// added makes a folder under a parent of its own and puts it on the list.
func added(t *testing.T, add usecase.Add, name string) domain.Vault {
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

// held is the list as the client is answered with it.
func (f *onTheList) held(t *testing.T) *v1.VaultsServiceListResponse {
	t.Helper()

	out, err := f.client.List(t.Context(), connect.NewRequest(&v1.VaultsServiceListRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg
}

// entry is the vault of that identity on the list the client was answered with.
func entry(t *testing.T, list *v1.VaultsServiceListResponse, id string) *v1.Known {
	t.Helper()

	for _, one := range list.GetVaults() {
		if one.GetId() == id {
			return one
		}
	}
	t.Fatalf("the list holds %v, and %s is not on it", list.GetVaults(), id)
	return nil
}

// TestTheListMarksAFolderThatIsGoneAndNamesTheVaultBeingShown.
func TestTheListMarksAFolderThatIsGoneAndNamesTheVaultBeingShown(t *testing.T) {
	f := onAList(t)
	if err := os.RemoveAll(f.second.Path); err != nil {
		t.Fatal(err)
	}

	list := f.held(t)
	if list.GetShowing() != string(f.first.ID) {
		t.Errorf("the window is showing %q, want %s", list.GetShowing(), string(f.first.ID))
	}
	if here := entry(t, list, string(f.first.ID)); here.GetMissing() {
		t.Errorf("%s is marked missing, and its folder is at %s", here.GetName(), here.GetPath())
	}
	gone := entry(t, list, string(f.second.ID))
	if !gone.GetMissing() {
		t.Errorf("%s is not marked missing, and there is nothing at %s", gone.GetName(), gone.GetPath())
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

	out, err := f.client.Add(t.Context(), connect.NewRequest(&v1.VaultsServiceAddRequest{Path: inner}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_OVERLAPS {
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

	out, err := f.client.Add(t.Context(), connect.NewRequest(&v1.VaultsServiceAddRequest{Path: copied}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_COPY {
		t.Errorf("a copy of a vault was answered %v", got)
	}
}

// TestAFolderJoinsTheListUnderANameAnotherVaultHas, numbered. Add does not
// refuse a taken name.
func TestAFolderJoinsTheListUnderANameAnotherVaultHas(t *testing.T) {
	f := onAList(t)

	out, err := f.client.Add(t.Context(), connect.NewRequest(&v1.VaultsServiceAddRequest{
		Path: folderNamed(t, "three"),
		Name: "one",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Refusal != nil {
		t.Fatalf("the folder was refused: %v", out.Msg.GetRefusal())
	}
	if got := out.Msg.GetVault().GetName(); got != "one 2" {
		t.Errorf("the vault that joined the list is called %q", got)
	}
	if len(f.held(t).GetVaults()) != 3 {
		t.Errorf("the list holds %v", f.held(t).GetVaults())
	}
}

// TestANameAnotherVaultHasIsNotGivenToASecond.
func TestANameAnotherVaultHasIsNotGivenToASecond(t *testing.T) {
	f := onAList(t)

	out, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.VaultsServiceRenameRequest{
		Id:   string(f.second.ID),
		Name: "one",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_NAME_TAKEN {
		t.Errorf("a name another vault has was answered %v", got)
	}
	if entry(t, f.held(t), string(f.second.ID)).GetName() != f.second.Name {
		t.Error("the vault was renamed all the same")
	}
}

// TestAVaultIsCalledWhatThePersonCallsIt.
func TestAVaultIsCalledWhatThePersonCallsIt(t *testing.T) {
	f := onAList(t)

	out, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.VaultsServiceRenameRequest{
		Id:   string(f.second.ID),
		Name: "journal",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Refusal != nil {
		t.Fatalf("the vault was not renamed: %v", out.Msg.GetRefusal())
	}
	if got := out.Msg.GetVault().GetName(); got != "journal" {
		t.Errorf("the vault is called %q", got)
	}
	if got := entry(t, f.held(t), string(f.second.ID)).GetName(); got != "journal" {
		t.Errorf("the list calls it %q", got)
	}
}

// TestTheVaultTheWindowIsShowingStaysOnTheList. The use cases are not told
// which vault is in front of the person; the handler is.
func TestTheVaultTheWindowIsShowingStaysOnTheList(t *testing.T) {
	f := onAList(t)

	forgot, err := f.client.Forget(t.Context(),
		connect.NewRequest(&v1.VaultsServiceForgetRequest{Id: string(f.first.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := forgot.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_SHOWING {
		t.Errorf("forgetting the vault being shown was answered %v", got)
	}

	erased, err := f.client.Erase(t.Context(),
		connect.NewRequest(&v1.VaultsServiceEraseRequest{Id: string(f.first.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := erased.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_SHOWING {
		t.Errorf("erasing the vault being shown was answered %v", got)
	}

	entry(t, f.held(t), string(f.first.ID))
	if got := f.rows.forgotten(); len(got) != 0 {
		t.Errorf("the index was told to forget %v", got)
	}
	if got := f.bin.trashed(); len(got) != 0 {
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

	gone, err := f.client.Forget(t.Context(),
		connect.NewRequest(&v1.VaultsServiceForgetRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if gone.Msg.Refusal != nil {
		t.Fatalf("the second vault was not forgotten: %v", gone.Msg.GetRefusal())
	}

	only, err := f.client.Forget(t.Context(),
		connect.NewRequest(&v1.VaultsServiceForgetRequest{Id: string(f.first.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := only.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_LAST_VAULT {
		t.Errorf("the only vault this installation has was answered %v", got)
	}
	entry(t, f.held(t), string(f.first.ID))
}

// TestAnIdentityOnNoListIsUnknown, whichever way it is asked about.
func TestAnIdentityOnNoListIsUnknown(t *testing.T) {
	f := onAList(t)
	const nobody = "01JZZZZZZZZZZZZZZZZZZZZZZZ"

	rename, err := f.client.Rename(t.Context(),
		connect.NewRequest(&v1.VaultsServiceRenameRequest{Id: nobody, Name: "journal"}))
	if err != nil {
		t.Fatal(err)
	}
	forget, err := f.client.Forget(t.Context(),
		connect.NewRequest(&v1.VaultsServiceForgetRequest{Id: nobody}))
	if err != nil {
		t.Fatal(err)
	}
	erase, err := f.client.Erase(t.Context(),
		connect.NewRequest(&v1.VaultsServiceEraseRequest{Id: nobody}))
	if err != nil {
		t.Fatal(err)
	}
	open, err := f.client.Open(t.Context(),
		connect.NewRequest(&v1.VaultsServiceOpenRequest{Id: nobody}))
	if err != nil {
		t.Fatal(err)
	}

	for what, got := range map[string]v1.VaultsRefusal{
		"rename": rename.Msg.GetRefusal(),
		"forget": forget.Msg.GetRefusal(),
		"erase":  erase.Msg.GetRefusal(),
		"open":   open.Msg.GetRefusal(),
	} {
		if got != v1.VaultsRefusal_VAULTS_REFUSAL_UNKNOWN {
			t.Errorf("%s of an identity on no list was answered %v", what, got)
		}
	}
	if shown := f.asked(); len(shown) != 0 {
		t.Errorf("the window was asked to show %v", shown)
	}
}

// TestAVaultGoesToTheTrashAndOffTheList.
func TestAVaultGoesToTheTrashAndOffTheList(t *testing.T) {
	f := onAList(t)

	out, err := f.client.Erase(t.Context(),
		connect.NewRequest(&v1.VaultsServiceEraseRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Refusal != nil {
		t.Fatalf("the vault was not erased: %v", out.Msg.GetRefusal())
	}
	if got := f.bin.trashed(); len(got) != 1 || got[0] != f.second.Path {
		t.Errorf("trashed %v, want %s", got, f.second.Path)
	}
	if got := f.rows.forgotten(); len(got) != 1 || got[0] != string(f.second.ID) {
		t.Errorf("the index was told to forget %v, want %s", got, string(f.second.ID))
	}
	if _, found, err := f.registry.Find(string(f.second.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
}

// TestAMachineWithNowhereToPutWhatIsDeletedErasesNothing.
func TestAMachineWithNowhereToPutWhatIsDeletedErasesNothing(t *testing.T) {
	f := onAList(t)
	f.bin.refuse(port.ErrNoTrash)

	out, err := f.client.Erase(t.Context(),
		connect.NewRequest(&v1.VaultsServiceEraseRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_NO_TRASH {
		t.Errorf("a machine with nowhere to put it was answered %v", got)
	}
	entry(t, f.held(t), string(f.second.ID))
	if got := f.rows.forgotten(); len(got) != 0 {
		t.Errorf("the index was told to forget %v", got)
	}
	if _, err := os.Stat(f.second.Path); err != nil {
		t.Errorf("the folder is gone: %v", err)
	}
}

// TestAnotherVaultIsPutInTheWindow.
func TestAnotherVaultIsPutInTheWindow(t *testing.T) {
	f := onAList(t)

	out, err := f.client.Open(t.Context(),
		connect.NewRequest(&v1.VaultsServiceOpenRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.Refusal != nil {
		t.Fatalf("the vault was not shown: %v", out.Msg.GetRefusal())
	}
	if shown := f.asked(); len(shown) != 1 || shown[0].ID != f.second.ID {
		t.Errorf("the window was asked to show %v", shown)
	}
}

// TestAPageHoldingAnUnansweredQuestionKeepsTheVaultItWasTypedIn.
func TestAPageHoldingAnUnansweredQuestionKeepsTheVaultItWasTypedIn(t *testing.T) {
	f := onAList(t)
	f.opening(errAsking)

	out, err := f.client.Open(t.Context(),
		connect.NewRequest(&v1.VaultsServiceOpenRequest{Id: string(f.second.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetRefusal(); got != v1.VaultsRefusal_VAULTS_REFUSAL_ASKING {
		t.Errorf("a page holding a question was answered %v", got)
	}
}

// TestAWindowThatIsGoingIsNotARefusalAboutTheVault. What stopped the swap is
// the window, so it is said as an error and not as something about the vault
// asked for.
func TestAWindowThatIsGoingIsNotARefusalAboutTheVault(t *testing.T) {
	for _, one := range []struct {
		name string
		why  error
	}{
		{"the window is closing", errGoing},
		{"the window is settling", errSettling},
	} {
		t.Run(one.name, func(t *testing.T) {
			f := onAList(t)
			f.opening(one.why)

			_, err := f.client.Open(t.Context(),
				connect.NewRequest(&v1.VaultsServiceOpenRequest{Id: string(f.second.ID)}))
			if got := connect.CodeOf(err); got != connect.CodeUnavailable {
				t.Errorf("a window that is going answered %v, want %v", got, connect.CodeUnavailable)
			}
		})
	}
}

// TestAPersonWhoClosedThePickerChoseNothing, which is an ordinary answer.
func TestAPersonWhoClosedThePickerChoseNothing(t *testing.T) {
	f := onAList(t)

	out, err := f.client.Choose(t.Context(), connect.NewRequest(&v1.VaultsServiceChooseRequest{
		Title:      "Where are your notes?",
		StartingAt: f.first.Path,
	}))
	if err != nil {
		t.Fatalf("a picker that was closed answered %v", err)
	}
	if out.Msg.GetChose() || out.Msg.GetPath() != "" {
		t.Errorf("the answer is %v", out.Msg)
	}
	if title, from := f.picker.asked(); title != "Where are your notes?" || from != f.first.Path {
		t.Errorf("the picker was titled %q and opened at %q", title, from)
	}
}

// TestTheFolderThePersonChoseIsAnswered.
func TestTheFolderThePersonChoseIsAnswered(t *testing.T) {
	f := onAList(t)
	f.picker.pick, f.picker.chose = f.second.Path, true

	out, err := f.client.Choose(t.Context(), connect.NewRequest(&v1.VaultsServiceChooseRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Msg.GetChose() || out.Msg.GetPath() != f.second.Path {
		t.Errorf("the answer is %v", out.Msg)
	}
}

// TestASecondPickerIsRefusedWhileOneIsUp. A person answers one at a time.
func TestASecondPickerIsRefusedWhileOneIsUp(t *testing.T) {
	f := onAList(t)
	f.picker.fails = port.ErrChoosing

	_, err := f.client.Choose(t.Context(), connect.NewRequest(&v1.VaultsServiceChooseRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Errorf("a second picker answered %v, want %v", got, connect.CodeUnavailable)
	}
}

// TestABuildWithoutThePieceSaysItCannotBeDoneHere, for every question about the
// list.
func TestABuildWithoutThePieceSaysItCannotBeDoneHere(t *testing.T) {
	api := &API{}
	server := httptest.NewUnstartedServer(api.Serving(http.NotFoundHandler()))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)
	client := numenv1connect.NewVaultsServiceClient(server.Client(), server.URL)

	asked := map[string]func() error{
		"list": func() error {
			_, err := client.List(t.Context(), connect.NewRequest(&v1.VaultsServiceListRequest{}))
			return err
		},
		"choose": func() error {
			_, err := client.Choose(t.Context(), connect.NewRequest(&v1.VaultsServiceChooseRequest{}))
			return err
		},
		"add": func() error {
			_, err := client.Add(t.Context(), connect.NewRequest(&v1.VaultsServiceAddRequest{Path: "/notes"}))
			return err
		},
		"rename": func() error {
			_, err := client.Rename(t.Context(), connect.NewRequest(&v1.VaultsServiceRenameRequest{Name: "journal"}))
			return err
		},
		"forget": func() error {
			_, err := client.Forget(t.Context(), connect.NewRequest(&v1.VaultsServiceForgetRequest{}))
			return err
		},
		"erase": func() error {
			_, err := client.Erase(t.Context(), connect.NewRequest(&v1.VaultsServiceEraseRequest{}))
			return err
		},
		"open": func() error {
			_, err := client.Open(t.Context(), connect.NewRequest(&v1.VaultsServiceOpenRequest{}))
			return err
		},
	}
	for what, ask := range asked {
		if got := connect.CodeOf(ask()); got != connect.CodeUnimplemented {
			t.Errorf("%s answered %v, want %v", what, got, connect.CodeUnimplemented)
		}
	}
}

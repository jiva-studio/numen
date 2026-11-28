package vault_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfolder"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type connection struct {
	remoteID string
	vaultID  domain.VaultID
}

type fakeProvider struct {
	listed       []vaultdomain.RemoteVault
	listFailure  error
	connectError error
	onConnect    func()
	connected    []connection
}

func (p *fakeProvider) ListRemoteVaults(context.Context) ([]vaultdomain.RemoteVault, error) {
	return p.listed, p.listFailure
}

func (p *fakeProvider) ConnectRemoteVault(_ context.Context, remoteID string, vaultID domain.VaultID) error {
	p.connected = append(p.connected, connection{remoteID: remoteID, vaultID: vaultID})
	if p.onConnect != nil {
		p.onConnect()
	}
	return p.connectError
}

type fixedFolders struct{ home, documents string }

func (f fixedFolders) GetHomeFolder() (string, error)      { return f.home, nil }
func (f fixedFolders) GetDocumentsFolder() (string, error) { return f.documents, nil }

type remoteFixture struct {
	t        *testing.T
	registry *appstate.VaultRegistry
	provider *fakeProvider
	parent   string
	open     *vaults.OpenRemoteVault
}

func newRemoteFixture(t *testing.T) *remoteFixture {
	t.Helper()
	registry := registryAt(t)
	clock := testsupport.GetMachine()
	add := vaults.NewAdd(vaultreader.Identity{}, registry, registry, registry, clock)
	provider := &fakeProvider{}
	parent := t.TempDir()
	user := fixedFolders{home: filepath.Join(parent, "home"), documents: filepath.Join(parent, "Documents")}
	open := vaults.NewOpenRemoteVault(registry, registry, vaults.RemoteVaultPlaces{Paths: vaultfolder.Folders{}, Folders: vaultfolder.Folders{}, User: user}, add, provider, clock)
	return &remoteFixture{t: t, registry: registry, provider: provider, parent: parent, open: open}
}

func (f *remoteFixture) listVaults() []domain.Vault {
	f.t.Helper()
	listed, err := f.registry.List()
	if err != nil {
		f.t.Fatal(err)
	}
	return listed
}

func (f *remoteFixture) makeFolder(name string, files ...string) string {
	f.t.Helper()
	path := filepath.Join(f.parent, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		f.t.Fatal(err)
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(path, file), []byte("x"), 0o600); err != nil {
			f.t.Fatal(err)
		}
	}
	return path
}

func TestARemoteVaultIsOpenedInANewFolderThatIsRegisteredAndConnected(t *testing.T) {
	f := newRemoteFixture(t)
	path := filepath.Join(f.parent, "Notes")

	id, err := f.open.Execute(t.Context(), "r-1", path)

	if err != nil {
		t.Fatal(err)
	}
	listed := f.listVaults()
	if len(listed) != 1 || listed[0].ID != id || filepath.Base(listed[0].Path) != "Notes" || listed[0].Name != "Notes" {
		t.Errorf("the list holds %+v, want the vault %s named Notes", listed, id)
	}
	if len(f.provider.connected) != 1 || f.provider.connected[0] != (connection{remoteID: "r-1", vaultID: id}) {
		t.Errorf("the provider connected %+v, want r-1 to %s", f.provider.connected, id)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Errorf("the folder of the vault is not there: %v", err)
	}
}

func TestARemoteVaultIsOpenedInAnEmptyFolderThatExists(t *testing.T) {
	f := newRemoteFixture(t)
	path := f.makeFolder("Empty")

	id, err := f.open.Execute(t.Context(), "r-1", path)

	if err != nil {
		t.Fatal(err)
	}
	if listed := f.listVaults(); len(listed) != 1 || listed[0].ID != id {
		t.Errorf("the list holds %+v", listed)
	}
}

func TestAPathThatCannotBeAVaultFolderIsRefusedBeforeAnythingIsMade(t *testing.T) {
	f := newRemoteFixture(t)
	existing := f.makeFolder("Existing")
	if _, err := f.open.Execute(t.Context(), "r-0", existing); err != nil {
		t.Fatal(err)
	}
	f.provider.connected = nil
	filled := f.makeFolder("Filled", "a.md")
	file := f.makeFolder("Holder", "file.md")
	home := filepath.Join(f.parent, "home")
	if err := os.Mkdir(home, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.VolumeName(f.parent) + string(os.PathSeparator)
	cases := map[string]string{
		"empty":                    "",
		"relative":                 "Notes",
		"a whole disk":             root,
		"the home folder":          home,
		"no parent":                filepath.Join(f.parent, "missing", "Notes"),
		"inside a vault":           filepath.Join(existing, "Notes"),
		"a folder holding a vault": f.parent,
		"a filled folder":          filled,
		"a file":                   filepath.Join(file, "file.md"),
	}

	for name, path := range cases {
		_, err := f.open.Execute(t.Context(), "r-1", path)

		if err == nil {
			t.Errorf("%s: the vault was opened at %q", name, path)
		}
		if len(f.provider.connected) != 0 {
			t.Errorf("%s: the provider was asked to connect %+v", name, f.provider.connected)
		}
	}
	if listed := f.listVaults(); len(listed) != 1 {
		t.Errorf("the list holds %d vaults after the refusals, want the one added before", len(listed))
	}
	if _, err := os.Stat(filepath.Join(f.parent, "missing")); !os.IsNotExist(err) {
		t.Errorf("a folder was made for a path that has no parent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filled, "a.md")); err != nil {
		t.Errorf("the file of the filled folder is gone: %v", err)
	}
}

func TestAPathThatIsNotValidIsAnInvalidPathError(t *testing.T) {
	f := newRemoteFixture(t)

	_, err := f.open.Execute(t.Context(), "r-1", "Notes")

	if !errors.Is(err, vaultdomain.ErrInvalidPath) {
		t.Errorf("the error is %v, want ErrInvalidPath", err)
	}
	_, err = f.open.Execute(t.Context(), "r-1", f.makeFolder("Filled", "a.md"))
	if !errors.Is(err, vaultdomain.ErrInvalidPath) {
		t.Errorf("the error for a filled folder is %v, want ErrInvalidPath", err)
	}
}

func TestOpenRemoteVault_ConnectFails_RemovesTheVaultAndTheCreatedFolder(t *testing.T) {
	f := newRemoteFixture(t)
	path := filepath.Join(f.parent, "Notes")
	failure := errors.New("the server is down")
	f.provider.connectError = failure
	f.provider.onConnect = func() {
		if err := os.WriteFile(filepath.Join(path, "from-provider.md"), []byte("x"), 0o600); err != nil {
			t.Error(err)
		}
	}

	_, err := f.open.Execute(t.Context(), "r-1", path)

	if !errors.Is(err, failure) {
		t.Fatalf("the error is %v, want the one of the provider", err)
	}
	if listed := f.listVaults(); len(listed) != 0 {
		t.Errorf("the list holds %+v after the failure", listed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the folder the call made is still there: %v", err)
	}
}

func TestAFailedConnectionKeepsAFolderThatExistedBeforeAndEmptiesIt(t *testing.T) {
	f := newRemoteFixture(t)
	path := f.makeFolder("Existing")
	f.provider.connectError = errors.New("the server is down")

	_, err := f.open.Execute(t.Context(), "r-1", path)

	if err == nil {
		t.Fatal("the connection did not fail")
	}
	entries, readErr := os.ReadDir(path)
	if readErr != nil {
		t.Fatalf("the folder that existed before is gone: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("%d entries of the call are left in the folder", len(entries))
	}
	if listed := f.listVaults(); len(listed) != 0 {
		t.Errorf("the list holds %+v after the failure", listed)
	}
}

func TestAFailedConnectionKeepsAFileThatIsOlderThanTheCall(t *testing.T) {
	f := newRemoteFixture(t)
	path := filepath.Join(f.parent, "Notes")
	f.provider.connectError = errors.New("the server is down")
	f.provider.onConnect = func() {
		kept := filepath.Join(path, "mine.md")
		if err := os.WriteFile(kept, []byte("x"), 0o600); err != nil {
			t.Error(err)
		}
		old := testsupport.GetMachine().Now().Add(-time.Hour)
		if err := os.Chtimes(kept, old, old); err != nil {
			t.Error(err)
		}
	}

	_, err := f.open.Execute(t.Context(), "r-1", path)

	if err == nil {
		t.Fatal("the connection did not fail")
	}
	if _, statErr := os.Stat(filepath.Join(path, "mine.md")); statErr != nil {
		t.Errorf("the file of the person was removed: %v", statErr)
	}
	if listed := f.listVaults(); len(listed) != 0 {
		t.Errorf("the list holds %+v after the failure", listed)
	}
}

func TestRemoteVaultsAreListedFromTheProvider(t *testing.T) {
	provider := &fakeProvider{listed: []vaultdomain.RemoteVault{{RemoteID: "r-1", Name: "Notes", SizeBytes: 5, VaultID: "v-1"}}}

	got := vaults.NewListRemoteVaults(provider).Execute(t.Context())

	if len(got) != 1 || got[0].RemoteID != "r-1" || got[0].VaultID != "v-1" || got[0].SizeBytes != 5 {
		t.Errorf("the list is %+v", got)
	}
}

func TestListRemoteVaults_ProviderFailsOrOffersNothing_ReturnsEmptyList(t *testing.T) {
	cases := map[string]*fakeProvider{
		"failure": {listFailure: errors.New("the plugin is not installed"), listed: []vaultdomain.RemoteVault{{RemoteID: "r-1"}}},
		"nothing": {},
	}

	for name, provider := range cases {
		got := vaults.NewListRemoteVaults(provider).Execute(t.Context())

		if got == nil || len(got) != 0 {
			t.Errorf("%s: the list is %#v, want an empty one", name, got)
		}
	}
}

func newSuggestion(parent, setting string) (*vaults.SuggestVaultPath, string) {
	documents := filepath.Join(parent, "Documents")
	user := fixedFolders{home: parent, documents: documents}
	return vaults.NewSuggestVaultPath(func() string { return setting }, user, vaultfolder.Folders{}, vaultfolder.Folders{}), documents
}

func TestTheDocumentsFolderIsWhereANewVaultGoesWhenNoFolderIsSet(t *testing.T) {
	suggest, documents := newSuggestion(t.TempDir(), "")

	got, err := suggest.Execute("Notes", "")

	if err != nil {
		t.Fatal(err)
	}
	if got.Folder != documents || got.Path != filepath.Join(documents, "Notes") || got.IsTaken {
		t.Errorf("the suggestion is %+v", got)
	}
}

func TestTheFolderOfTheSettingIsWhereANewVaultGoes(t *testing.T) {
	parent := t.TempDir()
	set := filepath.Join(parent, "Vaults")
	suggest, _ := newSuggestion(parent, set)

	got, err := suggest.Execute("Notes", "")

	if err != nil {
		t.Fatal(err)
	}
	if got.Folder != set || got.Path != filepath.Join(set, "Notes") {
		t.Errorf("the suggestion is %+v, want a path in %s", got, set)
	}
}

func TestATakenNameGetsTheLowestFreeNumber(t *testing.T) {
	parent := t.TempDir()
	suggest, documents := newSuggestion(parent, "")
	for _, name := range []string{"Notes", "Notes 2"} {
		if err := os.MkdirAll(filepath.Join(documents, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := suggest.Execute("Notes", "")

	if err != nil {
		t.Fatal(err)
	}
	if got.Path != filepath.Join(documents, "Notes 3") || !got.IsTaken {
		t.Errorf("the suggestion is %+v, want Notes 3 and a taken name", got)
	}
}

func TestANameThatCannotNameAFolderIsMadeOneThatCan(t *testing.T) {
	suggest, documents := newSuggestion(t.TempDir(), "")
	cases := map[string]string{"a/b": "a-b", `c:\d`: "c--d", "  ": vaultdomain.DefaultFolderName, "..": vaultdomain.DefaultFolderName, "Plain": "Plain"}

	for name, want := range cases {
		got, err := suggest.Execute(name, "")

		if err != nil {
			t.Fatal(err)
		}
		if got.Path != filepath.Join(documents, want) {
			t.Errorf("%q: the path is %q, want the folder %q", name, got.Path, want)
		}
	}
}

func TestAFolderForNewVaultsThatIsNotAbsoluteIsAnInvalidPath(t *testing.T) {
	suggest, _ := newSuggestion(t.TempDir(), "relative/vaults")

	_, err := suggest.Execute("Notes", "")

	if !errors.Is(err, vaultdomain.ErrInvalidPath) {
		t.Errorf("the error is %v, want ErrInvalidPath", err)
	}
}

func TestATakenNameInTheChosenFolderGetsTheLowestFreeNumberThere(t *testing.T) {
	parent := t.TempDir()
	suggest, documents := newSuggestion(parent, "")
	chosen := filepath.Join(parent, "Music")
	if err := os.MkdirAll(filepath.Join(chosen, "Notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := suggest.Execute("Notes", chosen)

	if err != nil {
		t.Fatal(err)
	}
	if got.Folder != chosen || got.Path != filepath.Join(chosen, "Notes 2") || !got.IsTaken {
		t.Errorf("the suggestion is %+v, want Notes 2 in %s and not in %s", got, chosen, documents)
	}
}

func TestAChosenFolderThatIsNotAbsoluteIsAnInvalidPath(t *testing.T) {
	suggest, _ := newSuggestion(t.TempDir(), "")

	_, err := suggest.Execute("Notes", "relative/music")

	if !errors.Is(err, vaultdomain.ErrInvalidPath) {
		t.Errorf("the error is %v, want ErrInvalidPath", err)
	}
}

package plugin

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// memoryVault is a vault of files kept in memory that answers as the host does.
type memoryVault struct {
	mu          sync.Mutex
	vault       *domain.Vault
	files       map[string]string
	stamp       int64
	data        map[string][]byte
	cache       map[string][]byte
	scanned     [][]string
	locks       int
	changes     chan []string
	droppedChan chan struct{}
	// others are the vaults beside the shown one, by identity.
	others map[domain.VaultID]*memoryVault
	state  port.PluginIndexState
	inUse  int
	moves  chan struct{}
}

func newMemoryVault(files map[string]string) *memoryVault {
	return &memoryVault{
		vault:       &domain.Vault{ID: "v1", Name: "Physics", Path: "/vaults/Physics"},
		files:       files,
		data:        map[string][]byte{},
		cache:       map[string][]byte{},
		changes:     make(chan []string, 4),
		droppedChan: make(chan struct{}),
		moves:       make(chan struct{}, 4),
		state:       port.PluginIndexState{Status: port.PluginIndexed},
	}
}

func (m *memoryVault) GetVaults() ([]domain.Vault, error) {
	out := []domain.Vault{}
	if m.vault != nil {
		out = append(out, *m.vault)
	}
	for _, other := range m.others {
		out = append(out, *other.vault)
	}
	slices.SortFunc(out, func(a, b domain.Vault) int { return strings.Compare(string(a.ID), string(b.ID)) })
	return out, nil
}

func (m *memoryVault) With(id domain.VaultID) (port.PluginVault, error) {
	if m.vault != nil && m.vault.ID == id {
		return m, nil
	}
	if other, there := m.others[id]; there {
		return other, nil
	}
	return nil, fs.ErrNotExist
}

func (m *memoryVault) EnsureIndexed(context.Context) (port.PluginIndexState, error) {
	return m.state, nil
}

func (m *memoryVault) SetInUse(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inUse++
	return nil
}

func (m *memoryVault) WatchMoves(context.Context) (<-chan struct{}, error) {
	return m.moves, nil
}

func (m *memoryVault) GetVault() (domain.Vault, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.vault == nil {
		return domain.Vault{}, false
	}
	return *m.vault, true
}

func (m *memoryVault) fingerprint(path string) (domain.Fingerprint, error) {
	body, there := m.files[path]
	if !there {
		return domain.Fingerprint{}, fs.ErrNotExist
	}
	return domain.Fingerprint{Path: path, Size: int64(len(body)), ModTime: time.Unix(0, 100+m.stamp)}, nil
}

func (m *memoryVault) GetFingerprint(_ context.Context, path string) (domain.Fingerprint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fingerprint(path)
}

func (m *memoryVault) Read(_ context.Context, path string) ([]byte, domain.Fingerprint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pos, err := m.fingerprint(path)
	if err != nil {
		return nil, domain.Fingerprint{}, err
	}
	return []byte(m.files[path]), pos, nil
}

func (m *memoryVault) Write(_ context.Context, path string, content []byte, baseline domain.Fingerprint) (domain.Fingerprint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now, err := m.fingerprint(path)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	if !now.ModTime.Equal(baseline.ModTime) {
		return domain.Fingerprint{}, port.ErrStale
	}
	m.stamp++
	m.files[path] = string(content)
	return m.fingerprint(path)
}

func (m *memoryVault) Create(_ context.Context, path string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, there := m.files[path]; there {
		return port.ErrOccupied
	}
	m.files[path] = string(content)
	return nil
}

func (m *memoryVault) Lock(context.Context) (func(), error) {
	m.mu.Lock()
	m.locks++
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		m.locks--
		m.mu.Unlock()
	}, nil
}

func (m *memoryVault) Scan(_ context.Context, paths []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanned = append(m.scanned, paths)
	return nil
}

func (m *memoryVault) GetNotes(_ context.Context, paths []string) (map[string]domain.NoteRef, error) {
	out := map[string]domain.NoteRef{}
	for _, path := range paths {
		if _, there := m.files[path]; there {
			out[path] = domain.NoteRef{Path: path, Title: "Title of " + path}
		}
	}
	return out, nil
}

func (m *memoryVault) GetTypes(_ context.Context, paths []string) (map[string]domain.NoteType, error) {
	out := map[string]domain.NoteType{}
	for _, path := range paths {
		if _, there := m.files[path]; there {
			out[path] = "deck"
		}
	}
	return out, nil
}

func (m *memoryVault) GetNamedPaths(_ context.Context, name string) ([]string, error) {
	var out []string
	for path := range m.files {
		if domain.Basename(path) == name {
			out = append(out, path)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (m *memoryVault) GetPathsOfType(_ context.Context, noteType domain.NoteType) ([]string, error) {
	if noteType != "deck" {
		return nil, nil
	}
	paths := make([]string, 0, len(m.files))
	for path := range m.files {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths, nil
}

func (m *memoryVault) link(from string) domain.ResolvedLink {
	return domain.ResolvedLink{
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Animal"}, Type: "preset", Label: "by"},
		From: from, To: "Animal.md", ToVault: "v1", IsAmbiguous: true,
	}
}

func (m *memoryVault) GetLinks(_ context.Context, from string) ([]domain.ResolvedLink, error) {
	return []domain.ResolvedLink{m.link(from)}, nil
}

func (m *memoryVault) GetBacklinks(_ context.Context, to string) ([]domain.ResolvedLink, error) {
	return []domain.ResolvedLink{m.link("decks/" + to)}, nil
}

func (m *memoryVault) ResolveLinks(_ context.Context, from string, output []string) (map[string]domain.ResolvedLink, error) {
	out := map[string]domain.ResolvedLink{}
	for _, one := range output {
		out[one] = m.link(from)
	}
	return out, nil
}

func (m *memoryVault) GetParseProblems(context.Context) ([]domain.VaultProblem, error) {
	return []domain.VaultProblem{{Path: "decks/A.md", Detail: "an entry has no role"}}, nil
}

type memoryData struct{ vault *memoryVault }

func (d memoryData) Read(_ context.Context, name string) ([]byte, error) {
	d.vault.mu.Lock()
	defer d.vault.mu.Unlock()
	content, there := d.vault.data[name]
	if !there {
		return nil, fs.ErrNotExist
	}
	return content, nil
}

func (d memoryData) Append(_ context.Context, name string, content []byte) error {
	d.vault.mu.Lock()
	defer d.vault.mu.Unlock()
	d.vault.data[name] = append(d.vault.data[name], content...)
	return nil
}

func (d memoryData) List(context.Context) ([]port.PluginDataFile, error) {
	d.vault.mu.Lock()
	defer d.vault.mu.Unlock()
	var out []port.PluginDataFile
	for name, content := range d.vault.data {
		out = append(out, port.PluginDataFile{Name: name, Size: len(content)})
	}
	slices.SortFunc(out, func(a, b port.PluginDataFile) int { return len(a.Name) - len(b.Name) })
	return out, nil
}

func (m *memoryVault) OpenData(string) (port.PluginDataStore, error) {
	return memoryData{vault: m}, nil
}

func (m *memoryVault) ReadCache(_ context.Context, pluginID, name string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	content, found := m.cache[pluginID+"/"+name]
	return content, found, nil
}

func (m *memoryVault) WriteCache(_ context.Context, pluginID, name string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[pluginID+"/"+name] = content
	return nil
}

func (m *memoryVault) Watch(context.Context) (<-chan []string, <-chan struct{}, error) {
	return m.changes, m.droppedChan, nil
}

// newVaultClient serves the host's vault endpoint over the callers and returns a client for the plugin the token names.
func newVaultClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.HostVaultServiceClient {
	t.Helper()
	endpoint := &HostEndpoint{callers: fakeCallers}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostVaultServiceHandler(newVaultEndpoint(endpoint)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(authInterceptor(token)))
}

func newVaultFixture(t *testing.T) (numenv1connect.HostVaultServiceClient, *memoryVault) {
	t.Helper()
	vault := newMemoryVault(map[string]string{"decks/A.md": "body", "Animal.md": "stencil"})
	hosted := newCallers(nil)
	hosted.vault = vault
	hosted.data = map[string]string{"md.numen.a": "flashcards"}
	return newVaultClient(t, hosted, "token-a"), vault
}

func TestTheVaultTheWindowShowsIsNamed(t *testing.T) {
	client, vault := newVaultFixture(t)

	res, err := client.GetHostVault(t.Context(), connect.NewRequest(&v1.GetHostVaultRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetId() != "v1" || res.Msg.GetName() != "Physics" || res.Msg.GetPath() != "/vaults/Physics" {
		t.Errorf("the vault was %v", res.Msg)
	}
	vault.mu.Lock()
	vault.vault = nil
	vault.mu.Unlock()
	res, err = client.GetHostVault(t.Context(), connect.NewRequest(&v1.GetHostVaultRequest{}))
	if err != nil || res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT {
		t.Errorf("a host showing none answered %v (%v)", res.Msg, err)
	}
}

func TestAHostWithNoVaultToOfferAnswersThatItHasNone(t *testing.T) {
	hosted := newCallers(nil)
	client := newVaultClient(t, hosted, "token-a")

	res, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "decks/A.md"}))

	if err != nil || res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT {
		t.Errorf("the answer was %v (%v)", res.Msg, err)
	}
}

func TestAFileIsReadWithWhatItWasWhenAskedAbout(t *testing.T) {
	client, _ := newVaultFixture(t)

	res, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "decks/A.md"}))
	if err != nil {
		t.Fatal(err)
	}

	if string(res.Msg.GetContent()) != "body" || res.Msg.GetFingerprint().GetPath() != "decks/A.md" ||
		res.Msg.GetFingerprint().GetSize() != 4 {
		t.Errorf("the file came back as %v", res.Msg)
	}
	missing, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "nothing.md"}))
	if err != nil || missing.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_MISSING {
		t.Errorf("a file that is not there was answered %v (%v)", missing.Msg, err)
	}
}

func TestAFileIsSavedOnlyOverWhatWasRead(t *testing.T) {
	client, vault := newVaultFixture(t)
	read, err := client.GetFileFingerprint(t.Context(), connect.NewRequest(&v1.GetFileFingerprintRequest{Path: "decks/A.md"}))
	if err != nil {
		t.Fatal(err)
	}

	first, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "decks/A.md", Content: []byte("new"), Baseline: read.Msg.GetFingerprint(),
	}))
	if err != nil || first.Msg.GetError() != 0 || first.Msg.GetFingerprint().GetSize() != 3 {
		t.Fatalf("the write was answered %v (%v)", first.Msg, err)
	}
	second, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "decks/A.md", Content: []byte("again"), Baseline: read.Msg.GetFingerprint(),
	}))

	if err != nil || second.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_STALE {
		t.Errorf("a write over what moved was answered %v (%v)", second.Msg, err)
	}
	if vault.files["decks/A.md"] != "new" {
		t.Errorf("the file holds %q", vault.files["decks/A.md"])
	}
}

func TestAFileIsMadeOnlyWhereNoneExists(t *testing.T) {
	client, _ := newVaultFixture(t)

	first, err := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "decks/B.md", Content: []byte("b")}))
	if err != nil || first.Msg.GetError() != 0 {
		t.Fatalf("the first was answered %v (%v)", first.Msg, err)
	}
	second, err := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "decks/B.md", Content: []byte("b")}))

	if err != nil || second.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OCCUPIED {
		t.Errorf("a file made over one was answered %v (%v)", second.Msg, err)
	}
}

func TestTheWritersAreHeldUntilTheyAreLetGo(t *testing.T) {
	client, vault := newVaultFixture(t)
	locked, err := client.LockVault(t.Context(), connect.NewRequest(&v1.LockVaultRequest{}))
	if err != nil || locked.Msg.GetLock() == "" {
		t.Fatalf("the lock was answered %v (%v)", locked.Msg, err)
	}
	if vault.locks != 1 {
		t.Fatalf("%d locks held", vault.locks)
	}

	for range 2 {
		if _, err := client.UnlockVault(t.Context(), connect.NewRequest(&v1.UnlockVaultRequest{Lock: locked.Msg.GetLock()})); err != nil {
			t.Fatal(err)
		}
	}

	vault.mu.Lock()
	defer vault.mu.Unlock()
	if vault.locks != 0 {
		t.Errorf("%d locks held after they were let go", vault.locks)
	}
}

func TestTheIndexIsBroughtLevelWithTheFilesNamed(t *testing.T) {
	client, vault := newVaultFixture(t)

	if _, err := client.ScanFiles(t.Context(), connect.NewRequest(&v1.ScanFilesRequest{Paths: []string{"decks/A.md"}})); err != nil {
		t.Fatal(err)
	}

	if len(vault.scanned) != 1 || vault.scanned[0][0] != "decks/A.md" {
		t.Errorf("the index was scanned for %v", vault.scanned)
	}
}

func TestWhatTheIndexReturnsOfTheNotesAndTheirLinksIsAnswered(t *testing.T) {
	client, _ := newVaultFixture(t)
	ctx := t.Context()

	refs, err := client.GetNoteRefs(ctx, connect.NewRequest(&v1.GetNoteRefsRequest{Paths: []string{"decks/A.md", "gone.md"}}))
	if err != nil || len(refs.Msg.GetNotes()) != 1 || refs.Msg.GetNotes()[0].GetTitle() != "Title of decks/A.md" {
		t.Errorf("the notes were %v (%v)", refs.Msg, err)
	}
	types, err := client.GetNoteTypes(ctx, connect.NewRequest(&v1.GetNoteTypesRequest{Paths: []string{"decks/A.md"}}))
	if err != nil || types.Msg.GetTypes()["decks/A.md"] != "deck" {
		t.Errorf("the types were %v (%v)", types.Msg, err)
	}
	paths, err := client.GetPathsOfType(ctx, connect.NewRequest(&v1.GetPathsOfTypeRequest{Type: "deck"}))
	if err != nil || !slices.Equal(paths.Msg.GetPaths(), []string{"Animal.md", "decks/A.md"}) {
		t.Errorf("the paths were %v (%v)", paths.Msg, err)
	}
	links, err := client.GetLinks(ctx, connect.NewRequest(&v1.GetLinksRequest{Path: "decks/A.md"}))
	if err != nil || len(links.Msg.GetLinks()) != 1 {
		t.Fatalf("the links were %v (%v)", links.Msg, err)
	}
	link := links.Msg.GetLinks()[0]
	if link.GetFrom() != "decks/A.md" || link.GetTo() != "Animal.md" || link.GetToVault() != "v1" || link.GetLinkType() != "preset" ||
		link.GetTargetScheme() != "name" || link.GetTargetValue() != "Animal" || link.GetLabel() != "by" || !link.GetIsAmbiguous() {
		t.Errorf("the link came out as %v", link)
	}
	back, err := client.GetBacklinks(ctx, connect.NewRequest(&v1.GetBacklinksRequest{Path: "Animal.md"}))
	if err != nil || len(back.Msg.GetLinks()) != 1 {
		t.Errorf("the backlinks were %v (%v)", back.Msg, err)
	}
	resolved, err := client.ResolveLinks(ctx, connect.NewRequest(&v1.ResolveLinksRequest{Path: "decks/A.md", Written: []string{"Animal"}}))
	if err != nil || resolved.Msg.GetLinks()["Animal"].GetTo() != "Animal.md" {
		t.Errorf("the resolved links were %v (%v)", resolved.Msg, err)
	}
	problems, err := client.GetParseProblems(ctx, connect.NewRequest(&v1.GetParseProblemsRequest{}))
	if err != nil || len(problems.Msg.GetProblems()) != 1 || problems.Msg.GetProblems()[0].GetDetail() != "an entry has no role" {
		t.Errorf("the problems were %v (%v)", problems.Msg, err)
	}
}

func TestADataFolderIsCachedOnlyWhereTheManifestNamesOne(t *testing.T) {
	client, vault := newVaultFixture(t)
	ctx := t.Context()

	if _, err := client.AppendData(ctx, connect.NewRequest(&v1.AppendDataRequest{Name: "run.log", Content: []byte("one\n")})); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AppendData(ctx, connect.NewRequest(&v1.AppendDataRequest{Name: "run.log", Content: []byte("two\n")})); err != nil {
		t.Fatal(err)
	}
	read, err := client.ReadData(ctx, connect.NewRequest(&v1.ReadDataRequest{Name: "run.log"}))
	items, listErr := client.ListData(ctx, connect.NewRequest(&v1.ListDataRequest{}))
	missing, missingErr := client.ReadData(ctx, connect.NewRequest(&v1.ReadDataRequest{Name: "other.log"}))

	if err != nil || string(read.Msg.GetContent()) != "one\ntwo\n" {
		t.Errorf("the data read back was %q (%v)", read.Msg.GetContent(), err)
	}
	if listErr != nil || len(items.Msg.GetFiles()) != 1 || items.Msg.GetFiles()[0].GetSize() != 8 {
		t.Errorf("the files were %v (%v)", items.Msg, listErr)
	}
	if missingErr != nil || missing.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_MISSING {
		t.Errorf("a missing file was answered %v (%v)", missing.Msg, missingErr)
	}

	hosted := newCallers(nil)
	hosted.vault = vault
	bareClient := newVaultClient(t, hosted, "token-a")
	none, err := bareClient.ReadData(ctx, connect.NewRequest(&v1.ReadDataRequest{Name: "run.log"}))
	if err != nil || none.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_DATA_FOLDER {
		t.Errorf("a plugin with no folder was answered %v (%v)", none.Msg, err)
	}
}

func TestACacheIsCachedPerPluginAndRemembersWhetherAnythingWasCached(t *testing.T) {
	client, _ := newVaultFixture(t)
	ctx := t.Context()

	before, err := client.ReadCache(ctx, connect.NewRequest(&v1.ReadCacheRequest{Name: "schedules"}))
	if err != nil || before.Msg.GetFound() {
		t.Fatalf("an empty cache answered %v (%v)", before.Msg, err)
	}
	if _, err := client.WriteCache(ctx, connect.NewRequest(&v1.WriteCacheRequest{Name: "schedules", Content: []byte("{}")})); err != nil {
		t.Fatal(err)
	}
	after, err := client.ReadCache(ctx, connect.NewRequest(&v1.ReadCacheRequest{Name: "schedules"}))

	if err != nil || !after.Msg.GetFound() || string(after.Msg.GetContent()) != "{}" {
		t.Errorf("the cache answered %v (%v)", after.Msg, err)
	}
}

func TestAWatchReturnsItIsOnThenTellsOfTheFilesThatChanged(t *testing.T) {
	client, vault := newVaultFixture(t)
	stream, err := client.WatchVault(t.Context(), connect.NewRequest(&v1.WatchVaultRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if !stream.Receive() || len(stream.Msg().GetPaths()) != 0 {
		t.Fatalf("the watch did not begin: %v", stream.Err())
	}
	vault.changes <- []string{"decks/A.md"}
	if !stream.Receive() || !slices.Equal(stream.Msg().GetPaths(), []string{"decks/A.md"}) {
		t.Fatalf("the change was not told: %v %v", stream.Msg(), stream.Err())
	}
	close(vault.droppedChan)
	if stream.Receive() {
		t.Errorf("a watch that was lost went on with %v", stream.Msg())
	}
}

func TestAWatchReturnsWhenTheWindowShowsAnotherVault(t *testing.T) {
	client, vault := newVaultFixture(t)
	stream, err := client.WatchVault(t.Context(), connect.NewRequest(&v1.WatchVaultRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if !stream.Receive() {
		t.Fatal(stream.Err())
	}

	vault.mu.Lock()
	vault.vault = &domain.Vault{ID: "v2", Path: "/vaults/Other"}
	vault.mu.Unlock()

	if !stream.Receive() || !stream.Msg().GetIsVaultChanged() {
		t.Errorf("the change of vault was not told: %v %v", stream.Msg(), stream.Err())
	}
}

func TestACallIsRefusedWithoutThePermissionItNeeds(t *testing.T) {
	vault := newMemoryVault(map[string]string{"a.md": "a"})
	hosted := newCallers(nil)
	hosted.vault = vault
	client := newVaultClient(t, hosted, "token-b")

	_, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "a.md"}))
	_, writeErr := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "b.md"}))
	_, anonymous := newVaultClient(t, hosted, "").GetHostVault(t.Context(), connect.NewRequest(&v1.GetHostVaultRequest{}))

	if got := readCode(t, err); got != connect.CodePermissionDenied {
		t.Errorf("a read without vault:read is %v", got)
	}
	if got := readCode(t, writeErr); got != connect.CodePermissionDenied {
		t.Errorf("a write without vault:write is %v", got)
	}
	if got := readCode(t, anonymous); got != connect.CodeUnauthenticated {
		t.Errorf("a call with no token is %v", got)
	}
	if !slices.Contains(hosted.checked, "md.numen.b "+port.PermissionVaultRead) || !slices.Contains(hosted.checked, "md.numen.b "+port.PermissionVaultWrite) {
		t.Errorf("the permissions asked were %v", hosted.checked)
	}
}

func TestAnErrorTheHostCannotNameIsInternal(t *testing.T) {
	if _, err := answer(t.Context(), errors.New("the disk is gone")); readCode(t, err) != connect.CodeInternal {
		t.Errorf("the failure was %v", err)
	}
}

func TestAHeaderNamesTheVaultACallWorks(t *testing.T) {
	client, vault := newVaultFixture(t)
	vault.others = map[domain.VaultID]*memoryVault{
		"v2": {vault: &domain.Vault{ID: "v2", Name: "Chemistry", Path: "/vaults/Chemistry"}},
	}
	req := connect.NewRequest(&v1.GetHostVaultRequest{})
	req.Header().Set("Numen-Vault", "v2")

	res, err := client.GetHostVault(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	unknown := connect.NewRequest(&v1.GetHostVaultRequest{})
	unknown.Header().Set("Numen-Vault", "v9")
	_, rejected := client.GetHostVault(t.Context(), unknown)

	if res.Msg.GetName() != "Chemistry" || connect.CodeOf(rejected) != connect.CodeNotFound {
		t.Errorf("the vault was %v, and an unknown one was answered %v", res.Msg, rejected)
	}
}

func TestEveryVaultTheInstallationKnowsIsListed(t *testing.T) {
	client, vault := newVaultFixture(t)
	vault.others = map[domain.VaultID]*memoryVault{
		"v2": {vault: &domain.Vault{ID: "v2", Name: "Chemistry", Path: "/vaults/Chemistry"}},
	}

	res, err := client.ListHostVaults(t.Context(), connect.NewRequest(&v1.ListHostVaultsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	got := []string{}
	for _, one := range res.Msg.GetVaults() {
		got = append(got, one.GetId()+" "+one.GetName())
	}
	if !slices.Equal(got, []string{"v1 Physics", "v2 Chemistry"}) {
		t.Errorf("the vaults were %v", got)
	}
}

func TestWhereTheIndexExistsIsSaid(t *testing.T) {
	client, vault := newVaultFixture(t)
	cases := map[port.PluginIndexStatus]v1.HostIndexState{
		port.PluginIndexed:     v1.HostIndexState_HOST_INDEX_STATE_INDEXED,
		port.PluginIndexing:    v1.HostIndexState_HOST_INDEX_STATE_INDEXING,
		port.PluginIndexFailed: v1.HostIndexState_HOST_INDEX_STATE_FAILED,
	}
	for status, want := range cases {
		vault.state = port.PluginIndexState{Status: status, Reason: "why"}

		res, err := client.EnsureVaultIndexed(t.Context(), connect.NewRequest(&v1.EnsureVaultIndexedRequest{}))

		if err != nil || res.Msg.GetState() != want || res.Msg.GetReason() != "why" {
			t.Errorf("status %v was answered %v (%v)", status, res.Msg, err)
		}
	}
}

func TestAWatchOfMovesTellsThatSomethingMoved(t *testing.T) {
	client, vault := newVaultFixture(t)
	stream, err := client.WatchMoves(t.Context(), connect.NewRequest(&v1.WatchMovesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if !stream.Receive() {
		t.Fatalf("the watch began with %v", stream.Err())
	}

	vault.moves <- struct{}{}

	if !stream.Receive() {
		t.Errorf("nothing received of the move: %v", stream.Err())
	}
}

func TestTheNotesFiledBelowANameAreListed(t *testing.T) {
	client, _ := newVaultFixture(t)

	res, err := client.GetNamedPaths(t.Context(), connect.NewRequest(&v1.GetNamedPathsRequest{Name: "Animal"}))

	if err != nil || !slices.Equal(res.Msg.GetPaths(), []string{"Animal.md"}) {
		t.Errorf("the paths were %v (%v)", res.Msg.GetPaths(), err)
	}
}

func TestAFingerprintCarriesWhatSortOfSourceTheFileIs(t *testing.T) {
	endpoint := writeFingerprint(domain.Fingerprint{Path: "B.epub", Kind: domain.KindBook, Size: 3, ModTime: time.Unix(5, 6)})

	got := readFingerprint(endpoint)

	if got.Kind != domain.KindBook || got.Path != "B.epub" || !got.ModTime.Equal(time.Unix(5, 6)) {
		t.Errorf("got %+v", got)
	}
}

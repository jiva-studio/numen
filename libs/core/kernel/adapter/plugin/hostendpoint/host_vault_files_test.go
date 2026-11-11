package hostendpoint_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/hold"
)

// channelWatcher is the watch of the filesystem: the paths it reports and the loss of the watch are
// channels a test writes to.
type channelWatcher struct {
	changes chan []string
	loss    chan struct{}
}

func (w channelWatcher) WatchVault(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return w.changes, w.loss, nil
}

// diskVaults is the vaults of an installation over folders on disk: the real readers and writers
// under the use cases of the host, and the index, the watch and the window as stand-ins the test
// controls.
type diskVaults struct {
	mu      sync.Mutex
	current *domain.Vault
	vaults  map[domain.VaultID]domain.Vault
	state   plugin.IndexState
	scanned [][]string

	watcher channelWatcher
	moves   chan struct{}
}

func (d *diskVaults) setCurrent(v *domain.Vault) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.current = v
}

func (d *diskVaults) setState(state plugin.IndexState) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.state = state
}

func (d *diskVaults) getScanned() [][]string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.scanned)
}

func (d *diskVaults) access(t *testing.T) hostvault.Vault {
	t.Helper()
	readers := vaultreader.Readers{}
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	return hostvault.Vault{
		Session: hostvault.Session{
			Current: func() (domain.Vault, bool) {
				d.mu.Lock()
				defer d.mu.Unlock()
				if d.current == nil {
					return domain.Vault{}, false
				}
				return *d.current, true
			},
			Watcher: vaults.NewWatchChanges(d.watcher).Execute,
			Vaults: func() ([]domain.Vault, error) {
				d.mu.Lock()
				defer d.mu.Unlock()
				out := []domain.Vault{}
				for _, one := range d.vaults {
					out = append(out, one)
				}
				slices.SortFunc(out, func(a, b domain.Vault) int { return compareIDs(a.ID, b.ID) })
				return out, nil
			},
			Find: func(id domain.VaultID) (domain.Vault, error) {
				d.mu.Lock()
				defer d.mu.Unlock()
				found, isKnown := d.vaults[id]
				if !isKnown {
					return domain.Vault{}, domain.ErrUnavailable
				}
				return found, nil
			},
			IndexState: func(context.Context, domain.Vault) (plugin.IndexState, error) {
				d.mu.Lock()
				defer d.mu.Unlock()
				return d.state, nil
			},
			Moved: func(context.Context) (<-chan struct{}, error) { return d.moves, nil },
		},
		Files: hostvault.Files{
			Stat:        file.NewStatFile(readers),
			ReadFile:    file.NewReadFile(readers),
			RangeReader: file.Read{Readers: readers},
			OpenWriter:  func(v domain.Vault) (hostvault.NoteWriter, error) { return writers.Open(v) },
			Index: func(_ context.Context, _ domain.Vault, paths []string) error {
				d.mu.Lock()
				defer d.mu.Unlock()
				d.scanned = append(d.scanned, slices.Clone(paths))
				return nil
			},
		},
		Stores: hostvault.Stores{Cache: appstate.PluginCachesAt(t.TempDir())},
		Locks:  hold.NewLocks(writers, testsupport.GetMachine(), hold.LockHold),
	}
}

func compareIDs(a, b domain.VaultID) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func newDiskVaults(t *testing.T, files map[string]string) (*diskVaults, numenv1connect.HostVaultServiceClient) {
	t.Helper()
	physics := vaultfixture.NewVault(t, files)
	physics.ID, physics.Name = "v1", "Physics"
	chemistry := vaultfixture.NewVault(t, map[string]string{"Baikal.md": "# Baikal\n"})
	chemistry.ID, chemistry.Name = "v2", "Chemistry"
	disk := &diskVaults{
		current: &physics,
		vaults:  map[domain.VaultID]domain.Vault{physics.ID: physics, chemistry.ID: chemistry},
		state:   plugin.IndexState{Status: plugin.Indexed},
		watcher: channelWatcher{changes: make(chan []string, 4), loss: make(chan struct{})},
		moves:   make(chan struct{}, 4),
	}
	return disk, openReadEndpoint(t, disk.access(t).Access())
}

func TestTheVaultTheWindowShowsIsNamed(t *testing.T) {
	disk, client := newDiskVaults(t, nil)

	res, err := client.GetHostVault(t.Context(), connect.NewRequest(&v1.GetHostVaultRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetId() != "v1" || res.Msg.GetName() != "Physics" || res.Msg.GetPath() == "" {
		t.Errorf("the vault was %v", res.Msg)
	}
	disk.setCurrent(nil)
	res, err = client.GetHostVault(t.Context(), connect.NewRequest(&v1.GetHostVaultRequest{}))
	if err != nil || res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT {
		t.Errorf("a host showing none answered %v (%v)", res.Msg, err)
	}
}

func TestAReadFileReturnsItsContentAndFingerprint(t *testing.T) {
	_, client := newDiskVaults(t, map[string]string{"samples/A.md": "body"})

	res, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "samples/A.md"}))
	if err != nil {
		t.Fatal(err)
	}

	if string(res.Msg.GetContent()) != "body" || res.Msg.GetStat().GetPath() != "samples/A.md" || res.Msg.GetStat().GetSize() != 4 {
		t.Errorf("the file came back as %v", res.Msg)
	}
	missing, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "nothing.md"}))
	if err != nil || missing.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND {
		t.Errorf("a file that is not there was answered %v (%v)", missing.Msg, err)
	}
}

func TestAFileIsSavedOnlyOverWhatWasRead(t *testing.T) {
	disk, client := newDiskVaults(t, map[string]string{"samples/A.md": "body"})
	read, err := client.StatFile(t.Context(), connect.NewRequest(&v1.StatFileRequest{Path: "samples/A.md"}))
	if err != nil {
		t.Fatal(err)
	}

	first, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "samples/A.md", Content: []byte("new"), Baseline: read.Msg.GetStat(),
	}))
	if err != nil || first.Msg.GetError() != 0 || first.Msg.GetStat().GetSize() != 3 {
		t.Fatalf("the write was answered %v (%v)", first.Msg, err)
	}
	second, err := client.WriteFile(t.Context(), connect.NewRequest(&v1.WriteFileRequest{
		Path: "samples/A.md", Content: []byte("again"), Baseline: read.Msg.GetStat(),
	}))

	if err != nil || second.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT {
		t.Errorf("a write over what moved was answered %v (%v)", second.Msg, err)
	}
	got, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "samples/A.md"}))
	if err != nil || string(got.Msg.GetContent()) != "new" {
		t.Errorf("the file holds %q (%v)", got.Msg.GetContent(), err)
	}
	if scanned := disk.getScanned(); len(scanned) != 1 || !slices.Equal(scanned[0], []string{"samples/A.md"}) {
		t.Errorf("the index was scanned for %v, want the one write that was kept", scanned)
	}
}

func TestAFileIsMadeOnlyWhereNoneExists(t *testing.T) {
	_, client := newDiskVaults(t, nil)

	first, err := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "samples/B.md", Content: []byte("b")}))
	if err != nil || first.Msg.GetError() != 0 {
		t.Fatalf("the first was answered %v (%v)", first.Msg, err)
	}
	second, err := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "samples/B.md", Content: []byte("b")}))

	if err != nil || second.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS {
		t.Errorf("a file made over one was answered %v (%v)", second.Msg, err)
	}
}

func TestLockVaultRefusesASecondLockUntilUnlockVault(t *testing.T) {
	_, client := newDiskVaults(t, nil)
	locked, err := client.LockVault(t.Context(), connect.NewRequest(&v1.LockVaultRequest{}))
	if err != nil || locked.Msg.GetLock() == "" {
		t.Fatalf("the lock was answered %v (%v)", locked.Msg, err)
	}

	waiting, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, heldBack := client.LockVault(waiting, connect.NewRequest(&v1.LockVaultRequest{}))
	if heldBack == nil {
		t.Fatal("a second lock was granted while the first was held")
	}
	for range 2 {
		if _, err := client.UnlockVault(t.Context(), connect.NewRequest(&v1.UnlockVaultRequest{Lock: locked.Msg.GetLock()})); err != nil {
			t.Fatal(err)
		}
	}

	again, err := client.LockVault(t.Context(), connect.NewRequest(&v1.LockVaultRequest{}))
	if err != nil || again.Msg.GetLock() == "" {
		t.Errorf("a lock after the release was answered %v (%v)", again.Msg, err)
	}
}

func TestTheIndexIsBroughtLevelWithTheFilesNamed(t *testing.T) {
	disk, client := newDiskVaults(t, map[string]string{"samples/A.md": "body"})

	if _, err := client.ScanFiles(t.Context(), connect.NewRequest(&v1.ScanFilesRequest{Paths: []string{"samples/A.md"}})); err != nil {
		t.Fatal(err)
	}

	if scanned := disk.getScanned(); len(scanned) != 1 || !slices.Equal(scanned[0], []string{"samples/A.md"}) {
		t.Errorf("the index was scanned for %v", scanned)
	}
}

func TestReadCacheAndWriteCacheKeepEntriesPerPluginAndName(t *testing.T) {
	_, client := newDiskVaults(t, nil)
	ctx := t.Context()

	before, err := client.ReadCache(ctx, connect.NewRequest(&v1.ReadCacheRequest{Name: "schedules"}))
	if err != nil || before.Msg.GetIsFound() {
		t.Fatalf("an empty cache answered %v (%v)", before.Msg, err)
	}
	if _, err := client.WriteCache(ctx, connect.NewRequest(&v1.WriteCacheRequest{Name: "schedules", Content: []byte("{}")})); err != nil {
		t.Fatal(err)
	}
	after, err := client.ReadCache(ctx, connect.NewRequest(&v1.ReadCacheRequest{Name: "schedules"}))

	if err != nil || !after.Msg.GetIsFound() || string(after.Msg.GetContent()) != "{}" {
		t.Errorf("the cache answered %v (%v)", after.Msg, err)
	}
}

func TestAWatchTellsOfTheFilesThatChangedUntilItIsLost(t *testing.T) {
	disk, client := newDiskVaults(t, nil)
	stream, err := client.WatchVault(t.Context(), connect.NewRequest(&v1.WatchVaultRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)

	if !stream.Receive() || len(stream.Msg().GetPaths()) != 0 {
		t.Fatalf("the watch did not begin: %v", stream.Err())
	}
	disk.watcher.changes <- []string{"samples/A.md"}
	if !stream.Receive() || !slices.Equal(stream.Msg().GetPaths(), []string{"samples/A.md"}) {
		t.Fatalf("the change was not told: %v %v", stream.Msg(), stream.Err())
	}
	close(disk.watcher.loss)
	if stream.Receive() {
		t.Errorf("a watch that was lost went on with %v", stream.Msg())
	}
}

func TestAWatchEndsWhenTheWindowShowsAnotherVault(t *testing.T) {
	disk, client := newDiskVaults(t, nil)
	stream, err := client.WatchVault(t.Context(), connect.NewRequest(&v1.WatchVaultRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)
	if !stream.Receive() {
		t.Fatal(stream.Err())
	}

	other := domain.Vault{ID: "v2", Path: t.TempDir()}
	disk.setCurrent(&other)

	if !stream.Receive() || !stream.Msg().GetIsVaultChanged() {
		t.Errorf("the change of vault was not told: %v %v", stream.Msg(), stream.Err())
	}
}

func TestAHeaderNamesTheVaultACallWorks(t *testing.T) {
	_, client := newDiskVaults(t, nil)
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

func TestEveryVaultOfTheInstallationIsListed(t *testing.T) {
	_, client := newDiskVaults(t, nil)

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

func TestTheIndexStateOfAVaultIsAnswered(t *testing.T) {
	disk, client := newDiskVaults(t, nil)
	cases := map[plugin.IndexStatus]v1.HostIndexState{
		plugin.Indexed:     v1.HostIndexState_HOST_INDEX_STATE_INDEXED,
		plugin.Indexing:    v1.HostIndexState_HOST_INDEX_STATE_INDEXING,
		plugin.IndexFailed: v1.HostIndexState_HOST_INDEX_STATE_FAILED,
	}
	for status, want := range cases {
		disk.setState(plugin.IndexState{Status: status, Reason: "why"})

		res, err := client.EnsureVaultIndexed(t.Context(), connect.NewRequest(&v1.EnsureVaultIndexedRequest{}))

		if err != nil || res.Msg.GetState() != want || res.Msg.GetReason() != "why" {
			t.Errorf("status %v was answered %v (%v)", status, res.Msg, err)
		}
	}
}

func TestAWatchOfMovesTellsThatSomethingMoved(t *testing.T) {
	disk, client := newDiskVaults(t, nil)
	stream, err := client.WatchMoves(t.Context(), connect.NewRequest(&v1.WatchMovesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)
	if !stream.Receive() {
		t.Fatalf("the watch began with %v", stream.Err())
	}

	disk.moves <- struct{}{}

	if !stream.Receive() {
		t.Errorf("nothing received of the move: %v", stream.Err())
	}
}

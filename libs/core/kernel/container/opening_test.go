package container_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

func getNote() map[string]string {
	return map[string]string{"Leaf.md": "---\ntitle: Leaf\n---\n\n# Leaf\n"}
}

// A note written while the vault is being walked is read again after it, so the
// newest copy of it lands last. A walk writes in groups from what it read, and
// its copy of a note lands whenever the group does.
func TestANoteSavedBelowTheWalkIsReadAgain(t *testing.T) {
	cfg, db, v := openVault(t, getNote())
	opening := container.VaultOpener(cfg, db)
	open := opening.Begin(t.Context(), v, nil, nil)

	// Written through UpdateIndex while the walk is running, as the window does when
	// the user saves.
	if _, err := open.Read(t.Context(), func() {
		write(t, v, "Leaf.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
		if err := opening.UpdateIndex(t.Context(), v, []string{"Leaf.md"}); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}

	if got := titleOf(t, db, v, "Leaf.md"); got != "Renamed" {
		t.Errorf("the index says %q", got)
	}
}

// A vault walked a second time is walked with the same guard as the first: a
// vault whose first walk failed is walked again on the opening it already has,
// and a note written under that walk is read again after it.
func TestASecondWalkContainsWhatIsSavedBelowIt(t *testing.T) {
	cfg, db, v := openVault(t, getNote())

	target := makeGatedReaders()
	opening := cfg.VaultOpenerWith(db, target, cfg.VaultWatcher())
	open := opening.Begin(t.Context(), v, nil, nil)

	// The first walk completes, and the second is held while it carries the
	// note's old bytes.
	target.release()
	if _, err := open.Read(t.Context(), nil); err != nil {
		t.Fatal(err)
	}

	// Changed on disk, so the second walk has the note to read again.
	write(t, v, "Leaf.md", "---\ntitle: Middle\n---\n\n# Middle\n")
	target.forget()
	target.hold()

	walked := make(chan error, 1)
	go func() {
		_, err := open.Read(t.Context(), nil)
		walked <- err
	}()

	if path := <-target.begun; path != "Leaf.md" {
		t.Fatalf("the walk is reading %q", path)
	}
	write(t, v, "Leaf.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	if err := opening.UpdateIndex(t.Context(), v, []string{"Leaf.md"}); err != nil {
		t.Fatal(err)
	}
	target.release()

	if err := <-walked; err != nil {
		t.Fatal(err)
	}
	if got := titleOf(t, db, v, "Leaf.md"); got != "Renamed" {
		t.Errorf("the index says %q after a second walk", got)
	}
}

// A vault that cannot be watched is still opened, and the error is reported.
func TestAVaultThatCannotBeWatchedIsOpenedAndReturnsSo(t *testing.T) {
	cfg, db, v := openVault(t, getNote())
	open := cfg.VaultOpenerWith(db, container.VaultReaders(cfg), refusing{}).Begin(t.Context(), v, nil, nil)

	if open.GetUnwatchedReason() == nil {
		t.Fatal("a vault nobody can follow says nothing about it")
	}
	// And it still reads: what cannot be followed can still be walked.
	if _, err := open.Read(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if got := titleOf(t, db, v, "Leaf.md"); got != "Leaf" {
		t.Errorf("the index says %q", got)
	}
}

// A vault is walked one at a time, and what the watch collected is processed
// after the first walk ends. The note on disk is what the index contains
// afterwards, however old the copy held by the first walk.
func TestARescanDoesNotRunAdjacentTheFirstWalk(t *testing.T) {
	cfg, db, v := openVault(t, getNote())

	watcher := makeWatcher()
	readers := makeStagedReaders()
	opening := cfg.VaultOpenerWith(db, readers, watcher)

	changesCh := make(chan vault.VaultChanges, 8)

	open := opening.Begin(t.Context(), v, func(m vault.VaultChanges) { changesCh <- m }, nil)
	go open.Run(t.Context())

	walked := make(chan error, 1)
	go func() {
		_, err := open.Read(t.Context(), nil)
		walked <- err
	}()

	// The first walk holds the note's old bytes.
	if path := receiveFrom(t, readers.read, "the walk read nothing"); path != "Leaf.md" {
		t.Fatalf("the walk is reading %q", path)
	}
	write(t, v, "Leaf.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")

	// More changed at once than the watch could follow, which is answered by
	// reading the vault again.
	watcher.lostChan <- struct{}{}
	select {
	case m := <-changesCh:
		t.Fatalf("the vault was read again beside the first walk: %+v", m)
	case <-time.After(250 * time.Millisecond):
	}

	readers.release()
	if err := receiveFrom(t, walked, "the first walk did not finish"); err != nil {
		t.Fatal(err)
	}
	if m := receiveFrom(t, changesCh, "the vault was never read again"); !m.ShouldReload {
		t.Fatalf("reported %+v", m)
	}
	if got := titleOf(t, db, v, "Leaf.md"); got != "Renamed" {
		t.Errorf("the index says %q", got)
	}
}

func receiveFrom[T any](t *testing.T, from <-chan T, what string) T {
	t.Helper()
	select {
	case value := <-from:
		return value
	case <-time.After(10 * time.Second):
		t.Fatal(what)
		var zero T
		return zero
	}
}

func makeWatcher() *waves {
	return &waves{changes: make(chan []string), lostChan: make(chan struct{}, 1)}
}

type waves struct {
	changes  chan []string
	lostChan chan struct{}
}

func (w *waves) WatchVault(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return w.changes, w.lostChan, nil
}

func makeStagedReaders() *staged {
	return &staged{
		VaultReaders: vaultreader.Readers{},
		read:         make(chan string, 1),
		first:        make(chan struct{}),
	}
}

type staged struct {
	vaultport.VaultReaders
	read  chan string
	first chan struct{}
	walks atomic.Int64
}

func (s *staged) release() { close(s.first) }

func (s *staged) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := s.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return stagedRead{VaultReader: reader, targetPath: s, isHolding: s.walks.Add(1) == 1}, nil
}

type stagedRead struct {
	vaultport.VaultReader
	targetPath *staged
	isHolding  bool
}

func (r stagedRead) Read(ctx context.Context, path string) ([]byte, error) {
	raw, err := r.VaultReader.Read(ctx, path)
	if !r.isHolding {
		return raw, err
	}
	select {
	case r.targetPath.read <- path:
	default:
	}
	select {
	case <-r.targetPath.first:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return raw, err
}

func makeGatedReaders() *gate {
	return &gate{
		VaultReaders: vaultreader.Readers{},
		begun:        make(chan string, 1),
		until:        make(chan struct{}),
	}
}

type gate struct {
	vaultport.VaultReaders
	begun chan string

	mu    sync.Mutex
	until chan struct{}
}

func (g *gate) forget() {
	select {
	case <-g.begun:
	default:
	}
}

func (g *gate) hold() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.until = make(chan struct{})
}

func (g *gate) release() {
	g.mu.Lock()
	defer g.mu.Unlock()
	select {
	case <-g.until:
	default:
		close(g.until)
	}
}

func (g *gate) getWaiting() chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.until
}

func (g *gate) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := g.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return gating{VaultReader: reader, targetPath: g}, nil
}

type gating struct {
	vaultport.VaultReader
	targetPath *gate
}

func (r gating) Read(ctx context.Context, path string) ([]byte, error) {
	raw, err := r.VaultReader.Read(ctx, path)
	select {
	case r.targetPath.begun <- path:
	default:
	}
	select {
	case <-r.targetPath.getWaiting():
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return raw, err
}

// refusing is a watcher on a machine that has no watches left to give.
type refusing struct{}

func (refusing) WatchVault(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return nil, nil, os.ErrPermission
}

func openVault(t *testing.T, notes map[string]string) (container.Config, *container.Index, domain.Vault) {
	t.Helper()

	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: indexfile.Path(t)},
	})
	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	return cfg, db, testsupport.NewVault(t, notes)
}

func write(t *testing.T, v domain.Vault, path, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Path, filepath.FromSlash(path)), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func titleOf(t *testing.T, db *container.Index, v domain.Vault, path string) string {
	t.Helper()

	target, err := db.NoteQueries().GetNotes(t.Context(), v.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	one, there := target[path]
	if !there {
		t.Fatalf("the index holds no note at %s", path)
	}
	return one.Title
}

package container_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// note is one vault of one note.
var note = map[string]string{"Leaf.md": "---\ntitle: Leaf\n---\n\n# Leaf\n"}

// A note written while the vault is being walked is read again after it, so the
// newest copy of it lands last. A walk writes in groups from what it read, and
// its copy of a note lands whenever the group does.
func TestANoteWrittenUnderTheWalkIsReadAgain(t *testing.T) {
	cfg, db, v := opened(t, note)
	opening := cfg.Opening(db)
	open := opening.Begin(t.Context(), v)

	// Written through the levelling while the walk is running, which is what the
	// window does when a person saves.
	if _, err := open.Read(t.Context(), func(usecase.ScanResult) {
		write(t, v, "Leaf.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
		if err := opening.Level(t.Context(), v, []string{"Leaf.md"}); err != nil {
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
func TestASecondWalkHoldsWhatIsWrittenUnderIt(t *testing.T) {
	cfg, db, v := opened(t, note)

	held := gated()
	opening := cfg.OpeningWith(db, held, cfg.VaultWatcher())
	open := opening.Begin(t.Context(), v)

	// The first walk goes through, and the second is held with the note's old
	// bytes in the walk's hand.
	held.release()
	if _, err := open.Read(t.Context(), func(usecase.ScanResult) {}); err != nil {
		t.Fatal(err)
	}

	// Changed on disk, so the second walk has the note to read again.
	write(t, v, "Leaf.md", "---\ntitle: Middle\n---\n\n# Middle\n")
	held.forget()
	held.hold()

	walked := make(chan error, 1)
	go func() {
		_, err := open.Read(t.Context(), func(usecase.ScanResult) {})
		walked <- err
	}()

	if path := <-held.begun; path != "Leaf.md" {
		t.Fatalf("the walk is reading %q", path)
	}
	write(t, v, "Leaf.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	if err := opening.Level(t.Context(), v, []string{"Leaf.md"}); err != nil {
		t.Fatal(err)
	}
	held.release()

	if err := <-walked; err != nil {
		t.Fatal(err)
	}
	if got := titleOf(t, db, v, "Leaf.md"); got != "Renamed" {
		t.Errorf("the index says %q after a second walk", got)
	}
}

// A vault that cannot be watched is opened all the same, and says why.
func TestAVaultThatCannotBeWatchedIsOpenedAndSaysSo(t *testing.T) {
	cfg, db, v := opened(t, note)
	open := cfg.OpeningWith(db, cfg.VaultReaders(), refusing{}).Begin(t.Context(), v)

	if open.Unwatched() == nil {
		t.Fatal("a vault nobody can follow says nothing about it")
	}
	// And it still reads: what cannot be followed can still be walked.
	if _, err := open.Read(t.Context(), func(usecase.ScanResult) {}); err != nil {
		t.Fatal(err)
	}
	if got := titleOf(t, db, v, "Leaf.md"); got != "Leaf" {
		t.Errorf("the index says %q", got)
	}
}

// gated is readers that hold each read open once the bytes are in hand, so a
// test can change the file the walk is holding a copy of.
func gated() *gate {
	return &gate{
		VaultReaders: filesystem.Readers{},
		begun:        make(chan string, 1),
		until:        make(chan struct{}),
	}
}

type gate struct {
	port.VaultReaders
	begun chan string

	mu    sync.Mutex
	until chan struct{}
}

// forget lets go of the last path read, so what is waited for next is a read
// that has not happened yet.
func (g *gate) forget() {
	select {
	case <-g.begun:
	default:
	}
}

// hold makes every read from now on wait to be released.
func (g *gate) hold() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.until = make(chan struct{})
}

// release lets every read waiting, and every read after it, through.
func (g *gate) release() {
	g.mu.Lock()
	defer g.mu.Unlock()
	select {
	case <-g.until:
	default:
		close(g.until)
	}
}

func (g *gate) waiting() chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.until
}

func (g *gate) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := g.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return gating{VaultReader: reader, at: g}, nil
}

type gating struct {
	port.VaultReader
	at *gate
}

func (r gating) Read(ctx context.Context, path string) ([]byte, error) {
	raw, err := r.VaultReader.Read(ctx, path)
	select {
	case r.at.begun <- path:
	default:
	}
	select {
	case <-r.at.waiting():
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return raw, err
}

// refusing is a watcher on a machine that has no watches left to give.
type refusing struct{}

func (refusing) Watch(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return nil, nil, os.ErrPermission
}

// opened is a vault and an index of a test's own.
func opened(t *testing.T, notes map[string]string) (container.Config, *container.Index, domain.Vault) {
	t.Helper()

	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	return cfg, db, testsupport.NewVault(t, notes)
}

func write(t *testing.T, v domain.Vault, path, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Path, filepath.FromSlash(path)), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// titleOf is what the index says a note is called.
func titleOf(t *testing.T, db *container.Index, v domain.Vault, path string) string {
	t.Helper()

	held, err := db.Queries().Notes(t.Context(), v.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	one, there := held[path]
	if !there {
		t.Fatalf("the index holds no note at %s", path)
	}
	return one.Title
}

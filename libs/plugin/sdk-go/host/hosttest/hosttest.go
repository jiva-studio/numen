// Package hosttest is a vault and a queue that answer as the host does, for the tests of a plugin
// that works them through package host. The vault's files are held in memory, or in a folder where
// a test can change them as another program would.
package hosttest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Vault is a vault the host shows. Its notes are in the index as soon as their files are there.
type Vault struct {
	world      *world
	vault      domain.Vault
	isShown    bool
	files      vaultFiles
	writes     uint64
	indexed    map[string]domain.Note
	feed       feed
	rejected   map[string]error
	unwritable map[string]error
	cache      map[string][]byte
	locked     chan struct{}
	scanned    [][]string
	unknown    bool
	isMissing  bool
	state      host.IndexState
	moves      []chan struct{}
	folder     *folder
	trashed    []string
	tb         testing.TB
}

var (
	_ host.VaultSelector = (*Vault)(nil)
	_ host.VaultIndex    = (*Vault)(nil)
	_ host.FileReader    = (*Vault)(nil)
	_ host.FileWriter    = (*Vault)(nil)
	_ host.NoteLookup    = (*Vault)(nil)
	_ host.LinkLookup    = (*Vault)(nil)
	_ host.CacheStore    = (*Vault)(nil)
)

// New is a vault of the name, shown by the host, whose files are held in memory.
func New(name string) *Vault {
	return newVault(domain.Vault{ID: domain.VaultID(name), Name: name, Path: "/vaults/" + name}, createMemoryFiles())
}

// CreateOnDisk returns a vault of the notes given, written into a folder of the test's own. The notes are
// files there, and a test that changes them changes the vault.
func CreateOnDisk(tb testing.TB, name string, notes map[string]string) *Vault {
	tb.Helper()
	root := tb.TempDir()
	for filePath, body := range notes {
		target := filepath.Join(root, filepath.FromSlash(filePath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	v := newVault(domain.Vault{ID: domain.VaultID(name), Name: name, Path: root}, createDiskFiles(root))
	v.tb = tb
	v.Reindex()
	return v
}

func newVault(v domain.Vault, files vaultFiles) *Vault {
	made := &Vault{
		vault:    v,
		isShown:  true,
		files:    files,
		indexed:  map[string]domain.Note{},
		feed:     newFeed(),
		rejected: map[string]error{},

		unwritable: map[string]error{},
		cache:      map[string][]byte{},
		locked:     make(chan struct{}, 1),
		state:      host.IndexState{Status: host.Indexed},
	}
	made.world = &world{members: []*Vault{made}}
	return made
}

// world is the set of vaults of a host. One lock guards them.
type world struct {
	mu                sync.Mutex
	members           []*Vault
	openedVault       *Vault
	hasUnsavedChanges bool
}

// Vault is the vault as the host names it: its identity, its name and where it is.
func (v *Vault) Vault() domain.Vault { return v.vault }

// RefuseWriting makes every write of the file fail with err, as a file in a folder the host may not
// change does, and leaves it readable. A nil err lets it be written again.
func (v *Vault) RefuseWriting(filePath string, err error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err == nil {
		delete(v.unwritable, filePath)
		return
	}
	v.unwritable[filePath] = err
}

// Refuse makes every read and write of the file fail with err, as a file the host may not open
// does. A nil err lets the file be worked again.
func (v *Vault) Refuse(filePath string, err error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err == nil {
		delete(v.rejected, filePath)
		return
	}
	v.rejected[filePath] = err
}

// Move notifies the move watches that a move happened.
func (v *Vault) Move() {
	v.world.mu.Lock()
	watches := append([]chan struct{}(nil), v.moves...)
	v.world.mu.Unlock()
	for _, w := range watches {
		w <- struct{}{}
	}
}

// errGone is what a call on the files of a vault gets when its folder is not there any longer.
var errGone = errors.New("the vault's folder is gone")

func (v *Vault) reach() error {
	if v.files.root != "" {
		if _, err := os.Stat(v.files.root); err != nil {
			return errGone
		}
	}
	return nil
}

func (v *Vault) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v.unknown {
		return host.ErrUnknownVault
	}
	if !v.isShown {
		return host.ErrNoVault
	}
	return nil
}

// Hide makes the host show no vault.
func (v *Vault) Hide() {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.isShown = false
}

// WriteFile creates or replaces a file, as another window would, and notifies the index
// as the watcher would..
func (v *Vault) WriteFile(filePath string, content string) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if _, err := v.put(filePath, []byte(content)); err != nil {
		v.report(fmt.Errorf("write %s: %w", filePath, err))
	}
	v.rescan(filePath)
}

// Remove takes a file away, as another window would.
func (v *Vault) Remove(filePath string) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.files.remove(filePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		v.report(fmt.Errorf("remove %s: %w", filePath, err))
	}
	v.rescan(filePath)
}

// ReadContent returns what a file holds now.
func (v *Vault) ReadContent(filePath string) (string, bool) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	content, err := v.files.read(filePath)
	return string(content), err == nil
}

func (v *Vault) check(ctx context.Context, filePath string) error {
	if err := v.checkContext(ctx); err != nil {
		return err
	}
	if err := v.reach(); err != nil {
		return err
	}
	if filePath == "" {
		return fmt.Errorf("%w: it is empty", host.ErrOutside)
	}
	if strings.HasPrefix(filePath, "/") || path.Clean(filePath) != filePath || strings.HasPrefix(filePath, "..") {
		return host.ErrOutside
	}
	return v.rejected[filePath]
}

func (v *Vault) StatFile(ctx context.Context, filePath string) (domain.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return domain.FileStat{}, err
	}
	at, err := v.files.stat(filePath)
	if err == nil && isIgnored(filePath) {
		return domain.FileStat{}, host.ErrNotANote
	}
	if err != nil {
		return domain.FileStat{}, err
	}
	return v.hashStat(at)
}

func (v *Vault) Read(ctx context.Context, filePath string) ([]byte, domain.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return nil, domain.FileStat{}, err
	}
	version := v.writes
	fp, err := v.files.stat(filePath)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	content, err := v.files.read(filePath)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	fp.Hash, fp.Version = domain.HashContent(content), version
	return content, fp, nil
}

func (v *Vault) Write(ctx context.Context, filePath string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return domain.FileStat{}, err
	}
	if failErr := v.unwritable[filePath]; failErr != nil {
		return domain.FileStat{}, failErr
	}
	now, err := v.files.stat(filePath)
	if err != nil {
		return domain.FileStat{}, err
	}
	stale, err := v.isStale(now, baseline)
	if err != nil {
		return domain.FileStat{}, err
	}
	if stale {
		return domain.FileStat{}, host.ErrStale
	}
	written, err := v.put(filePath, content)
	if err == nil {
		v.rescan(filePath)
	}
	return written, err
}

func (v *Vault) Create(ctx context.Context, filePath string, content []byte) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return err
	}
	if failErr := v.unwritable[filePath]; failErr != nil {
		return failErr
	}
	if _, err := v.files.create(filePath, content); err != nil {
		return err
	}
	v.rescan(filePath)
	return nil
}

func (v *Vault) Lock(ctx context.Context) (func() error, error) {
	select {
	case v.locked <- struct{}{}:
		return func() error {
			<-v.locked
			return nil
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (v *Vault) ReadCache(ctx context.Context, name string) ([]byte, bool, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return nil, false, err
	}
	if err := v.reach(); err != nil {
		return nil, false, err
	}
	content, ok := v.cache[name]
	return append([]byte(nil), content...), ok, nil
}

func (v *Vault) WriteCache(ctx context.Context, name string, content []byte) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return err
	}
	if err := v.reach(); err != nil {
		return err
	}
	v.cache[name] = append([]byte(nil), content...)
	return nil
}

func (v *Vault) WatchIndexChanges(ctx context.Context) (<-chan host.Move, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if v.unknown {
		return nil, host.ErrUnknownVault
	}
	in := make(chan struct{}, 16)
	out := make(chan host.Move)
	v.moves = append(v.moves, in)
	go func() {
		defer close(out)
		for {
			select {
			case <-in:
				select {
				case out <- host.Move{Err: nil}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func (v *Vault) report(err error) {
	if v.tb != nil {
		v.tb.Helper()
		v.tb.Error(err)
		return
	}
	if _, writeErr := fmt.Fprintln(os.Stderr, "hosttest:", err); writeErr != nil {
		return
	}
}

func (v *Vault) hashStat(at domain.FileStat) (domain.FileStat, error) {
	version := v.writes
	content, err := v.files.read(at.Path)
	if err != nil {
		return domain.FileStat{}, err
	}
	at.Hash, at.Version = domain.HashContent(content), version
	return at, nil
}

func (v *Vault) isStale(now, baseline domain.FileStat) (bool, error) {
	if baseline.IsZero() {
		return false, nil
	}
	content, err := v.files.read(now.Path)
	if err != nil {
		return false, err
	}
	return domain.HashContent(content) != baseline.Hash, nil
}

func (v *Vault) put(filePath string, content []byte) (domain.FileStat, error) {
	written, err := v.files.write(filePath, content)
	if err != nil {
		return domain.FileStat{}, err
	}
	v.writes++
	written.Hash, written.Version = domain.HashContent(content), v.writes
	return written, nil
}

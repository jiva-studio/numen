package derivedfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/atomicfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
)

// DerivedStore stores the application's own files inside one vault: files it
// generated and cannot regenerate.
// It is separate from VaultWriter, which rejects the application's own folder,
// and rejects every path VaultWriter writes.
// Every name begins with an area name; others are rejected, including the vault's identity.
type DerivedStore struct {
	vault   string
	service string
	id      string
	root    string
	areas   []string
	last    atomic.Pointer[fileInfo]
	where   atomic.Pointer[places]
}

// places is the store's folder as it is on this machine, and its areas as the
// folders directly inside it.
type places struct {
	root  string
	areas map[string]string
}

func (p *places) isInside(resolved, area string) bool {
	linked, named := p.areas[area]
	return named && vaultconfig.IsUnderRoot(resolved, linked)
}

func (d *DerivedStore) isCurrent(p *places) bool {
	found, err := os.Lstat(p.root)
	if err != nil {
		return false
	}
	now, err := os.Stat(d.root)
	return err == nil && os.SameFile(found, now)
}

func (d *DerivedStore) locate() (*places, error) {
	root, err := vaultconfig.GetDeepestExisting(d.root)
	if err != nil {
		return nil, err
	}
	found := &places{root: root, areas: make(map[string]string, len(d.areas))}
	for _, area := range d.areas {
		found.areas[area] = filepath.Join(root, area)
	}
	return found, nil
}

// fileInfo is the recorded state of a file, compared to detect changes to the
// file.
type fileInfo struct{ stat os.FileInfo }

func (s *fileInfo) isSameFile(now os.FileInfo) bool {
	return s != nil && os.SameFile(s.stat, now) &&
		s.stat.Size() == now.Size() && s.stat.ModTime().Equal(now.ModTime())
}

// ErrNotThisVault is returned for a name when the vault folder has a different
// identity from the one the store was opened with.
var ErrNotThisVault = errors.New("the folder is not the vault this store was opened on")

// OpenDerived opens one vault's store and holds on to the identity that vault carries.
// Nothing is written: a store that created its folder on opening would put one
// in every vault the application looks at.
// The store answers for every area named; the first is its name, so at least one is required.
func OpenDerived(vaultRoot string, opts vaultconfig.Options, areas ...string) (*DerivedStore, error) {
	validAreas := make([]string, 0, len(areas))
	for _, area := range areas {
		if area == "" {
			continue
		}
		if strings.ContainsAny(area, `/\`) || area == "." || area == ".." {
			return nil, fmt.Errorf("%q is not one folder", area)
		}
		validAreas = append(validAreas, area)
	}
	if len(validAreas) == 0 {
		return nil, errors.New("a store names at least one area")
	}
	abs, err := filepath.Abs(vaultRoot)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, err
	}
	id, was, err := readIdentity(abs, opts.GetServiceDir())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", abs, err)
	}
	d := &DerivedStore{
		vault:   abs,
		service: opts.GetServiceDir(),
		id:      id,
		root:    filepath.Join(abs, opts.GetServiceDir()),
		areas:   validAreas,
	}
	d.last.Store(was)
	if found, err := d.locate(); err == nil {
		d.where.Store(found)
	}
	return d, nil
}

// Area is the folder inside the service folder this store is called by, which
// is the first part of every name the index records.
func (d *DerivedStore) Area() string { return d.areas[0] }

func (d *DerivedStore) Read(_ context.Context, name string) (_ []byte, err error) {
	target, err := d.getPath(name)
	if err != nil {
		return nil, err
	}
	root, pos, err := d.openRoot(target)
	if err != nil {
		return nil, err
	}
	defer closeAndJoin(&err, root)
	raw, err := root.ReadFile(pos)
	if err != nil {
		return nil, wrapLockedByAnother(err)
	}
	return raw, nil
}

// Open is one file of the store to read a part of, and how many bytes it holds.
// A copy of a video is played from the middle, and holding an hour of it in
// memory to answer for a second is what this is not.
func (d *DerivedStore) Open(_ context.Context, name string) (_ io.ReadSeekCloser, _ int64, err error) {
	target, err := d.getPath(name)
	if err != nil {
		return nil, 0, err
	}
	root, pos, err := d.openRoot(target)
	if err != nil {
		return nil, 0, err
	}
	defer closeAndJoin(&err, root)
	file, err := root.Open(pos)
	if err != nil {
		return nil, 0, err
	}
	about, err := file.Stat()
	if err != nil {
		return nil, 0, errors.Join(err, file.Close())
	}
	return file, about.Size(), nil
}

// WriteStream puts what a reader gives under a name, through the same rename every
// other write here lands by: what a fetch was still writing when a machine
// stopped is not a file anything reads afterwards.
func (d *DerivedStore) WriteStream(ctx context.Context, name string, from io.Reader) (_ int64, err error) {
	target, err := d.getPath(name)
	if err != nil {
		return 0, err
	}
	root, pos, err := d.createRoot(target)
	if err != nil {
		return 0, err
	}
	defer closeAndJoin(&err, root)
	if err := root.MkdirAll(filepath.Dir(pos), 0o755); err != nil {
		return 0, err
	}
	content, err := atomicfile.Replace(ctx, root, pos, from, 0o644)
	if err != nil {
		return 0, err
	}
	return content.Size, atomicfile.SyncDirectory(root, filepath.Dir(pos))
}

func (d *DerivedStore) Write(ctx context.Context, name string, content []byte) (err error) {
	target, err := d.getPath(name)
	if err != nil {
		return err
	}
	root, pos, err := d.createRoot(target)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, root)
	if err := root.MkdirAll(filepath.Dir(pos), 0o755); err != nil {
		return err
	}
	if _, err := atomicfile.Replace(ctx, root, pos, bytes.NewReader(content), 0o644); err != nil {
		return err
	}
	return atomicfile.SyncDirectory(root, filepath.Dir(pos))
}

// Append adds to the end of the file in place, completely or not at all: a
// partial write is truncated back to the original length.
// A name that several callers append to is guarded by a claim, because the
// truncation removes everything written after this append's own bytes.
// A crash can still leave a torn tail; readers keep the complete pages.
func (d *DerivedStore) Append(_ context.Context, name string, content []byte) (err error) {
	target, err := d.getPath(name)
	if err != nil {
		return err
	}
	root, pos, err := d.createRoot(target)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, root)
	if err := root.MkdirAll(filepath.Dir(pos), 0o755); err != nil {
		return err
	}
	file, err := root.OpenFile(pos, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if n, err := file.Write(content); err != nil || n != len(content) {
		return errors.Join(newShortAppendError(name, n, len(content), err), closeAndTruncate(root, file, pos, n))
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	return file.Close()
}

func newShortAppendError(name string, output, wanted int, why error) error {
	if why == nil {
		why = io.ErrShortWrite
	}
	return fmt.Errorf("%s: %d of %d bytes: %w", name, output, wanted, why)
}

func closeAndTruncate(root *os.Root, file *os.File, name string, wrote int) error {
	if wrote <= 0 {
		return file.Close()
	}
	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	return truncateFile(root, name, pos-int64(wrote))
}

func truncateFile(root *os.Root, name string, to int64) error {
	file, err := root.OpenFile(name, os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := file.Truncate(to); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	return file.Close()
}

// claimSuffix names the file a claim on a name is held on. It outlives the
// name: a claim is held on an open file, and unlinking one lets a second caller
// make another at the same path and hold it too.
const claimSuffix = ".claim"

// ClaimName holds a name until the returned function is called, and fails
// with port.ErrClaimed when another caller holds it.
// The claim is a kernel lock on `<name>.claim` next to the artifact, so it
// applies across processes and is released when a holder is killed. A claim
// file with no lock on it does not hold the name.
func (d *DerivedStore) ClaimName(_ context.Context, name string) (func() error, error) {
	target, err := d.getPath(name + claimSuffix)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, err
	}
	return acquireFileLock(target)
}

// List returns the files directly under a name, as names of this store,
// sorted. Folders are omitted because the store contains only files.
func (d *DerivedStore) List(_ context.Context, name string) ([]Listed, error) {
	target, err := d.getPath(name)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(target)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	clean, err := vaultconfig.CleanPath(name)
	if err != nil {
		return nil, err
	}
	out := make([]Listed, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// A file removed between listing and stat, for example by another
		// machine's synchronizer, is omitted.
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Listed{Name: clean + "/" + e.Name(), Size: int(info.Size())})
	}
	slices.SortFunc(out, func(a, b Listed) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// Remove deletes a name from the store, together with the file that holds a
// claim on it, so no claim file remains.
func (d *DerivedStore) Remove(_ context.Context, name string) error {
	for _, one := range []string{name, name + claimSuffix} {
		target, err := d.getPath(one)
		if err != nil {
			return err
		}
		root, pos, err := d.openRoot(target)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		err = root.Remove(pos)
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		if err := errors.Join(err, root.Close()); err != nil {
			return err
		}
	}
	return nil
}

// RemoveAll takes a name out of the store with everything under it. A name
// that is not there is removed.
func (d *DerivedStore) RemoveAll(_ context.Context, name string) (err error) {
	target, err := d.getPath(name)
	if err != nil {
		return err
	}
	root, pos, err := d.openRoot(target)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, root)
	return root.RemoveAll(pos)
}

func (d *DerivedStore) checkVaultIdentity() error {
	if now, err := os.Stat(vaultconfig.ConfigAt(d.vault, d.service)); err == nil && d.last.Load().isSameFile(now) {
		return nil
	}
	id, was, err := readIdentity(d.vault, d.service)
	if err != nil {
		return fmt.Errorf("%s: %w", d.vault, err)
	}
	if id != d.id {
		return fmt.Errorf("%s has %q and not %q: %w", d.vault, id, d.id, ErrNotThisVault)
	}
	// A folder without an identity has none after it is removed, so the
	// folder's existence is checked directly.
	if id == "" {
		if _, err := os.Stat(d.vault); err != nil {
			return fmt.Errorf("%s: %w", d.vault, err)
		}
	}
	d.last.Store(was)
	// The identity file differs from the last one read, so the folder locations
	// are resolved again.
	if found, err := d.locate(); err == nil {
		d.where.Store(found)
	}
	return nil
}

func readIdentity(root, serviceDir string) (string, *fileInfo, error) {
	if serviceDir == "" {
		serviceDir = vaultconfig.DefaultServiceDir
	}
	var was *fileInfo
	if info, err := os.Stat(vaultconfig.ConfigAt(root, serviceDir)); err == nil {
		was = &fileInfo{stat: info}
	}
	cfg, err := vaultconfig.ReadConfig(root, serviceDir)
	if errors.Is(err, vaultconfig.ErrNotAVault) || errors.Is(err, fs.ErrNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	return cfg.ID, was, nil
}

func (d *DerivedStore) openRoot(target string) (*os.Root, string, error) {
	name, err := filepath.Rel(d.root, target)
	if err != nil || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("%s: %w", target, vault.ErrOutside)
	}
	root, err := os.OpenRoot(d.root)
	if err != nil {
		return nil, "", err
	}
	return root, name, nil
}

func (d *DerivedStore) createRoot(target string) (*os.Root, string, error) {
	if err := os.MkdirAll(d.root, 0o755); err != nil {
		return nil, "", err
	}
	return d.openRoot(target)
}

func (d *DerivedStore) findArea(clean string) (string, bool) {
	for _, area := range d.areas {
		if clean == area || strings.HasPrefix(clean, area+"/") {
			return area, true
		}
	}
	return "", false
}

func (d *DerivedStore) getPath(name string) (string, error) {
	clean, err := vaultconfig.CleanPath(name)
	if err != nil {
		return "", err
	}
	if err := d.checkVaultIdentity(); err != nil {
		return "", err
	}
	// A name belongs to one area, and the store handles only its own areas.
	area, targetVal := d.findArea(clean)
	if !targetVal {
		return "", fmt.Errorf("%s is not in the %s store: %w", name, d.Area(), vault.ErrOutside)
	}
	target := filepath.Join(d.root, filepath.FromSlash(clean))

	resolved, err := vaultconfig.GetDeepestExisting(target)
	if err != nil {
		return "", err
	}
	enclosed, err := d.isEnclosedBy(resolved, area)
	if err != nil {
		return "", err
	}
	if !enclosed {
		return "", fmt.Errorf("%s: %w", name, vault.ErrOutside)
	}
	return target, nil
}

func (d *DerivedStore) isEnclosedBy(resolved, area string) (bool, error) {
	if where := d.where.Load(); where != nil && where.isInside(resolved, area) && d.isCurrent(where) {
		return true, nil
	}
	where, err := d.locate()
	if err != nil {
		return false, err
	}
	d.where.Store(where)
	return where.isInside(resolved, area), nil
}

// Listed is a file a folder of the store holds: its name in the store, and how
// many bytes are under it.
type Listed struct {
	Name string
	Size int
}

// Rename moves what is under one name to another, replacing what the other held. Both names are
// in the store, and the file lands under the new name whole.
func (d *DerivedStore) Rename(_ context.Context, from, to string) (err error) {
	source, err := d.getPath(from)
	if err != nil {
		return err
	}
	target, err := d.getPath(to)
	if err != nil {
		return err
	}
	root, oldPos, err := d.openRoot(source)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, root)
	newPos, err := filepath.Rel(d.root, target)
	if err != nil {
		return err
	}
	if err := root.Rename(oldPos, newPos); err != nil {
		return wrapLockedByAnother(err)
	}
	return atomicfile.SyncDirectory(root, filepath.Dir(newPos))
}

// IndexClaim is the hold on a vault's vectors: a claim on one name of the store.
type IndexClaim struct {
	store *DerivedStore
	name  string
}

// NewIndexClaim returns a claim on a named entry in this store.
func (d *DerivedStore) NewIndexClaim(name string) IndexClaim {
	return IndexClaim{store: d, name: name}
}

// ClaimIndex acquires the file lock for the configured name.
func (c IndexClaim) ClaimIndex(ctx context.Context) (func() error, error) {
	return c.store.ClaimName(ctx, c.name)
}

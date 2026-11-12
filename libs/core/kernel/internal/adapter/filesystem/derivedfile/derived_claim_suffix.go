package derivedfile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/atomicfile"
)

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
func (d *DerivedStore) Rename(ctx context.Context, from, to string) (err error) {
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
	if err := atomicfile.Rename(ctx, root, oldPos, newPos); err != nil {
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

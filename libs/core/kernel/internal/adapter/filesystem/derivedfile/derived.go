package derivedfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/atomicfile"
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

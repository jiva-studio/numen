package vaultreader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// VaultReader reads one vault from disk.
type VaultReader struct {
	root    string
	opts    vaultconfig.Options
	ignored *vaultconfig.Ignoring
}

// Open prepares a vault for reading. It does not write anything: looking at a
// folder and adding a vault are different acts, and only the second may touch
// the user's files.
func Open(root string, opts vaultconfig.Options) (*VaultReader, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// The vault is where the links lead. The paths the operating system reports
	// changes at are resolved, and they are named against this.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	// The vault's own configuration decides what is ignored. A folder that has
	// never been added has no configuration and uses the defaults.
	if cfg, err := vaultconfig.ReadConfig(abs, opts.ServiceDir); err == nil && len(cfg.Ignore) > 0 {
		opts.Ignore = cfg.Ignore
	}
	return &VaultReader{root: abs, opts: opts, ignored: opts.CompileIgnoring()}, nil
}

func (s *VaultReader) Root() string { return s.root }

// WalkFiles reports every source in the vault with its kind. A file of no kind
// the application reads is not reported.
// What the vault's ignore rules name is skipped, and so is the service folder,
// named separately because its name is a setting and a name without a leading
// dot must still be skipped.
func (s *VaultReader) WalkFiles(ctx context.Context, fn func(domain.FileStat) error) error {
	return filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			// The vault is edited while it is walked. A file or folder that has
			// gone between reading the directory and reaching it is not an
			// error: the next scan sees whatever it became.
			return nil
		}
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if p != s.root && s.isSkipped(s.getRelPath(p), d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		stat, err := s.getWalkedFileStat(p, d)
		if err != nil || stat == nil {
			return err
		}
		return fn(*stat)
	})
}

func (s *VaultReader) getWalkedFileStat(p string, d fs.DirEntry) (*domain.FileStat, error) {
	kind := s.opts.Kind(d.Name())
	rel := s.getRelPath(p)
	if s.ignored.MatchesPath(rel) {
		return nil, nil
	}
	found, err := getEntryInfo(p, d)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !found.Mode().IsRegular() {
		return nil, nil
	}
	return &domain.FileStat{
		Path:    rel,
		Kind:    kind,
		Size:    found.Size(),
		ModTime: found.ModTime(),
	}, nil
}

func (s *VaultReader) getRelPath(p string) string {
	if len(p) <= len(s.root) {
		return ""
	}
	if strings.HasSuffix(s.root, string(filepath.Separator)) {
		return filepath.ToSlash(p[len(s.root):])
	}
	return filepath.ToSlash(p[len(s.root)+1:])
}

// List reports what one folder holds, without descending. Every file and
// folder is reported, whether anything reads it or not; ignored names and the
// service folder are left out.
// Folders come first, then files, each by name compared without regard to case.
// A path holding a file is answered as a folder that is not there.
func (s *VaultReader) List(ctx context.Context, folder string) ([]domain.Entry, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	target := s.root
	if folder != "" {
		var err error
		if target, err = vaultconfig.GetContainedPath(s.root, folder, s.opts.GetServiceDir()); err != nil {
			return nil, err
		}
		folder = pathpkg.Clean(filepath.ToSlash(folder))
	}
	switch info, err := os.Stat(target); {
	case err != nil:
		return nil, err
	case !info.IsDir():
		return nil, fmt.Errorf("list %s: %w", folder, fs.ErrNotExist)
	}
	read, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}

	entries := make([]domain.Entry, 0, len(read))
	for _, d := range read {
		if entry, items := s.getEntry(target, folder, d); items {
			entries = append(entries, entry)
		}
	}
	slices.SortFunc(entries, inOrder)
	return entries, nil
}

func (s *VaultReader) getEntry(target, folder string, d fs.DirEntry) (domain.Entry, bool) {
	rel := pathpkg.Join(folder, d.Name())
	entry := domain.Entry{Path: rel, Name: d.Name(), IsFolder: d.IsDir()}
	if d.IsDir() {
		if found, err := d.Info(); err == nil {
			entry.ModTime = found.ModTime()
		}
		return entry, !s.opts.IsService(d.Name()) && !s.ignored.MatchesPath(rel+"/")
	}
	if s.ignored.MatchesPath(rel) {
		return domain.Entry{}, false
	}
	if found, err := getEntryInfo(filepath.Join(target, d.Name()), d); err == nil {
		entry.ModTime = found.ModTime()
		if found.Mode().IsRegular() {
			entry.Kind = s.opts.Kind(d.Name())
		}
	}
	return entry, true
}

func getEntryInfo(path string, d fs.DirEntry) (fs.FileInfo, error) {
	if d.Type()&fs.ModeSymlink == 0 {
		return d.Info()
	}
	found, err := os.Stat(path)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	return found, nil
}

func inOrder(a, b domain.Entry) int {
	if a.IsFolder != b.IsFolder {
		if a.IsFolder {
			return -1
		}
		return 1
	}
	if delta := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); delta != 0 {
		return delta
	}
	return strings.Compare(a.Name, b.Name)
}

func (s *VaultReader) Read(ctx context.Context, path string) (_ []byte, err error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	target, err := vaultconfig.GetContainedPath(s.root, path, s.opts.GetServiceDir())
	if err != nil {
		return nil, err
	}
	f, err := openReadableFile(target)
	if err != nil {
		return nil, err
	}
	defer closeAndJoin(&err, f)

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if err := checkFileReadable(path, info); err != nil {
		return nil, err
	}
	if err := s.checkSize(path, info); err != nil {
		return nil, err
	}
	return readAll(f, info.Size())
}

func readAll(r io.Reader, size int64) ([]byte, error) {
	var capSize int
	if int64(int(size)) == size && size > 0 {
		capSize = int(size) + 1
	}
	if capSize < 512 {
		capSize = 512
	}
	data := make([]byte, 0, capSize)
	for {
		if len(data) >= cap(data) {
			data = append(data[:cap(data)], 0)
		}
		n, err := r.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err != nil {
			if errors.Is(err, io.EOF) {
				return data, nil
			}
			return nil, err
		}
	}
}

func (s *VaultReader) checkSize(path string, info fs.FileInfo) error {
	linked := s.opts.GetMaxNoteBytes()
	if !s.opts.IsNote(pathpkg.Base(path)) || info.Size() <= linked {
		return nil
	}
	return fmt.Errorf("%s is %d bytes, and %d is the most a note is read at: %w",
		path, info.Size(), linked, edit.ErrNotANote)
}

// Open is one file, to read a part of. The path is checked by the same rule
// Read checks it by, and what is returned is the file itself.
func (s *VaultReader) Open(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return s.openWith(ctx, path, openReadableFile)
}

func (s *VaultReader) openWith(
	ctx context.Context,
	path string,
	open func(string) (*os.File, error),
) (io.ReadSeekCloser, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	target, err := vaultconfig.GetContainedPath(s.root, path, s.opts.GetServiceDir())
	if err != nil {
		return nil, err
	}
	f, err := open(target)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	if err := checkFileReadable(path, info); err != nil {
		return nil, errors.Join(err, f.Close())
	}
	return f, nil
}

func checkFileReadable(path string, info fs.FileInfo) error {
	if info.Mode().IsRegular() || info.IsDir() {
		return nil
	}
	return fmt.Errorf("%s: %w", path, edit.ErrNotANote)
}

// StatFile answers the same question about one path that Walk answers about all of
// them, and by the same rules: a path the vault ignores holds no note. Where
// something is there all the same, the answer names it as a file the vault
// leaves alone.
func (s *VaultReader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	if ctx.Err() != nil {
		return domain.FileStat{}, ctx.Err()
	}
	kind, targetVal := s.getKind(path)
	if !targetVal {
		return domain.FileStat{}, s.getNotANoteError(path)
	}
	target, err := vaultconfig.GetContainedPath(s.root, path, s.opts.GetServiceDir())
	if err != nil {
		return domain.FileStat{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return domain.FileStat{}, err
	}
	if !info.Mode().IsRegular() {
		return domain.FileStat{}, s.getNotANoteError(path)
	}
	return domain.FileStat{
		Path:    path,
		Kind:    kind,
		Size:    info.Size(),
		ModTime: info.ModTime(),
		FileID:  getFileID(info),
	}, nil
}

func (s *VaultReader) getNotANoteError(path string) error {
	target, err := vaultconfig.GetContainedPath(s.root, path, s.opts.GetServiceDir())
	if err != nil {
		return fs.ErrNotExist
	}
	if _, err := os.Lstat(target); err != nil {
		return fs.ErrNotExist
	}
	return fmt.Errorf("%s: %w", path, edit.ErrNotANote)
}

func (s *VaultReader) isSkipped(path, name string) bool {
	return s.opts.IsService(name) || s.ignored.MatchesPath(path+"/")
}

func (s *VaultReader) relative(absolute string) (path string, inside bool) {
	rel, err := filepath.Rel(s.root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func (s *VaultReader) getKind(path string) (domain.SourceKind, bool) {
	if _, err := vaultconfig.GetContainedPath(s.root, path, s.opts.GetServiceDir()); err != nil {
		return "", false
	}
	kind := s.opts.Kind(pathpkg.Base(path))
	if s.ignored.MatchesPath(path) {
		return "", false
	}
	for dir := pathpkg.Dir(path); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if s.opts.IsService(pathpkg.Base(dir)) || s.ignored.MatchesPath(dir+"/") {
			return "", false
		}
	}
	return kind, true
}

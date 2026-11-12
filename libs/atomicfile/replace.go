package atomicfile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// nameMax is how many bytes one component of a path may be. It is 255 on every
// filesystem the library is used on.
const nameMax = 255

// Stat is the size and modification time of the file that Replace wrote.
type Stat struct {
	Size    int64
	ModTime time.Time
}

// WriteFile writes the content of from to the file at path, whole or not at all: a reader of the
// path sees the old file or the new one, and a failed read of from or a failed write leaves the old
// file and no temporary file. The folder of the path must exist.
//
// The new file has the mode, whatever the old one had. A from that returns an error before its
// end, as one does that checks what it read, stops the write. A rename that fails because another
// process holds the target is not tried again; Replace waits that out.
func WriteFile(path string, from io.Reader, mode fs.FileMode) (err error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	_, err = replace(root, filepath.Base(path), from, mode, root.Rename)
	return err
}

// Replace writes content beside the target and renames it over the top.
// The temporary file is named with a leading dot. The contents are flushed
// before the rename, so a power loss leaves the old file or the new one. The
// stat comes from the temporary file's descriptor; the rename carries the file
// across whole, so size and modification time are the target's.
func Replace(ctx context.Context, root *os.Root, target string, from io.Reader, mode fs.FileMode) (Stat, error) {
	return replace(root, target, from, mode, func(from, to string) error { return Rename(ctx, root, from, to) })
}

func replace(root *os.Root, target string, from io.Reader, mode fs.FileMode, rename func(from, to string) error) (stat Stat, err error) {
	dir := filepath.Dir(target)
	tmp, pos, err := createTempFile(root, dir, getTempPattern(filepath.Base(target)))
	if err != nil {
		return Stat{}, err
	}
	defer func() {
		if removeErr := root.Remove(pos); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()

	if _, err := io.Copy(tmp, from); err != nil {
		return Stat{}, errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return Stat{}, errors.Join(err, tmp.Close())
	}
	info, err := tmp.Stat()
	if err != nil {
		return Stat{}, errors.Join(err, tmp.Close())
	}
	// Changing the mode does not change the modification time. It is applied to the
	// open file, which is this one.
	if err := tmp.Chmod(mode); err != nil {
		return Stat{}, errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return Stat{}, err
	}
	if err := rename(pos, target); err != nil {
		return Stat{}, err
	}
	stat = Stat{Size: info.Size(), ModTime: info.ModTime()}
	return stat, SyncDirectory(root, dir)
}

// getTempPattern returns the pattern a temporary file next to a target is
// created under. The name is cut on a rune boundary, leaving space for the
// leading dot and for the digits that replace the star; the rename uses the full
// name.
func getTempPattern(name string) string {
	const room = len(".") + len(".") + 10
	for len(name)+room > nameMax {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return "." + name + ".*"
}

// createTempFile creates a file next to a target, under the root's own handle.
// It is os.CreateTemp for a root: the name is unique to the creating call and
// is claimed by creating the file, not by checking for it first.
func createTempFile(root *os.Root, dir, pattern string) (*os.File, string, error) {
	prefix, suffix, _ := strings.Cut(pattern, "*")
	for range 10_000 {
		name := filepath.Join(dir, prefix+strconv.FormatUint(uint64(rand.Uint32()), 10)+suffix)
		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		return file, name, nil
	}
	return nil, "", fmt.Errorf("%s: no free name beside it", dir)
}

// SyncDirectory flushes the folder the rename was recorded in.
// Flushing the file keeps its contents whole; flushing the folder keeps the rename.
// A folder that cannot be opened or flushed on this platform or filesystem is
// not an error: the file is written either way.
func SyncDirectory(root *os.Root, dir string) (err error) {
	if runtime.GOOS == "windows" {
		return nil
	}
	folder, err := root.Open(dir)
	if err != nil {
		return nil
	}
	defer func() { err = errors.Join(err, folder.Close()) }()
	if err := folder.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, errors.ErrUnsupported) {
		return err
	}
	return nil
}

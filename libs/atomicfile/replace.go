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
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// nameMax is how many bytes one component of a path may be. It is 255 on every
// filesystem the library is used on.
const (
	nameMax          = 255
	tempFileAttempts = 10_000
)

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
func WriteFile(path string, from io.Reader, mode fs.FileMode) error {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	_, err = replace(root, filepath.Base(path), from, mode, root.Rename, (*os.File).Sync)
	return errors.Join(err, root.Close())
}

// Replace writes content beside the target and renames it over the top.
// The temporary file is named with a leading dot. The contents are flushed
// before the rename, so a power loss leaves the old file or the new one. The
// stat comes from the temporary file's descriptor; the rename carries the file
// across whole, so size and modification time are the target's.
// The temporary name is the target name cut on a rune boundary to fit the limit of a path component,
// so a target whose name fills the limit can still be written.
func Replace(ctx context.Context, root *os.Root, target string, from io.Reader, mode fs.FileMode) (Stat, error) {
	return replace(
		root,
		target,
		from,
		mode,
		func(from, to string) error { return Rename(ctx, root, from, to) },
		(*os.File).Sync,
	)
}

func replace(
	root *os.Root,
	target string,
	from io.Reader,
	mode fs.FileMode,
	rename func(from, to string) error,
	sync func(*os.File) error,
) (stat Stat, err error) {
	dir := filepath.Dir(target)
	tmp, pos, err := createTempFile(root, dir, getTempPattern(filepath.Base(target)), getRandomSuffix)
	if err != nil {
		return Stat{}, err
	}
	defer func() {
		if removeErr := root.Remove(pos); !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()

	if _, err := io.Copy(tmp, from); err != nil {
		return Stat{}, errors.Join(err, tmp.Close())
	}
	if err := sync(tmp); err != nil {
		return Stat{}, errors.Join(err, tmp.Close())
	}
	info, statErr := tmp.Stat()
	// Changing the mode does not change the modification time. It is applied to the
	// open file, which is this one.
	if err := errors.Join(statErr, tmp.Chmod(mode), tmp.Close()); err != nil {
		return Stat{}, err
	}
	if err := rename(pos, target); err != nil {
		return Stat{}, err
	}
	return Stat{Size: info.Size(), ModTime: info.ModTime()}, syncDirectory(root, dir, runtime.GOOS, sync)
}

func getTempPattern(name string) string {
	const room = len(".") + len(".") + 10
	for len(name)+room > nameMax {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return "." + name + ".*"
}

func createTempFile(root *os.Root, dir, pattern string, getSuffix func() string) (*os.File, string, error) {
	prefix, suffix, _ := strings.Cut(pattern, "*")
	for range tempFileAttempts {
		name := filepath.Join(dir, prefix+getSuffix()+suffix)
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

func getRandomSuffix() string {
	return fmt.Sprint(rand.Uint32())
}

// SyncDirectory flushes the folder the rename was recorded in.
// Flushing the file keeps its contents whole; flushing the folder keeps the rename.
// A folder that cannot be opened or flushed on this platform or filesystem is
// not an error: the file is written either way.
func SyncDirectory(root *os.Root, dir string) error {
	return syncDirectory(root, dir, runtime.GOOS, (*os.File).Sync)
}

func syncDirectory(root *os.Root, dir, goos string, sync func(*os.File) error) error {
	if goos == "windows" {
		return nil
	}
	folder, err := root.Open(dir)
	if err != nil {
		if errors.Is(err, os.ErrClosed) {
			return err
		}
		return nil
	}
	return errors.Join(ignoreSyncError(sync(folder)), folder.Close())
}

func ignoreSyncError(err error) error {
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, errors.ErrUnsupported) {
		return nil
	}
	return err
}

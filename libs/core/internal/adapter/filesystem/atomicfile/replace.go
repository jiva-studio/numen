// Putting a file where another one stands, whole or not at all: the temporary
// file it is written to first, the folders above it, and the flush that keeps
// the swap.

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
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// nameMax is how many bytes one component of a path may be. It is 255 on every
// filesystem a vault is kept on.
const nameMax = 255

// GetTempPattern is the pattern a temporary file next to a target is created
// under. The name is cut on a rune boundary, leaving room for the leading dot
// and for the digits that go where the star is; the rename lands on the full
// name.
func GetTempPattern(name string) string {
	const room = len(".") + len(".") + 10
	for len(name)+room > nameMax {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return "." + name + ".*"
}

// CreateParents puts the folders above a name there, and says a file standing where
// one of them would go in the words a caller acts on.
func CreateParents(root *os.Root, dir, path string) error {
	err := root.MkdirAll(dir, 0o755)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("make %s: %w", path, vaultport.ErrAlreadyExists)
	}
	return err
}

// CreateTempFile is a file created beside a target, under the vault's own handle.
// It is what os.CreateTemp is, for a root: a name nobody else holds, taken by
// creating it and not by looking first.
func CreateTempFile(root *os.Root, dir, pattern string) (*os.File, string, error) {
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

// Replace writes content beside the target and renames it over the top.
//
// The temporary file is named with a leading dot, and a name beginning with a
// dot is not a note. The contents are flushed before the rename, so a machine
// that loses power comes back to the old file or the new one.
//
// The file stat comes from the temporary file's own descriptor. The rename
// carries the file across whole, so its size and its modification time are the
// ones at the target from the moment the rename lands.
func Replace(ctx context.Context, root *os.Root, target string, from io.Reader, mode fs.FileMode) (content domain.FileStat, err error) {
	dir := filepath.Dir(target)
	tmp, pos, err := CreateTempFile(root, dir, GetTempPattern(filepath.Base(target)))
	if err != nil {
		return domain.FileStat{}, err
	}
	defer func() {
		if removeErr := root.Remove(pos); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()

	if _, err := io.Copy(tmp, from); err != nil {
		return domain.FileStat{}, errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return domain.FileStat{}, errors.Join(err, tmp.Close())
	}
	info, err := tmp.Stat()
	if err != nil {
		return domain.FileStat{}, errors.Join(err, tmp.Close())
	}
	// Changing the mode moves no modification time. It is asked of the open
	// file, which is this one and can be no other.
	if err := tmp.Chmod(mode); err != nil {
		return domain.FileStat{}, errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return domain.FileStat{}, err
	}
	if err := Rename(ctx, root, pos, target); err != nil {
		return domain.FileStat{}, err
	}
	content = domain.FileStat{Size: info.Size(), ModTime: info.ModTime()}
	return content, SyncDirectory(root, dir)
}

// SyncDirectory flushes the folder the rename was recorded in.
//
// Flushing the file is what keeps its contents whole; flushing the folder is
// what keeps the rename itself. A folder that cannot be opened or flushed on
// this platform or filesystem is not an error: the note is written either way.
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

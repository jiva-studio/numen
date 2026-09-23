// Putting a file where another one stands, whole or not at all: the temporary
// file it is written to first, the folders above it, and the flush that keeps
// the swap.

package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// nameMax is how many bytes one component of a path may be. It is 255 on every
// filesystem a vault is kept on.
const nameMax = 255

// getTempPattern is the pattern a temporary file next to a target is created
// under. The name is cut on a rune boundary, leaving room for the leading dot
// and for the digits that go where the star is; the rename lands on the full
// name.
func getTempPattern(name string) string {
	const room = len(".") + len(".") + 10
	for len(name)+room > nameMax {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return "." + name + ".*"
}

// parents puts the folders above a name there, and says a file standing where
// one of them would go in the words a caller acts on.
func parents(root *os.Root, dir, path string) error {
	err := root.MkdirAll(dir, 0o755)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("make %s: %w", path, port.ErrOccupied)
	}
	return err
}

// temporary is a file created beside a target, under the vault's own handle.
// It is what os.CreateTemp is, for a root: a name nobody else holds, taken by
// creating it and not by looking first.
func temporary(root *os.Root, dir, pattern string) (*os.File, string, error) {
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

// replace writes content beside the target and renames it over the top.
//
// The temporary file is named with a leading dot, and a name beginning with a
// dot is not a note. The contents are flushed before the rename, so a machine
// that loses power comes back to the old file or the new one.
//
// The fingerprint comes from the temporary file's own descriptor. The rename
// carries the file across whole, so its size and its modification time are the
// ones at the target from the moment the rename lands.
func replace(ctx context.Context, root *os.Root, target string, from io.Reader, mode fs.FileMode) (domain.Fingerprint, error) {
	dir := filepath.Dir(target)
	tmp, at, err := temporary(root, dir, getTempPattern(filepath.Base(target)))
	if err != nil {
		return domain.Fingerprint{}, err
	}
	defer root.Remove(at)

	if _, err := io.Copy(tmp, from); err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	info, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	// Changing the mode moves no modification time. It is asked of the open
	// file, which is this one and can be no other.
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	if err := tmp.Close(); err != nil {
		return domain.Fingerprint{}, err
	}
	if err := rename(ctx, root, at, target); err != nil {
		return domain.Fingerprint{}, err
	}
	written := domain.Fingerprint{Size: info.Size(), ModTime: info.ModTime()}
	return written, settle(root, dir)
}

// settle flushes the folder the rename was recorded in.
//
// Flushing the file is what keeps its contents whole; flushing the folder is
// what keeps the rename itself.
//
// Not every filesystem lets a directory be opened for this, and a refusal is
// not an error to hand back: the note is written either way.
func settle(root *os.Root, dir string) error {
	folder, err := root.Open(dir)
	if err != nil {
		return nil
	}
	defer folder.Close()
	_ = folder.Sync()
	return nil
}

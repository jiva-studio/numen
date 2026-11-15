// Package packagecache keeps downloaded plugin packages on disk under the SHA-256 of their content.
//
// The folder is shared by every application of the user, so the content of a file is hashed on each
// use. A package is written to a temporary file in the folder and renamed to its checksum, which
// leaves a reader the whole package or none of it, whichever application is writing.
package packagecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jiva-studio/numen/libs/atomicfile"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
)

var checksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Cache is a folder of packages named by checksum.
type Cache struct {
	dir string
}

var errNoDir = errors.New("the package cache names no folder")

// New returns the cache in the folder, which is created on the first Put.
func New(dir string) (*Cache, error) {
	if dir == "" {
		return nil, errNoDir
	}
	return &Cache{dir: dir}, nil
}

// DefaultDir is the folder every application shares: numen/packages in the user cache folder.
func DefaultDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find the user cache folder: %w", err)
	}
	return filepath.Join(base, "numen", "packages"), nil
}

func (c *Cache) pathOf(checksum string) (string, error) {
	if !checksumPattern.MatchString(checksum) {
		return "", fmt.Errorf("%w: %q", catalog.ErrInvalidChecksum, checksum)
	}
	return filepath.Join(c.dir, checksum), nil
}

// Put stores the content under the checksum after hashing it. Content whose hash differs is
// refused with catalog.ErrChecksumMismatch and leaves no file.
func (c *Cache) Put(ctx context.Context, checksum string, content io.Reader) error {
	destination, err := c.pathOf(checksum)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return fmt.Errorf("create the package cache folder: %w", err)
	}
	checked := &checkedReader{getContextError: ctx.Err, content: content, hash: sha256.New(), checksum: checksum}
	if err := atomicfile.WriteFile(destination, checked, 0o600); err != nil {
		if errors.Is(err, catalog.ErrChecksumMismatch) {
			return err
		}
		return fmt.Errorf("write the package: %w", err)
	}
	return nil
}

// Open returns the package after hashing it. A package that fails the check is removed.
func (c *Cache) Open(ctx context.Context, checksum string) (io.ReadCloser, error) {
	source, err := c.pathOf(checksum)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(source)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", catalog.ErrPackageNotCached, checksum)
	}
	if err != nil {
		return nil, fmt.Errorf("open the package: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, errors.Join(fmt.Errorf("read the package: %w", err), file.Close())
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != checksum {
		return nil, errors.Join(
			fmt.Errorf("%w: %s hashes to %s", catalog.ErrChecksumMismatch, checksum, got),
			file.Close(), removeIfPresent(source))
	}
	// gomutants:disable-next-line BRANCH_IF reason="seeking to start of open regular file does not fail"
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Join(fmt.Errorf("rewind the package: %w", err), file.Close())
	}
	return file, nil
}

func (c *Cache) Remove(ctx context.Context, checksum string) error {
	target, err := c.pathOf(checksum)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return removeIfPresent(target)
}

func removeIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// checkedReader reads the content under a context, hashes what it reads, and at the end of the
// content fails with ErrChecksumMismatch when the hash is not the checksum, so a write from it
// stops before the file is put in place.
type checkedReader struct {
	getContextError func() error
	content         io.Reader
	hash            hash.Hash
	checksum        string
}

func (r *checkedReader) Read(buffer []byte) (int, error) {
	if err := r.getContextError(); err != nil {
		return 0, err
	}
	count, err := r.content.Read(buffer)
	r.hash.Write(buffer[:count])
	if errors.Is(err, io.EOF) {
		if got := hex.EncodeToString(r.hash.Sum(nil)); got != r.checksum {
			return count, fmt.Errorf("%w: content hashes to %s, stored under %s", catalog.ErrChecksumMismatch, got, r.checksum)
		}
	}
	return count, err
}

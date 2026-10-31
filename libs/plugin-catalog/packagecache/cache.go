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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
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
func (c *Cache) Put(ctx context.Context, checksum string, content io.Reader) (err error) {
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
	temporary, err := os.CreateTemp(c.dir, ".incoming-*")
	if err != nil {
		return fmt.Errorf("create a package file: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, removeIfPresent(temporary.Name()))
		}
	}()
	hash := sha256.New()
	if err := copyContext(ctx, io.MultiWriter(temporary, hash), content); err != nil {
		return errors.Join(fmt.Errorf("write the package: %w", err), temporary.Close())
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != checksum {
		return errors.Join(fmt.Errorf("%w: content hashes to %s, stored under %s", catalog.ErrChecksumMismatch, got, checksum), temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(fmt.Errorf("flush the package: %w", err), temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close the package: %w", err)
	}
	if err := os.Rename(temporary.Name(), destination); err != nil {
		return fmt.Errorf("move the package into place: %w", err)
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
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Join(fmt.Errorf("rewind the package: %w", err), file.Close())
	}
	return file, nil
}

// Remove deletes the package.
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

// copyChunkSize is the number of bytes copied between two checks of the context.
const copyChunkSize = 64 * 1024

// copyContext copies the source to the writer and stops when the context ends.
func copyContext(ctx context.Context, destination io.Writer, source io.Reader) error {
	buffer := make([]byte, copyChunkSize)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			if _, writeErr := destination.Write(buffer[:count]); writeErr != nil {
				return writeErr
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

// maxCachedBooks is the most books a bookCache keeps open.
const maxCachedBooks = 4

// errCacheClosed is what a closed bookCache answers.
var errCacheClosed = errors.New("the book cache is closed")

// errReadFile marks a failure to open or stat the file of a book, as distinct from a
// file that is no book.
var errReadFile = errors.New("read file")

// A bookCache keeps open the most recently read books, each opened once while its
// file is unchanged.
type bookCache struct {
	// readFile reads the file at a path of the vault.
	readFile func(ctx context.Context, files host.Vault, path string) ([]byte, host.FileStat, error)

	mu       sync.Mutex
	books    []*cachedBook // least recently read first
	isClosed bool
}

func newBookCache() *bookCache {
	return &bookCache{readFile: func(ctx context.Context, files host.Vault, path string) ([]byte, host.FileStat, error) {
		return files.Read(ctx, path)
	}}
}

// A cachedBook is a book kept in the cache until it is evicted and has no readers.
type cachedBook struct {
	key     string
	size    int64
	mtime   int64
	archive *epub.Archive

	// readers is how many requests are reading, and evicted is whether the cache
	// dropped the book.
	readers int
	evicted bool
}

// acquire returns the cached book at path, opening it when the cache has none for the
// file as it is now. The caller returns it with release.
func (c *bookCache) acquire(ctx context.Context, files host.Vault, vaultID host.VaultID, path string) (*cachedBook, error) {
	info, err := files.StatFile(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errReadFile, err)
	}
	key := string(vaultID) + "\x00" + path

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isClosed {
		return nil, errCacheClosed
	}
	for i, book := range c.books {
		if book.key != key {
			continue
		}
		c.books = append(c.books[:i:i], c.books[i+1:]...)
		if book.size == info.Size && book.mtime == info.ModTime.UnixNano() {
			c.books = append(c.books, book)
			book.readers++
			return book, nil
		}
		c.evict(book)
		break
	}

	book, err := c.openBook(ctx, files, key, path)
	if err != nil {
		return nil, err
	}
	if len(c.books) >= maxCachedBooks {
		c.evict(c.books[0])
		c.books = c.books[1:]
	}
	c.books = append(c.books, book)
	book.readers++
	return book, nil
}

func (c *bookCache) openBook(ctx context.Context, files host.Vault, key, path string) (*cachedBook, error) {
	raw, info, err := c.readFile(ctx, files, path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errReadFile, err)
	}
	archive, err := epub.OpenArchive(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	return &cachedBook{
		key:     key,
		size:    info.Size,
		mtime:   info.ModTime.UnixNano(),
		archive: archive,
	}, nil
}

// evict drops a book from the cache. The caller has the lock.
func (c *bookCache) evict(book *cachedBook) {
	book.evicted = true
	if book.readers == 0 {
		book.archive = nil
	}
}

// release returns a book acquired.
func (c *bookCache) release(book *cachedBook) {
	c.mu.Lock()
	defer c.mu.Unlock()
	book.readers--
	if book.evicted && book.readers == 0 {
		book.archive = nil
	}
}

// Close evicts every book and rejects later requests. Requests still reading finish
// first.
func (c *bookCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isClosed = true
	for _, book := range c.books {
		c.evict(book)
	}
	c.books = nil
}

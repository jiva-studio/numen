package service

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub"
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
	// openFile opens the file at a path.
	openFile func(path string) (*os.File, error)

	mu       sync.Mutex
	books    []*cachedBook // least recently read first
	isClosed bool
}

func newBookCache() *bookCache {
	return &bookCache{openFile: openShared}
}

// A cachedBook is a book whose file stays open until it is evicted and has no readers.
type cachedBook struct {
	path    string
	size    int64
	mtime   int64
	file    *os.File
	archive *epub.Archive

	// readers is how many requests are reading, and evicted is whether the cache
	// dropped the book. The file closes when an evicted book has no readers.
	readers int
	evicted bool
}

// acquire returns the cached book at path, opening it when the cache has none for the
// file as it is now. The caller returns it with release.
func (c *bookCache) acquire(path string) (*cachedBook, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errReadFile, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isClosed {
		return nil, errCacheClosed
	}
	for i, book := range c.books {
		if book.path != path {
			continue
		}
		c.books = append(c.books[:i:i], c.books[i+1:]...)
		if book.size == info.Size() && book.mtime == info.ModTime().UnixNano() {
			c.books = append(c.books, book)
			book.readers++
			return book, nil
		}
		c.evict(book)
		break
	}

	book, err := c.openBook(path)
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

func (c *bookCache) openBook(path string) (*cachedBook, error) {
	file, err := c.openFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errReadFile, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("%w: %w", errReadFile, err)
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, fmt.Errorf("%w: %s is a directory", errReadFile, path)
	}
	archive, err := epub.OpenArchive(file, info.Size())
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &cachedBook{
		path:    path,
		size:    info.Size(),
		mtime:   info.ModTime().UnixNano(),
		file:    file,
		archive: archive,
	}, nil
}

// evict drops a book from the cache. The file of a book with no readers closes at once, and the
// file of a book with readers closes when the last one releases it. The caller has the lock.
func (c *bookCache) evict(book *cachedBook) {
	book.evicted = true
	if book.readers == 0 {
		_ = book.file.Close()
	}
}

// release returns a book acquired.
func (c *bookCache) release(book *cachedBook) {
	c.mu.Lock()
	defer c.mu.Unlock()
	book.readers--
	if book.evicted && book.readers == 0 {
		_ = book.file.Close()
	}
}

// Close evicts every book and refuses later requests. Requests still reading finish
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

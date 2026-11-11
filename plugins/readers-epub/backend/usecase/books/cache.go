package books

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub"
)

// maxCachedBooks is the most books a bookCache keeps open.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="maxCachedBooks is the capacity bound for open book archives"
const maxCachedBooks = 4

// errCacheClosed is what a closed bookCache answers.
var errCacheClosed = errors.New("the book cache is closed")

// errReadFile marks a failure to open or stat the file of a book, as distinct from a
// file that is no book.
var errReadFile = errors.New("read file")

// A bookCache keeps open the most recently read books, each opened once while its
// file is unchanged.
type bookCache struct {
	readFile func(ctx context.Context, files host.FileReader, path string) ([]byte, domain.FileStat, error)

	mu       sync.Mutex
	books    []*cachedBook
	isClosed bool
}

func newBookCache() *bookCache {
	return &bookCache{
		readFile: func(ctx context.Context, files host.FileReader, path string) ([]byte, domain.FileStat, error) {
			return files.Read(ctx, path)
		},
		mu:       sync.Mutex{},
		books:    nil,
		isClosed: false,
	}
}

// A cachedBook is a book kept in the cache until it is evicted and has no readers.
type cachedBook struct {
	key     string
	size    int64
	mtime   int64
	archive *epub.Archive

	readers int
	evicted bool
}

func (c *bookCache) acquire(ctx context.Context, files host.FileReader, vaultID domain.VaultID, path string) (*cachedBook, error) {
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
		// gomutants:disable-next-line INVERT_LOOP_CTRL reason="break exits the loop once the matching key has been removed and evicted"
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

func (c *bookCache) openBook(ctx context.Context, files host.FileReader, key, path string) (*cachedBook, error) {
	raw, info, err := c.readFile(ctx, files, path)
	if err != nil {
		// gomutants:disable-next-line ERRORF_WRAP reason="errReadFile is the primary wrapped sentinel error"
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
		readers: 0,
		evicted: false,
	}, nil
}

func (c *bookCache) evict(book *cachedBook) {
	book.evicted = true
	if book.readers == 0 {
		book.archive = nil
	}
}

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

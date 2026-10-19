package service

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/epub"
)

// mostOpenBooks is how many books a shelf holds open at once.
const mostOpenBooks = 4

// errShelfClosed is what a closed shelf answers.
var errShelfClosed = errors.New("the shelf is closed")

// errReadFile marks a failure to open or stat the file of a book, as apart from a
// file that is no book.
var errReadFile = errors.New("read file")

// A shelf holds the books that entries are being read from, each open once for as
// long as its file is unchanged and it is among the most recently read.
type shelf struct {
	// openFile opens the file at a path.
	openFile func(path string) (*os.File, error)

	mu     sync.Mutex
	open   []*shelvedBook // least recently read first
	closed bool
}

func newShelf() *shelf {
	return &shelf{openFile: os.Open}
}

// A shelvedBook is a book whose file stays open while it is on the shelf or read.
type shelvedBook struct {
	path    string
	size    int64
	mtime   int64
	file    *os.File
	archive *epub.Archive

	// readers is how many requests are reading, and retired is whether the shelf
	// has let the book go. The file is closed once a retired book has no readers.
	readers int
	retired bool
}

// fingerprint is the size and the stamp of the file the book was opened from.
func (b *shelvedBook) fingerprint() (size, mtime int64) {
	return b.size, b.mtime
}

// take returns the book at path, opened if the shelf holds none of the file as it
// is now. The caller gives it back with release.
func (s *shelf) take(path string) (*shelvedBook, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errReadFile, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errShelfClosed
	}
	for i, book := range s.open {
		if book.path != path {
			continue
		}
		if book.size == info.Size() && book.mtime == info.ModTime().UnixNano() {
			s.open = append(append(s.open[:i:i], s.open[i+1:]...), book)
			book.readers++
			return book, nil
		}
		s.open = append(s.open[:i:i], s.open[i+1:]...)
		s.retire(book)
		break
	}

	book, err := s.openBook(path)
	if err != nil {
		return nil, err
	}
	if len(s.open) >= mostOpenBooks {
		s.retire(s.open[0])
		s.open = s.open[1:]
	}
	s.open = append(s.open, book)
	book.readers++
	return book, nil
}

func (s *shelf) openBook(path string) (*shelvedBook, error) {
	file, err := s.openFile(path)
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
	return &shelvedBook{
		path:    path,
		size:    info.Size(),
		mtime:   info.ModTime().UnixNano(),
		file:    file,
		archive: archive,
	}, nil
}

// retire lets a book go: its file is closed now when no request reads it, and by
// the last request that does otherwise. The caller holds the lock.
func (s *shelf) retire(book *shelvedBook) {
	book.retired = true
	if book.readers == 0 {
		_ = book.file.Close()
	}
}

// release gives back a book taken.
func (s *shelf) release(book *shelvedBook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	book.readers--
	if book.retired && book.readers == 0 {
		_ = book.file.Close()
	}
}

// Close lets every book go and refuses later requests. Requests still reading
// finish first.
func (s *shelf) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	for _, book := range s.open {
		s.retire(book)
	}
	s.open = nil
}

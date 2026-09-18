package pagecache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// A Disk is the pages drawn before, kept where this machine keeps what it can
// make again.
//
// A page of a scan is the same half second of decoding every time it is turned
// back to, whatever size was asked for. What is here is a Disk in the strict
// sense: it is made from the document alone, so it may be deleted at any moment
// and the only cost is drawing the page again.
//
// It is not in the vault and not beside the document. A drawn page is nobody's
// work, and the folder a person keeps their notes in is not where an
// application puts what it can remake.
type Disk struct {
	dir   string
	limit int64
	// every is what is written between sweeps.
	every int64

	// going is the sweep that has not ended. A sweep outlives the page whose
	// writing started it, and the window waits here for it: it is unlink calls
	// for the files above the bound, and a folder left half swept is one nothing
	// is left to sweep.
	going sync.WaitGroup

	mu sync.Mutex
	// written is what has been written since the last sweep. Counting the folder
	// on every write is a folder read per page turned.
	written int64
	// isShut is the window gone. A sweep started after it would be one nobody is
	// left to wait for.
	isShut bool
}

const (
	// mostKept is what the drawn pages may come to on disk. A book of five
	// hundred pages at one width is under this, so reading one through does not
	// throw away its own beginning.
	mostKept = 512 << 20
	// sweptEvery is how much is written between sweeps.
	sweptEvery = mostKept / 8
)

// NewDisk is where this machine keeps pages already drawn, and nothing where it
// says it keeps nothing.
func NewDisk() *Disk {
	under, err := os.UserCacheDir()
	if err != nil {
		return nil
	}
	dir := filepath.Join(under, "numen", "pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil
	}
	return NewDiskIn(dir)
}

// NewDiskIn is a Disk in one folder, with nothing written to it yet and no sweep
// running.
func NewDiskIn(dir string) *Disk {
	return &Disk{dir: dir, limit: mostKept, every: sweptEvery}
}

// Close waits for the sweep that is running, and starts no more.
//
// A sweep is one pass of unlink calls over the files above the bound, so what
// it costs to wait for it is that. What it costs not to is a folder swept as
// far as the process happened to get, with the count that would sweep it again
// already back at nothing.
func (s *Disk) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.isShut = true
	s.mu.Unlock()
	s.going.Wait()
}

// Get is a page drawn before, and nothing where it was not.
//
// The file's time is set to now, so what is dropped when the folder is swept is
// what has gone longest without being looked at.
func (s *Disk) Get(key ID) []byte {
	if s == nil {
		return nil
	}
	at := filepath.Join(s.dir, s.getFileName(key))
	body, err := os.ReadFile(at)
	if err != nil {
		return nil
	}
	now := time.Now()
	_ = os.Chtimes(at, now, now)
	return body
}

// Put keeps one drawn page. A page that cannot be written is a page drawn again
// next time and nothing else, so nothing here is reported.
func (s *Disk) Put(key ID, body []byte) {
	if s == nil || len(body) == 0 {
		return
	}
	at := filepath.Join(s.dir, s.getFileName(key))
	// Written under another name and moved into place, so a reader never opens
	// half a picture.
	temp, err := os.CreateTemp(s.dir, "drawing-")
	if err != nil {
		return
	}
	if _, err := temp.Write(body); err != nil {
		temp.Close()
		os.Remove(temp.Name())
		return
	}
	if err := temp.Close(); err != nil {
		os.Remove(temp.Name())
		return
	}
	if err := os.Rename(temp.Name(), at); err != nil {
		os.Remove(temp.Name())
		return
	}

	s.mu.Lock()
	s.written += int64(len(body))
	due := !s.isShut && s.written >= s.every
	if due {
		s.written = 0
		// Counted before the sweep is started, so a close cannot come between
		// the two and go without waiting for it.
		s.going.Add(1)
	}
	s.mu.Unlock()
	if due {
		go func() {
			defer s.going.Done()
			s.sweep()
		}()
	}
}

// sweep drops the pages that have gone longest without being looked at, until
// what is kept is under the bound.
func (s *Disk) sweep() {
	if s == nil {
		return
	}
	held, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	type page struct {
		name  string
		size  int64
		mtime time.Time
	}
	pages := make([]page, 0, len(held))
	var total int64
	for _, one := range held {
		info, err := one.Info()
		if err != nil {
			continue
		}
		pages = append(pages, page{name: one.Name(), size: info.Size(), mtime: info.ModTime()})
		total += info.Size()
	}
	if total <= s.limit {
		return
	}
	sort.Slice(pages, func(a, b int) bool { return pages[a].mtime.Before(pages[b].mtime) })
	for _, one := range pages {
		if total <= s.limit {
			return
		}
		if os.Remove(filepath.Join(s.dir, one.name)) == nil {
			total -= one.size
		}
	}
}

// getFileName is what one drawn page is called: the bytes it was drawn from, which
// page of them, and how wide.
//
// The file's own name says nothing about the vault. A folder listing is
// readable by whatever else runs as this person, and what they are reading is
// theirs.
func (s *Disk) getFileName(key ID) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s\x00%d\x00%d\x00%d\x00%d",
		key.Document.Path, key.Document.Size, key.Document.Mtime, key.Page, key.Width))
	return hex.EncodeToString(sum[:]) + ".jpg"
}

// GetDir is the folder the drawings are kept in.
func (s *Disk) GetDir() string { return s.dir }

// GetPath is where one drawing is kept, whether or not it has been drawn yet.
func (s *Disk) GetPath(key ID) string { return filepath.Join(s.dir, s.getFileName(key)) }

// SweepTo brings what is kept down to this many bytes, oldest first, and waits
// for the sweep. What is kept is a cache, so a folder emptied further than the
// bound is a folder whose pages are drawn again.
func (s *Disk) SweepTo(bytes int64) {
	if s == nil {
		return
	}
	s.limit = bytes
	s.sweep()
}

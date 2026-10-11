package vault

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Refresh brings named notes up to date, for when something says which files
// changed. A scan asks the whole vault what it looks like; this asks a handful
// of files.
//
// The names it is handed come from a watcher, which reports every kind of
// source. A path that is not a note is said back and left alone.
type Refresh struct {
	readers port.VaultReaders
	// vaults is where the vault gets the row every note of it points at.
	vaults port.VaultRepository
	notes  port.NoteRepository
	// queries says which kind of source the index holds at a path, and sources
	// takes those rows out.
	queries port.SourceQueries
	sources port.SourceRepository

	// Derived is where what was downloaded for a link note is kept. A run given
	// none indexes every note as the prose in its file.
	Derived port.DerivedStores

	// Ledger holds the content of notes indexed while their modification time
	// was too recent to tell a second edit from the first. Refreshes that share
	// an index share one.
	Ledger *Ledger

	// Now is when a note counts as recently modified. A refresh given none treats
	// every note as recent.
	Now port.Clock

	// Force reads, parses and saves every named note, whatever the index holds
	// about it.
	Force bool
}

// TimestampGranularity is the coarsest step in which a supported file system
// records modification times. A file modified less than this long ago can be
// edited again without its modification time changing.
const TimestampGranularity = 2 * time.Second

// Ledger is the content hash of each note that was indexed while its
// modification time was within TimestampGranularity of the clock. A refresh
// skips such a note only when its bytes still hash to the recorded value.
type Ledger struct {
	mu   sync.Mutex
	held map[ledgerKey]ledgerEntry
}

type ledgerKey struct {
	vault domain.VaultID
	path  string
}

type ledgerEntry struct {
	fingerprint domain.Fingerprint
	hash        [sha256.Size]byte
}

// NewLedger is a ledger holding nothing.
func NewLedger() *Ledger { return &Ledger{held: make(map[ledgerKey]ledgerEntry)} }

func (l *Ledger) get(v domain.VaultID, path string) (ledgerEntry, bool) {
	if l == nil {
		return ledgerEntry{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.held[ledgerKey{v, path}]
	return entry, ok
}

func (l *Ledger) put(v domain.VaultID, path string, entry ledgerEntry) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.held[ledgerKey{v, path}] = entry
}

func (l *Ledger) drop(v domain.VaultID, path string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.held, ledgerKey{v, path})
}

// isRecent reports whether a file modified at mtime can still be edited again
// under the same modification time.
func (u Refresh) isRecent(mtime time.Time) bool {
	if u.Now == nil {
		return true
	}
	return u.Now().Sub(mtime) < TimestampGranularity
}

// NewRefresh is how the index is brought level with a handful of files: the
// vault they are read out of, the row it is filed under, where a note is filed,
// and the two the rows of a book and a recording are swept through.
func NewRefresh(
	readers port.VaultReaders,
	vaults port.VaultRepository,
	notes port.NoteRepository,
	queries port.SourceQueries,
	sources port.SourceRepository,
) Refresh {
	return Refresh{
		readers: readers, vaults: vaults, notes: notes, queries: queries, sources: sources,
		Ledger: NewLedger(),
	}
}

// RefreshResult is what happened, in the terms a caller acts on: the notes that
// are now different from what was shown, and the ones that could not be read.
type RefreshResult struct {
	Indexed []string
	Removed []string
	// LeftAlone is the paths holding a file the vault does not hold as a note.
	// Nothing was there to look at again, and nothing is missing either.
	LeftAlone  []string
	Unreadable []string
	// Assets are the paths of sources that are not notes. They changed, and
	// bringing them up to date is its own step.
	Assets []string
}

// GetChangedPaths is every note the caller may need to look at again.
func (r RefreshResult) GetChangedPaths() []string {
	return append(append([]string(nil), r.Indexed...), r.Removed...)
}

// Execute brings the named notes up to date.
//
// A note whose size and modification time equal the index's row is reported in
// Indexed without being saved, unless Force is set. It is skipped without a read
// when its modification time is at least TimestampGranularity old; a more recent
// one is skipped only when its bytes hash to what the ledger recorded at its last
// save.
func (u Refresh) Execute(ctx context.Context, v domain.Vault, paths []string) (RefreshResult, error) {
	var res RefreshResult
	if len(paths) == 0 {
		return res, nil
	}

	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	// A note is filed under the vault's row, and a vault nothing has walked yet
	// has none: a note written into a vault the moment it opens is levelled
	// here or nowhere.
	if err := u.vaults.Register(ctx, v.ID); err != nil {
		return res, fmt.Errorf("register vault: %w", err)
	}
	// The same bounds a scan writes in. One event can name a whole folder — a
	// checkout, a restore, a sync client unpacking an archive — so the number of
	// paths handed here is not small because they were named individually.
	group := grouping{write: func(ctx context.Context, notes []domain.Note) error {
		return u.notes.Save(ctx, v.ID, notes)
	}}

	recorded := make(map[string]*ledgerEntry)
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		ref, err := reader.Stat(ctx, path)
		// A file the vault leaves alone is at this path — an attachment, an
		// export, a note somebody renamed out of the vault's sight. It is not
		// a note that vanished: nothing was lost, so nobody is told to look
		// again, and the index holds nothing at a path that is not a note.
		if errors.Is(err, port.ErrNotANote) {
			res.LeftAlone = append(res.LeftAlone, path)
			continue
		}
		if errors.Is(err, fs.ErrNotExist) {
			res.Removed = append(res.Removed, path)
			continue
		}
		if err != nil {
			// One file nobody can read — a permission, a broken link, a device
			// that went away — must not cost the others. A watcher's event
			// arrives once, so anything dropped here is dropped until the next
			// scan.
			res.Unreadable = append(res.Unreadable, path)
			continue
		}
		if ref.Kind != domain.KindNote {
			// The vault holds it, so it is not gone and its row stays where it
			// is. Reading it is the business of whatever extracts its text.
			res.Assets = append(res.Assets, path)
			continue
		}
		if !u.Force && u.isCurrent(ctx, reader, v, ref) {
			res.Indexed = append(res.Indexed, path)
			continue
		}
		raw, err := reader.Read(ctx, path)
		if errors.Is(err, fs.ErrNotExist) {
			// It was there a moment ago, when it was looked at. A file that
			// briefly disappears between the two is what an editor saving
			// through a temporary file looks like, and taking the note out of
			// the index would take it out of search until something put it
			// back. The save that follows arrives as its own event.
			continue
		}
		if err != nil {
			res.Unreadable = append(res.Unreadable, path)
			continue
		}
		if err := group.add(ctx, markdown.Parse(ref, raw), len(raw)); err != nil {
			return res, fmt.Errorf("index: %w", err)
		}
		res.Indexed = append(res.Indexed, path)
		if u.isRecent(ref.ModTime) {
			recorded[path] = &ledgerEntry{fingerprint: ref, hash: sha256.Sum256(raw)}
		} else {
			recorded[path] = nil
		}
	}

	if err := group.flush(ctx); err != nil {
		return res, fmt.Errorf("index: %w", err)
	}
	for path, entry := range recorded {
		if entry == nil {
			u.Ledger.drop(v.ID, path)
			continue
		}
		u.Ledger.put(v.ID, path, *entry)
	}
	// Both leave the index, and they leave it for different reasons: one path
	// has nothing at it, the other has something that is not a note.
	gone := append(append([]string(nil), res.Removed...), res.LeftAlone...)
	if err := u.notes.Remove(ctx, v.ID, gone); err != nil {
		return res, fmt.Errorf("remove: %w", err)
	}
	if err := u.dropSources(ctx, v, gone); err != nil {
		return res, fmt.Errorf("remove: %w", err)
	}
	return res, nil
}

// dropSources takes out the rows of every source at a path the vault no longer
// holds. A note leaves through the note repository; a book and a recording are
// filed by kind and leave through their own, with their chunks and their vectors.
func (u Refresh) dropSources(ctx context.Context, v domain.Vault, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	// A path names one file, and a folder names everything under it.
	held := make(map[domain.SourceKind][]string)
	for _, path := range paths {
		under, err := u.queries.GetSourcesUnder(ctx, v.ID, path)
		if err != nil {
			return err
		}
		for _, ref := range under {
			if ref.Kind == domain.KindNote {
				continue
			}
			held[ref.Kind] = append(held[ref.Kind], ref.Path)
		}
	}
	for _, kind := range slices.Sorted(maps.Keys(held)) {
		if err := u.sources.RemoveSources(ctx, v.ID, kind, held[kind]); err != nil {
			return err
		}
	}
	return nil
}

// isCurrent reports whether the index already holds the note as the file is.
func (u Refresh) isCurrent(ctx context.Context, reader port.VaultReader, v domain.Vault, ref domain.Fingerprint) bool {
	held, err := u.queries.GetSourcesUnder(ctx, v.ID, ref.Path)
	if err != nil {
		return false
	}
	at := slices.IndexFunc(held, func(f domain.Fingerprint) bool {
		return f.Path == ref.Path && f.Kind == domain.KindNote
	})
	if at < 0 || !held[at].IsUnchanged(ref) {
		return false
	}
	entry, ok := u.Ledger.get(v.ID, ref.Path)
	if !ok {
		return !u.isRecent(ref.ModTime)
	}
	if !entry.fingerprint.IsUnchanged(ref) {
		return false
	}
	raw, err := reader.Read(ctx, ref.Path)
	if err != nil || sha256.Sum256(raw) != entry.hash {
		return false
	}
	if !u.isRecent(ref.ModTime) {
		u.Ledger.drop(v.ID, ref.Path)
	}
	return true
}

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

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Refresh brings named notes up to date, for when something says which files
// changed. A scan asks the whole vault what it looks like; this asks a handful
// of files.
//
// The names it is handed come from a watcher, which reports every kind of
// source. A path that is not a note is said back and left alone.
type Refresh struct {
	readers vaultport.VaultReaders
	// vaults is where the vault gets the row every note of it points at.
	vaults vaultport.VaultRepository
	notes  noteport.NoteRepository
	// queries says which kind of source the index holds at a path, and sources
	// takes those rows out.
	queries sourceport.SourceQueries
	sources sourceport.SourceRepository
	// now says which notes were modified recently enough to be edited again under
	// the same modification time.
	now machineport.Clock

	// Cut is how the notes named are cut into the chunks the index keeps.
	Cut note.CutNotes

	// Ledger holds the content of notes indexed while their modification time
	// was too recent to tell a second edit from the first. Refreshes that share
	// an index share one.
	Ledger *Ledger

	// Force reads, parses and saves every named note, whatever the index holds
	// about it.
	Force bool
}

// TimestampGranularity is the coarsest step in which a supported file system
// records modification times; FAT stores them in two-second steps. A file
// modified less than this long ago can be edited again without its modification
// time changing.
const TimestampGranularity = 2 * time.Second

// Ledger is the content hash of each note indexed while its modification time was
// within TimestampGranularity of the clock. A refresh skips such a note only when
// its bytes still hash to the recorded value. An entry is held only while its
// file is inside that window.
type Ledger struct {
	mu   sync.Mutex
	item map[ledgerKey]ledgerEntry
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
func NewLedger() *Ledger { return &Ledger{item: make(map[ledgerKey]ledgerEntry)} }

func (l *Ledger) get(now time.Time, v domain.VaultID, path string) (ledgerEntry, bool) {
	if l == nil {
		return ledgerEntry{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	entry, ok := l.item[ledgerKey{v, path}]
	return entry, ok
}

func (l *Ledger) put(now time.Time, v domain.VaultID, path string, entry ledgerEntry) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.item[ledgerKey{v, path}] = entry
	l.prune(now)
}

// Len is how many notes the ledger holds.
func (l *Ledger) Len() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.item)
}

// prune forgets the entries whose file has left the window. The caller holds the
// lock.
func (l *Ledger) prune(now time.Time) {
	for key, entry := range l.item {
		if now.Sub(entry.fingerprint.ModTime) >= TimestampGranularity {
			delete(l.item, key)
		}
	}
}

func (l *Ledger) drop(v domain.VaultID, path string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.item, ledgerKey{v, path})
}

// isRecent reports whether a file modified at mtime can still be edited again
// under the same modification time.
func (u Refresh) isRecent(mtime time.Time) bool {
	return u.now().Sub(mtime) < TimestampGranularity
}

// NewRefresh is how the index is brought level with a handful of files: the
// vault they are read out of, the row it is filed under, where a note is filed,
// the two the rows of a book and a recording are swept through, and the clock
// that says which notes were modified recently.
func NewRefresh(
	readers vaultport.VaultReaders,
	vaults vaultport.VaultRepository,
	notes noteport.NoteRepository,
	queries sourceport.SourceQueries,
	sources sourceport.SourceRepository,
	now machineport.Clock,
) Refresh {
	return Refresh{
		readers: readers, vaults: vaults, notes: notes, queries: queries, sources: sources,
		now:    now,
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
// save. A recent note is hashed only when the ledger knows it. A scan decides on
// size and modification time alone.
func (u Refresh) Execute(ctx context.Context, v domain.Vault, paths []string) (RefreshResult, error) {
	if len(paths) == 0 {
		return RefreshResult{}, nil
	}

	reader, err := u.readers.Open(v)
	if err != nil {
		return RefreshResult{}, err
	}
	// A note is filed under the vault's row, and a vault nothing has walked yet
	// has none: a note written into a vault the moment it opens is levelled
	// here or nowhere.
	if err := u.vaults.Register(ctx, v.ID); err != nil {
		return RefreshResult{}, fmt.Errorf("register vault: %w", err)
	}
	// The same bounds a scan writes in. One event can name a whole folder — a
	// checkout, a restore, a sync client unpacking an archive — so the number of
	// paths handed here is not small because they were named individually.
	var res RefreshResult
	group := grouping{write: func(ctx context.Context, notes []domain.Note) error {
		return u.notes.Save(ctx, v.ID, u.Cut.Execute(notes))
	}}

	recorded := make(map[string]*ledgerEntry)
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if err := u.refreshPath(ctx, v, reader, path, &group, recorded, &res); err != nil {
			return res, err
		}
	}

	if err := group.flush(ctx); err != nil {
		return res, fmt.Errorf("index: %w", err)
	}
	u.writeLedger(v, recorded)
	// Both leave the index, and they leave it for different reasons: one path
	// has nothing at it, the other has something that is not a note.
	removed := append(append([]string(nil), res.Removed...), res.LeftAlone...)
	if err := u.notes.Remove(ctx, v.ID, removed); err != nil {
		return res, fmt.Errorf("remove: %w", err)
	}
	if err := u.dropSources(ctx, v, removed); err != nil {
		return res, fmt.Errorf("remove: %w", err)
	}
	return res, nil
}

// refreshPath files one path under the list of res it belongs to, and adds the
// note it holds to the group when the index is behind the file. A note indexed
// is recorded for the ledger: its entry when it is recent, nil when it is not.
func (u Refresh) refreshPath(
	ctx context.Context,
	v domain.Vault,
	reader vaultport.VaultReader,
	path string,
	group *grouping,
	recorded map[string]*ledgerEntry,
	res *RefreshResult,
) error {
	ref, isNote := u.statNote(ctx, reader, path, res)
	if !isNote {
		return nil
	}
	if !u.Force && u.isCurrent(ctx, reader, v, ref) {
		res.Indexed = append(res.Indexed, path)
		return nil
	}
	raw, isRead := readListedFile(ctx, reader, path, res)
	if !isRead {
		return nil
	}
	if err := group.add(ctx, markdown.Parse(ref, raw), len(raw)); err != nil {
		return fmt.Errorf("index: %w", err)
	}
	res.Indexed = append(res.Indexed, path)
	if u.isRecent(ref.ModTime) {
		recorded[path] = &ledgerEntry{fingerprint: ref, hash: sha256.Sum256(raw)}
	} else {
		recorded[path] = nil
	}
	return nil
}

// readListedFile is the bytes of the note at path and whether they were read. A
// file gone since it was looked at is an editor saving through a temporary
// file, and the save that follows arrives as its own event; any other failure is
// filed as unreadable.
func readListedFile(ctx context.Context, reader vaultport.VaultReader, path string, res *RefreshResult) ([]byte, bool) {
	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		res.Unreadable = append(res.Unreadable, path)
	default:
		return raw, true
	}
	return nil, false
}

// statNote is the fingerprint of the file at path and whether it is a note. A
// path that is not one is filed in res under the list that says why, and nothing
// is lost by it: one file nobody can read costs that file and not the others.
func (u Refresh) statNote(
	ctx context.Context, reader vaultport.VaultReader, path string, res *RefreshResult,
) (domain.Fingerprint, bool) {
	ref, err := reader.GetFingerprint(ctx, path)
	switch {
	case errors.Is(err, vaultport.ErrNotANote):
		// An attachment, an export or a note renamed out of the vault's sight is
		// not a note that vanished, and the index holds nothing at its path.
		res.LeftAlone = append(res.LeftAlone, path)
	case errors.Is(err, fs.ErrNotExist):
		res.Removed = append(res.Removed, path)
	case err != nil:
		res.Unreadable = append(res.Unreadable, path)
	case ref.Kind != domain.KindNote:
		// The vault holds it, so its row stays where it is. Reading it is the
		// business of whatever extracts its text.
		res.Assets = append(res.Assets, path)
	default:
		return ref, true
	}
	return domain.Fingerprint{}, false
}

// writeLedger puts the recorded entries in the ledger and drops the paths
// recorded as nil.
func (u Refresh) writeLedger(v domain.Vault, recorded map[string]*ledgerEntry) {
	for path, entry := range recorded {
		if entry == nil {
			u.Ledger.drop(v.ID, path)
			continue
		}
		u.Ledger.put(u.now(), v.ID, path, *entry)
	}
}

// dropSources takes out the rows of every source at a path the vault no longer
// holds. A note leaves through the note repository; a book and a recording are
// filed by kind and leave through their own, with their chunks and their vectors.
func (u Refresh) dropSources(ctx context.Context, v domain.Vault, paths []string) error {
	// A path names one file, and a folder names everything under it.
	target := make(map[domain.SourceKind][]string)
	for _, path := range paths {
		limit, err := u.queries.GetSourcesUnder(ctx, v.ID, path)
		if err != nil {
			return err
		}
		for _, ref := range limit {
			if ref.Kind == domain.KindNote {
				continue
			}
			target[ref.Kind] = append(target[ref.Kind], ref.Path)
		}
	}
	for _, kind := range slices.Sorted(maps.Keys(target)) {
		if err := u.sources.RemoveSources(ctx, v.ID, kind, target[kind]); err != nil {
			return err
		}
	}
	return nil
}

// isCurrent reports whether the index already holds the note as the file is.
func (u Refresh) isCurrent(ctx context.Context, reader vaultport.VaultReader, v domain.Vault, ref domain.Fingerprint) bool {
	target, err := u.queries.GetSourcesUnder(ctx, v.ID, ref.Path)
	if err != nil {
		return false
	}
	pos := slices.IndexFunc(target, func(f domain.Fingerprint) bool {
		return f.Path == ref.Path && f.Kind == domain.KindNote
	})
	if pos < 0 || !target[pos].IsUnchanged(ref) {
		return false
	}
	entry, ok := u.Ledger.get(u.now(), v.ID, ref.Path)
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

package vault

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Refresh brings named notes up to date when a watcher reports which files
// changed. A scan inspects the whole vault; Refresh only the named files.
// The paths come from a watcher, which reports every source type. A path that is
// not a note is reported in the result and skipped.
type Refresh struct {
	readers vaultport.VaultReaders
	vaults  vaultport.VaultRepository
	notes   noteport.NoteRepository
	queries sourceport.RemovalLookup
	sources sourceport.SourceRepository
	now     machineport.Clock

	Cut chunking.CutNotes

	Ledger *Ledger

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
	stat domain.FileStat
	hash [sha256.Size]byte
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

func (l *Ledger) prune(now time.Time) {
	for key, entry := range l.item {
		if now.Sub(entry.stat.ModTime) >= TimestampGranularity {
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

func (u Refresh) isRecent(mtime time.Time) bool {
	return u.now.Now().Sub(mtime) < TimestampGranularity
}

// NewRefresh creates a Refresh from the vault readers, the vault repository,
// the note repository, what the index holds about sources, the source repository that deletes the rows
// of other files, and the clock that identifies recently modified notes.
func NewRefresh(
	readers vaultport.VaultReaders,
	vaults vaultport.VaultRepository,
	notes noteport.NoteRepository,
	queries sourceport.RemovalLookup,
	sources sourceport.SourceRepository,
	now machineport.Clock,
) Refresh {
	return Refresh{
		readers: readers, vaults: vaults, notes: notes, queries: queries, sources: sources,
		now:    now,
		Ledger: NewLedger(),
	}
}

// RefreshDeps are what NewRefreshFrom builds a Refresh over: the collaborators of NewRefresh, the sizes the notes are
// cut at, and the ledger the refreshes of one index share.
type RefreshDeps struct {
	Readers vaultport.VaultReaders
	Vaults  vaultport.VaultRepository
	Notes   noteport.NoteRepository
	Queries sourceport.RemovalLookup
	Sources sourceport.SourceRepository
	Now     machineport.Clock
	Cut     chunking.CutNotes
	Ledger  *Ledger
}

// NewRefreshFrom creates a Refresh that cuts notes at deps.Cut and records recent notes in deps.Ledger.
func NewRefreshFrom(deps RefreshDeps) Refresh {
	return Refresh{
		readers: deps.Readers, vaults: deps.Vaults, notes: deps.Notes, queries: deps.Queries, sources: deps.Sources,
		now: deps.Now, Cut: deps.Cut, Ledger: deps.Ledger,
	}
}

// RefreshResult is what happened, in the terms a caller acts on: the notes that
// are now different from what was shown, and the ones that could not be read.
type RefreshResult struct {
	Indexed    []string
	Removed    []string
	LeftAlone  []string
	Unreadable []string
	Assets     []string
}

// GetChangedPaths is every note the caller may need to look at again.
func (r RefreshResult) GetChangedPaths() []string {
	return append(append([]string(nil), r.Indexed...), r.Removed...)
}

// Execute brings the named notes up to date.
// A note whose size and modification time equal the index's row is reported in
// Indexed without being saved, unless Force is set. It is skipped without a read
// when its modification time is at least TimestampGranularity old; a more recent
// one only when its bytes hash to what the ledger recorded at its last save.
func (u Refresh) Execute(ctx context.Context, v domain.Vault, paths []string) (RefreshResult, error) {
	if len(paths) == 0 {
		return RefreshResult{}, nil
	}

	reader, err := u.readers.Open(v)
	if err != nil {
		return RefreshResult{}, err
	}
	// A note is filed under the vault's row, and a vault nothing has walked yet
	// has none: a note written into a vault the moment it opens is indexed
	// here or nowhere.
	if err := u.vaults.Register(ctx, v.ID); err != nil {
		return RefreshResult{}, fmt.Errorf("the index cannot record the vault %s: %w", v.Name, err)
	}
	// The same bounds a scan writes in. One event can name a whole folder — a
	// checkout, a restore, a sync client unpacking an archive — so the number of
	// paths handed here is not small because they were named individually.
	var res RefreshResult
	group := grouping{write: func(ctx context.Context, notes []domain.Note) error {
		return u.notes.Save(ctx, v.ID, u.Cut.Execute(notes))
	}}

	recorded := make(map[string]*ledgerEntry)
	run := refreshRun{v: v, reader: reader, group: &group, recorded: recorded, res: &res}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if err := u.refreshPath(ctx, &run, path); err != nil {
			return res, err
		}
	}

	if err := group.flush(ctx); err != nil {
		return res, fmt.Errorf("the notes of %s cannot be indexed: %w", v.Name, err)
	}
	u.writeLedger(v, recorded)
	if err := u.followRenames(ctx, v, reader, res); err != nil {
		return res, fmt.Errorf("the renamed notes of %s cannot be tracked: %w", v.Name, err)
	}
	// Both leave the index, and they leave it for different reasons: one path
	// has nothing at it, the other has something that is not a note.
	removed := append(append([]string(nil), res.Removed...), res.LeftAlone...)
	if err := u.notes.Remove(ctx, v.ID, removed); err != nil {
		return res, fmt.Errorf("the removed notes of %s cannot be removed from the index: %w", v.Name, err)
	}
	if err := u.dropSources(ctx, v, removed); err != nil {
		return res, fmt.Errorf("the sources of the removed notes of %s cannot be removed from the index: %w", v.Name, err)
	}
	return res, nil
}

// refreshRun is what one Execute shares across the paths it refreshes.
type refreshRun struct {
	v        domain.Vault
	reader   vaultport.VaultReader
	group    *grouping
	recorded map[string]*ledgerEntry
	res      *RefreshResult
}

func (u Refresh) refreshPath(ctx context.Context, run *refreshRun, path string) error {
	v, reader, group, recorded, res := run.v, run.reader, run.group, run.recorded, run.res
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
		return fmt.Errorf("%s cannot be indexed: %w", path, err)
	}
	res.Indexed = append(res.Indexed, path)
	if u.isRecent(ref.ModTime) {
		recorded[path] = &ledgerEntry{stat: ref, hash: sha256.Sum256(raw)}
	} else {
		recorded[path] = nil
	}
	return nil
}

func readListedFile(ctx context.Context, reader vaultport.VaultReader, path string, res *RefreshResult) ([]byte, bool) {
	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, edit.ErrNotExist):
	case err != nil:
		res.Unreadable = append(res.Unreadable, path)
	default:
		return raw, true
	}
	return nil, false
}

func (u Refresh) statNote(
	ctx context.Context, reader vaultport.VaultReader, path string, res *RefreshResult,
) (domain.FileStat, bool) {
	ref, err := reader.StatFile(ctx, path)
	switch {
	case errors.Is(err, edit.ErrNotANote):
		// An attachment, an export or a note renamed out of the vault's sight is
		// not a note that vanished, and the index holds nothing at its path.
		res.LeftAlone = append(res.LeftAlone, path)
	case errors.Is(err, edit.ErrNotExist):
		res.Removed = append(res.Removed, path)
	case err != nil:
		res.Unreadable = append(res.Unreadable, path)
	case ref.Kind != domain.KindNote:
		// The vault holds it, so its row keeps its position. Reading it is the
		// business of whatever extracts its text.
		res.Assets = append(res.Assets, path)
	default:
		return ref, true
	}
	return domain.FileStat{}, false
}

func (u Refresh) writeLedger(v domain.Vault, recorded map[string]*ledgerEntry) {
	for path, entry := range recorded {
		if entry == nil {
			u.Ledger.drop(v.ID, path)
			continue
		}
		u.Ledger.put(u.now.Now(), v.ID, path, *entry)
	}
}

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

func (u Refresh) followRenames(ctx context.Context, v domain.Vault, reader vaultport.VaultReader, res RefreshResult) error {
	if len(res.Removed) == 0 || len(res.Assets) == 0 {
		return nil
	}
	removed, err := sourcedomain.FindRemoved(res.Removed,
		func(path string) ([]domain.FileStat, error) { return u.queries.GetSourcesUnder(ctx, v.ID, path) },
		func(path string) (sourcedomain.Text, bool, error) { return u.queries.GetSourceText(ctx, v.ID, path) })
	if err != nil {
		return err
	}
	_, err = sourcedomain.MoveRenamed(removed, res.Assets,
		func(path string) (domain.FileStat, error) { return reader.StatFile(ctx, path) },
		func(path string) (string, error) {
			raw, err := reader.Read(ctx, path)
			if err != nil {
				return "", err
			}
			return domain.HashContent(raw), nil
		},
		func(from, to string) error { return u.sources.MoveSources(ctx, v.ID, from, to) })
	return err
}

func (u Refresh) isCurrent(ctx context.Context, reader vaultport.VaultReader, v domain.Vault, ref domain.FileStat) bool {
	target, err := u.queries.GetSourcesUnder(ctx, v.ID, ref.Path)
	if err != nil {
		return false
	}
	pos := slices.IndexFunc(target, func(f domain.FileStat) bool {
		return f.Path == ref.Path && f.Kind == domain.KindNote
	})
	if pos < 0 || !target[pos].IsUnchanged(ref) {
		return false
	}
	entry, ok := u.Ledger.get(u.now.Now(), v.ID, ref.Path)
	if !ok {
		return !u.isRecent(ref.ModTime)
	}
	if !entry.stat.IsUnchanged(ref) {
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

// UpdateIndex brings the notes a write touched up to date in the index, through
// the Refresh of the vault the caller has running.
type UpdateIndex struct {
	refresh func() Refreshing
}

// NewUpdateIndex creates an UpdateIndex that asks getRefresh for the refresh to
// use at each call.
func NewUpdateIndex(getRefresh func() Refreshing) UpdateIndex {
	return UpdateIndex{refresh: getRefresh}
}

// Execute refreshes the notes at the paths of the vault.
func (u UpdateIndex) Execute(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := u.refresh().Execute(ctx, v, paths)
	return err
}

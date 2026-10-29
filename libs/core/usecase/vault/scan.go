package vault

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/libs/core/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Scan updates the index to match one vault. The vault is authoritative: after
// a scan the index holds exactly what the scan found.
type Scan struct {
	readers     vaultport.VaultReaders
	vaults      vaultport.VaultRepository
	notes       noteport.NoteRepository
	indexedVal  FileStatQueries
	maintenance indexport.IndexMaintenance

	// Cut is how the notes found are cut into the chunks the index keeps.
	Cut chunking.CutNotes

	// Walks serializes the walks of the vaults being scanned. A scan without it
	// runs without waiting for any other scan.
	Walks *Walks

	// ShouldRebuildIndex reads every file and puts it in the index again, whatever the
	// index remembers about it.
	//
	// What it remembers is a path, a size and a modification time, so a file whose
	// content changed while those did not is skipped — an archive restored by
	// `unzip`, a tree brought over by `rsync -tc`. Rebuilding is the way out of
	// that, and the only one.
	ShouldRebuildIndex bool

	// OnProgress, if set, is called each time a group of notes is written. A
	// scan of a large vault takes a minute, and something has to be able to say
	// so while it happens. What is done with that is the caller's business.
	OnProgress func(ScanResult)
}

// NewScan returns a Scan that reads and writes through: the vault reader, the
// vault repository, the note repository, the file stat queries of the index,
// and the maintenance run after the index changed.
//
// Maintenance runs only when the walk found a change.
func NewScan(
	readers vaultport.VaultReaders,
	vaults vaultport.VaultRepository,
	notes noteport.NoteRepository,
	queries FileStatQueries,
	maintenance indexport.IndexMaintenance,
) Scan {
	return Scan{
		readers: readers, vaults: vaults, notes: notes,
		indexedVal: queries, maintenance: maintenance,
	}
}

// ScanResult reports what a scan did, in the terms the user cares about.
type ScanResult = vaultreading.ScanResult

// ScanWithProgress runs Execute with onProgress reporting each group when non-nil.
func (u Scan) ScanWithProgress(ctx context.Context, v domain.Vault, onProgress func(ScanResult)) (ScanResult, error) {
	if onProgress != nil {
		u.OnProgress = onProgress
	}
	return u.Execute(ctx, v)
}

// FileStatQueries is the single query a scan makes of the index. It exposes
// which files have changed and nothing about notes.
type FileStatQueries interface {
	// GetFileStats is what the index believes about each file, keyed by path,
	// so a scan can decide what to reparse without reading anything.
	GetFileStats(ctx context.Context, vaultID domain.VaultID) (map[string]domain.FileStat, error)
}

// Execute walks the vault once.
//
// Only files whose size or modification time differ from what the index holds
// are read and parsed; the rest are not opened at all. That is what keeps a scan
// of an unchanged vault cheap enough to run at startup.
//
// One walk of a vault runs at a time among the scans sharing its turns. A walk
// writes in groups from what it read, so its copy of a note lands last however
// early the note was read.
func (u Scan) Execute(ctx context.Context, v domain.Vault) (ScanResult, error) {
	var res ScanResult

	finished, err := u.Walks.startOne(ctx, v.ID)
	if err != nil {
		return ScanResult{}, err
	}
	defer finished()

	reader, err := u.readers.Open(v)
	if err != nil {
		return ScanResult{}, err
	}
	// The rows the walk writes point at the vault's own row. What the vault is
	// called and where it is stay as the list has them.
	if err := u.vaults.Register(ctx, v.ID); err != nil {
		return ScanResult{}, fmt.Errorf("register vault: %w", err)
	}

	recognized, err := u.indexedVal.GetFileStats(ctx, v.ID)
	if err != nil {
		return ScanResult{}, fmt.Errorf("read index: %w", err)
	}
	found, err := walkVault(ctx, reader)
	if err != nil {
		return ScanResult{}, err
	}

	group := u.newGroup(v, &res)
	observed, err := u.indexChanged(ctx, reader, found, recognized, &group, &res)
	if err != nil {
		return res, err
	}
	if err := group.flush(ctx); err != nil {
		return res, err
	}

	if err := u.removeGone(ctx, v, recognized, observed, &res); err != nil {
		return res, err
	}

	// A scan that stored nothing changed nothing, and a scan of an unchanged
	// vault has to stay cheap enough to run at startup.
	if res.Indexed > 0 || res.Removed > 0 {
		if err := u.maintenance.ReportChanges(ctx); err != nil {
			return res, fmt.Errorf("index upkeep for %s: %w", v.Name, err)
		}
	}

	return res, nil
}

// walkVault is every file of the vault, the most recently modified first. The walk
// is collected before any file is read so the order can be applied: recently
// modified notes are scanned first.
func walkVault(ctx context.Context, reader vaultport.VaultReader) ([]domain.FileStat, error) {
	var found []domain.FileStat
	if err := reader.WalkFiles(ctx, func(ref domain.FileStat) error {
		found = append(found, ref)
		return nil
	}); err != nil {
		return nil, err
	}
	slices.SortFunc(found, func(a, b domain.FileStat) int { return b.ModTime.Compare(a.ModTime) })
	return found, nil
}

// newGroup is the grouping that saves a vault's notes and adds each group to
// res.Indexed.
func (u Scan) newGroup(v domain.Vault, res *ScanResult) grouping {
	return grouping{write: func(ctx context.Context, notes []domain.Note) error {
		if err := u.notes.Save(ctx, v.ID, u.Cut.Execute(notes)); err != nil {
			// The failure is somewhere in a group, so say which one.
			return fmt.Errorf("index %d notes of %s, %s to %s: %w",
				len(notes), v.Name, notes[0].FileStat.Path, notes[len(notes)-1].FileStat.Path, err)
		}
		res.Indexed += len(notes)
		if u.OnProgress != nil {
			u.OnProgress(*res)
		}
		return nil
	}}
}

// indexChanged reads and groups every note whose file stat differs from the
// index's, and is the paths of the notes the walk found.
func (u Scan) indexChanged(
	ctx context.Context,
	reader vaultport.VaultReader,
	found []domain.FileStat,
	indexed map[string]domain.FileStat,
	group *grouping,
	res *ScanResult,
) (map[string]bool, error) {
	observed := make(map[string]bool, len(indexed))
	for _, ref := range found {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ref.Kind != domain.KindNote {
			// Taking the text out of a source of another kind is its own step,
			// on its own schedule.
			res.Assets++
			continue
		}
		res.Notes++
		observed[ref.Path] = true

		if previous, ok := indexed[ref.Path]; ok && !u.ShouldRebuildIndex && previous.IsUnchanged(ref) {
			res.Unchanged++
			continue
		}
		raw, isRead := readNoteBytes(ctx, reader, ref.Path, res)
		if !isRead {
			continue
		}
		if err := group.add(ctx, markdown.Parse(ref, raw), len(raw)); err != nil {
			return nil, err
		}
	}
	return observed, nil
}

// readNoteBytes is the bytes of the note at path and whether they were read. A
// note that cannot be read is counted in res and the scan goes on, and its path
// stays among those seen: the row is still about the file.
func readNoteBytes(ctx context.Context, reader vaultport.VaultReader, path string, res *ScanResult) ([]byte, bool) {
	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, vaultport.ErrNotExist):
		// A note saved, moved or deleted during a scan does not end the scan; the
		// next one sees whatever it became. A file briefly absent is what an
		// editor saving through a temporary file looks like.
		res.Vanished++
	case err != nil:
		// A permission, a broken link or a device that went away costs one file
		// and not the others.
		res.Unreadable++
	default:
		return raw, true
	}
	return nil, false
}

// removeGone takes out of the index the notes the walk did not find.
func (u Scan) removeGone(
	ctx context.Context, v domain.Vault, indexed map[string]domain.FileStat, visited map[string]bool, res *ScanResult,
) error {
	var removed []string
	for path := range indexed {
		if !visited[path] {
			removed = append(removed, path)
		}
	}
	if err := u.notes.Remove(ctx, v.ID, removed); err != nil {
		return fmt.Errorf("remove deleted notes: %w", err)
	}
	res.Removed = len(removed)
	return nil
}

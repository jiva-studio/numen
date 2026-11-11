package vault

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Scan updates the index to match one vault. The vault is authoritative: after
// a scan the index holds exactly what the scan found.
type Scan struct {
	readers     vaultport.VaultReaders
	vaults      vaultport.VaultRepository
	notes       noteport.NoteRepository
	indexedVal  noteport.FileStats
	maintenance indexport.IndexMaintenance
	backlog     sourceport.SourceBacklog

	Cut chunking.CutNotes

	ShouldRecut bool

	Walks *Walks

	ShouldRebuildIndex bool
}

// NewScan returns a Scan that reads and writes through: the vault reader, the
// vault repository, the note repository, the file stat queries of the index,
// the maintenance run after the index changed, and the backlog that names the notes cut by another recipe.
//
// Maintenance runs only when the walk found a change.
func NewScan(
	readers vaultport.VaultReaders,
	vaults vaultport.VaultRepository,
	notes noteport.NoteRepository,
	queries noteport.FileStats,
	maintenance indexport.IndexMaintenance,
	backlog sourceport.SourceBacklog,
) Scan {
	return Scan{
		readers: readers, vaults: vaults, notes: notes,
		indexedVal: queries, maintenance: maintenance, backlog: backlog,
	}
}

// ScanWithProgress runs Execute with onProgress reporting each group when non-nil.
func (u Scan) ScanWithProgress(ctx context.Context, v domain.Vault, onProgress func(vaultreading.ScanResult)) (vaultreading.ScanResult, error) {
	return u.run(ctx, v, onProgress)
}

// Execute traverses the vault once.
// Only files whose size or modification time differ from what the index holds
// are read and parsed, which keeps a scan of an unchanged vault cheap.
// One walk of a vault runs at a time among the scans sharing its turns. A walk
// writes in groups, so its copy of a note lands last however early it was read.
func (u Scan) Execute(ctx context.Context, v domain.Vault) (vaultreading.ScanResult, error) {
	return u.run(ctx, v, nil)
}

func (u Scan) run(
	ctx context.Context, v domain.Vault, onProgress func(vaultreading.ScanResult),
) (vaultreading.ScanResult, error) {
	var res vaultreading.ScanResult

	finished, err := u.Walks.startOne(ctx, v.ID)
	if err != nil {
		return vaultreading.ScanResult{}, err
	}
	defer finished()

	walk, err := u.readWalk(ctx, v)
	if err != nil {
		return vaultreading.ScanResult{}, err
	}

	group := u.newGroup(v, &res, onProgress)
	observed, err := u.indexChanged(ctx, walk, &group, &res)
	if err != nil {
		return res, err
	}
	if err := group.flush(ctx); err != nil {
		return res, err
	}

	if err := u.removeGone(ctx, v, walk.recognized, observed, &res); err != nil {
		return res, err
	}

	// A scan that stored nothing changed nothing, and a scan of an unchanged
	// vault has to stay cheap enough to run at startup.
	if res.Indexed > 0 || res.Removed > 0 {
		if err := u.maintenance.ReportChanges(ctx); err != nil {
			return res, fmt.Errorf("the index upkeep of %s failed: %w", v.Name, err)
		}
	}

	return res, nil
}

// vaultWalk is what a scan has read of a vault before it saves a note: the reader of its files, the
// files found, the file stats the index holds, and the paths of the notes to cut again.
type vaultWalk struct {
	reader     vaultport.VaultReader
	found      []domain.FileStat
	recognized map[string]domain.FileStat
	recutPaths map[string]bool
}

func (u Scan) readWalk(ctx context.Context, v domain.Vault) (vaultWalk, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return vaultWalk{}, err
	}
	// The rows the walk writes point at the vault's own row. What the vault is
	// called and where it is stay as the list has them.
	if err := u.vaults.Register(ctx, v.ID); err != nil {
		return vaultWalk{}, fmt.Errorf("the index cannot record the vault %s: %w", v.Name, err)
	}
	recognized, err := u.indexedVal.GetFileStats(ctx, v.ID)
	if err != nil {
		return vaultWalk{}, fmt.Errorf("the index of %s cannot be read: %w", v.Name, err)
	}
	found, err := walkVault(ctx, reader)
	if err != nil {
		return vaultWalk{}, err
	}
	recutPaths, err := u.getRecutPaths(ctx, v)
	if err != nil {
		return vaultWalk{}, err
	}
	return vaultWalk{reader: reader, found: found, recognized: recognized, recutPaths: recutPaths}, nil
}

func (u Scan) getRecutPaths(ctx context.Context, v domain.Vault) (map[string]bool, error) {
	if !u.ShouldRecut || u.backlog == nil {
		return nil, nil
	}
	recipes := []string{u.Cut.Recipe(), chunking.GetTextVersionRecipe(u.Cut.Sizes)}
	paths, err := u.backlog.GetSourcesWithOtherRecipe(ctx, v.ID, domain.KindNote, recipes, math.MaxInt32)
	if err != nil {
		return nil, fmt.Errorf("the notes of %s chunked with another method cannot be read: %w", v.Name, err)
	}
	recutPaths := make(map[string]bool, len(paths))
	for _, path := range paths {
		recutPaths[path] = true
	}
	return recutPaths, nil
}

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

func (u Scan) newGroup(v domain.Vault, res *vaultreading.ScanResult, onProgress func(vaultreading.ScanResult)) grouping {
	return grouping{write: func(ctx context.Context, notes []domain.Note) error {
		if err := u.notes.Save(ctx, v.ID, u.Cut.Execute(notes)); err != nil {
			// The failure is somewhere in a group, so say which one.
			return fmt.Errorf("%d notes of %s, %s to %s, cannot be indexed: %w",
				len(notes), v.Name, notes[0].FileStat.Path, notes[len(notes)-1].FileStat.Path, err)
		}
		res.Indexed += len(notes)
		if onProgress != nil {
			onProgress(*res)
		}
		return nil
	}}
}

func (u Scan) indexChanged(
	ctx context.Context, walk vaultWalk, group *grouping, res *vaultreading.ScanResult,
) (map[string]bool, error) {
	reader, found, indexed, recutPaths := walk.reader, walk.found, walk.recognized, walk.recutPaths
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

		if previous, ok := indexed[ref.Path]; ok && !u.ShouldRebuildIndex && !recutPaths[ref.Path] && previous.IsUnchanged(ref) {
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

func readNoteBytes(ctx context.Context, reader vaultport.VaultReader, path string, res *vaultreading.ScanResult) ([]byte, bool) {
	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, edit.ErrNotExist):
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

func (u Scan) removeGone(
	ctx context.Context, v domain.Vault, indexed map[string]domain.FileStat, visited map[string]bool, res *vaultreading.ScanResult,
) error {
	var removed []string
	for path := range indexed {
		if !visited[path] {
			removed = append(removed, path)
		}
	}
	if err := u.notes.Remove(ctx, v.ID, removed); err != nil {
		return fmt.Errorf("the deleted notes of %s cannot be removed from the index: %w", v.Name, err)
	}
	res.Removed = len(removed)
	return nil
}

// Rebuilding returns the walk set to rebuild the index, or to trust it where a note is unchanged.
func (u Scan) Rebuilding(rebuild bool) Scan {
	u.ShouldRebuildIndex = rebuild
	return u
}

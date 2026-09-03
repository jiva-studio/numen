package vault

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Scan brings the index up to date with one vault. The vault is
// authoritative: whatever the scan finds is what the index says afterwards.
type Scan struct {
	Readers     port.VaultReaders
	Vaults      port.VaultRepository
	Notes       port.NoteRepository
	Known       port.NoteQueries
	Maintenance port.IndexMaintenance

	// RebuildIndex reads every file and puts it in the index again, whatever the
	// index remembers about it.
	//
	// What it remembers is a path, a size and a modification time, so a file whose
	// content changed while those did not is skipped — an archive restored by
	// `unzip`, a tree brought over by `rsync -tc`. Rebuilding is the way out of
	// that, and the only one.
	RebuildIndex bool

	// OnProgress, if set, is called each time a group of notes is written. A
	// scan of a large vault takes a minute, and something has to be able to say
	// so while it happens. What is done with that is the caller's business.
	OnProgress func(ScanResult)
}

// ScanResult reports what a scan did, in the terms the user cares about.
//
// `Seen` counts notes and nothing else, and every other number here is about
// those notes. What the walk found that is not a note is `Assets`.
type ScanResult struct {
	Seen       int // notes found in the vault
	Assets     int // sources of another kind found in the vault
	Indexed    int // parsed and written, because they were new or had changed
	Unchanged  int // skipped on size and modification time alone
	Removed    int // in the index, no longer on disk
	Vanished   int // walked, but gone by the time it was read
	Unreadable int // walked, still there, and the read refused
}

// Execute walks the vault once.
//
// Only files whose size or modification time differ from what the index holds
// are read and parsed; the rest are not opened at all. That is what keeps a scan
// of an unchanged vault cheap enough to run at startup.
func (u Scan) Execute(ctx context.Context, v domain.Vault) (ScanResult, error) {
	var res ScanResult

	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	if err := u.Vaults.Save(ctx, v); err != nil {
		return res, fmt.Errorf("register vault: %w", err)
	}

	known, err := u.Known.Fingerprints(ctx, v.ID)
	if err != nil {
		return res, fmt.Errorf("read index: %w", err)
	}

	// The walk is collected before anything is read, so that the order can be
	// chosen. A vault has a working set and an archive, and they are not the
	// same size: notes touched recently are what the person is looking for while
	// the scan runs, so they are indexed first.
	var found []domain.FileRef
	if err := reader.Walk(ctx, func(ref domain.FileRef) error {
		found = append(found, ref)
		return nil
	}); err != nil {
		return res, err
	}
	slices.SortFunc(found, func(a, b domain.FileRef) int { return cmp.Compare(b.MTime, a.MTime) })

	group := grouping{write: func(ctx context.Context, notes []domain.Note) error {
		if err := u.Notes.Save(ctx, v.ID, notes); err != nil {
			// The failure is somewhere in a group, so say which one.
			return fmt.Errorf("index %d notes of %s, %s to %s: %w",
				len(notes), v.Name, notes[0].Ref.Path, notes[len(notes)-1].Ref.Path, err)
		}
		res.Indexed += len(notes)
		if u.OnProgress != nil {
			u.OnProgress(res)
		}
		return nil
	}}

	seen := make(map[string]bool, len(known))
	for _, ref := range found {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if ref.Kind != domain.KindNote {
			// A source of another kind is what the vault holds, said out loud.
			// Taking its text out of it is its own step, on its own schedule.
			res.Assets++
			continue
		}
		res.Seen++
		seen[ref.Path] = true

		if previous, ok := known[ref.Path]; ok && !u.RebuildIndex && previous.Unchanged(ref) {
			res.Unchanged++
			continue
		}

		raw, err := reader.Read(ctx, ref.Path)
		if errors.Is(err, fs.ErrNotExist) {
			// The vault is edited while it is read — that is what it means for
			// files to be the source of truth. A note saved, moved or deleted
			// during a scan must not end the scan; the next one will see
			// whatever it became.
			// The path stays in `seen`: a file that was there a moment ago and
			// is briefly absent is what every editor that saves through a
			// temporary file looks like. Removing its row would take the note
			// out of search until the next scan.
			res.Vanished++
			continue
		}
		if err != nil {
			// One file nobody can read — a permission, a broken link, a device
			// that went away — must not cost the others. The path stays in
			// `seen`: the file is there and its row is still about it.
			res.Unreadable++
			continue
		}
		if err := group.add(ctx, markdown.Parse(ref, raw), len(raw)); err != nil {
			return res, err
		}
	}
	if err := group.flush(ctx); err != nil {
		return res, err
	}

	var gone []string
	for path := range known {
		if !seen[path] {
			gone = append(gone, path)
		}
	}
	if err := u.Notes.Remove(ctx, v.ID, gone); err != nil {
		return res, fmt.Errorf("remove deleted notes: %w", err)
	}
	res.Removed = len(gone)

	// A scan that stored nothing changed nothing, and a scan of an unchanged
	// vault has to stay cheap enough to run at startup.
	if res.Indexed > 0 || res.Removed > 0 {
		if err := u.Maintenance.Changed(ctx); err != nil {
			return res, fmt.Errorf("index upkeep for %s: %w", v.Name, err)
		}
	}

	return res, nil
}

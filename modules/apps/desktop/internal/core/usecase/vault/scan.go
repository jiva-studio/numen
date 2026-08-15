package vault

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Scan brings the index up to date with one vault. The vault is
// authoritative: whatever the scan finds is what the index says afterwards.
type Scan struct {
	Readers    port.VaultReaders
	Vaults     port.VaultRepository
	Notes      port.NoteRepository
	Known      port.NoteQueries
	Statistics port.IndexStatistics

	// OnProgress, if set, is called each time a group of notes is written. A
	// scan of a large vault takes a minute, and something has to be able to say
	// so while it happens. What is done with that is the caller's business.
	OnProgress func(ScanResult)
}

// Notes are written in groups rather than one at a time, because the cost of
// storing a note is mostly the cost of the boundary around it.
//
// The count is where the gain flattens out: measured on a vault of ten thousand
// notes, groups of fifty take less than half the time of one note at a time,
// groups of five hundred a third, and ten times that buys almost nothing more.
//
// The size is a second bound for the same group, and it is the one that matters
// on a vault this was not sized for: a folder of long transcripts would
// otherwise hold five hundred whole documents in memory before writing any of
// them.
const (
	notesPerWrite = 500
	bytesPerWrite = 8 << 20
)

// ScanResult reports what a scan did, in the terms the user cares about.
type ScanResult struct {
	Seen      int // markdown files found in the vault
	Indexed   int // parsed and written, because they were new or had changed
	Unchanged int // skipped on size and modification time alone
	Removed   int // in the index, no longer on disk
	Vanished  int // walked, but gone by the time it was read
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

	var (
		pending      []domain.Note
		pendingBytes int
	)
	write := func() error {
		if len(pending) == 0 {
			return nil
		}
		if err := u.Notes.Save(ctx, v.ID, pending); err != nil {
			return fmt.Errorf("index: %w", err)
		}
		res.Indexed += len(pending)
		pending, pendingBytes = pending[:0], 0
		if u.OnProgress != nil {
			u.OnProgress(res)
		}
		return nil
	}

	seen := make(map[string]bool, len(known))
	for _, ref := range found {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		res.Seen++
		seen[ref.Path] = true

		if previous, ok := known[ref.Path]; ok && previous.Unchanged(ref) {
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
			return res, fmt.Errorf("read %s: %w", ref.Path, err)
		}
		pending = append(pending, markdown.Parse(ref, raw))
		pendingBytes += len(raw)
		if len(pending) >= notesPerWrite || pendingBytes >= bytesPerWrite {
			if err := write(); err != nil {
				return res, err
			}
		}
	}
	if err := write(); err != nil {
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

	// A scan that changed nothing changed nothing to measure, and a scan of an
	// unchanged vault has to stay cheap enough to run at startup.
	if res.Indexed > 0 || res.Removed > 0 {
		if err := u.Statistics.Update(ctx); err != nil {
			return res, fmt.Errorf("measure index: %w", err)
		}
	}

	return res, nil
}

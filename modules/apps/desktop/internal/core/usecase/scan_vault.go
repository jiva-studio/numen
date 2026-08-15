// Package service holds the use cases. It orchestrates ports and knows no more
// about a filesystem or a database than the domain does.
package usecase

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ScanVault brings the index up to date with one vault. The vault is
// authoritative: whatever the scan finds is what the index says afterwards.
type ScanVault struct {
	Vaults port.VaultReaders
	Notes  port.NoteRepository
}

// ScanResult reports what a scan did, in the terms the user cares about.
type ScanResult struct {
	Seen      int // markdown files found in the vault
	Indexed   int // parsed and written, because they were new or had changed
	Unchanged int // skipped on size and modification time alone
	Removed   int // in the index, no longer on disk
}

func (r ScanResult) String() string {
	return fmt.Sprintf("%d notes: %d indexed, %d unchanged, %d removed",
		r.Seen, r.Indexed, r.Unchanged, r.Removed)
}

// Scan walks the vault once.
//
// Only files whose size or modification time differ from what the index holds
// are read and parsed; the rest are not opened at all. That is what keeps a scan
// of an unchanged vault cheap enough to run at startup.
func (u ScanVault) Execute(ctx context.Context, v domain.Vault) (ScanResult, error) {
	var res ScanResult

	reader, err := u.Vaults.Open(v)
	if err != nil {
		return res, err
	}

	if err := u.Notes.RegisterVault(ctx, v); err != nil {
		return res, fmt.Errorf("register vault: %w", err)
	}

	known, err := u.Notes.Known(ctx, v.ID)
	if err != nil {
		return res, fmt.Errorf("read index: %w", err)
	}

	seen := make(map[string]bool, len(known))
	walkErr := reader.Walk(ctx, func(ref domain.FileRef) error {
		res.Seen++
		seen[ref.Path] = true

		if prev, ok := known[ref.Path]; ok && prev.Unchanged(ref) {
			res.Unchanged++
			return nil
		}

		raw, err := reader.Read(ctx, ref.Path)
		if err != nil {
			return fmt.Errorf("read %s: %w", ref.Path, err)
		}
		if err := u.Notes.Put(ctx, v.ID, markdown.Parse(ref, raw)); err != nil {
			return fmt.Errorf("index %s: %w", ref.Path, err)
		}
		res.Indexed++
		return nil
	})
	if walkErr != nil {
		return res, walkErr
	}

	var gone []string
	for path := range known {
		if !seen[path] {
			gone = append(gone, path)
		}
	}
	if err := u.Notes.Delete(ctx, v.ID, gone); err != nil {
		return res, fmt.Errorf("remove deleted notes: %w", err)
	}
	res.Removed = len(gone)

	return res, nil
}

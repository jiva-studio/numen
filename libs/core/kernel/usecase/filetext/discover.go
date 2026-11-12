package filetext

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"

	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// sourceQueries are the queries on the sources the index records.
type sourceQueries struct {
	records sourceport.SourceRecords
	backlog sourceport.SourceBacklog
	lookup  sourceport.RemovalLookup
}

// discoverer is the first pass: it records the files that have text, and moves or removes the sources of
// the files that went.
type discoverer struct {
	sources       sourceport.SourceRepository
	records       sourceport.SourceRecords
	lookup        sourceport.RemovalLookup
	shouldRebuild bool
}

func newDiscoverer(u Index) discoverer {
	return discoverer{sources: u.sources, records: u.queries.records, lookup: u.queries.lookup, shouldRebuild: u.ShouldRebuildIndex}
}

func (u discoverer) discover(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	res *vaultreading.IndexResult,
) (map[string]domain.FileStat, []string, error) {
	recognized, err := u.readKnownStats(ctx, v)
	if err != nil {
		return nil, nil, err
	}
	noteStats, err := u.readNoteStats(ctx, v)
	if err != nil {
		return nil, nil, err
	}
	found, notePaths, err := u.walkAndRecord(ctx, v, on, recognized, noteStats, res)
	if err != nil {
		return nil, nil, err
	}
	return noteStats, notePaths, u.removeGoneFiles(ctx, v, on, recognized, found, res)
}

func (u discoverer) readKnownStats(ctx context.Context, v domain.Vault) (map[string]domain.FileStat, error) {
	stats, err := u.records.GetFileStats(ctx, v.ID, domain.KindFile)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	return stats, nil
}

func (u discoverer) readNoteStats(ctx context.Context, v domain.Vault) (map[string]domain.FileStat, error) {
	sources, err := u.records.GetRecognizedSources(ctx, v.ID, domain.KindNote)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	if len(sources) == 0 {
		return nil, nil
	}
	stats, err := u.records.GetFileStats(ctx, v.ID, domain.KindNote)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	noteStats := make(map[string]domain.FileStat, len(sources))
	for _, one := range sources {
		if stat, ok := stats[one.Path]; ok {
			noteStats[one.Path] = stat
		}
	}
	return noteStats, nil
}

func (u discoverer) walkAndRecord(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	stats map[string]domain.FileStat,
	noteStats map[string]domain.FileStat,
	res *vaultreading.IndexResult,
) (map[string]bool, []string, error) {
	found := make(map[string]bool)
	var notePaths []string
	err := on.files.WalkFiles(ctx, func(ref domain.FileStat) error {
		switch ref.Kind {
		case domain.KindFile:
			found[ref.Path] = true
			return u.recordFile(ctx, v, on, stats, ref, res)
		case domain.KindNote:
			hasCut, err := u.checkNote(ctx, on, noteStats, ref, res)
			if err != nil {
				return err
			}
			if hasCut {
				notePaths = append(notePaths, ref.Path)
			}
			return nil
		default:
			return nil
		}
	})
	return found, notePaths, err
}

func (u discoverer) recordFile(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	stats map[string]domain.FileStat,
	ref domain.FileStat,
	res *vaultreading.IndexResult,
) error {
	previous, isKnown := stats[ref.Path]
	if !isKnown {
		hasText, err := on.texts.IsTextSource(ctx, ref.Path)
		if err != nil {
			return fmt.Errorf("find the text of %s: %w", ref.Path, err)
		}
		if !hasText {
			return nil
		}
	}
	res.Seen++
	if isKnown && !u.shouldRebuild && previous.IsUnchanged(ref) {
		res.Unchanged++
		return nil
	}
	if err := u.sources.SaveSource(ctx, v.ID, domain.Source{FileStat: ref}); err != nil {
		return fmt.Errorf("record %s: %w", ref.Path, err)
	}
	res.Recorded++
	return nil
}

func (u discoverer) checkNote(
	ctx context.Context,
	on openVault,
	noteStats map[string]domain.FileStat,
	ref domain.FileStat,
	res *vaultreading.IndexResult,
) (bool, error) {
	previous, isKnown := noteStats[ref.Path]
	if !isKnown {
		hasText, err := on.texts.IsTextSource(ctx, ref.Path)
		if err != nil {
			return false, fmt.Errorf("find the text of %s: %w", ref.Path, err)
		}
		if !hasText {
			return false, nil
		}
	}
	res.Seen++
	if isKnown && !u.shouldRebuild && previous.IsUnchanged(ref) {
		res.Unchanged++
		return false, nil
	}
	res.Recorded++
	return true, nil
}

func (u discoverer) removeGoneFiles(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	stats map[string]domain.FileStat,
	found map[string]bool,
	res *vaultreading.IndexResult,
) error {
	var removed []string
	for path := range stats {
		if !found[path] {
			removed = append(removed, path)
		}
	}
	if len(removed) == 0 {
		return nil
	}
	moved, err := u.moveRenamed(ctx, v, on, removed, getAddedPaths(stats, found))
	if err != nil {
		return err
	}
	paths := slices.DeleteFunc(removed, func(path string) bool { return slices.Contains(moved, path) })
	if len(paths) == 0 {
		return nil
	}
	slices.Sort(paths)
	if err := u.sources.RemoveSources(ctx, v.ID, domain.KindFile, paths); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	res.Removed += len(paths)
	return nil
}

func (u discoverer) moveRenamed(ctx context.Context, v domain.Vault, on openVault, removed, added []string) ([]string, error) {
	if len(added) == 0 {
		return nil, nil
	}
	slices.Sort(added)
	sources, err := sourcedomain.FindRemoved(removed,
		func(path string) ([]domain.FileStat, error) { return u.lookup.GetSourcesUnder(ctx, v.ID, path) },
		func(path string) (sourcedomain.Text, bool, error) { return u.lookup.GetSourceText(ctx, v.ID, path) })
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	moved, err := sourcedomain.MoveRenamed(sources, added,
		func(path string) (domain.FileStat, error) { return on.files.StatFile(ctx, path) },
		func(path string) (string, error) { return getContentHash(ctx, on.files, path) },
		func(from, to string) error { return u.sources.MoveSources(ctx, v.ID, from, to) })
	if err != nil {
		return nil, fmt.Errorf("move: %w", err)
	}
	return moved, nil
}

func getContentHash(ctx context.Context, reader vaultport.VaultReader, path string) (string, error) {
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", err
	}
	return domain.HashContent(raw), nil
}

func getAddedPaths(stats map[string]domain.FileStat, found map[string]bool) []string {
	var added []string
	for path := range found {
		if _, isKnown := stats[path]; !isKnown {
			added = append(added, path)
		}
	}
	return added
}

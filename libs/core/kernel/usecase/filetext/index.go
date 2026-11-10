package filetext

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/note/domain"

	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// sourcesPerQuery is the page size of the query for sources that need text,
// so a library of any size is processed in pages.
const sourcesPerQuery = 100

// versionReader names the text a source is cut from: the live text version of
// the file. It is part of every recipe, and changes when the text or the
// offsets a version gives do.
const versionReader = "text-version-1"

// Index cuts the live text versions of the files a vault holds into chunks, in two passes.
// The first records the files that have text or that a plugin can extract text
// from, without opening them. The second reads the live text of every file that
// needs it and cuts it into chunks, one write per file.
type Index struct {
	readers vaultport.VaultReaders
	sources sourceport.SourceRepository
	records sourceport.SourceRecords
	backlog sourceport.SourceBacklog
	lookup  sourceport.RemovalLookup
	texts   text.Readers

	Sizes chunking.Sizes

	Legibility chunking.Legibility

	ShouldRebuildIndex bool

	OnProgress func(indexdomain.IndexResult)

	IsSkipped func(path string) bool

	ErrorHandler window.ErrorHandler

	ReadNote func(ctx context.Context, v domain.Vault, path string) error
}

// NewIndex creates an Index from the vault readers, the source repository
// that stores sources and chunks, the queries that report which sources the
// index records, which still need text and the text kept for the ones that left,
// and the text readers. Nil texts means no file has text.
func NewIndex(
	readers vaultport.VaultReaders,
	sources sourceport.SourceRepository,
	records sourceport.SourceRecords,
	backlog sourceport.SourceBacklog,
	lookup sourceport.RemovalLookup,
	texts text.Readers,
) Index {
	return Index{readers: readers, sources: sources, records: records, backlog: backlog, lookup: lookup, texts: texts}
}

// IndexWithProgress runs Execute with onProgress reporting each step when
// non-nil, and leaves uncut every source isSkipped names when non-nil.
func (u Index) IndexWithProgress(
	ctx context.Context, v domain.Vault, onProgress func(indexdomain.IndexResult), isSkipped func(path string) bool,
) (indexdomain.IndexResult, error) {
	if onProgress != nil {
		u.OnProgress = onProgress
	}
	if isSkipped != nil {
		u.IsSkipped = isSkipped
	}
	return u.Execute(ctx, v)
}

// Execute brings the chunks of one vault's file texts up to date with what is
// on disk.
func (u Index) Execute(ctx context.Context, v domain.Vault) (indexdomain.IndexResult, error) {
	var res indexdomain.IndexResult

	on, err := u.open(v)
	if err != nil {
		return res, err
	}
	noteStats, notePaths, err := u.discover(ctx, v, on, &res)
	if err != nil {
		return res, err
	}
	notePaths, err = u.dropOutdatedText(ctx, v, on, notePaths, &res)
	if err != nil {
		return res, err
	}
	return res, u.cut(ctx, v, on, noteStats, notePaths, &res)
}

func (u Index) discover(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	res *indexdomain.IndexResult,
) (map[string]domain.FileStat, []string, error) {
	recognized, err := u.readKnown(ctx, v)
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
	return noteStats, notePaths, u.removeGone(ctx, v, on, recognized, found, res)
}

func (u Index) readKnown(ctx context.Context, v domain.Vault) (map[string]domain.FileStat, error) {
	stats, err := u.records.GetFileStats(ctx, v.ID, domain.KindFile)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	return stats, nil
}

func (u Index) readNoteStats(ctx context.Context, v domain.Vault) (map[string]domain.FileStat, error) {
	sources, err := u.records.GetRecognisedSources(ctx, v.ID, domain.KindNote)
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

func (u Index) walkAndRecord(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	stats map[string]domain.FileStat,
	noteStats map[string]domain.FileStat,
	res *indexdomain.IndexResult,
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

func (u Index) recordFile(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	stats map[string]domain.FileStat,
	ref domain.FileStat,
	res *indexdomain.IndexResult,
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
	if isKnown && !u.ShouldRebuildIndex && previous.IsUnchanged(ref) {
		res.Unchanged++
		return nil
	}
	if err := u.sources.SaveSource(ctx, v.ID, domain.Source{FileStat: ref}); err != nil {
		return fmt.Errorf("record %s: %w", ref.Path, err)
	}
	res.Recorded++
	return nil
}

func (u Index) checkNote(
	ctx context.Context,
	on openVault,
	noteStats map[string]domain.FileStat,
	ref domain.FileStat,
	res *indexdomain.IndexResult,
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
	if isKnown && !u.ShouldRebuildIndex && previous.IsUnchanged(ref) {
		res.Unchanged++
		return false, nil
	}
	res.Recorded++
	return true, nil
}

func (u Index) removeGone(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	stats map[string]domain.FileStat,
	found map[string]bool,
	res *indexdomain.IndexResult,
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
	moved, err := u.moveRenamed(ctx, v, on, removed, getAdded(stats, found))
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

func (u Index) moveRenamed(ctx context.Context, v domain.Vault, on openVault, removed, added []string) ([]string, error) {
	if len(added) == 0 {
		return nil, nil
	}
	slices.Sort(added)
	sources, err := findRemovedSources(ctx, u.lookup, v.ID, removed)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	moved, err := moveRenamedSources(ctx, u.sources, on.files, v.ID, sources, added)
	if err != nil {
		return nil, fmt.Errorf("move: %w", err)
	}
	return moved, nil
}

func getAdded(stats map[string]domain.FileStat, found map[string]bool) []string {
	var added []string
	for path := range found {
		if _, isKnown := stats[path]; !isKnown {
			added = append(added, path)
		}
	}
	return added
}

func (u Index) dropOutdatedText(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	notePaths []string,
	res *indexdomain.IndexResult,
) ([]string, error) {
	cut, err := u.records.GetRecognisedSources(ctx, v.ID, domain.KindFile)
	if err != nil {
		return notePaths, fmt.Errorf("read index: %w", err)
	}
	for _, one := range cut {
		if err := ctx.Err(); err != nil {
			return notePaths, err
		}
		if err := u.recordIfOutdated(ctx, v, on, one, res); err != nil {
			return notePaths, err
		}
	}
	notes, err := u.records.GetRecognisedSources(ctx, v.ID, domain.KindNote)
	if err != nil {
		return notePaths, fmt.Errorf("read index: %w", err)
	}
	for _, one := range notes {
		if err := ctx.Err(); err != nil {
			return notePaths, err
		}
		outdated, err := u.isOutdated(ctx, on, one)
		if err != nil {
			return notePaths, err
		}
		if outdated {
			res.Outdated++
			notePaths = append(notePaths, one.Path)
		}
	}
	return notePaths, nil
}

func (u Index) recordIfOutdated(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	source sourcedomain.Text,
	res *indexdomain.IndexResult,
) error {
	ref, outdated, err := u.statIfOutdated(ctx, on, source)
	if err != nil || !outdated {
		return err
	}
	if err := u.sources.SaveSource(ctx, v.ID, domain.Source{FileStat: ref}); err != nil {
		return fmt.Errorf("record %s: %w", source.Path, err)
	}
	res.Outdated++
	return nil
}

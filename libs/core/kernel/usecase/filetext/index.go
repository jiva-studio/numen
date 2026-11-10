package filetext

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

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

func (u Index) isOutdated(ctx context.Context, on openVault, source sourcedomain.Text) (bool, error) {
	_, outdated, err := u.statIfOutdated(ctx, on, source)
	return outdated, err
}

func (u Index) statIfOutdated(
	ctx context.Context,
	on openVault,
	source sourcedomain.Text,
) (domain.FileStat, bool, error) {
	live, err := on.texts.FindLiveVersion(ctx, source.Path, source.ContentHash)
	switch {
	case err == nil && string(live) == source.Producer:
		return domain.FileStat{}, false, nil
	case err != nil && !errors.Is(err, text.ErrNoText):
		return domain.FileStat{}, false, fmt.Errorf("find the live version of %s: %w", source.Path, err)
	}
	ref, err := on.files.StatFile(ctx, source.Path)
	if edit.IsNoNote(err) {
		return domain.FileStat{}, false, nil
	}
	if err != nil {
		return domain.FileStat{}, false, fmt.Errorf("stat %s: %w", source.Path, err)
	}
	return ref, true, nil
}

func (u Index) cut(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	noteStats map[string]domain.FileStat,
	notePaths []string,
	res *indexdomain.IndexResult,
) error {
	sizes := u.sizes()
	recognized := []string{recipe(versionReader, sizes)}
	run := cutRun{tried: map[string]bool{}, sizes: sizes, res: res}

	questions := []func(context.Context) ([]string, error){
		func(ctx context.Context) ([]string, error) {
			return u.backlog.GetUnchunkedSources(ctx, v.ID, domain.KindFile, sourcesPerQuery)
		},
		func(ctx context.Context) ([]string, error) {
			return u.backlog.GetSourcesWithOtherRecipe(ctx, v.ID, domain.KindFile, recognized, sourcesPerQuery)
		},
	}
	if len(notePaths) > 0 || len(noteStats) > 0 {
		questions = append(questions, func(ctx context.Context) ([]string, error) {
			return u.getNotePathsToCut(ctx, v, noteStats, notePaths, recognized)
		})
	}
	for _, ask := range questions {
		for {
			paths, err := ask(ctx)
			if err != nil {
				return err
			}
			freshCount, err := u.chunkNewPaths(ctx, v, on, paths, run)
			if err != nil {
				return err
			}
			if freshCount == 0 {
				break
			}
		}
	}
	return nil
}

func (u Index) getNotePathsToCut(
	ctx context.Context,
	v domain.Vault,
	noteStats map[string]domain.FileStat,
	notePaths []string,
	recognized []string,
) ([]string, error) {
	if len(noteStats) == 0 {
		return notePaths, nil
	}
	other, err := u.backlog.GetSourcesWithOtherRecipe(ctx, v.ID, domain.KindNote, recognized, sourcesPerQuery)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(notePaths)
	for _, path := range other {
		if _, ok := noteStats[path]; ok {
			out = append(out, path)
		}
	}
	return out, nil
}

// cutRun is the state one cut shares across its queries: the paths tried, the chunk
// sizes and the result.
type cutRun struct {
	tried map[string]bool
	sizes chunking.Sizes
	res   *indexdomain.IndexResult
}

func (u Index) chunkNewPaths(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	paths []string,
	run cutRun,
) (int, error) {
	freshCount := 0
	for _, path := range paths {
		if run.tried[path] {
			continue
		}
		run.tried[path] = true
		if u.IsSkipped != nil && u.IsSkipped(path) {
			continue
		}
		freshCount++
		if err := ctx.Err(); err != nil {
			return freshCount, err
		}
		if err := u.source(ctx, v, on, path, run.sizes, run.res); err != nil {
			return freshCount, err
		}
	}
	return freshCount, nil
}

// IndexOne brings a single file's chunks up to date with its live text
// version. It is what the index does when the versions of a file change.
func (u Index) IndexOne(ctx context.Context, v domain.Vault, path string) (indexdomain.IndexResult, error) {
	var res indexdomain.IndexResult
	on, err := u.open(v)
	if err != nil {
		return res, err
	}
	return res, u.source(ctx, v, on, path, u.sizes(), &res)
}

func (u Index) source(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	path string,
	sizes chunking.Sizes,
	res *indexdomain.IndexResult,
) error {
	res.CurrentSource = path
	u.progress(*res)

	ref, err := on.files.StatFile(ctx, path)
	if edit.IsNoNote(err) {
		res.Vanished++
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	hash, err := on.texts.GetContentHash(ctx, path)
	if errors.Is(err, edit.ErrNotExist) {
		res.Vanished++
		return nil
	}
	if err != nil {
		if ctx.Err() != nil {
			return err
		}
		// A file that will not open is one file. Stopping here records nothing,
		// and every later run stops at the same place, leaving every file after
		// it unread.
		res.Unreadable++
		return nil
	}
	doc, err := on.texts.GetLiveText(ctx, path, hash)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if ref.Kind == domain.KindNote && errors.Is(err, text.ErrNoText) {
			return u.dropVersionOfNote(ctx, v, path)
		}
		return u.saveWithoutText(ctx, v, domain.Source{FileStat: ref, ContentHash: hash}, err, res)
	}

	chunks := chunksOf(doc, sizes, u.Legibility)
	extraction := domain.SourceChunks{
		Source: domain.Source{
			FileStat:    ref,
			ContentHash: hash,
			Recipe:      recipe(versionReader, sizes),
			Producer:    string(doc.Version().GetID()),
		},
		Chunks: chunks,
	}
	if err := u.sources.SaveExtraction(ctx, v.ID, extraction); err != nil {
		return fmt.Errorf("write the chunks of %s: %w", path, err)
	}
	res.Extracted++
	res.Chunks += countChunks(chunks)
	u.progress(*res)
	return nil
}

func (u Index) saveWithoutText(
	ctx context.Context,
	v domain.Vault,
	source domain.Source,
	readErr error,
	res *indexdomain.IndexResult,
) error {
	res.Unreadable++
	if !errors.Is(readErr, text.ErrNoText) {
		u.reportError(readErr)
	}
	if err := u.sources.SaveExtraction(ctx, v.ID, domain.SourceChunks{Source: source}); err != nil {
		return fmt.Errorf("write the chunks of %s: %w", source.FileStat.Path, err)
	}
	return nil
}

func (u Index) dropVersionOfNote(ctx context.Context, v domain.Vault, path string) error {
	indexed, ok, err := u.lookup.GetSourceText(ctx, v.ID, path)
	if err != nil {
		return fmt.Errorf("read the index of %s: %w", path, err)
	}
	if !ok || indexed.Producer == "" || u.ReadNote == nil {
		return nil
	}
	if err := u.ReadNote(ctx, v, path); err != nil {
		return fmt.Errorf("read the note %s again: %w", path, err)
	}
	return nil
}

func (u Index) reportError(err error) {
	if u.ErrorHandler != nil {
		u.ErrorHandler(err)
	}
}

func chunksOf(doc domain.VersionText, sizes chunking.Sizes, reads chunking.Legibility) []domain.Chunk {
	return chunking.Chunks(doc.Text, doc.GetHeadings(), sizes, reads, func(c chunking.Chunk, isLarge bool) domain.Chunk {
		out := chunkAt(doc, c)
		// The name of a section is kept on the chunk that begins it, and on that
		// one only.
		if isLarge {
			out.Opens = doc.GetHeadingsAt(c.Start)
		}
		return out
	})
}

func chunkAt(doc domain.VersionText, c chunking.Chunk) domain.Chunk {
	return domain.Chunk{
		Start:    c.Start,
		Length:   c.Length,
		Location: doc.Locate(c.Start),
		Text:     c.ExtractText(doc.Text),
	}
}

func countChunks(chunks []domain.Chunk) int {
	n := len(chunks)
	for _, c := range chunks {
		n += len(c.Small)
	}
	return n
}

func recipe(reader string, s chunking.Sizes) string {
	return fmt.Sprintf("%s/large=%d+%d/small=%d+%d/limit=%d",
		reader, s.Large, s.LargeOverlap, s.Small, s.SmallOverlap, s.Limit)
}

// openVault is one vault as a cut reads it: its files and the text versions of them.
type openVault struct {
	files vaultport.VaultReader
	texts text.VaultTexts
}

func (u Index) open(v domain.Vault) (openVault, error) {
	files, err := u.readers.Open(v)
	if err != nil {
		return openVault{}, err
	}
	if u.texts == nil {
		return openVault{files: files, texts: noTexts{files: files}}, nil
	}
	texts, err := u.texts.Open(v)
	if err != nil {
		return openVault{}, err
	}
	return openVault{files: files, texts: texts}, nil
}

// noTexts is a vault with no text versions: every file has no text, and no
// text is requested.
type noTexts struct {
	files vaultport.VaultReader
}

// GetContentHash hashes the file from a stream on every call: with no versions there is no record to keep.
func (t noTexts) GetContentHash(ctx context.Context, path string) (_ string, err error) {
	file, err := t.files.Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return domain.HashReader(file)
}

func (noTexts) FindLiveVersion(context.Context, string, string) (domain.TextVersionID, error) {
	return "", text.ErrNoText
}

func (noTexts) GetLiveText(context.Context, string, string) (domain.VersionText, error) {
	return domain.VersionText{}, text.ErrNoText
}

func (noTexts) GetVersionText(context.Context, string, domain.TextVersionID) (domain.VersionText, error) {
	return domain.VersionText{}, text.ErrNoText
}

func (noTexts) IsTextSource(context.Context, string) (bool, error) { return false, nil }

// Recipe returns the recipe that names the version reader and the chunk sizes.
func (u Index) Recipe() string { return recipe(versionReader, u.sizes()) }

func (u Index) sizes() chunking.Sizes { return u.Sizes.Resolve() }

func (u Index) progress(res indexdomain.IndexResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}

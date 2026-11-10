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

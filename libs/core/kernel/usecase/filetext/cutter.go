package filetext

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
)

// cutter is the second pass: it cuts the live text of the files that need it into chunks.
type cutter struct {
	sources      sourceport.SourceRepository
	backlog      sourceport.SourceBacklog
	lookup       sourceport.RemovalLookup
	sizes        chunking.Sizes
	legibility   chunking.Legibility
	isSkipped    func(path string) bool
	errorHandler window.ErrorHandler
	readNote     func(ctx context.Context, v domain.Vault, path string) error
	onProgress   func(vaultreading.IndexResult)
}

func newCutter(u Index, onProgress func(vaultreading.IndexResult), isSkipped func(path string) bool) cutter {
	if onProgress == nil {
		onProgress = func(vaultreading.IndexResult) {}
	}
	if isSkipped == nil {
		isSkipped = func(string) bool { return false }
	}
	errorHandler, readNote := u.errorHandler, u.readNote
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	if readNote == nil {
		readNote = func(context.Context, domain.Vault, string) error { return nil }
	}
	return cutter{
		sources: u.sources, backlog: u.queries.backlog, lookup: u.queries.lookup, sizes: u.sizes.Resolve(), legibility: u.legibility,
		isSkipped: isSkipped, errorHandler: errorHandler, readNote: readNote, onProgress: onProgress,
	}
}

func (u cutter) cut(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	noteStats map[string]domain.FileStat,
	notePaths []string,
	res *vaultreading.IndexResult,
) error {
	sizes := u.sizes
	recognized := []string{chunking.GetTextVersionRecipe(sizes)}
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

func (u cutter) getNotePathsToCut(
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
	res   *vaultreading.IndexResult
}

func (u cutter) chunkNewPaths(
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
		if u.isSkipped(path) {
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

func (u cutter) source(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	path string,
	sizes chunking.Sizes,
	res *vaultreading.IndexResult,
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

	chunks := chunksOf(doc, sizes, u.legibility)
	extraction := domain.SourceChunks{
		Source: domain.Source{
			FileStat:    ref,
			ContentHash: hash,
			Recipe:      chunking.GetTextVersionRecipe(sizes),
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

func (u cutter) saveWithoutText(
	ctx context.Context,
	v domain.Vault,
	source domain.Source,
	readErr error,
	res *vaultreading.IndexResult,
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

func (u cutter) dropVersionOfNote(ctx context.Context, v domain.Vault, path string) error {
	indexed, ok, err := u.lookup.GetSourceText(ctx, v.ID, path)
	if err != nil {
		return fmt.Errorf("read the index of %s: %w", path, err)
	}
	if !ok || indexed.Producer == "" {
		return nil
	}
	if err := u.readNote(ctx, v, path); err != nil {
		return fmt.Errorf("read the note %s again: %w", path, err)
	}
	return nil
}

func (u cutter) reportError(err error) {
	u.errorHandler(err)
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

func (u cutter) progress(res vaultreading.IndexResult) {
	u.onProgress(res)
}

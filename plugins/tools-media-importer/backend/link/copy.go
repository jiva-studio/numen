package link

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
)

const copyChunk = 1 << 20

// Copy copies the media a link's address publishes, the video or the sound alone, into the plugin's
// folder, through the core's queue, writes its address into the note under media, and returns its
// size. A copy of the same media already stored is shared and not downloaded again; a copy no link
// note refers to any more is deleted.
func (l *Links) Copy(ctx context.Context, link Link, media download.Media) (int64, error) {
	task, size, err := l.beginCopy(ctx, link, media)
	if err != nil || task == nil {
		return size, err
	}
	return l.endCopy(ctx, link, media, *task)
}

// StartMediaCopy shows the task of the copy of the media of a link and copies them in the
// background. Where the requested media are already stored it starts nothing and returns their size
// and false. It returns ErrBeingFetched where another run copies the media, and ErrNotCopyable where
// nothing copies what the address publishes. A failure of the copy is current as the error of the task.
func (l *Links) StartMediaCopy(ctx context.Context, link Link, media download.Media) (int64, bool, error) {
	task, size, err := l.beginCopy(ctx, link, media)
	if err != nil || task == nil {
		return size, false, err
	}
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		_, err := l.endCopy(runCtx, link, media, *task)
		l.reportBackgroundError(runCtx, "copy the video of", link, err)
	})
	return 0, true, nil
}

func (l *Links) beginCopy(ctx context.Context, link Link, media download.Media) (*host.Task, int64, error) {
	if !l.claim("copy", link, link.Key) {
		return nil, 0, ErrBeingFetched
	}
	task, size, err := l.prepareCopy(ctx, link, media)
	if err != nil || task == nil {
		l.release("copy", link, link.Key)
	}
	return task, size, err
}

func (l *Links) prepareCopy(ctx context.Context, link Link, media download.Media) (*host.Task, int64, error) {
	kept, isKept, err := folder{store: l.getStore(link)}.findCopy(ctx, link.Key, media)
	if err != nil {
		return nil, 0, err
	}
	if isKept && kept.Size > 0 {
		// The copy is kept by the address: the link notes of the address share it, and a note made
		// again after the first was deleted points at it.
		return nil, kept.Size, l.publishCopy(ctx, link, kept.Name)
	}
	if !l.fetcher().CanCopy(string(link.Address)) {
		return nil, 0, ErrNotCopyable
	}
	task := host.Task{
		ID: GetCopyTaskID(link.Vault, link.Path), Title: copyingVideo, Detail: link.Path,
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitBytes, FailureReason: "", IsUserStarted: true,
	}
	l.reportTask(ctx, task)
	return &task, 0, nil
}

func (l *Links) endCopy(ctx context.Context, link Link, media download.Media, task host.Task) (int64, error) {
	defer l.release("copy", link, link.Key)
	size, err := l.copy(ctx, l.fetcher(), l.settings(), link, media, task)
	l.endTask(ctx, task, err)
	return size, err
}

func (l *Links) copy(ctx context.Context, fetcher Fetcher, settings Settings, link Link, media download.Media, task host.Task) (size int64, err error) {
	meta, err := fetcher.GetMetadata(ctx, string(link.Address))
	if err != nil {
		return 0, err
	}
	if settings.CopyMaxSize > 0 && meta.Bytes > settings.CopyMaxSize {
		return meta.Bytes, fmt.Errorf("%w: %d bytes", ErrTooLarge, meta.Bytes)
	}
	turn, err := l.host.Queue.EnqueueWork(ctx, host.Work{IsUserStarted: true, OnWaiting: nil})
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, turn.Release()) }()

	part := link.Key + partExt
	store := l.getStore(link)
	if err := store.Writer.Delete(ctx, part); err != nil {
		return 0, err
	}
	into := &storeWriter{
		limit: settings.CopyMaxSize,
		appendChunk: func(chunk []byte) error {
			return store.Writer.Append(ctx, part, chunk)
		},
		report: func(byteCount int64) {
			task.CompletedCount, task.TotalCount = byteCount, meta.Bytes
			l.reportTask(ctx, task)
		},
		pending:   nil,
		byteCount: 0,
	}
	copied, err := fetcher.Download(ctx, string(link.Address), media, into)
	if err == nil {
		err = into.flush()
	}
	if err != nil {
		return 0, errors.Join(err, store.Writer.Delete(context.WithoutCancel(ctx), part))
	}
	ext := strings.ToLower(copied.Extension)
	if !isCopyExt(ext) {
		return 0, errors.Join(fmt.Errorf("a copy in a %q file is not played", ext), store.Writer.Delete(ctx, part))
	}
	name := getCopyName(link.Key, media, ext)
	if err := store.Writer.Rename(ctx, part, name); err != nil {
		return 0, err
	}
	return into.byteCount, l.publishCopy(ctx, link, name)
}

func (l *Links) publishCopy(ctx context.Context, link Link, name string) error {
	if err := l.setAddress(ctx, link, mediaKey, l.getAddress(name)); err != nil {
		return err
	}
	return l.deleteUnreferencedCopies(ctx, link)
}

// DeleteCopy takes the address of the copy out of the note, and deletes the copy where no other
// link note refers to it. It waits for a running copy of the same address to end.
func (l *Links) DeleteCopy(ctx context.Context, link Link) error {
	if err := l.waitClaim(ctx, "copy", link, link.Key); err != nil {
		return err
	}
	defer l.release("copy", link, link.Key)
	if err := l.setAddress(ctx, link, mediaKey, ""); err != nil {
		return err
	}
	return l.deleteUnreferencedCopies(ctx, link)
}

func (l *Links) deleteUnreferencedCopies(ctx context.Context, link Link) error {
	store := l.getStore(link)
	names, err := folder{store: store}.listCopies(ctx, link.Key)
	if err != nil || len(names) == 0 {
		return err
	}
	referred, err := l.listMediaAddresses(ctx, link)
	if err != nil {
		return err
	}
	var errs []error
	for _, name := range names {
		if !referred[l.getAddress(name)] {
			errs = append(errs, store.Writer.Delete(ctx, name))
		}
	}
	return errors.Join(errs...)
}

func (l *Links) listMediaAddresses(ctx context.Context, link Link) (map[string]bool, error) {
	vault := l.getVault(link)
	paths, err := vault.Files.GetPathsOfType(ctx, NoteType)
	if err != nil {
		return nil, err
	}
	referred := map[string]bool{}
	for _, path := range append(paths, link.Path) {
		raw, _, err := vault.Files.Read(ctx, path)
		if errors.Is(err, host.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="markdown.Parse ignores FileStat.Size"
		note := markdown.Parse(domain.FileStat{Path: path, Kind: "", Size: 0, ModTime: time.Time{}}, raw)
		if note.Type == NoteType {
			referred[getText(note.Frontmatter[mediaKey])] = true
		}
	}
	return referred, nil
}

// TextInfo describes the text fetched for the address of a link.

type storeWriter struct {
	limit       int64
	appendChunk func(chunk []byte) error
	report      func(byteCount int64)
	pending     []byte
	byteCount   int64
}

func (w *storeWriter) Write(p []byte) (int, error) {
	if w.limit > 0 && w.byteCount+int64(len(w.pending))+int64(len(p)) > w.limit {
		return 0, ErrTooLarge
	}
	w.pending = append(w.pending, p...)
	for len(w.pending) >= copyChunk {
		if err := w.send(w.pending[:copyChunk]); err != nil {
			return 0, err
		}
		w.pending = w.pending[copyChunk:]
	}
	return len(p), nil
}

func (w *storeWriter) flush() error {
	if len(w.pending) == 0 {
		return nil
	}
	err := w.send(w.pending)
	w.pending = nil
	return err
}

func (w *storeWriter) send(chunk []byte) error {
	if err := w.appendChunk(chunk); err != nil {
		return err
	}
	w.byteCount += int64(len(chunk))
	w.report(w.byteCount)
	return nil
}

// Package link keeps what is published at the addresses the vault's links hold. A link is a `.url`
// file holding one address. The text published there, the prose of an article or the captions of a
// video, is stored in the plugin's folder of the vault's store and given to the host as a text
// version of the link; a video is copied into the same folder and played through the host in place
// of the link.
package link

import (
	"context"
	"errors"
	"fmt"
	"io"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/urlfile"
)

var (
	// ErrNotALink is a file that is not a `.url` file holding an address.
	ErrNotALink = errors.New("this file holds no web address")

	// ErrBeingFetched is a link another run is fetching for. Nothing was done.
	ErrBeingFetched = errors.New("what this link points at is already being fetched")

	// ErrNotCopyable is an address nothing on this machine copies.
	ErrNotCopyable = errors.New("nothing on this machine copies what this address publishes")

	// ErrTooLarge is a copy larger than the settings let one be.
	ErrTooLarge = errors.New("the copy is larger than a copy may be")
)

// Fetcher reaches what is published at an address.
type Fetcher interface {
	CanCopy(address string) bool
	GetDownloadModel(address string) download.DownloadModel
	GetMetadata(ctx context.Context, address string) (download.Metadata, error)
	GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error)
	Download(ctx context.Context, address string, into io.Writer) (download.Copy, error)
}

// Host is what the plugin works through the host.
type Host struct {
	Vault host.Vault
	Store host.Store
	Texts host.Texts
	Queue host.Queue
}

// Settings are what the person set about fetching. CopyMaxSize is in bytes, and zero is no limit.
type Settings struct {
	Captions    []string
	IsAutomatic bool
	CopyMaxSize int64
}

// Links fetches for the links of the vault the host shows.
type Links struct {
	host     Host
	fetcher  func() Fetcher
	settings func() Settings

	// ctx is what every fetch started in the background runs under, and it ends with Close.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

// New is the links worked through the host. fetcher and settings are asked at every fetch, so a
// setting changed is taken by the next one.
func New(h Host, fetcher func() Fetcher, settings func() Settings) *Links {
	ctx, cancel := context.WithCancel(context.Background())
	return &Links{host: h, fetcher: fetcher, settings: settings, ctx: ctx, cancel: cancel, running: map[string]bool{}}
}

// Close stops every fetch in the background and waits for them.
func (l *Links) Close() {
	l.cancel()
	l.wg.Wait()
}

// Wait waits for every fetch started in the background.
func (l *Links) Wait() { l.wg.Wait() }

// Link is one link: the address its file holds and the key of what is stored for it.
type Link struct {
	Path    string
	Address domain.URL
	Key     string
}

// Read is the link at a path of the vault the host shows.
func (l *Links) Read(ctx context.Context, path string) (Link, error) {
	return readLink(ctx, l.host.Vault, path)
}

// readLink is the link at a path of the vault.
func readLink(ctx context.Context, vault host.Vault, path string) (Link, error) {
	if !strings.EqualFold(pathpkg.Ext(path), domain.URLExtension) {
		return Link{}, fmt.Errorf("%s: %w", path, ErrNotALink)
	}
	raw, _, err := vault.Read(ctx, path)
	if err != nil {
		return Link{}, err
	}
	address, err := urlfile.Read(raw)
	if err != nil {
		return Link{}, fmt.Errorf("%s: %w: %w", path, ErrNotALink, err)
	}
	return Link{Path: path, Address: address, Key: domain.HashContent(raw)}, nil
}

// Create makes the file an address is stored in, in a folder of the vault the host shows, named by
// the address.
func (l *Links) Create(ctx context.Context, folder, address string) (Link, error) {
	parsed, err := domain.ParseURL(strings.TrimSpace(address))
	if err != nil {
		return Link{}, err
	}
	name, _, err := domain.Filename(string(parsed))
	if err != nil {
		return Link{}, err
	}
	path := pathpkg.Join(folder, name+domain.URLExtension)
	content := urlfile.Write(parsed)
	if err := l.host.Vault.Create(ctx, path, content); err != nil {
		return Link{}, err
	}
	return Link{Path: path, Address: parsed, Key: domain.HashContent(content)}, nil
}

// claim takes one kind of work on a link for this run alone, and answers false where another run
// has it.
func (l *Links) claim(work, key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running[work+key] {
		return false
	}
	l.running[work+key] = true
	return true
}

func (l *Links) release(work, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.running, work+key)
}

// IsCopyable reports whether something on this machine copies what the address publishes.
func (l *Links) IsCopyable(address domain.URL) bool { return l.fetcher().CanCopy(string(address)) }

// FindCopy is the size of the copy stored for a link, and zero where none is.
func (l *Links) FindCopy(ctx context.Context, link Link) (int64, error) {
	found, ok, err := texts{store: l.host.Store}.findCopy(ctx, link.Key)
	if err != nil || !ok {
		return 0, err
	}
	return found.Size, nil
}

// StartFetch fetches the text of a link in the background, and a copy of the video where one is
// published. A failure is shown as the task of the work that failed.
func (l *Links) StartFetch(link Link) {
	l.wg.Go(func() {
		_ = l.FetchText(l.ctx, link, true)
		if l.IsCopyable(link.Address) {
			_, _ = l.Copy(l.ctx, link)
		}
	})
}

// StartFetchText fetches the text of a link in the background. A failure is shown as its task.
func (l *Links) StartFetchText(link Link, isUserStarted bool) {
	l.wg.Go(func() {
		_ = l.FetchText(l.ctx, link, isUserStarted)
	})
}

// taskTitles are what each kind of work on a link is called in the list of tasks.
const (
	fetchingText = "Fetching the text of a link"
	copyingVideo = "Copying a video"
)

func getTaskID(work, path string) string { return work + ":" + path }

// FetchText fetches the text published at a link's address and puts it as a text version of the
// link. An address that publishes none is marked as such. A fetch nobody asked for leaves an address
// marked as publishing none, or a text a person deleted, as it is.
func (l *Links) FetchText(ctx context.Context, link Link, isUserStarted bool) error {
	if !l.claim("text", link.Key) {
		return ErrBeingFetched
	}
	defer l.release("text", link.Key)
	folder := texts{store: l.host.Store}
	if !isUserStarted {
		if isNone, err := folder.isNone(ctx, link.Key); err != nil || isNone {
			return err
		}
	}
	taskID := getTaskID("text", link.Path)
	l.reportTask(ctx, host.Task{ID: taskID, Title: fetchingText, About: link.Path, IsUserStarted: isUserStarted})
	err := l.fetchText(ctx, folder, link)
	l.endTask(ctx, host.Task{ID: taskID, Title: fetchingText, About: link.Path, IsUserStarted: isUserStarted}, err)
	return err
}

func (l *Links) fetchText(ctx context.Context, folder texts, link Link) error {
	fetcher, settings := l.fetcher(), l.settings()
	address := string(link.Address)
	fetched, err := fetcher.GetText(ctx, address, download.PreferredCaptions{
		Languages: settings.Captions, IsAutomatic: settings.IsAutomatic,
	})
	if errors.Is(err, download.ErrNothingDownloaded) {
		return folder.markNone(ctx, link.Key)
	}
	if err != nil {
		return err
	}
	model := fetcher.GetDownloadModel(address)
	rec := record{Address: address, Title: fetched.Title, Producer: fetched.Producer, Downloader: model.Tool + " " + model.Version}
	if err := folder.writeText(ctx, link.Key, fetched, rec); err != nil {
		return err
	}
	return l.putVersion(ctx, folder, link)
}

// putVersion reports the text stored for a link to the host as the link's version.
func (l *Links) putVersion(ctx context.Context, folder texts, link Link) error {
	parts, format, err := folder.readParts(ctx, link.Key)
	if err != nil {
		return err
	}
	_, err = l.host.Texts.PutVersion(ctx, host.TextPut{
		Path: link.Path, Key: link.Key, Format: format, InputHash: link.Key, Hash: hashParts(parts),
	})
	return err
}

// copyChunk is how much of a copy is sent to the store in one call.
const copyChunk = 1 << 20

// Copy copies the video a link's address publishes into the plugin's folder, through the core's
// queue, and answers its size. A copy already stored is left as it is.
func (l *Links) Copy(ctx context.Context, link Link) (int64, error) {
	if !l.claim("copy", link.Key) {
		return 0, ErrBeingFetched
	}
	defer l.release("copy", link.Key)
	folder := texts{store: l.host.Store}
	if found, ok, err := folder.findCopy(ctx, link.Key); err != nil || ok {
		return found.Size, err
	}
	fetcher, settings := l.fetcher(), l.settings()
	address := string(link.Address)
	if !fetcher.CanCopy(address) {
		return 0, ErrNotCopyable
	}
	task := host.Task{ID: getTaskID("copy", link.Path), Title: copyingVideo, About: link.Path, Unit: host.UnitBytes, IsUserStarted: true}
	l.reportTask(ctx, task)
	size, err := l.copy(ctx, folder, fetcher, settings, link, task)
	l.endTask(ctx, task, err)
	return size, err
}

func (l *Links) copy(
	ctx context.Context, folder texts, fetcher Fetcher, settings Settings, link Link, task host.Task,
) (int64, error) {
	address := string(link.Address)
	meta, err := fetcher.GetMetadata(ctx, address)
	if err != nil {
		return 0, err
	}
	if settings.CopyMaxSize > 0 && meta.Bytes > settings.CopyMaxSize {
		return meta.Bytes, fmt.Errorf("%w: %d bytes", ErrTooLarge, meta.Bytes)
	}
	turn, err := l.host.Queue.EnqueueWork(ctx, host.Work{IsUserStarted: true})
	if err != nil {
		return 0, err
	}
	defer turn.Release()

	part := link.Key + partExt
	if err := l.host.Store.Delete(ctx, part); err != nil {
		return 0, err
	}
	into := &storeWriter{
		ctx: ctx, store: l.host.Store, name: part, limit: settings.CopyMaxSize,
		report: func(byteCount int64) {
			task.Done, task.Total = byteCount, meta.Bytes
			l.reportTask(ctx, task)
		},
	}
	copied, err := fetcher.Download(ctx, address, into)
	if err == nil {
		err = into.flush()
	}
	if err != nil {
		return 0, errors.Join(err, l.host.Store.Delete(context.WithoutCancel(ctx), part))
	}
	ext := strings.ToLower(copied.Extension)
	if !containsExt(ext) {
		return 0, errors.Join(fmt.Errorf("a copy folder as %q is not played", ext), l.host.Store.Delete(ctx, part))
	}
	if err := l.host.Store.Rename(ctx, part, link.Key+ext); err != nil {
		return 0, err
	}
	return into.byteCount, nil
}

// DeleteCopy takes away the copy stored for a link.
func (l *Links) DeleteCopy(ctx context.Context, link Link) error {
	return texts{store: l.host.Store}.deleteCopy(ctx, link.Key)
}

// ReadArticle is the text of the live version of a link, and nothing where it has none.
func (l *Links) ReadArticle(ctx context.Context, path string) (string, error) {
	listing, err := l.host.Texts.ListVersions(ctx, path)
	if err != nil || listing.Live == "" {
		return "", err
	}
	for _, version := range listing.Versions {
		if version.GetID() != listing.Live {
			continue
		}
		whole, err := host.ReadWholeText(ctx, l.host.Texts, version)
		if err != nil {
			return "", err
		}
		return whole.Text, nil
	}
	return "", nil
}

// GetTitle is what the page at a link's address calls itself, and nothing where no text was
// fetched.
func (l *Links) GetTitle(ctx context.Context, link Link) (string, error) {
	rec, err := texts{store: l.host.Store}.readRecord(ctx, link.Key)
	return rec.Title, err
}

// reportTask shows a task. A host that cannot show it leaves the work to go on unseen.
func (l *Links) reportTask(ctx context.Context, task host.Task) {
	_ = l.host.Queue.ReportTask(ctx, task)
}

// endTask takes a task out of the list, or leaves it shown with the error that ended the work.
func (l *Links) endTask(ctx context.Context, task host.Task, err error) {
	ctx = context.WithoutCancel(ctx)
	if err == nil || errors.Is(err, context.Canceled) {
		_ = l.host.Queue.FinishTask(ctx, task.ID)
		return
	}
	task.Error = err.Error()
	l.reportTask(ctx, task)
}

// storeWriter appends what is written to a file of the store, a chunk at a time, and stops past a
// limit.
type storeWriter struct {
	ctx       context.Context
	store     host.Store
	name      string
	limit     int64
	report    func(byteCount int64)
	pending   []byte
	byteCount int64
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
	if err := w.store.Append(w.ctx, w.name, chunk); err != nil {
		return err
	}
	w.byteCount += int64(len(chunk))
	w.report(w.byteCount)
	return nil
}

func msToDuration(ms int) time.Duration { return time.Duration(ms) * time.Millisecond }

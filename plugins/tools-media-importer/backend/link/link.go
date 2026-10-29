// Package link keeps what is published at the addresses the vault's links hold. A link is a note of
// type `link` whose frontmatter holds one address under `url`. The text published there, the prose
// of an article or the captions of a video, is stored in the plugin's folder of the vault's store
// and put to the host as a text version of the note; a video is copied into the same folder and
// played through the host in place of the note.
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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
)

// NoteType is the `type` a link note carries in its frontmatter.
const NoteType domain.NoteType = "link"

// The frontmatter keys a link note is written with.
const (
	typeKey = "type"
	urlKey  = "url"
)

var (
	// ErrNotALink is a file that is not a link note holding a web address.
	ErrNotALink = errors.New("this file is no link note holding a web address")

	// ErrBeingFetched is a link another run is fetching for. Nothing was done.
	ErrBeingFetched = errors.New("what this link points at is already being fetched")

	// ErrNotCopyable is an address nothing on this machine copies.
	ErrNotCopyable = errors.New("nothing on this machine copies what this address publishes")

	// ErrTooLarge is a copy larger than the settings let one be.
	ErrTooLarge = errors.New("the copy is larger than a copy may be")
)

// Fetcher retrieves the content published at an address.
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
	Now   func() time.Time
}

// Settings are the user's fetch settings. CopyMaxSize is in bytes; zero means
// no limit.
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
	now      func() time.Time

	closing   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

// New returns the links service for the host. fetcher and settings are
// evaluated on every fetch, so a changed setting applies to the next one.
func New(h Host, fetcher func() Fetcher, settings func() Settings) *Links {
	now := h.Now
	if now == nil {
		now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	}
	return &Links{
		host: h, fetcher: fetcher, settings: settings, now: now,
		closing: make(chan struct{}), running: map[string]bool{},
	}
}

// Close stops every fetch in the background and waits for them.
func (l *Links) Close() {
	l.closeOnce.Do(func() { close(l.closing) })
	l.wg.Wait()
}

func (l *Links) withClosing(ctx context.Context) (context.Context, context.CancelFunc) {
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-l.closing:
			cancel()
		case <-runCtx.Done():
		}
	}()
	return runCtx, cancel
}

// Wait waits for every fetch started in the background.
func (l *Links) Wait() { l.wg.Wait() }

// Link is one link note. Key names the text stored for its address, and Hash is the content hash of
// the note, which names the copy of its video and is the source hash of its text version. Vault is
// the identity of the vault the note is in, read from the host when the note is read or made, so
// the work on a link of the shown vault is keyed as that vault's and not as no vault.
type Link struct {
	Vault   domain.VaultID
	Path    string
	Address address.URL
	Key     string
	Hash    string
}

// Read is the link at a path of the vault the host shows.
func (l *Links) Read(ctx context.Context, path string) (Link, error) {
	return readLink(ctx, l.host.Vault, path)
}

// readLink is the link note at a path of the vault.
func readLink(ctx context.Context, vault host.Vault, path string) (Link, error) {
	if !strings.EqualFold(pathpkg.Ext(path), domain.NoteExtension) {
		return Link{}, fmt.Errorf("%s: %w", path, ErrNotALink)
	}
	raw, _, err := vault.Read(ctx, path)
	if err != nil {
		return Link{}, err
	}
	note := markdown.Parse(domain.FileStat{Path: path}, raw)
	value, _ := note.Frontmatter[urlKey].(string)
	if note.Type != NoteType || value == "" {
		return Link{}, fmt.Errorf("%s: %w", path, ErrNotALink)
	}
	parsed, err := address.Parse(value)
	if err != nil {
		return Link{}, fmt.Errorf("%s: %w: %w", path, ErrNotALink, err)
	}
	return newLinkIn(ctx, vault, path, parsed, raw)
}

func newLink(path string, parsed address.URL, raw []byte) Link {
	return Link{Path: path, Address: parsed, Key: getKey(parsed), Hash: domain.HashContent(raw)}
}

// getKey names what is stored for the text of an address.
func getKey(parsed address.URL) string { return domain.HashContent([]byte(parsed)) }

// Create makes the link note of an address in a folder of the vault the host shows, named by the
// address.
func (l *Links) Create(ctx context.Context, folder, raw string) (Link, error) {
	parsed, err := address.Parse(raw)
	if err != nil {
		return Link{}, err
	}
	name, _, err := domain.Filename(string(parsed))
	if err != nil {
		return Link{}, err
	}
	doc, err := markdown.Open(markdown.Create(ulid.New(l.now()), ""))
	if err != nil {
		return Link{}, err
	}
	if err := errors.Join(doc.SetScalar(typeKey, string(NoteType)), doc.SetScalar(urlKey, string(parsed))); err != nil {
		return Link{}, err
	}
	path := pathpkg.Join(folder, name+domain.NoteExtension)
	content := doc.Bytes()
	if err := l.host.Vault.Create(ctx, path, content); err != nil {
		return Link{}, err
	}
	return newLinkIn(ctx, l.host.Vault, path, parsed, content)
}

// getStore is the plugin's folder in the vault of the link.
func (l *Links) getStore(link Link) host.Store {
	if link.Vault == "" {
		return l.host.Store
	}
	return l.host.Store.With(link.Vault)
}

// getTexts is the text versions of the vault of the link.
func (l *Links) getTexts(link Link) host.Texts {
	if link.Vault == "" {
		return l.host.Texts
	}
	return l.host.Texts.With(link.Vault)
}

// claim reserves one kind of work on a key of a link's vault for this run and
// returns false when another run holds it. The same address in another vault is
// separate work.
func (l *Links) claim(work string, link Link, key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	name := getWorkName(work, link, key)
	if l.running[name] {
		return false
	}
	l.running[name] = true
	return true
}

func (l *Links) release(work string, link Link, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.running, getWorkName(work, link, key))
}

// getWorkName returns the name of one kind of work on a key in the vault of a
// link.
func getWorkName(work string, link Link, key string) string {
	return work + "\x00" + string(link.Vault) + "\x00" + key
}

// IsCopyable reports whether something on this machine copies what the address publishes.
func (l *Links) IsCopyable(parsed address.URL) bool { return l.fetcher().CanCopy(string(parsed)) }

// FindCopy is the size of the copy stored for a link, and zero where none is. A copy kept under the
// hash the note had before an edit is renamed to the hash it has now, so the host finds it.
func (l *Links) FindCopy(ctx context.Context, link Link) (int64, error) {
	store := l.getStore(link)
	folder := folder{store: store}
	if found, ok, err := folder.findCopy(ctx, link.Hash); err != nil || ok {
		return found.Size, err
	}
	rec, err := folder.readRecord(ctx, link.Key)
	if err != nil || rec.Copy == "" {
		return 0, err
	}
	files, err := store.List(ctx)
	if err != nil {
		return 0, err
	}
	for _, one := range files {
		if one.Name != rec.Copy {
			continue
		}
		target := link.Hash + pathpkg.Ext(one.Name)
		if err := store.Rename(ctx, one.Name, target); err != nil {
			return 0, err
		}
		rec.Copy = target
		return one.Size, folder.writeRecord(ctx, link.Key, rec)
	}
	return 0, nil
}

// StartFetch fetches the text of a link in the background, and a copy of the video where one is
// published. A failure is shown as the task of the work that failed.
func (l *Links) StartFetch(link Link) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.Background())
		defer cancel()
		_ = l.FetchText(runCtx, link, true)
		if l.IsCopyable(link.Address) {
			_, _ = l.Copy(runCtx, link)
		}
	})
}

// StartFetchText fetches the text of a link in the background. A failure is shown as its task.
func (l *Links) StartFetchText(link Link, isUserStarted bool) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.Background())
		defer cancel()
		_ = l.FetchText(runCtx, link, isUserStarted)
	})
}

// StartCopy copies the video of a link in the background. A failure is shown as its task.
func (l *Links) StartCopy(link Link) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.Background())
		defer cancel()
		_, _ = l.Copy(runCtx, link)
	})
}

// taskTitles are the names of each kind of work on a link in the task list.
const (
	fetchingText = "Fetching the text of a link"
	copyingVideo = "Copying a video"
)

func getTaskID(work, path string) string { return work + ":" + path }

// FetchText fetches the text published at a link's address and stores it as a
// text version of the link. An address that publishes no text is marked as such.
// A fetch not requested by the user skips an address marked as publishing no
// text and a text whose version the user deleted, and re-stores an existing text
// as a version of the note as it currently reads, without fetching it.
func (l *Links) FetchText(ctx context.Context, link Link, isUserStarted bool) error {
	if !l.claim("text", link, link.Key) {
		return ErrBeingFetched
	}
	defer l.release("text", link, link.Key)
	folder := folder{store: l.getStore(link)}
	if !isUserStarted {
		isDone, err := l.reuseText(ctx, folder, link)
		if err != nil || isDone {
			return err
		}
	}
	task := host.Task{ID: getTaskID("text", link.Path), Title: fetchingText, About: link.Path, IsUserStarted: isUserStarted}
	l.reportTask(ctx, task)
	err := l.fetchText(ctx, folder, link)
	l.endTask(ctx, task, err)
	return err
}

// reuseText completes a fetch not requested by the user from stored data and
// reports whether it did.
func (l *Links) reuseText(ctx context.Context, folder folder, link Link) (bool, error) {
	if isNone, err := folder.isNone(ctx, link.Key); err != nil || isNone {
		return true, err
	}
	rec, err := folder.readRecord(ctx, link.Key)
	if err != nil || rec.Kind == "" {
		return false, err
	}
	hasText, err := folder.hasText(ctx, link.Key)
	if err != nil || !hasText {
		// A text that was fetched earlier and is now absent is a version the user
		// deleted.
		return true, err
	}
	return true, l.ensureVersion(ctx, link, rec.Kind)
}

func (l *Links) fetchText(ctx context.Context, folder folder, link Link) error {
	fetcher, settings := l.fetcher(), l.settings()
	fetched, err := fetcher.GetText(ctx, string(link.Address), download.PreferredCaptions{
		Languages: settings.Captions, IsAutomatic: settings.IsAutomatic,
	})
	if errors.Is(err, download.ErrNothingDownloaded) {
		return folder.markNone(ctx, link.Key)
	}
	if err != nil {
		return err
	}
	model := fetcher.GetDownloadModel(string(link.Address))
	rec, err := folder.readRecord(ctx, link.Key)
	if err != nil {
		return err
	}
	rec.Address, rec.Title, rec.Producer = string(link.Address), fetched.Title, fetched.Producer
	rec.Downloader = model.Tool + " " + model.Version
	rec.Kind = getKind(fetched)
	if err := folder.writeText(ctx, link.Key, getSegments(fetched), rec); err != nil {
		return err
	}
	return l.putVersion(ctx, link, rec.Kind)
}

// ensureVersion puts the stored text of a link as its version where the note has none made from it
// as it reads now.
func (l *Links) ensureVersion(ctx context.Context, link Link, kind domain.TextKind) error {
	listing, err := l.getTexts(link).ListVersions(ctx, link.Path)
	if err != nil {
		return err
	}
	for _, version := range listing.Versions {
		if version.Key == link.Key && version.SourceHash == link.Hash {
			return nil
		}
	}
	return l.putVersion(ctx, link, kind)
}

// putVersion puts the text stored for a link to the host as the link's version.
func (l *Links) putVersion(ctx context.Context, link Link, kind domain.TextKind) error {
	_, err := l.getTexts(link).PutVersion(ctx, host.TextPut{
		Path: link.Path, Key: link.Key, Kind: kind, SourceHash: link.Hash, File: link.Key + textExt,
	})
	return err
}

// IsTextDeleted reports whether the text of a link was fetched and its version deleted since.
func (l *Links) IsTextDeleted(ctx context.Context, link Link) (bool, error) {
	folder := folder{store: l.getStore(link)}
	if rec, err := folder.readRecord(ctx, link.Key); err != nil || rec.Kind == "" {
		return false, err
	}
	hasText, err := folder.hasText(ctx, link.Key)
	return !hasText, err
}

// copyChunk is how much of a copy is sent to the store in one call.
const copyChunk = 1 << 20

// Copy copies the video a link's address publishes into the plugin's folder, through the core's
// queue, and returns its size. A copy already stored is left as it is.
func (l *Links) Copy(ctx context.Context, link Link) (int64, error) {
	if !l.claim("copy", link, link.Hash) {
		return 0, ErrBeingFetched
	}
	defer l.release("copy", link, link.Hash)
	if size, err := l.FindCopy(ctx, link); err != nil || size > 0 {
		return size, err
	}
	fetcher, settings := l.fetcher(), l.settings()
	if !fetcher.CanCopy(string(link.Address)) {
		return 0, ErrNotCopyable
	}
	task := host.Task{ID: getTaskID("copy", link.Path), Title: copyingVideo, About: link.Path, Unit: host.UnitBytes, IsUserStarted: true}
	l.reportTask(ctx, task)
	size, err := l.copy(ctx, fetcher, settings, link, task)
	l.endTask(ctx, task, err)
	return size, err
}

func (l *Links) copy(ctx context.Context, fetcher Fetcher, settings Settings, link Link, task host.Task) (int64, error) {
	meta, err := fetcher.GetMetadata(ctx, string(link.Address))
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

	part := link.Hash + partExt
	store := l.getStore(link)
	if err := store.Delete(ctx, part); err != nil {
		return 0, err
	}
	into := &storeWriter{
		limit: settings.CopyMaxSize,
		appendChunk: func(chunk []byte) error {
			return store.Append(ctx, part, chunk)
		},
		report: func(byteCount int64) {
			task.Done, task.Total = byteCount, meta.Bytes
			l.reportTask(ctx, task)
		},
	}
	copied, err := fetcher.Download(ctx, string(link.Address), into)
	if err == nil {
		err = into.flush()
	}
	if err != nil {
		return 0, errors.Join(err, store.Delete(context.WithoutCancel(ctx), part))
	}
	ext := strings.ToLower(copied.Extension)
	if !isCopyExt(ext) {
		return 0, errors.Join(fmt.Errorf("a copy in a %q file is not played", ext), store.Delete(ctx, part))
	}
	if err := store.Rename(ctx, part, link.Hash+ext); err != nil {
		return 0, err
	}
	return into.byteCount, l.recordCopy(ctx, link, link.Hash+ext)
}

// recordCopy writes down the file the copy of a link is kept in.
func (l *Links) recordCopy(ctx context.Context, link Link, name string) error {
	folder := folder{store: l.getStore(link)}
	rec, err := folder.readRecord(ctx, link.Key)
	if err != nil {
		return err
	}
	rec.Address, rec.Copy = string(link.Address), name
	return folder.writeRecord(ctx, link.Key, rec)
}

// DeleteCopy takes away the copy stored for a link.
func (l *Links) DeleteCopy(ctx context.Context, link Link) error {
	folder := folder{store: l.getStore(link)}
	if err := folder.deleteCopy(ctx, link.Hash); err != nil {
		return err
	}
	rec, hasRecord, err := folder.findRecord(ctx, link.Key)
	if err != nil || !hasRecord || rec.Copy == "" {
		return err
	}
	rec.Copy = ""
	return folder.writeRecord(ctx, link.Key, rec)
}

// ReadArticle is the text of the live version of a link, read in the link's vault, and nothing
// where it has none.
func (l *Links) ReadArticle(ctx context.Context, link Link) (string, error) {
	texts := l.getTexts(link)
	listing, err := texts.ListVersions(ctx, link.Path)
	if err != nil || listing.Live == "" {
		return "", err
	}
	for _, version := range listing.Versions {
		if version.GetID() != listing.Live {
			continue
		}
		whole, err := host.ReadWholeText(ctx, texts, version)
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
	rec, err := folder{store: l.getStore(link)}.readRecord(ctx, link.Key)
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

// newLinkIn is the link note at a path of the vault, carrying the identity of the vault.
func newLinkIn(ctx context.Context, vault host.Vault, path string, parsed address.URL, raw []byte) (Link, error) {
	v, err := vault.GetVault(ctx)
	if err != nil {
		return Link{}, err
	}
	made := newLink(path, parsed, raw)
	made.Vault = v.ID
	return made, nil
}

// Package link keeps what is published at the addresses the vault's links hold. A link is a note of
// type `link` whose frontmatter holds one address under `url`. The text published there, the prose
// of an article or the captions of a video, is stored in the plugin's folder of the vault's store
// and put to the host as a text version of the note. A video is downloaded into the same folder, and
// the page that frames its player is written there; the note names both by their plugin:// addresses
// under `media` and `page`.
package link

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
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
	typeKey  = "type"
	urlKey   = "url"
	mediaKey = "media"
	pageKey  = "page"
)

// pluginScheme starts the address of a file in a plugin's store of the vault.
const pluginScheme = "plugin://"

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
	GetDownloadModel(address string) download.Model
	GetMetadata(ctx context.Context, address string) (download.Metadata, error)
	GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error)
	Download(ctx context.Context, address string, media download.Media, into io.Writer) (download.Copy, error)
}

// Clock gets the current time.
type Clock interface {
	Now() time.Time
}

// Host is what the plugin works through the host.
type Host struct {
	PluginID string
	Vault    host.Vault
	Store    host.Store
	Texts    host.Texts
	Queue    host.Queue
	Clock    Clock
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
	clock    Clock

	closing   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

// New returns the links service for the host. fetcher and settings are
// evaluated on every fetch, so a changed setting applies to the next one.
func New(h Host, fetcher func() Fetcher, settings func() Settings) *Links {
	return &Links{
		host: h, fetcher: fetcher, settings: settings, clock: h.Clock,
		closing: make(chan struct{}), closeOnce: sync.Once{}, wg: sync.WaitGroup{},
		mu: sync.Mutex{}, running: map[string]bool{},
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

// Link is one link note. Key names the text stored for its address, and URL is the url property as
// the note writes it, the root of its text version. Vault is the identity of the vault the note is
// in, read from the host when the note is read or made, so the work on a link of the shown vault is
// keyed as that vault's and not as no vault.
type Link struct {
	Vault   domain.VaultID
	Path    string
	Address address.URL
	Key     string
	URL     string
}

// Read is the link at a path of the vault the host shows.
func (l *Links) Read(ctx context.Context, path string) (Link, error) {
	return readLink(ctx, l.host.Vault, path)
}

func readLink(ctx context.Context, vault host.Vault, path string) (Link, error) {
	if !strings.EqualFold(pathpkg.Ext(path), domain.NoteExtension) {
		return Link{}, fmt.Errorf("%s: %w", path, ErrNotALink)
	}
	raw, _, err := vault.Read(ctx, path)
	if err != nil {
		return Link{}, err
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="markdown.Parse ignores FileStat.Size"
	note := markdown.Parse(domain.FileStat{Path: path, Kind: "", Size: 0, ModTime: time.Time{}}, raw)
	value, isText := note.Frontmatter[urlKey].(string)
	if note.Type != NoteType || !isText || value == "" {
		return Link{}, fmt.Errorf("%s: %w", path, ErrNotALink)
	}
	parsed, err := address.Parse(value)
	if err != nil {
		return Link{}, fmt.Errorf("%s: %w: %w", path, ErrNotALink, err)
	}
	return newLinkIn(ctx, vault, path, parsed, raw)
}

func newLink(path string, parsed address.URL, raw []byte) Link {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="markdown.Parse ignores FileStat.Size"
	note := markdown.Parse(domain.FileStat{Path: path, Kind: "", Size: 0, ModTime: time.Time{}}, raw)
	return Link{
		Vault: "", Path: path, Address: parsed, Key: getKey(parsed), URL: getText(note.Frontmatter[urlKey]),
	}
}

func getText(value any) string {
	text, isText := value.(string)
	if !isText {
		return ""
	}
	return text
}

func getKey(parsed address.URL) string { return domain.HashContent([]byte(parsed)) }

// Create creates the link note of an address in a folder of a vault, named by the address. An empty
// vault ID means the vault the host shows.
func (l *Links) Create(ctx context.Context, vault domain.VaultID, folder, raw string) (Link, error) {
	parsed, err := address.Parse(raw)
	if err != nil {
		return Link{}, err
	}
	name, _, err := domain.Filename(string(parsed))
	// gomutants:disable-next-line BRANCH_IF reason="domain.Filename cannot fail on a validated http or https URL"
	if err != nil {
		return Link{}, err
	}
	identifier, err := ulid.NewGenerator(rand.Reader).NewID(l.clock.Now())
	// gomutants:disable-next-line BRANCH_IF reason="crypto/rand.Reader does not fail in ulid generation"
	if err != nil {
		return Link{}, err
	}
	doc, err := markdown.Open(markdown.Create(identifier, ""))
	// gomutants:disable-next-line BRANCH_IF reason="markdown.Create output is always valid for markdown.Open"
	if err != nil {
		return Link{}, err
	}
	// gomutants:disable-next-line BRANCH_IF reason="SetScalar on valid frontmatter keys of a new document cannot fail"
	if err := errors.Join(doc.SetScalar(typeKey, string(NoteType)), doc.SetScalar(urlKey, string(parsed))); err != nil {
		return Link{}, err
	}
	target, store := l.getVaultByID(vault), l.getStoreByID(vault)
	path := pathpkg.Join(folder, name+domain.NoteExtension)
	key := getKey(parsed)
	if page, ok, err := writePage(ctx, store, key, parsed); err != nil {
		return Link{}, err
	} else if ok {
		// gomutants:disable-next-line BRANCH_IF reason="SetScalar on a valid frontmatter key of a new document cannot fail"
		if err := doc.SetScalar(pageKey, l.getAddress(page)); err != nil {
			return Link{}, err
		}
	}
	content := doc.Bytes()
	if err := target.Create(ctx, path, content); err != nil {
		return Link{}, err
	}
	return newLinkIn(ctx, target, path, parsed, content)
}

// FindOrCreate calls Create and, when a file is already at the path of the note, returns the link
// note there if it has the same address. Any other file at the path is an error naming the path,
// and the file is unchanged.
func (l *Links) FindOrCreate(ctx context.Context, vault domain.VaultID, folder, raw string) (Link, error) {
	made, err := l.Create(ctx, vault, folder, raw)
	if !errors.Is(err, host.ErrAlreadyExists) {
		return made, err
	}
	parsed, err := address.Parse(raw)
	// gomutants:disable-next-line BRANCH_IF reason="Create parsed the same address before it opened the file"
	if err != nil {
		return Link{}, err
	}
	name, _, err := domain.Filename(string(parsed))
	// gomutants:disable-next-line BRANCH_IF reason="Create named the same address before it opened the file"
	if err != nil {
		return Link{}, err
	}
	path := pathpkg.Join(folder, name+domain.NoteExtension)
	found, err := readLink(ctx, l.getVaultByID(vault), path)
	if err != nil && !errors.Is(err, ErrNotALink) {
		return Link{}, err
	}
	if err != nil || found.Address != parsed {
		return Link{}, fmt.Errorf("%s already exists and is not the link note of %s", path, parsed)
	}
	return found, nil
}

func (l *Links) getVaultByID(vault domain.VaultID) host.Vault {
	if vault == "" {
		return l.host.Vault
	}
	return l.host.Vault.With(vault)
}

func (l *Links) getStoreByID(vault domain.VaultID) host.Store {
	if vault == "" {
		return l.host.Store
	}
	return l.host.Store.With(vault)
}

func (l *Links) getAddress(name string) string {
	return pluginScheme + l.host.PluginID + "/" + name
}

func (l *Links) setAddress(ctx context.Context, link Link, key, value string) error {
	vault := l.getVault(link)
	raw, stat, err := vault.Read(ctx, link.Path)
	if err != nil {
		return err
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return err
	}
	if err := doc.SetScalar(key, value); err != nil {
		return err
	}
	content := doc.Bytes()
	if bytes.Equal(content, raw) {
		return nil
	}
	_, err = vault.Write(ctx, link.Path, content, stat)
	return err
}

func (l *Links) getVault(link Link) host.Vault { return l.getVaultByID(link.Vault) }

func (l *Links) getStore(link Link) host.Store { return l.getStoreByID(link.Vault) }

func (l *Links) getTexts(link Link) host.Texts {
	if link.Vault == "" {
		return l.host.Texts
	}
	return l.host.Texts.With(link.Vault)
}

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

func getWorkName(work string, link Link, key string) string {
	return work + "\x00" + string(link.Vault) + "\x00" + key
}

// IsCopyable reports whether something on this machine copies what the address publishes.
func (l *Links) IsCopyable(parsed address.URL) bool { return l.fetcher().CanCopy(string(parsed)) }

// StartFetch fetches the text of a link in the background, and a copy of the video where one is
// published. A failure is shown as the task of the work that failed.
func (l *Links) StartFetch(ctx context.Context, link Link) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		reportBackgroundError(runCtx, "fetch the text of", link, l.FetchText(runCtx, link))
		if l.IsCopyable(link.Address) {
			_, err := l.Copy(runCtx, link, download.MediaVideo)
			reportBackgroundError(runCtx, "copy the video of", link, err)
		}
	})
}

// StartCopy copies the media of a link in the background. A failure is shown as its task.
func (l *Links) StartCopy(ctx context.Context, link Link, media download.Media) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		_, err := l.Copy(runCtx, link, media)
		reportBackgroundError(runCtx, "copy the video of", link, err)
	})
}

func reportBackgroundError(ctx context.Context, work string, link Link, err error) {
	if err == nil || ctx.Err() != nil || errors.Is(err, ErrBeingFetched) {
		return
	}
	fmt.Fprintf(os.Stderr, "tools-media-importer: %s %s: %v\n", work, link.Path, err)
}

// taskTitles are the names of each type of work on a link in the task list.
const (
	fetchingText = "Fetching the text of a link"
	copyingVideo = "Copying a video"
)

func getTaskID(work, path string) string { return work + ":" + path }

// GetCopyTaskID returns the id of the copy of the video of a link note in the list of tasks.
func GetCopyTaskID(path string) string { return getTaskID("copy", path) }

// FetchText fetches the text published at a link's address and stores it as a
// text version of the link. An address that publishes no text stores nothing.
func (l *Links) FetchText(ctx context.Context, link Link) error {
	task, err := l.beginTextFetch(ctx, link)
	if err != nil {
		return err
	}
	return l.endTextFetch(ctx, link, task)
}

// StartTextFetch shows the task of the text of a link and fetches the text in the background. It
// returns ErrBeingFetched, and starts nothing, when another run fetches the text. A failure of the
// fetch is shown as the error of the task.
func (l *Links) StartTextFetch(ctx context.Context, link Link) error {
	task, err := l.beginTextFetch(ctx, link)
	if err != nil {
		return err
	}
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		reportBackgroundError(runCtx, "fetch the text of", link, l.endTextFetch(runCtx, link, task))
	})
	return nil
}

func (l *Links) beginTextFetch(ctx context.Context, link Link) (host.Task, error) {
	if !l.claim("text", link, link.Key) {
		return host.Task{}, ErrBeingFetched
	}
	task := host.Task{
		ID: getTaskID("text", link.Path), Title: fetchingText, Detail: link.Path,
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "", IsUserStarted: true,
	}
	l.reportTask(ctx, task)
	return task, nil
}

func (l *Links) endTextFetch(ctx context.Context, link Link, task host.Task) error {
	defer l.release("text", link, link.Key)
	err := l.fetchText(ctx, folder{store: l.getStore(link)}, link)
	l.endTask(ctx, task, err)
	return err
}

func (l *Links) fetchText(ctx context.Context, folder folder, link Link) error {
	fetcher, settings := l.fetcher(), l.settings()
	fetched, err := fetcher.GetText(ctx, string(link.Address), download.PreferredCaptions{
		Languages: settings.Captions, IsAutomatic: settings.IsAutomatic,
	})
	if errors.Is(err, download.ErrNothingDownloaded) {
		return nil
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
	segments, err := getSegments(fetched)
	// gomutants:disable-next-line BRANCH_IF reason="getSegments cannot fail for integer cue timestamps"
	if err != nil {
		return err
	}
	if err := folder.writeText(ctx, link.Key, segments, rec); err != nil {
		return err
	}
	return l.putVersion(ctx, link, rec.Kind)
}

func (l *Links) putVersion(ctx context.Context, link Link, kind domain.TextKind) error {
	_, err := l.getTexts(link).PutVersion(ctx, host.TextPut{
		Path: link.Path, Key: link.Key, SourceProperty: domain.TextSourceProperty{Key: urlKey, Value: link.URL},
		Kind: kind, File: link.Key + textExt,
	})
	return err
}

// copyChunk is how much of a copy is sent to the store in one call.
const copyChunk = 1 << 20

// Copy copies the media a link's address publishes, the video or the sound alone, into the plugin's
// folder, through the core's queue, writes its address into the note under media, and returns its
// size. A copy of the same media already stored is left as it is; a copy of the other is replaced.
func (l *Links) Copy(ctx context.Context, link Link, media download.Media) (int64, error) {
	task, size, err := l.beginCopy(ctx, link, media)
	if err != nil || task == nil {
		return size, err
	}
	return l.endCopy(ctx, link, media, *task)
}

// StartMediaCopy shows the task of the copy of the media of a link and copies them in the
// background. Where the media asked for are already stored it starts nothing and returns their size
// and false. It returns ErrBeingFetched where another run copies the media, and ErrNotCopyable where
// nothing copies what the address publishes. A failure of the copy is shown as the error of the task.
func (l *Links) StartMediaCopy(ctx context.Context, link Link, media download.Media) (int64, bool, error) {
	task, size, err := l.beginCopy(ctx, link, media)
	if err != nil || task == nil {
		return size, false, err
	}
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		_, err := l.endCopy(runCtx, link, media, *task)
		reportBackgroundError(runCtx, "copy the video of", link, err)
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
	folder := folder{store: l.getStore(link)}
	rec, err := folder.readRecord(ctx, link.Key)
	if err != nil {
		return nil, 0, err
	}
	kept, isKept, err := folder.findCopy(ctx, link.Key)
	if err != nil {
		return nil, 0, err
	}
	if isKept && kept.Size > 0 && rec.CopyMedia == media {
		// The copy is kept by the address and outlives a deleted note of it: a note made again
		// points at it.
		return nil, kept.Size, l.setAddress(ctx, link, mediaKey, l.getAddress(kept.Name))
	}
	if !l.fetcher().CanCopy(string(link.Address)) {
		return nil, 0, ErrNotCopyable
	}
	task := host.Task{
		ID: GetCopyTaskID(link.Path), Title: copyingVideo, Detail: link.Path,
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
	if err := store.Delete(ctx, part); err != nil {
		return 0, err
	}
	into := &storeWriter{
		limit: settings.CopyMaxSize,
		appendChunk: func(chunk []byte) error {
			return store.Append(ctx, part, chunk)
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
		return 0, errors.Join(err, store.Delete(context.WithoutCancel(ctx), part))
	}
	ext := strings.ToLower(copied.Extension)
	if !isCopyExt(ext) {
		return 0, errors.Join(fmt.Errorf("a copy in a %q file is not played", ext), store.Delete(ctx, part))
	}
	name := link.Key + ext
	if err := store.Rename(ctx, part, name); err != nil {
		return 0, err
	}
	if err := l.recordCopy(ctx, link, name, media); err != nil {
		return 0, err
	}
	if err := l.setAddress(ctx, link, mediaKey, l.getAddress(name)); err != nil {
		return 0, err
	}
	return into.byteCount, folder{store: store}.deleteOtherCopies(ctx, link.Key, name)
}

func (l *Links) recordCopy(ctx context.Context, link Link, name string, media download.Media) error {
	folder := folder{store: l.getStore(link)}
	rec, err := folder.readRecord(ctx, link.Key)
	if err != nil {
		return err
	}
	rec.Address, rec.Copy, rec.CopyMedia = string(link.Address), name, media
	return folder.writeRecord(ctx, link.Key, rec)
}

// DeleteCopy takes away the copy stored for a link, and its address from the note.
func (l *Links) DeleteCopy(ctx context.Context, link Link) error {
	folder := folder{store: l.getStore(link)}
	if err := folder.deleteCopy(ctx, link.Key); err != nil {
		return err
	}
	rec, hasRecord, err := folder.findRecord(ctx, link.Key)
	if err != nil {
		return err
	}
	if hasRecord && rec.Copy != "" {
		rec.Copy, rec.CopyMedia = "", ""
		if err := folder.writeRecord(ctx, link.Key, rec); err != nil {
			return err
		}
	}
	return l.setAddress(ctx, link, mediaKey, "")
}

// TextInfo describes the text fetched for the address of a link.
type TextInfo struct {
	Title    string
	Producer string
}

// GetTextInfo returns the title and the producer of the text fetched for the address of a link. A
// link with no text fetched has an empty title and producer.
func (l *Links) GetTextInfo(ctx context.Context, link Link) (TextInfo, error) {
	rec, err := folder{store: l.getStore(link)}.readRecord(ctx, link.Key)
	if err != nil {
		return TextInfo{}, err
	}
	return TextInfo{Title: rec.Title, Producer: rec.Producer}, nil
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

func (l *Links) reportTask(ctx context.Context, task host.Task) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr diagnostic log on queue failure has no observable effect"
	if err := l.host.Queue.ReportTask(ctx, task); err != nil {
		fmt.Fprintf(os.Stderr, "tools-media-importer: show the task %s: %v\n", task.ID, err)
	}
}

func (l *Links) endTask(ctx context.Context, task host.Task, err error) {
	ctx = context.WithoutCancel(ctx)
	if err == nil || errors.Is(err, context.Canceled) {
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr diagnostic log on queue failure has no observable effect"
		if finishErr := l.host.Queue.FinishTask(ctx, task.ID); finishErr != nil {
			fmt.Fprintf(os.Stderr, "tools-media-importer: finish the task %s: %v\n", task.ID, finishErr)
		}
		return
	}
	task.FailureReason = err.Error()
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

func newLinkIn(ctx context.Context, vault host.Vault, path string, parsed address.URL, raw []byte) (Link, error) {
	v, err := vault.GetVault(ctx)
	if err != nil {
		return Link{}, err
	}
	made := newLink(path, parsed, raw)
	made.Vault = v.ID
	return made, nil
}

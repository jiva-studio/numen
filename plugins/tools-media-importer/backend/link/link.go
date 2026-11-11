// Package link keeps what is published at the addresses the vault's links hold. A link is a note of
// type `link` whose frontmatter holds one address under `url`. The text published there, the prose
// of an article or the captions of a video, is stored in the plugin's folder of the vault's store
// and put to the host as a text version of the note. One copy of the media of an address, the video
// or the sound alone, is downloaded into the same folder and shared by the link notes of the
// address, and the page that frames its player is written there; the note names both by their
// plugin:// addresses under `media` and `page`.
package link

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/url"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
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

// Files is what the links read of a vault: its link notes, its name and its index.
type Files interface {
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	GetVault(ctx context.Context) (domain.Vault, error)
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
	Scan(ctx context.Context, paths []string) error
}

// Vault is the calls the links make on one vault, each through the narrow interface it needs.
type Vault struct {
	Files  Files
	Writer host.FileWriter
}

// Folder is the plugin's folder in the store of a vault, through the narrow interface of each of its calls.
type Folder struct {
	Files  host.StoreReader
	Writer host.StoreWriter
}

// Texts is the text versions of a vault: recorded, listed and read.
type Texts interface {
	host.TextVersions
	host.TextReader
}

// Host is what the plugin works through the host.
type Host struct {
	PluginID string
	Vaults   func(domain.VaultID) Vault
	Store    func(domain.VaultID) Folder
	Texts    func(domain.VaultID) Texts
	Queue    host.Queue
	Clock    Clock
	Report   func(what string, err error)
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
	running map[string]chan struct{}
}

// New returns the links service for the host. fetcher and settings are
// evaluated on every fetch, so a changed setting applies to the next one.
func New(h Host, fetcher func() Fetcher, settings func() Settings) *Links {
	return &Links{
		host: h, fetcher: fetcher, settings: settings, clock: h.Clock,
		closing: make(chan struct{}), closeOnce: sync.Once{}, wg: sync.WaitGroup{},
		mu: sync.Mutex{}, running: map[string]chan struct{}{},
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
// in, read from the host when the note is read or made, so the work on a link of the current vault is
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
	return readLink(ctx, l.host.Vaults("").Files, path)
}

func readLink(ctx context.Context, vault Files, path string) (Link, error) {
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
	if page, ok, err := writePage(ctx, store.Writer, key, parsed); err != nil {
		return Link{}, err
	} else if ok {
		// gomutants:disable-next-line BRANCH_IF reason="SetScalar on a valid frontmatter key of a new document cannot fail"
		if err := doc.SetScalar(pageKey, l.getAddress(page)); err != nil {
			return Link{}, err
		}
	}
	content := doc.Bytes()
	if err := target.Writer.Create(ctx, path, content); err != nil {
		return Link{}, err
	}
	// The index takes the note in at once, so a lookup by type finds it before the watch reports it.
	if err := target.Files.Scan(ctx, []string{path}); err != nil {
		return Link{}, err
	}
	return newLinkIn(ctx, target.Files, path, parsed, content)
}

// FindOrCreate returns the link note of the address in a vault, wherever it is, and true; it calls
// Create where the vault has none. Where a file other than the link note of the address is at the
// path of the note, the error names the path, and the file is unchanged.
func (l *Links) FindOrCreate(ctx context.Context, vault domain.VaultID, folder, raw string) (Link, bool, error) {
	parsed, err := address.Parse(raw)
	if err != nil {
		return Link{}, false, err
	}
	if found, isFound, err := l.findByAddress(ctx, l.getVaultByID(vault).Files, parsed); err != nil || isFound {
		return found, isFound, err
	}
	made, err := l.Create(ctx, vault, folder, raw)
	if !errors.Is(err, host.ErrAlreadyExists) {
		return made, false, err
	}
	name, _, err := domain.Filename(string(parsed))
	// gomutants:disable-next-line BRANCH_IF reason="Create named the same address before it opened the file"
	if err != nil {
		return Link{}, false, err
	}
	path := pathpkg.Join(folder, name+domain.NoteExtension)
	found, err := readLink(ctx, l.getVaultByID(vault).Files, path)
	if err != nil && !errors.Is(err, ErrNotALink) {
		return Link{}, false, err
	}
	if err != nil || found.Address != parsed {
		return Link{}, false, fmt.Errorf("%s already exists and is not the link note of %s", path, parsed)
	}
	return found, true, nil
}

func (l *Links) findByAddress(ctx context.Context, vault Files, parsed address.URL) (Link, bool, error) {
	paths, err := vault.GetPathsOfType(ctx, NoteType)
	if err != nil {
		return Link{}, false, err
	}
	for _, path := range paths {
		found, err := readLink(ctx, vault, path)
		if errors.Is(err, ErrNotALink) || errors.Is(err, host.ErrNotExist) {
			continue
		}
		if err != nil {
			return Link{}, false, err
		}
		if found.Address == parsed {
			return found, true, nil
		}
	}
	return Link{}, false, nil
}

func (l *Links) getVaultByID(vault domain.VaultID) Vault { return l.host.Vaults(vault) }

func (l *Links) getStoreByID(vault domain.VaultID) Folder { return l.host.Store(vault) }

func (l *Links) getAddress(name string) string {
	// The core resolves plugin://<plugin id>/<path> to the file at path in that plugin's store.
	return (&url.URL{Scheme: "plugin", Host: l.host.PluginID, Path: "/" + name}).String()
}

func (l *Links) setAddress(ctx context.Context, link Link, key, value string) error {
	vault := l.getVault(link)
	raw, stat, err := vault.Files.Read(ctx, link.Path)
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
	_, err = vault.Writer.Write(ctx, link.Path, content, stat)
	return err
}

func (l *Links) getVault(link Link) Vault { return l.getVaultByID(link.Vault) }

func (l *Links) getStore(link Link) Folder { return l.getStoreByID(link.Vault) }

func (l *Links) getTexts(link Link) Texts { return l.host.Texts(link.Vault) }

func (l *Links) claim(work string, link Link, key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	name := getWorkName(work, link, key)
	if _, isRunning := l.running[name]; isRunning {
		return false
	}
	l.running[name] = make(chan struct{})
	return true
}

func (l *Links) waitClaim(ctx context.Context, work string, link Link, key string) error {
	name := getWorkName(work, link, key)
	for {
		l.mu.Lock()
		released, isRunning := l.running[name]
		if !isRunning {
			l.running[name] = make(chan struct{})
			l.mu.Unlock()
			return nil
		}
		l.mu.Unlock()
		select {
		case <-released:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *Links) release(work string, link Link, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	name := getWorkName(work, link, key)
	if released, isRunning := l.running[name]; isRunning {
		close(released)
		delete(l.running, name)
	}
}

func getWorkName(work string, link Link, key string) string {
	return work + "\x00" + string(link.Vault) + "\x00" + key
}

// IsCopyable reports whether something on this machine copies what the address publishes.
func (l *Links) IsCopyable(parsed address.URL) bool { return l.fetcher().CanCopy(string(parsed)) }

// StartFetch fetches the text of a link in the background, and a copy of the video where one is
// published. A failure is current as the task of the work that failed.
func (l *Links) StartFetch(ctx context.Context, link Link) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		l.reportBackgroundError(runCtx, "fetch the text of", link, l.FetchText(runCtx, link))
		if l.IsCopyable(link.Address) {
			_, err := l.Copy(runCtx, link, download.MediaVideo)
			l.reportBackgroundError(runCtx, "copy the video of", link, err)
		}
	})
}

// StartCopy copies the media of a link in the background. A failure is current as its task.
func (l *Links) StartCopy(ctx context.Context, link Link, media download.Media) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		_, err := l.Copy(runCtx, link, media)
		l.reportBackgroundError(runCtx, "copy the video of", link, err)
	})
}

func (l *Links) reportBackgroundError(ctx context.Context, work string, link Link, err error) {
	if err == nil || ctx.Err() != nil || errors.Is(err, ErrBeingFetched) {
		return
	}
	l.reportError(work+" "+link.Path, err)
}

func (l *Links) reportError(what string, err error) {
	if l.host.Report != nil && err != nil {
		l.host.Report(what, err)
	}
}

// taskTitles are the names of each type of work on a link in the task list.
const (
	fetchingText = "Fetching the text of a link"
	copyingVideo = "Copying a video"
)

func getTaskID(work string, vault domain.VaultID, path string) string {
	return work + ":" + string(vault) + ":" + path
}

// GetCopyTaskID returns the id of the copy of the video of a link note of a vault in the list of tasks.
func GetCopyTaskID(vault domain.VaultID, path string) string { return getTaskID("copy", vault, path) }

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
// fetch is current as the error of the task.
func (l *Links) StartTextFetch(ctx context.Context, link Link) error {
	task, err := l.beginTextFetch(ctx, link)
	if err != nil {
		return err
	}
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		l.reportBackgroundError(runCtx, "fetch the text of", link, l.endTextFetch(runCtx, link, task))
	})
	return nil
}

func (l *Links) beginTextFetch(ctx context.Context, link Link) (host.Task, error) {
	if !l.claim("text", link, link.Key) {
		return host.Task{}, ErrBeingFetched
	}
	task := host.Task{
		ID: getTaskID("text", link.Vault, link.Path), Title: fetchingText, Detail: link.Path,
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
		l.reportError("show the task "+task.ID, err)
	}
}

func (l *Links) endTask(ctx context.Context, task host.Task, err error) {
	ctx = context.WithoutCancel(ctx)
	if err == nil || errors.Is(err, context.Canceled) {
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr diagnostic log on queue failure has no observable effect"
		if finishErr := l.host.Queue.FinishTask(ctx, task.ID); finishErr != nil {
			l.reportError("finish the task "+task.ID, finishErr)
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

func newLinkIn(ctx context.Context, vault Files, path string, parsed address.URL, raw []byte) (Link, error) {
	v, err := vault.GetVault(ctx)
	if err != nil {
		return Link{}, err
	}
	made := newLink(path, parsed, raw)
	made.Vault = v.ID
	return made, nil
}

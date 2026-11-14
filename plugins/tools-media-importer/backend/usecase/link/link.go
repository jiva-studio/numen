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
	"net/url"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
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

// Links fetches for the links of the vault the host shows.
type Links struct {
	host     Host
	fetcher  func() ports.Fetcher
	settings func() fetch.Settings
	clock    Clock

	closing   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

	mu      sync.Mutex
	running map[string]chan struct{}
}

// New returns the links service for the host. fetcher and settings are
// evaluated on every fetch, so a changed setting applies to the next one.
func New(h Host, fetcher func() ports.Fetcher, settings func() fetch.Settings) *Links {
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

func newLinkIn(ctx context.Context, vault Files, path string, parsed address.URL, raw []byte) (Link, error) {
	v, err := vault.GetVault(ctx)
	if err != nil {
		return Link{}, err
	}
	made := newLink(path, parsed, raw)
	made.Vault = v.ID
	return made, nil
}

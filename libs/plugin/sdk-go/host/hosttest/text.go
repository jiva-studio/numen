package hosttest

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"time"

	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	notesmarkdown "github.com/jiva-studio/numen/libs/note/markdown"
	"github.com/jiva-studio/numen/libs/note/textfile"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Texts is the text versions of a vault in memory, used as a plugin uses them through the
// host. A version's file is read from the folder of the plugin that made it, as the host reads it:
// the store SetStore names, or a folder in memory. The live version is the one the core would show:
// the newest whose sources are unchanged and whose file still has the hash it was recorded with.
// With returns the texts of another vault of the same host, empty on first use.
type Texts struct {
	world    *textWorld
	pluginID string
}

// defaultTextPageSize is the number of segments a page of ReadText holds, as in the core.
const defaultTextPageSize = 1000

// textHost is the texts of every vault one host keeps, by identity, and the number of segments
// a page of ReadText holds in each of them.
type textHost struct {
	mu       sync.Mutex
	worlds   map[domain.VaultID]*textWorld
	pageSize int
}

// textWorld is what every plugin's view of one vault's texts shares.
type textWorld struct {
	mu       sync.Mutex
	all      *textHost
	vault    domain.VaultID
	versions []domain.TextVersion
	files    map[string][]byte
	stores   map[string]*Store
	changes  []string
	clock    time.Time
}

var (
	_ host.TextVersions = (*Texts)(nil)
	_ host.TextReader   = (*Texts)(nil)
)

// NewTexts is the texts of the vault of the identity, as the plugin of the id works them.
func NewTexts(vault domain.VaultID, pluginID string) *Texts {
	vaults := &textHost{worlds: map[domain.VaultID]*textWorld{}, pageSize: defaultTextPageSize}
	return &Texts{world: vaults.getWorld(vault), pluginID: pluginID}
}

// SetPageSize sets the number of segments a page of ReadText holds, in every vault of the host.
func (t *Texts) SetPageSize(segments int) {
	h := t.world.all
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pageSize = segments
}

func (h *textHost) getPageSize() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pageSize
}

func (h *textHost) getWorld(vault domain.VaultID) *textWorld {
	h.mu.Lock()
	defer h.mu.Unlock()
	w, ok := h.worlds[vault]
	if !ok {
		w = &textWorld{
			all: h, vault: vault, files: map[string][]byte{}, stores: map[string]*Store{},
			clock: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		h.worlds[vault] = w
	}
	return w
}

// With returns the texts of the vault of the identity. The empty identity is the texts themselves.
func (t *Texts) With(id domain.VaultID) *Texts {
	if id == "" {
		return t
	}
	return &Texts{world: t.world.all.getWorld(id), pluginID: t.pluginID}
}

// As is the same texts, as another plugin works them.
func (t *Texts) As(pluginID string) *Texts { return &Texts{world: t.world, pluginID: pluginID} }

// File puts a file's bytes in the vault: what a version made from the file is compared with.
func (t *Texts) File(path string, raw []byte) {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	t.world.files[path] = raw
}

// SetStore is the folder the host reads a plugin's version files from.
func (t *Texts) SetStore(pluginID string, store *Store) {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	t.world.stores[pluginID] = store
}

// GetStore is the folder of a plugin's version files: the one SetStore named, or a folder in
// memory made on first use.
func (t *Texts) GetStore(pluginID string) *Store {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	return t.world.getStore(pluginID)
}

func (w *textWorld) getStore(pluginID string) *Store {
	store, ok := w.stores[pluginID]
	if !ok {
		store = New(pluginID).Store()
		w.stores[pluginID] = store
	}
	return store
}

// TakeChanges returns every file whose versions changed since the last call, in order.
func (t *Texts) TakeChanges() []string {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	out := t.world.changes
	t.world.changes = nil
	return out
}

func (w *textWorld) isValidPut(params host.TextPut, version domain.TextVersion) bool {
	if params.Path == "" || params.Key == "" || params.Kind == "" || params.File == "" {
		return false
	}
	if version.IsFromProperty() && (params.Source != "" || params.InputHash != "") {
		return false
	}
	return params.Source != version.GetID() && (params.Source == "" || w.indexOf(params.Path, params.Source) >= 0)
}

func (t *Texts) PutVersion(ctx context.Context, params host.TextPut) (domain.TextVersion, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	version := domain.TextVersion{
		Path: params.Path, PluginID: t.pluginID, Key: params.Key, Source: params.Source, SourceProperty: params.SourceProperty,
		Kind: params.Kind, File: params.File, InputHash: params.InputHash,
	}
	if !w.isValidPut(params, version) {
		return domain.TextVersion{}, host.ErrInvalidVersion
	}
	_, hash, err := w.readFile(ctx, version)
	if err != nil {
		return domain.TextVersion{}, errors.Join(host.ErrInvalidVersion, err)
	}
	version.TextHash = hash
	if i := w.indexOf(params.Path, version.GetID()); i >= 0 {
		version.CreateTime = w.versions[i].CreateTime
		w.versions[i] = version
	} else {
		w.clock = w.clock.Add(time.Second)
		version.CreateTime = w.clock
		w.versions = append(w.versions, version)
	}
	w.changes = append(w.changes, params.Path)
	return version, nil
}

func (t *Texts) RemoveVersion(_ context.Context, path, key string) error {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	id := notesdomain.NewTextVersionID(t.pluginID, key)
	w.versions = slices.DeleteFunc(w.versions, func(v domain.TextVersion) bool { return v.Path == path && v.GetID() == id })
	w.changes = append(w.changes, path)
	return nil
}

func (t *Texts) ListVersions(ctx context.Context, path string) (host.TextListing, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	var listing host.TextListing
	for _, v := range w.versions {
		if v.Path == path {
			listing.Versions = append(listing.Versions, v)
		}
	}
	isIntact := func(v domain.TextVersion) bool {
		_, hash, err := w.readFile(ctx, v)
		return err == nil && hash == v.TextHash
	}
	if live, ok := notesdomain.FindLiveVersion(listing.Versions, domain.HashContent(w.files[path]), w.readProperties(path), isIntact); ok {
		listing.Live = live.GetID()
	}
	return listing, nil
}

// ReadText returns the page of the version's segments that starts at the cursor: at most the page
// size of segments, and the cursor of the next segment when more remain. A cursor is the decimal
// index of a segment, empty for the first.
func (t *Texts) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (host.TextPage, error) {
	start := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return host.TextPage{}, host.ErrNotExist
		}
		start = n
	}
	segments, err := t.readSegments(ctx, path, id)
	if err != nil {
		return host.TextPage{}, err
	}
	if start > len(segments) {
		return host.TextPage{}, host.ErrNotExist
	}
	end := min(start+t.world.all.getPageSize(), len(segments))
	page := host.TextPage{Segments: segments[start:end]}
	if end < len(segments) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

func (t *Texts) readSegments(ctx context.Context, path string, id domain.TextVersionID) ([]domain.TextSegment, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	i := w.indexOf(path, id)
	if i < 0 {
		return nil, host.ErrNotExist
	}
	segments, hash, err := w.readFile(ctx, w.versions[i])
	if err != nil || hash != w.versions[i].TextHash {
		return nil, host.ErrNotExist
	}
	return segments, nil
}

func (w *textWorld) readFile(ctx context.Context, v domain.TextVersion) ([]domain.TextSegment, string, error) {
	raw, err := w.getStore(v.PluginID).Read(ctx, v.File)
	if err != nil {
		return nil, "", err
	}
	segments, err := textfile.Decode(raw)
	if err != nil {
		return nil, "", err
	}
	return segments, domain.HashContent(raw), nil
}

func (w *textWorld) indexOf(path string, id domain.TextVersionID) int {
	return slices.IndexFunc(w.versions, func(v domain.TextVersion) bool { return v.Path == path && v.GetID() == id })
}

func (w *textWorld) readProperties(path string) map[string]any {
	return notesmarkdown.Parse(notesdomain.FileStat{Path: path}, w.files[path]).Frontmatter
}

// ReadTextRange returns the segments of the version that cover the range of its whole text, truncated at
// whole characters.
func (t *Texts) ReadTextRange(ctx context.Context, path string, id domain.TextVersionID, start, length int) (host.TextRange, error) {
	segments, err := t.readSegments(ctx, path, id)
	if err != nil {
		return host.TextRange{}, err
	}
	text := notesdomain.NewVersionText(domain.TextVersion{}, segments)
	rangeStart, rangeEnd := getRange([]byte(text.Text), start, length)
	return host.TextRange{Segments: text.GetSegmentsIn(rangeStart, rangeEnd), Start: rangeStart, Size: len(text.Text)}, nil
}

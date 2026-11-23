package synctest

import (
	"context"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// PlaceCall is a call of PlaceFile as the host got it.
type PlaceCall struct {
	Name     string
	Path     string
	Hash     string
	Baseline *domain.FileStat
}

// MoveCall is a call of MoveFile as the host got it.
type MoveCall struct {
	From     string
	To       string
	Baseline *domain.FileStat
}

// RemoveCall is a call of RemoveFile as the host got it.
type RemoveCall struct {
	Path     string
	Baseline *domain.FileStat
}

// SpyHost is the host of a computer that writes down the calls that move the bytes of files, and serves a file that
// is never held whole when one is given.
type SpyHost struct {
	inner ports.Host

	mu            sync.Mutex
	sources       map[string]*Source
	places        []PlaceCall
	moves         []MoveCall
	removes       []RemoveCall
	maxRead       int64
	placeCallback func(PlaceCall)
}

// NewSpyHost returns the host over the inner host.
func NewSpyHost(inner ports.Host) *SpyHost {
	return &SpyHost{inner: inner, mu: sync.Mutex{}, sources: map[string]*Source{}, places: nil, moves: nil, removes: nil, maxRead: 0, placeCallback: nil}
}

// Serve makes the host read the file at the path of the source from the source, not from the vault.
func (h *SpyHost) Serve(source *Source) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sources[source.GetPath()] = source
}

// GetPlaces returns the calls of PlaceFile, in order.
func (h *SpyHost) GetPlaces() []PlaceCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.places)
}

// GetMoves returns the calls of MoveFile, in order.
func (h *SpyHost) GetMoves() []MoveCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.moves)
}

// GetRemoves returns the calls of RemoveFile, in order.
func (h *SpyHost) GetRemoves() []RemoveCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.removes)
}

// GetMaxRead returns the most bytes one ranged read asked for.
func (h *SpyHost) GetMaxRead() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.maxRead
}

func (h *SpyHost) GetVaults(ctx context.Context) ([]domain.Vault, error) {
	return h.inner.GetVaults(ctx)
}

func (h *SpyHost) GetFiles(id domain.VaultID) ports.Files { return h.inner.GetFiles(id) }

func (h *SpyHost) GetIndexer(id domain.VaultID) ports.Indexer { return h.inner.GetIndexer(id) }

// Store returns the store folder of the plugin in the vault.
func (h *SpyHost) Store(id domain.VaultID) ports.Store { return h.inner.Store(id) }

// GetRemover returns the remover of the inner host.
func (h *SpyHost) GetRemover(id domain.VaultID) ports.Remover { return h.inner.GetRemover(id) }

// Transfers returns what the host does with the bytes of the files of the vault, written down.
func (h *SpyHost) GetTransfers(id domain.VaultID) ports.Transfers {
	return &spyTransfers{spy: h, inner: h.inner.GetTransfers(id)}
}

type spyTransfers struct {
	spy   *SpyHost
	inner ports.Transfers
}

func (t *spyTransfers) ReadBytes(ctx context.Context, path string, start, length int64) (host.ByteRange, error) {
	t.spy.mu.Lock()
	t.spy.maxRead = max(t.spy.maxRead, length)
	source := t.spy.sources[path]
	t.spy.mu.Unlock()
	if source != nil {
		return source.ReadBytes(ctx, start, length)
	}
	return t.inner.ReadBytes(ctx, path, start, length)
}

func (t *spyTransfers) MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (host.MoveFileResult, error) {
	t.spy.mu.Lock()
	t.spy.moves = append(t.spy.moves, MoveCall{From: from, To: to, Baseline: baseline})
	t.spy.mu.Unlock()
	return t.inner.MoveFile(ctx, from, to, baseline)
}

func (t *spyTransfers) RemoveFile(ctx context.Context, path string, baseline *domain.FileStat) (host.RemoveFileResult, error) {
	t.spy.mu.Lock()
	t.spy.removes = append(t.spy.removes, RemoveCall{Path: path, Baseline: baseline})
	t.spy.mu.Unlock()
	return t.inner.RemoveFile(ctx, path, baseline)
}

func (t *spyTransfers) PlaceFile(ctx context.Context, name, path, expectedHash string, baseline *domain.FileStat) (domain.FileStat, error) {
	t.spy.mu.Lock()
	call := PlaceCall{Name: name, Path: path, Hash: expectedHash, Baseline: baseline}
	t.spy.places = append(t.spy.places, call)
	before := t.spy.placeCallback
	t.spy.mu.Unlock()
	if before != nil {
		before(call)
	}
	return t.inner.PlaceFile(ctx, name, path, expectedHash, baseline)
}

// RunBeforePlace runs the hook in every call of PlaceFile before the host gets it, as a change of the vault does that
// lands between the staging of a file and its placement.
func (h *SpyHost) RunBeforePlace(hook func(PlaceCall)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.placeCallback = hook
}

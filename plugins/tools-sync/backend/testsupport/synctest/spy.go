package synctest

import (
	"context"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type PlaceCall struct {
	Name     string
	Path     string
	Hash     string
	Baseline *domain.FileStat
}

type MoveCall struct {
	From     string
	To       string
	Baseline *domain.FileStat
}

type RemoveCall struct {
	Path     string
	Baseline *domain.FileStat
}

// SpyHost is the host of a computer that writes down the calls that move the bytes of files, and serves a file that
// is never held whole when one is given.
type SpyHost struct {
	inner ports.Host
	notes ports.NoteHost

	mu            sync.Mutex
	sources       map[string]*Source
	places        []PlaceCall
	moves         []MoveCall
	removes       []RemoveCall
	maxRead       int64
	readCalls     int
	placeCallback func(PlaceCall)
	readCallback  func(path string, start, length int64)
}

func NewSpyHost(inner ports.Host, notes ports.NoteHost) *SpyHost {
	return &SpyHost{inner: inner, notes: notes, mu: sync.Mutex{}, sources: map[string]*Source{}, places: nil, moves: nil, removes: nil, maxRead: 0, readCalls: 0, placeCallback: nil, readCallback: nil}
}

func (h *SpyHost) Serve(source *Source) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sources[source.GetPath()] = source
}

func (h *SpyHost) GetPlaces() []PlaceCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.places)
}

func (h *SpyHost) GetMoves() []MoveCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.moves)
}

func (h *SpyHost) GetRemoves() []RemoveCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.removes)
}

func (h *SpyHost) GetReadCalls() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.readCalls
}

func (h *SpyHost) GetMaxRead() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.maxRead
}

func (h *SpyHost) GetVaults(ctx context.Context) ([]domain.Vault, error) {
	return h.inner.GetVaults(ctx)
}

func (h *SpyHost) GetFiles(id domain.VaultID) ports.Files { return h.inner.GetFiles(id) }

func (h *SpyHost) GetAbsent(id domain.VaultID) ports.Absent { return h.inner.GetAbsent(id) }

func (h *SpyHost) GetNotes(id domain.VaultID) ports.Notes { return h.notes.GetNotes(id) }

func (h *SpyHost) Store(id domain.VaultID) ports.Store { return h.inner.Store(id) }

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
	t.spy.readCalls++
	source, after := t.spy.sources[path], t.spy.readCallback
	t.spy.mu.Unlock()
	if after != nil {
		defer after(path, start, length)
	}
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

// RunAfterRead runs the hook after every ranged read the host served, as a change of the file does that lands after the
// stat of that read was taken.
func (h *SpyHost) RunAfterRead(hook func(path string, start, length int64)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.readCallback = hook
}

// RunBeforePlace runs the hook in every call of PlaceFile before the host gets it, as a change of the vault does that
// lands between the staging of a file and its placement.
func (h *SpyHost) RunBeforePlace(hook func(PlaceCall)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.placeCallback = hook
}

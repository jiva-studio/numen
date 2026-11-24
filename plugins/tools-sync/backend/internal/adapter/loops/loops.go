// Package loops keeps the background work of a plugin running: the session over the server of a setting, the
// interval between passes, the wake of a vault and the entropy of the machine.
package loops

import (
	"context"
	"crypto/rand"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

const streamConnectTimeout = 10 * time.Second

// Random reads the entropy of the machine.
func NewRandom() io.Reader { return rand.Reader }

// Interval is the time between two passes, in whole seconds of at least one.
type Interval struct {
	secs     atomic.Int64
	fallback int64
}

// NewInterval returns an interval that is fallback seconds until it is set.
func NewInterval(fallback int64) *Interval {
	i := &Interval{secs: atomic.Int64{}, fallback: fallback}
	i.secs.Store(fallback)
	return i
}

// Set reads seconds from text, and takes the fallback for a text that is not a number of at least one.
func (i *Interval) Set(text string) {
	if secs, err := strconv.ParseInt(text, 10, 64); err == nil && secs >= 1 && secs <= int64(math.MaxInt64/int64(time.Second)) {
		i.secs.Store(secs)
		return
	}
	i.secs.Store(i.fallback)
}

// Get returns the time between two passes.
func (i *Interval) Get() time.Duration { return time.Duration(i.secs.Load()) * time.Second }

// Wakes carries the vaults whose files changed to the loop that passes over them.
type Wakes struct{ vaults chan domain.VaultID }

// NewWakes returns wakes that hold up to buffer vaults.
func NewWakes(buffer int) *Wakes { return &Wakes{vaults: make(chan domain.VaultID, buffer)} }

// Wake names a vault to pass over now. A vault named while the buffer is full is dropped.
func (w *Wakes) Wake(vault domain.VaultID) {
	select {
	case w.vaults <- vault:
	default:
	}
}

// Vaults returns the vaults named by Wake.
func (w *Wakes) WatchVaults() <-chan domain.VaultID { return w.vaults }

// Start builds the session over address: the value its callers use, and the loops that run until ctx ends.
type Start[T any] func(ctx context.Context, address string) (T, map[string]func(context.Context) error)

// Supervisor runs the loops of one session at a time and runs them again when the address changes.
type Supervisor[T any] struct {
	start Start[T]

	mu        sync.Mutex
	address   string
	cancel    context.CancelFunc
	done      chan struct{}
	current   T
	isStarted bool
}

// NewSupervisor returns a supervisor that runs nothing until it is given an address.
func NewSupervisor[T any](start Start[T]) *Supervisor[T] {
	var none T
	return &Supervisor[T]{
		start: start, mu: sync.Mutex{}, address: "", cancel: nil, done: nil, current: none, isStarted: false,
	}
}

// Get returns the value of the running session, and false when none runs.
func (s *Supervisor[T]) Get() (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current, s.isStarted
}

// Configure runs the session over address; an empty address runs none, and the address that runs is left alone.
func (s *Supervisor[T]) Configure(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if address == s.address && s.cancel != nil {
		return
	}
	s.stopWithLock()
	if address == "" {
		return
	}
	s.startWithLock(address)
}

// Stop ends the loops and waits for them.
func (s *Supervisor[T]) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopWithLock()
}

func (s *Supervisor[T]) startWithLock(address string) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	current, loops := s.start(ctx, address)
	s.address, s.cancel, s.done, s.current, s.isStarted = address, cancel, done, current, true
	var group sync.WaitGroup
	for name, run := range loops {
		group.Go(func() { report(ctx, name, run(ctx)) })
	}
	go func() {
		group.Wait()
		cancel()
		close(done)
	}()
}

func (s *Supervisor[T]) stopWithLock() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
	var none T
	s.address, s.cancel, s.done, s.current, s.isStarted = "", nil, nil, none, false
}

func report(ctx context.Context, name string, err error) {
	if ctx.Err() == nil {
		slog.Warn("a loop of Numen Sync ended", "loop", name, "error", err)
	}
}

// NewStreamHTTP returns the client of a long response: it has no Timeout, which would end the read of the body,
// and ends an open whose headers do not come within headerTimeout.
func NewStreamHTTP(headerTimeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: streamConnectTimeout}
	return &http.Client{Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   streamConnectTimeout,
		ResponseHeaderTimeout: headerTimeout,
	}}
}

// Text is a text setting that the loops read while the host may set it again.
type Text struct{ value atomic.Pointer[string] }

// NewText returns an empty text.
func NewText() *Text { return &Text{value: atomic.Pointer[string]{}} }

// Set replaces the text.
func (t *Text) Set(text string) { t.value.Store(&text) }

// Get returns the text, empty until it is set.
func (t *Text) Get() string {
	if text := t.value.Load(); text != nil {
		return *text
	}
	return ""
}

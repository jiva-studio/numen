package plugintest

import (
	"context"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/note/domain"
	manifestdomain "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// LeaseHost resolves every capability to one stub plugin and counts the references taken and given back.
// A question lease counts among them and among the question leases.
type LeaseHost struct {
	Endpoint          transport.Endpoint
	EnsureErr         error
	Acquired          atomic.Int32
	Released          atomic.Int32
	QuestionsAcquired atomic.Int32
	QuestionsReleased atomic.Int32
}

func (h *LeaseHost) ResolveCapability(string) (manifestdomain.Discovered, error) {
	return manifestdomain.Discovered{Manifest: manifestdomain.Manifest{ID: "org.example.stub"}}, nil
}

func (h *LeaseHost) ResolveService(string) (manifestdomain.Discovered, bool) {
	return manifestdomain.Discovered{}, false
}

func (h *LeaseHost) GetPlugin(string) (manifestdomain.Discovered, bool) {
	return manifestdomain.Discovered{Manifest: manifestdomain.Manifest{ID: "org.example.stub"}}, true
}

func (h *LeaseHost) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	if h.EnsureErr != nil {
		return transport.Endpoint{}, h.EnsureErr
	}
	h.Acquired.Add(1)
	return h.Endpoint, nil
}

func (h *LeaseHost) Acquire(string) { h.Acquired.Add(1) }

func (h *LeaseHost) Release(string) { h.Released.Add(1) }

func (h *LeaseHost) EnsureActiveForQuestion(context.Context, string) (transport.Endpoint, error) {
	if h.EnsureErr != nil {
		return transport.Endpoint{}, h.EnsureErr
	}
	h.Acquired.Add(1)
	h.QuestionsAcquired.Add(1)
	return h.Endpoint, nil
}

func (h *LeaseHost) ReleaseQuestion(string) {
	h.Released.Add(1)
	h.QuestionsReleased.Add(1)
}

// WatchPluginConfig registers nothing: the stub is never configured.
func (h *LeaseHost) WatchPluginConfig(string, func()) func() { return func() {} }

func (h *LeaseHost) CheckPluginPermission(_, _ string) error { return nil }

func (h *LeaseHost) GetVaultPath() string { return "" }

func (h *LeaseHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

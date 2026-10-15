package main

import (
	"context"
	"os/exec"
	"sync"
	"syscall"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Work represents one in-flight agent run.
type Work struct {
	Output       chan *numenv1.AskAgentResponse
	conversation string
	stop         func()
	stopped      chan struct{}
	once         sync.Once
	reader       sync.WaitGroup
}

// NewWork initializes an active work handle.
func NewWork(conversation string, buffer int, stop func()) *Work {
	return &Work{
		Output:       make(chan *numenv1.AskAgentResponse, buffer),
		conversation: conversation,
		stop:         stop,
		stopped:      make(chan struct{}),
	}
}

// Begin marks reader startup.
func (w *Work) Begin() { w.reader.Add(1) }

// End marks reader completion.
func (w *Work) End() { w.reader.Done() }

// Stopped returns a channel closed upon work termination.
func (w *Work) Stopped() <-chan struct{} { return w.stopped }

// Stop terminates the underlying run and waits for reader completion.
func (w *Work) Stop() {
	w.once.Do(func() {
		close(w.stopped)
		if w.stop != nil {
			w.stop()
		}
	})
	w.reader.Wait()
}

// Registry tracks active work items and conversation sessions.
type Registry struct {
	mu       sync.Mutex
	works    map[*Work]struct{}
	sessions map[string]string
	isClosed bool
}

// NewRegistry initializes a Registry.
func NewRegistry() *Registry {
	return &Registry{
		works:    make(map[*Work]struct{}),
		sessions: make(map[string]string),
	}
}

// Add registers an active work item.
func (r *Registry) Add(w *Work) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isClosed {
		return false
	}
	r.works[w] = struct{}{}
	return true
}

// Remove unregisters an active work item.
func (r *Registry) Remove(w *Work) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.works, w)
}

// GetSession returns the external agent session ID associated with a conversation.
func (r *Registry) GetSession(conversation string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[conversation]
}

// SetSession records the external agent session ID for a conversation.
func (r *Registry) SetSession(conversation, sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[conversation] = sessionID
}

// EndConversation stops any active work for the conversation and drops its session.
func (r *Registry) EndConversation(_ context.Context, conversation string) error {
	r.mu.Lock()
	delete(r.sessions, conversation)
	var matching []*Work
	for w := range r.works {
		if w.conversation == conversation {
			matching = append(matching, w)
		}
	}
	r.mu.Unlock()

	for _, w := range matching {
		w.Stop()
	}
	return nil
}

// Close stops all active work items and closes the registry.
func (r *Registry) Close() error {
	r.mu.Lock()
	r.isClosed = true
	var all []*Work
	for w := range r.works {
		all = append(all, w)
	}
	r.mu.Unlock()

	for _, w := range all {
		w.Stop()
	}
	return nil
}

// Supervise sets process group attributes to ensure clean child termination.
func Supervise(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

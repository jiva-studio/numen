package agentrun

import (
	"context"
	"sync"
)

// Registry is the work an agent started and has not been told is over, and the session each
// conversation is on. The zero value is ready.
type Registry struct {
	mu       sync.Mutex
	running  map[*Work]bool
	isClosed bool
	sessions map[string]string
}

// Add keeps the work so that Close can reach it, and counts its reader, which calls End on the work
// when it is done. It is false once the registry is closed.
func (r *Registry) Add(w *Work) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isClosed {
		return false
	}
	if r.running == nil {
		r.running = map[*Work]bool{}
	}
	w.reader.Add(1)
	r.running[w] = true
	return true
}

// Remove drops the work once its reader is done.
func (r *Registry) Remove(w *Work) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, w)
}

// Close stops every work and waits for them. Work added afterwards is refused.
func (r *Registry) Close() error {
	r.mu.Lock()
	r.isClosed = true
	running := make([]*Work, 0, len(r.running))
	for w := range r.running {
		running = append(running, w)
	}
	r.running = nil
	r.mu.Unlock()

	stopAll(running)
	return nil
}

// EndConversation stops everything still answered in the conversation, then lets go of its
// session. Empty is no conversation.
func (r *Registry) EndConversation(_ context.Context, conversation string) error {
	if conversation == "" {
		return nil
	}

	r.mu.Lock()
	answering := make([]*Work, 0, len(r.running))
	for w := range r.running {
		if w.conversation == conversation {
			answering = append(answering, w)
		}
	}
	r.mu.Unlock()

	stopAll(answering)

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, conversation)
	return nil
}

// GetSession returns the session the next task of the conversation is asked in, empty where
// nothing has been asked yet.
func (r *Registry) GetSession(conversation string) string {
	if conversation == "" {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[conversation]
}

// SetSession keeps the session a conversation is on. A task that named no conversation keeps
// none.
func (r *Registry) SetSession(conversation, session string) {
	if conversation == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions == nil {
		r.sessions = map[string]string{}
	}
	r.sessions[conversation] = session
}

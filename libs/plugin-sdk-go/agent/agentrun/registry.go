package agentrun

import (
	"context"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// Registry tracks the work an agent started that has not ended, and the session of each
// conversation. The zero value is ready to use.
type Registry struct {
	mu       sync.Mutex
	running  map[*Work]bool
	isClosed bool
	sessions map[string]string
}

// Add registers the work for Close, and counts its reader, which calls End on the work
// when it finishes. It returns false once the registry is closed.
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

// Close stops the registered work, refuses further work, and returns the joined stop errors.
func (r *Registry) Close() error {
	r.mu.Lock()
	r.isClosed = true
	running := make([]*Work, 0, len(r.running))
	for w := range r.running {
		running = append(running, w)
	}
	r.running = nil
	r.mu.Unlock()

	return stopAll(running)
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

	stopErr := stopAll(answering)

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, conversation)
	return stopErr
}

// GetSession returns the session in which the next task of the conversation runs, empty when
// nothing has run yet.
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

// FinishConversation answers FinishConversation: it ends the conversation of the request.
func (r *Registry) FinishConversation(
	ctx context.Context,
	req *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	if err := r.EndConversation(ctx, req.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

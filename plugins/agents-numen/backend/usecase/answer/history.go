package answer

import (
	"sync"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

// history keeps the transcript of each conversation.
type history struct {
	mu            sync.Mutex
	conversations map[string][]conversation.Message
}

func newHistory() *history {
	return &history{mu: sync.Mutex{}, conversations: map[string][]conversation.Message{}}
}

func (h *history) get(id string) []conversation.Message {
	if id == "" {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]conversation.Message(nil), h.conversations[id]...)
}

func (h *history) append(id string, transcript []conversation.Message) {
	if id == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conversations[id] = append(h.conversations[id], transcript...)
}

func (h *history) discard(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conversations, id)
}

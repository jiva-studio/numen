package answer

import (
	"sync"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

const (
	// maxConversations is how many conversations the history keeps; a new one forgets the one used longest ago.
	maxConversations = 8
	// idleLimit is how long a conversation is kept after its last turn.
	idleLimit = 30 * time.Minute
)

// kept is the transcript of one conversation and when its last turn ended.
type kept struct {
	messages []conversation.Message
	usedAt   time.Time
}

// history keeps the transcript of each conversation.
type history struct {
	mu            sync.Mutex
	clock         ports.Clock
	conversations map[string]*kept
}

func newHistory(clock ports.Clock) *history {
	return &history{mu: sync.Mutex{}, clock: clock, conversations: map[string]*kept{}}
}

func (h *history) get(id string) []conversation.Message {
	if id == "" {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.forgetIdle(h.clock.Now())
	one, isKept := h.conversations[id]
	if !isKept {
		return nil
	}
	return append([]conversation.Message(nil), one.messages...)
}

func (h *history) append(id string, transcript []conversation.Message) {
	if id == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.clock.Now()
	h.forgetIdle(now)
	one, isKept := h.conversations[id]
	if !isKept {
		if len(h.conversations) >= maxConversations {
			h.forgetOldest()
		}
		one = &kept{messages: nil, usedAt: now}
		h.conversations[id] = one
	}
	one.messages = append(one.messages, transcript...)
	one.usedAt = now
}

func (h *history) forgetIdle(now time.Time) {
	for id, one := range h.conversations {
		if now.Sub(one.usedAt) >= idleLimit {
			delete(h.conversations, id)
		}
	}
}

func (h *history) forgetOldest() {
	oldest := ""
	for id, one := range h.conversations {
		if oldest == "" || one.usedAt.Before(h.conversations[oldest].usedAt) {
			oldest = id
		}
	}
	delete(h.conversations, oldest)
}

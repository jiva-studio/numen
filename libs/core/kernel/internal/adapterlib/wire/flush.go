package wire

import (
	"strconv"
	"sync"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// flushState is what a page has left when it answers.
type flushState int

func parseFlushResult(result v1.FlushResult) flushState {
	switch result {
	case v1.FlushResult_FLUSH_RESULT_NOTHING, v1.FlushResult_FLUSH_RESULT_WRITTEN:
		return flushWritten
	case v1.FlushResult_FLUSH_RESULT_UNSAVED:
		return flushUnsaved
	default:
		return flushUnanswered
	}
}

// flushCoordinator tracks the connected clients and the running flush request.
//
// A client holds unwritten work in a buffer this process cannot read, so it
// receives a write request and responds when done. A disconnected client is not
// awaited, except when its stream ended in the flushUnsaved state: its work is
// still unwritten, so it counts as unanswered until a client reconnects.
type flushCoordinator struct {
	mu      sync.Mutex
	next    int
	clients map[string]*client
	current *flushRequest
}

// client is one browser tab connected to the window, with its flush state.
type client struct {
	flushTokens    chan string
	state          flushState
	isDisconnected bool
}

// flushRequest is one flush request to all clients, with its progress.
type flushRequest struct {
	allWritten     chan struct{}
	questionRaised chan struct{}
	superseded     chan struct{}

	isAllWrittenSignaled bool
	isQuestionSignaled   bool
	isSuperseded         bool
}

func (r *flushRequest) isPending() bool {
	select {
	case <-r.questionRaised:
		return true
	default:
		return false
	}
}

func (r *flushRequest) isWritten() bool {
	select {
	case <-r.allWritten:
		return true
	default:
		return false
	}
}

func (l *flushCoordinator) registerClient() (string, <-chan string, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.clients == nil {
		l.clients = map[string]*client{}
	}
	// One window shows one vault, so a connecting client replaces the disconnected
	// client and takes over its pending work.
	for token, p := range l.clients {
		if p.isDisconnected {
			delete(l.clients, token)
		}
	}
	token := strconv.Itoa(l.next)
	l.next++
	p := &client{flushTokens: make(chan string, 1)}
	l.clients[token] = p
	if l.isRunning() {
		p.flushTokens <- token
	}
	l.updateProgress()

	return token, p.flushTokens, func() { l.disconnectClient(token, p) }
}

func (l *flushCoordinator) isRunning() bool {
	return l.current != nil && !l.current.isSuperseded
}

func (l *flushCoordinator) disconnectClient(token string, p *client) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.clients[token] != p || p.isDisconnected {
		return
	}
	close(p.flushTokens)
	// A page that went with a question standing is the only place that work
	// exists. It is remembered, and it is remembered as silence, which is what
	// a request waits its bound for.
	if p.state == flushUnsaved {
		p.state = flushUnanswered
		p.isDisconnected = true
	} else {
		delete(l.clients, token)
	}
	l.updateProgress()
}

func (l *flushCoordinator) startRequest() *flushRequest {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isRunning() {
		l.current.isSuperseded = true
		close(l.current.superseded)
	}
	l.current = &flushRequest{
		allWritten:     make(chan struct{}),
		questionRaised: make(chan struct{}),
		superseded:     make(chan struct{}),
	}
	for token, p := range l.clients {
		p.state = flushUnanswered
		if p.isDisconnected {
			continue
		}
		select {
		case p.flushTokens <- token:
		default:
		}
	}
	l.updateProgress()
	return l.current
}

func (l *flushCoordinator) endRequest() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isRunning() {
		return
	}
	l.current.isSuperseded = true
	close(l.current.superseded)
	l.current = nil
}

func (l *flushCoordinator) getRunningRequest() *flushRequest {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isRunning() {
		return nil
	}
	return l.current
}

func (l *flushCoordinator) recordFlush(token string, response flushState) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if p, found := l.clients[token]; found && !p.isDisconnected {
		p.state = response
	}
	l.updateProgress()
}

func (l *flushCoordinator) updateProgress() {
	r := l.current
	if r == nil || r.isSuperseded {
		return
	}
	for _, p := range l.clients {
		if p.state == flushUnsaved && !r.isQuestionSignaled {
			r.isQuestionSignaled = true
			close(r.questionRaised)
		}
	}
	if r.isAllWrittenSignaled {
		return
	}
	for _, p := range l.clients {
		if p.state != flushWritten {
			return
		}
	}
	r.isAllWrittenSignaled = true
	close(r.allWritten)
}

const (
	// flushUnanswered is a client that has not responded to this request.
	flushUnanswered flushState = iota
	// flushWritten is a client with no unsaved work left.
	flushWritten
	// flushUnsaved is a client with unwritten work while the user decides what to
	// do with it.
	flushUnsaved
)

package task

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"
)

// maxSessions is how many conversations the plugin keeps a session for.
const maxSessions = 8

// idleLimit is how long a kept session may go unused before it is closed.
const idleLimit = 30 * time.Minute

var errClosed = errors.New("the plugin is stopping")

// pool keeps the sessions of conversations by key. The errors of closing a session outside closeIdle
// and close are kept until one of them returns them: a task does not fail on the session of another, nor
// on its own after its turn has ended.
type pool struct {
	clock     Clock
	mu        sync.Mutex
	kept      map[string]*conversation
	isClosed  bool
	closeErrs error
}

func newPool(clock Clock) *pool {
	return &pool{clock: clock, mu: sync.Mutex{}, kept: map[string]*conversation{}, isClosed: false, closeErrs: nil}
}

// keepError keeps the error of closing a session for the next closeIdle or close.
func (p *pool) keepError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closeErrs = errors.Join(p.closeErrs, err)
}

// takeErrors returns the kept errors of closing sessions and keeps none.
func (p *pool) takeErrors() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	err := p.closeErrs
	p.closeErrs = nil
	return err
}

// acquire returns the conversation of key with its token held, and whether the pool keeps it. A
// conversation the pool has no session for is new and holds none yet; the caller opens it. An empty key
// is a conversation of its own that the pool does not keep.
func (p *pool) acquire(ctx context.Context, key string) (c *conversation, isNew, isKept bool, err error) {
	if key == "" {
		return newConversation("", p.clock.Now()), true, false, nil
	}
	for {
		p.mu.Lock()
		if p.isClosed {
			p.mu.Unlock()
			return nil, false, false, errClosed
		}
		found := p.kept[key]
		if found == nil {
			created := newConversation(key, p.clock.Now())
			evicted, hasRoom := p.makeRoom()
			if hasRoom {
				p.kept[key] = created
			}
			p.mu.Unlock()
			p.keepError(closeConversation(ctx, evicted))
			return created, true, hasRoom, nil
		}
		p.mu.Unlock()
		select {
		case found.turn <- struct{}{}:
		case <-ctx.Done():
			return nil, false, false, ctx.Err()
		}
		p.mu.Lock()
		isGone := found.isGone
		p.mu.Unlock()
		if !isGone {
			return found, false, true, nil
		}
		<-found.turn
	}
}

// makeRoom takes out the least recently used idle conversation when the pool is full, and reports whether
// there is room. Every conversation of a full pool may be running a turn, and then there is none. The
// caller holds the lock and closes the one taken out.
func (p *pool) makeRoom() (*conversation, bool) {
	if len(p.kept) < maxSessions {
		return nil, true
	}
	kept := slices.Collect(maps.Values(p.kept))
	slices.SortFunc(kept, func(a, b *conversation) int { return a.lastUsed.Compare(b.lastUsed) })
	for _, one := range kept {
		if p.tryTake(one) {
			return one, true
		}
	}
	return nil, false
}

// tryTake takes the conversation out of the pool when no turn runs in it. The caller holds the lock and
// then holds the token of the conversation.
func (p *pool) tryTake(c *conversation) bool {
	select {
	case c.turn <- struct{}{}:
	default:
		return false
	}
	delete(p.kept, c.key)
	c.isGone = true
	return true
}

// release ends the turn of the conversation. A conversation the pool does not keep is taken out and
// closed, and one whose session broke has its program stopped.
func (p *pool) release(ctx context.Context, c *conversation, isKept, isBroken bool) {
	p.mu.Lock()
	c.lastUsed = p.clock.Now()
	isClosing := !isKept || isBroken || c.isGone
	if isKept && isClosing && p.kept[c.key] == c {
		delete(p.kept, c.key)
	}
	if isClosing {
		c.isGone = true
	}
	p.mu.Unlock()
	<-c.turn
	if isClosing {
		p.keepError(c.close(ctx, isBroken))
	}
}

// closeIdle closes the conversations that have gone unused for the idle limit, and returns the errors
// of closing sessions since the last call.
func (p *pool) closeIdle(ctx context.Context) error {
	now := p.clock.Now()
	var idle []*conversation
	p.mu.Lock()
	for _, one := range p.kept {
		if now.Sub(one.lastUsed) >= idleLimit && p.tryTake(one) {
			idle = append(idle, one)
		}
	}
	p.mu.Unlock()
	err := p.takeErrors()
	for _, one := range idle {
		err = errors.Join(err, closeConversation(ctx, one))
	}
	return err
}

// close closes every conversation and keeps no other. A conversation whose turn still runs is stopped.
func (p *pool) close(ctx context.Context) error {
	p.mu.Lock()
	p.isClosed = true
	kept := p.kept
	p.kept = map[string]*conversation{}
	var idle, running []*conversation
	for _, one := range kept {
		one.isGone = true
		select {
		case one.turn <- struct{}{}:
			idle = append(idle, one)
		default:
			running = append(running, one)
		}
	}
	p.mu.Unlock()
	err := p.takeErrors()
	for _, one := range idle {
		err = errors.Join(err, closeConversation(ctx, one))
	}
	for _, one := range running {
		err = errors.Join(err, one.close(ctx, true))
	}
	return err
}

// closeConversation closes a conversation taken out of the pool whose token the caller holds.
func closeConversation(ctx context.Context, c *conversation) error {
	if c == nil {
		return nil
	}
	<-c.turn
	return c.close(ctx, false)
}

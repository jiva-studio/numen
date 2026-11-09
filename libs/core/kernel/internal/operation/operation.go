// Package operation holds what one operation learned about an address for as
// long as that operation runs.
//
// An operation begins at the use case that requests an address more than once,
// and ends when that use case returns. The stored state is released then, and no
// later operation sees it.
package operation

import (
	"context"
	"sync"
)

type scopeKey struct{}

type entry struct {
	value   string
	release func()
}

type scope struct {
	mu      sync.Mutex
	entries map[string]entry
}

// Begin starts an operation and answers the context it runs under and the
// function that ends it. Beginning inside an operation joins it, and only the
// outermost end releases what was remembered.
func Begin(ctx context.Context) (context.Context, func()) {
	if _, has := ctx.Value(scopeKey{}).(*scope); has {
		return ctx, func() {}
	}
	s := &scope{entries: map[string]entry{}}
	return context.WithValue(ctx, scopeKey{}, s), s.end
}

func (s *scope) end() {
	s.mu.Lock()
	entries := s.entries
	s.entries = map[string]entry{}
	s.mu.Unlock()
	for _, one := range entries {
		if one.release != nil {
			one.release()
		}
	}
}

// Recall answers what the operation remembered under a key, and whether it
// remembered anything.
func Recall(ctx context.Context, key string) (string, bool) {
	s, has := ctx.Value(scopeKey{}).(*scope)
	if !has {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	one, found := s.entries[key]
	return one.value, found
}

// Remember keeps a value under a key until the operation ends, and runs
// release then. Outside an operation nothing is kept and release runs at once.
func Remember(ctx context.Context, key, value string, release func()) {
	s, has := ctx.Value(scopeKey{}).(*scope)
	if !has {
		if release != nil {
			release()
		}
		return
	}
	s.mu.Lock()
	previous, replaced := s.entries[key]
	s.entries[key] = entry{value: value, release: release}
	s.mu.Unlock()
	if replaced && previous.release != nil {
		previous.release()
	}
}

// IsActive reports whether the context belongs to an operation.
func IsActive(ctx context.Context) bool {
	_, has := ctx.Value(scopeKey{}).(*scope)
	return has
}

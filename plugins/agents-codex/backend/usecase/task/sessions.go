package task

import (
	"slices"
	"sync"
)

// maxSessions is how many conversations the plugin keeps the thread of.
const maxSessions = 64

// sessions keeps the thread of the program each conversation continues, the most recently recorded ones
// up to maxSessions.
type sessions struct {
	mu    sync.Mutex
	byKey map[string]string
	order []string
}

func (s *sessions) get(key string) string {
	if key == "" {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byKey[key]
}

func (s *sessions) set(key, session string) {
	if key == "" || session == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byKey == nil {
		s.byKey = map[string]string{}
	}
	s.order = slices.DeleteFunc(s.order, func(one string) bool { return one == key })
	s.order = append(s.order, key)
	s.byKey[key] = session
	for len(s.order) > maxSessions {
		delete(s.byKey, s.order[0])
		s.order = s.order[1:]
	}
}

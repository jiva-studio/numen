package source

import "github.com/jiva-studio/numen/modules/libs/core/domain"

// queue is the sources a person named, in the order they named them. A run
// empties it before it takes up anything nobody asked for, and a source named
// twice waits in it once.
//
// Nothing here is filtered. A person naming a file by hand has said that this
// file is worth the machine's time, so the size a queue leaves alone and an
// installation that transcribes nothing on its own both hold no sway here.
//
// It locks nothing: it is held by the run that owns it, under that run's lock.
type queue struct {
	line   []wanted
	queued map[string]bool
}

// wanted is one source a person named: the vault it is in, and where.
type wanted struct {
	vault domain.Vault
	path  string
}

// add puts a source at the back of the line and says whether it went in.
func (q *queue) add(v domain.Vault, path string) bool {
	if q.queued == nil {
		q.queued = map[string]bool{}
	}
	key := getKey(v, path)
	if q.queued[key] {
		return false
	}
	q.queued[key] = true
	q.line = append(q.line, wanted{vault: v, path: path})
	return true
}

// take is the source at the front of the line, and false where nobody is
// waiting.
func (q *queue) take() (wanted, bool) {
	if len(q.line) == 0 {
		return wanted{}, false
	}
	one := q.line[0]
	q.line = q.line[1:]
	delete(q.queued, getKey(one.vault, one.path))
	return one, true
}

// countWaiting is how many sources are named and not yet taken up.
func (q *queue) countWaiting() int { return len(q.line) }

// getKey is one source of one vault, as the one string a set is keyed by.
func getKey(v domain.Vault, path string) string { return string(v.ID) + "\x00" + path }

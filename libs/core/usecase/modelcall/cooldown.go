package modelcall

import (
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

type connectionVersion struct {
	id       string
	revision int
}

type cooldownEntry struct {
	until     time.Time
	isForever bool
	reason    connectionport.FailureReason
}

// cooldown is the connections set aside, by id and revision. An edit changes the revision, so the entry
// of the old revision no longer matches. It is kept in memory.
type cooldown struct {
	mu      sync.Mutex
	entries map[connectionVersion]cooldownEntry
}

func newCooldown() *cooldown { return &cooldown{entries: map[connectionVersion]cooldownEntry{}} }

func (c *cooldown) get(version connectionVersion, now time.Time) (cooldownEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, found := c.entries[version]
	if !found {
		return cooldownEntry{}, false
	}
	if !entry.isForever && !now.Before(entry.until) {
		delete(c.entries, version)
		return cooldownEntry{}, false
	}
	return entry, true
}

func (c *cooldown) start(version connectionVersion, reason connectionport.FailureReason, retryAfter time.Duration, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for other := range c.entries {
		if other.id == version.id && other.revision != version.revision {
			delete(c.entries, other)
		}
	}
	entry := cooldownEntry{reason: reason}
	switch {
	case lastsUntilEdited(reason):
		entry.isForever = true
	case retryAfter > 0:
		entry.until = now.Add(min(retryAfter, MaxCooldown))
	default:
		entry.until = now.Add(DefaultCooldown)
	}
	c.entries[version] = entry
}

func lastsUntilEdited(reason connectionport.FailureReason) bool {
	return reason == connectionport.FailureReasonKeyRejected || reason == connectionport.FailureReasonPaymentRequired
}

type notice struct {
	version connectionVersion
	reason  connectionport.FailureReason
}

// told is the notices already given, for each connection revision and reason.
type told struct {
	mu   sync.Mutex
	done map[notice]bool
}

func newTold() *told { return &told{done: map[notice]bool{}} }

// claim reports whether the notice was not given yet, and marks it given.
func (t *told) claim(one notice) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done[one] {
		return false
	}
	t.done[one] = true
	return true
}

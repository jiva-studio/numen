package changefeed

import (
	"sync"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Changes tells the watchers of a vault that a write of the index that can store changes in the vault's feed
// was committed. A watcher reads the head of the feed to learn whether it moved.
type Changes struct {
	mu        sync.Mutex
	next      int
	watchers  map[int]changeWatcher
	onPublish []func()
}

// SetPublishHandler registers a callback invoked whenever any write is published.
func (c *Changes) SetPublishHandler(report func()) {
	if c == nil || report == nil {
		return
	}
	c.mu.Lock()
	c.onPublish = append(c.onPublish, report)
	c.mu.Unlock()
}

type changeWatcher struct {
	vault  domain.VaultID
	report func()
}

// Watch calls report after each committed write of the vault that can store changes in its feed, until the
// function it returns is called. report runs inside the write's caller and does not block.
func (c *Changes) Watch(vaultID domain.VaultID, report func()) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.watchers == nil {
		c.watchers = map[int]changeWatcher{}
	}
	id := c.next
	c.next++
	c.watchers[id] = changeWatcher{vault: vaultID, report: report}
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.watchers, id)
	}
}

// Publish calls the watchers of the vault.
func (c *Changes) Publish(vaultID domain.VaultID) {
	if c == nil {
		return
	}
	c.mu.Lock()
	var reports []func()
	reports = append(reports, c.onPublish...)
	for _, one := range c.watchers {
		if one.vault == vaultID {
			reports = append(reports, one.report)
		}
	}
	c.mu.Unlock()
	for _, report := range reports {
		report()
	}
}

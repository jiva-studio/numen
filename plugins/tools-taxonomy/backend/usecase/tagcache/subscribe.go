package tagcache

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Change is the tags of the notes whose tags an update changed, by path: a note left with no tags,
// removed or moved away has an empty list.
type Change struct {
	Changed map[string][]string
}

// GetMap returns the tags of the notes of the vault that have any, by path, after bringing the tags up
// to date.
func (c *Caches) GetMap(ctx context.Context, id domain.VaultID) (map[string][]string, error) {
	one := c.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	if err := c.update(ctx, id, one); err != nil {
		return nil, err
	}
	return one.getTagsByPath(), nil
}

// Subscribe returns the changes of the tags of the vault, one for each update that changed any, and the
// function that ends the subscription and closes the channel. A subscriber that reads slowly is sent the
// changes it missed as one, so it never holds up an update.
func (c *Caches) Subscribe(id domain.VaultID) (<-chan Change, func()) {
	return c.getVault(id).follow.add()
}

// follow is the subscribers to the changes of the tags of one vault and the tags last published to them;
// the first tags published are the baseline and change nothing.
type follow struct {
	mu        sync.Mutex
	published map[string][]string
	listeners map[*listener]struct{}
}

func newFollow() follow {
	return follow{mu: sync.Mutex{}, published: nil, listeners: map[*listener]struct{}{}}
}

// listener is one subscriber: the changes it has not taken yet, merged by path, and the channel it takes them from.
type listener struct {
	mu      sync.Mutex
	pending map[string][]string
	wake    chan struct{}
	done    chan struct{}
	out     chan Change
}

func (f *follow) add() (<-chan Change, func()) {
	one := &listener{
		mu:      sync.Mutex{},
		pending: map[string][]string{},
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		out:     make(chan Change),
	}
	f.mu.Lock()
	f.listeners[one] = struct{}{}
	f.mu.Unlock()
	go one.run()
	var once sync.Once
	return one.out, func() {
		once.Do(func() {
			f.mu.Lock()
			delete(f.listeners, one)
			f.mu.Unlock()
			close(one.done)
		})
	}
}

func (f *follow) publish(current map[string][]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	previous := f.published
	f.published = current
	if previous == nil {
		return
	}
	changed := diffTags(previous, current)
	if len(changed) == 0 {
		return
	}
	for one := range f.listeners {
		one.push(changed)
	}
}

func diffTags(previous, current map[string][]string) map[string][]string {
	changed := map[string][]string{}
	for path, tags := range current {
		if !slices.Equal(previous[path], tags) {
			changed[path] = tags
		}
	}
	for path := range previous {
		if _, has := current[path]; !has {
			changed[path] = []string{}
		}
	}
	return changed
}

func (l *listener) push(changed map[string][]string) {
	l.mu.Lock()
	maps.Copy(l.pending, changed)
	l.mu.Unlock()
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

func (l *listener) take() map[string][]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	taken := l.pending
	l.pending = map[string][]string{}
	return taken
}

func (l *listener) run() {
	defer close(l.out)
	for {
		select {
		case <-l.done:
			return
		case <-l.wake:
		}
		taken := l.take()
		if len(taken) == 0 {
			continue
		}
		select {
		case <-l.done:
			return
		case l.out <- Change{Changed: taken}:
		}
	}
}

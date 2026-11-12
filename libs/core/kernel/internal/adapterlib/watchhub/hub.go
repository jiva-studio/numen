// Package watchhub shares the operating system's folder watches between everything in the process
// that follows a folder.
//
// Watches never nest: a folder requested inside a folder already watched
// recursively is served by that watch, and a folder requested over existing
// watches replaces them. notify's recursive tree, which FSEvents uses, removes
// the outer watch when a watch nested inside it stops.
package watchhub

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rjeczalik/notify"
)

// handover is how many events a placed watch holds before they are handed to the followers.
const handover = 1024

// follower is one caller following a folder.
type follower struct {
	dir         string
	isRecursive bool
	c           chan<- notify.EventInfo
	watch       *placed
}

func (f *follower) isFollowing(path string) bool {
	if f.isRecursive {
		return isWithin(path, f.dir)
	}
	return path == f.dir || filepath.Dir(path) == f.dir
}

// placed is one watch of the operating system and the followers it serves.
type placed struct {
	dir         string
	isRecursive bool
	raw         chan notify.EventInfo
	quit        chan struct{}

	mu        sync.Mutex
	followers map[*follower]struct{}
}

func (p *placed) isCovering(dir string, isRecursive bool) bool {
	if p.isRecursive {
		return isWithin(dir, p.dir)
	}
	return dir == p.dir && !isRecursive
}

func (p *placed) forward() {
	for {
		select {
		case <-p.quit:
			return
		case event := <-p.raw:
			p.mu.Lock()
			for one := range p.followers {
				if !one.isFollowing(event.Path()) {
					continue
				}
				select {
				case one.c <- event:
				default:
				}
			}
			p.mu.Unlock()
		}
	}
}

// system is the operating system's watches, as notify places them.
type system interface {
	Watch(path string, c chan<- notify.EventInfo, events ...notify.Event) error
	Stop(c chan<- notify.EventInfo)
}

// notifySystem places watches through notify.
type notifySystem struct{}

func (notifySystem) Watch(path string, c chan<- notify.EventInfo, events ...notify.Event) error {
	return notify.Watch(path, c, events...)
}

func (notifySystem) Stop(c chan<- notify.EventInfo) { notify.Stop(c) }

// Hub places the watches of one container. notify keeps one watcher per process, so the hub that
// merges overlapping watches is one per container, and a container is one per process.
type Hub struct {
	system system

	mu     sync.Mutex
	placed map[*placed]struct{}
}

func newHub(watches system) *Hub {
	return &Hub{system: watches, placed: make(map[*placed]struct{})}
}

// ErrNoHub is returned by a watch asked of no hub.
var ErrNoHub = errors.New("a watch was asked of no hub")

// NewHub returns a hub placing its watches through notify.
func NewHub() *Hub { return newHub(notifySystem{}) }

// Watch reports to c each event at dir and the entries in it, or anywhere under it when recursive,
// until stop is called. dir names an existing folder.
func (h *Hub) Watch(dir string, isRecursive bool, c chan<- notify.EventInfo) (stop func(), err error) {
	if h == nil {
		return nil, ErrNoHub
	}
	canonical, err := getCanonicalPath(dir)
	if err != nil {
		return nil, err
	}
	one := &follower{dir: canonical, isRecursive: isRecursive, c: c}

	h.mu.Lock()
	defer h.mu.Unlock()
	for p := range h.placed {
		if p.isCovering(canonical, isRecursive) {
			p.add(one)
			return h.newStop(one), nil
		}
	}
	if err := h.place(canonical, isRecursive, one); err != nil {
		return nil, err
	}
	return h.newStop(one), nil
}

func (h *Hub) place(dir string, isRecursive bool, one *follower) error {
	dir, isRecursive, inner := h.getInnerWatches(dir, isRecursive)

	// The watches inner over stop before the new one is placed: notify takes the new watch away
	// when one nested inside it stops.
	for _, p := range inner {
		h.unplace(p)
	}
	next := &placed{dir: dir, isRecursive: isRecursive, followers: make(map[*follower]struct{})}
	if err := h.start(next); err != nil {
		for _, p := range inner {
			err = errors.Join(err, h.start(p))
		}
		return err
	}
	for _, p := range inner {
		for moved := range p.followers {
			next.add(moved)
		}
	}
	next.add(one)
	return nil
}

func (h *Hub) getInnerWatches(dir string, isRecursive bool) (string, bool, []*placed) {
	for p := range h.placed {
		if isWithin(dir, p.dir) {
			dir, isRecursive = p.dir, true
		}
	}
	var inner []*placed
	for p := range h.placed {
		if !isWithin(p.dir, dir) {
			continue
		}
		inner = append(inner, p)
		if p.dir != dir || p.isRecursive {
			isRecursive = true
		}
	}
	return dir, isRecursive, inner
}

func (h *Hub) start(p *placed) error {
	target := p.dir
	if p.isRecursive {
		target = filepath.Join(p.dir, "...")
	}
	p.raw = make(chan notify.EventInfo, handover)
	p.quit = make(chan struct{})
	if err := h.system.Watch(target, p.raw, notify.All); err != nil {
		return fmt.Errorf("watch %s: %w", p.dir, err)
	}
	h.placed[p] = struct{}{}
	go p.forward()
	return nil
}

func (h *Hub) unplace(p *placed) {
	h.system.Stop(p.raw)
	close(p.quit)
	delete(h.placed, p)
}

func (h *Hub) newStop(one *follower) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			p := one.watch
			p.mu.Lock()
			delete(p.followers, one)
			isEmpty := len(p.followers) == 0
			p.mu.Unlock()
			if isEmpty {
				h.unplace(p)
			}
		})
	}
}

func (p *placed) add(one *follower) {
	p.mu.Lock()
	defer p.mu.Unlock()
	one.watch = p
	p.followers[one] = struct{}{}
}

func getCanonicalPath(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("watch %s: %w", dir, err)
	}
	return resolved, nil
}

func isWithin(path, dir string) bool {
	if path == dir {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(dir, string(filepath.Separator))+string(filepath.Separator))
}

// Package watchhub shares the operating system's folder watches between everything in the process
// that follows a folder.
//
// Watches never nest: a folder requested inside a folder already watched
// recursively is served by that watch, and a folder requested over existing
// watches replaces them. notify's recursive tree, which FSEvents uses, removes
// the outer watch when a watch nested inside it stops.
package watchhub

import (
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

// isFollowing reports whether an event at path is the follower's.
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

// isCovering reports whether the watch delivers the events a follower of dir requires.
func (p *placed) isCovering(dir string, isRecursive bool) bool {
	if p.isRecursive {
		return isWithin(dir, p.dir)
	}
	return dir == p.dir && !isRecursive
}

// forward hands each event to the followers it is for, until the watch is inner away. A follower
// that is not reading loses the event, as notify itself does.
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

// hub is every watch the process placed.
type hub struct {
	system system

	mu     sync.Mutex
	placed map[*placed]struct{}
}

// newHub is a hub placing its watches through the system given.
func newHub(watches system) *hub {
	return &hub{system: watches, placed: make(map[*placed]struct{})}
}

var shared = newHub(notifySystem{})

// Watch reports to c each event at dir and the entries in it, or anywhere under it when recursive,
// until stop is called. dir names an existing folder.
func Watch(dir string, isRecursive bool, c chan<- notify.EventInfo) (stop func(), err error) {
	return shared.watch(dir, isRecursive, c)
}

func (h *hub) watch(dir string, isRecursive bool, c chan<- notify.EventInfo) (func(), error) {
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

// place puts a watch over dir that takes over every watch at or under it, and over the one above it
// when that one watches its own folder alone.
func (h *hub) place(dir string, isRecursive bool, one *follower) error {
	dir, isRecursive, inner := h.getInnerWatches(dir, isRecursive)

	// The watches inner over stop before the new one is placed: notify takes the new watch away
	// when one nested inside it stops.
	for _, p := range inner {
		h.unplace(p)
	}
	next := &placed{dir: dir, isRecursive: isRecursive, followers: make(map[*follower]struct{})}
	if err := h.start(next); err != nil {
		for _, p := range inner {
			_ = h.start(p)
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

// getInnerWatches is the folder and the depth a new watch is placed at to serve a follower of dir,
// and the watches it takes over.
func (h *hub) getInnerWatches(dir string, isRecursive bool) (string, bool, []*placed) {
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

// start registers the watch with the system and forwards its events. A stopped
// watch is started again on new channels.
func (h *hub) start(p *placed) error {
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

// unplace takes a watch away from the operating system and from the hub.
func (h *hub) unplace(p *placed) {
	h.system.Stop(p.raw)
	close(p.quit)
	delete(h.placed, p)
}

// newStop returns the function that detaches the follower; repeated calls have
// no further effect. A watch with no remaining followers is stopped.
func (h *hub) newStop(one *follower) func() {
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

// add puts a follower on the watch.
func (p *placed) add(one *follower) {
	p.mu.Lock()
	defer p.mu.Unlock()
	one.watch = p
	p.followers[one] = struct{}{}
}

// getCanonicalPath is the absolute path of dir with its links resolved, which is how notify names
// the paths of its events.
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

// isWithin reports whether path is dir or under it.
func isWithin(path, dir string) bool {
	if path == dir {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(dir, string(filepath.Separator))+string(filepath.Separator))
}

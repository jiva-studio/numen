package watchhub

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rjeczalik/notify"
)

// event is one change the fake system reports.
type event struct{ path string }

func (e event) Event() notify.Event { return notify.Create }
func (e event) Path() string        { return e.path }
func (e event) Sys() any            { return nil }

// fakeSystem places watches in memory and fails a test that places one inside another.
type fakeSystem struct {
	t *testing.T

	mu      sync.Mutex
	watched map[chan<- notify.EventInfo]string
	calls   []string
}

func newFakeSystem(t *testing.T) *fakeSystem {
	return &fakeSystem{t: t, watched: make(map[chan<- notify.EventInfo]string)}
}

func getBase(target string) (string, bool) {
	base, isRecursive := strings.CutSuffix(target, string(filepath.Separator)+"...")
	return base, isRecursive
}

func (f *fakeSystem) Watch(target string, c chan<- notify.EventInfo, _ ...notify.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	base, _ := getBase(target)
	for _, other := range f.watched {
		otherBase, _ := getBase(other)
		if isWithin(base, otherBase) || isWithin(otherBase, base) {
			f.t.Errorf("%s is placed while %s is watched", target, other)
		}
	}
	f.watched[c] = target
	f.calls = append(f.calls, "watch "+target)
	return nil
}

func (f *fakeSystem) Stop(c chan<- notify.EventInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "stop "+f.watched[c])
	delete(f.watched, c)
}

// getWatched are the watches placed now.
func (f *fakeSystem) getWatched() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, target := range f.watched {
		out = append(out, target)
	}
	slices.Sort(out)
	return out
}

// emit reports a change at path to the watch that covers it.
func (f *fakeSystem) emit(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for c, target := range f.watched {
		base, isRecursive := getBase(target)
		if (isRecursive && isWithin(path, base)) || path == base || filepath.Dir(path) == base {
			c <- event{path: path}
		}
	}
}

// makeTree is a folder holding the folders named, by name.
func makeTree(t *testing.T, names ...string) map[string]string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dirs := map[string]string{"": root}
	for _, name := range names {
		dirs[name] = filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(dirs[name], 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dirs
}

func follow(t *testing.T, h *hub, dir string, isRecursive bool) (chan notify.EventInfo, func()) {
	t.Helper()
	c := make(chan notify.EventInfo, 8)
	stop, err := h.watch(dir, isRecursive, c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)
	return c, stop
}

func expectEvent(t *testing.T, c chan notify.EventInfo, path string) {
	t.Helper()
	select {
	case got := <-c:
		if got.Path() != path {
			t.Errorf("heard %s, want %s", got.Path(), path)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("heard nothing of %s", path)
	}
}

func expectNoEvent(t *testing.T, c chan notify.EventInfo) {
	t.Helper()
	select {
	case got := <-c:
		t.Errorf("heard %s", got.Path())
	case <-time.After(50 * time.Millisecond):
	}
}

// A folder followed inside a tree already followed is served from the tree's watch, and letting it
// go leaves the tree watched.
func TestAFolderInsideAFollowedTreeSharesItsWatch(t *testing.T) {
	dirs := makeTree(t, "themes")
	system := newFakeSystem(t)
	h := newHub(system)
	vault, _ := follow(t, h, dirs[""], true)
	themes, stopThemes := follow(t, h, dirs["themes"], false)

	system.emit(filepath.Join(dirs["themes"], "dark.css"))
	expectEvent(t, themes, filepath.Join(dirs["themes"], "dark.css"))
	expectEvent(t, vault, filepath.Join(dirs["themes"], "dark.css"))

	stopThemes()
	if got := system.getWatched(); !slices.Equal(got, []string{filepath.Join(dirs[""], "...")}) {
		t.Fatalf("watched %v", got)
	}
	system.emit(filepath.Join(dirs[""], "Rivers.md"))
	expectEvent(t, vault, filepath.Join(dirs[""], "Rivers.md"))
	expectNoEvent(t, themes)
}

// A tree followed over a folder already followed takes its watch over: the inner watch stops before
// the tree's is placed, and its follower goes on hearing from the tree's.
func TestAWatchOverAnInnerWatchServesItsFollower(t *testing.T) {
	dirs := makeTree(t, "themes")
	system := newFakeSystem(t)
	h := newHub(system)
	themes, stopThemes := follow(t, h, dirs["themes"], false)
	vault, stopVault := follow(t, h, dirs[""], true)

	want := []string{"watch " + dirs["themes"], "stop " + dirs["themes"], "watch " + filepath.Join(dirs[""], "...")}
	if !slices.Equal(system.calls, want) {
		t.Fatalf("the system was asked %v", system.calls)
	}
	system.emit(filepath.Join(dirs["themes"], "dark.css"))
	expectEvent(t, themes, filepath.Join(dirs["themes"], "dark.css"))
	expectEvent(t, vault, filepath.Join(dirs["themes"], "dark.css"))

	stopThemes()
	system.emit(filepath.Join(dirs[""], "Rivers.md"))
	expectEvent(t, vault, filepath.Join(dirs[""], "Rivers.md"))

	stopVault()
	if got := system.getWatched(); len(got) != 0 {
		t.Errorf("still watched %v", got)
	}
}

// A folder followed inside one followed alone turns the watch on the outer folder into one over its
// tree, and each follower hears only what it follows.
func TestAFolderInsideAFolderFollowedAloneWidensItsWatch(t *testing.T) {
	dirs := makeTree(t, "settings")
	system := newFakeSystem(t)
	h := newHub(system)
	outer, _ := follow(t, h, dirs[""], false)
	inner, _ := follow(t, h, dirs["settings"], false)

	if got := system.getWatched(); !slices.Equal(got, []string{filepath.Join(dirs[""], "...")}) {
		t.Fatalf("watched %v", got)
	}
	system.emit(filepath.Join(dirs["settings"], "numen.json"))
	expectEvent(t, inner, filepath.Join(dirs["settings"], "numen.json"))
	expectNoEvent(t, outer)

	system.emit(filepath.Join(dirs[""], "index.db"))
	expectEvent(t, outer, filepath.Join(dirs[""], "index.db"))
	expectNoEvent(t, inner)
}

// Folders apart from each other are watched apart.
func TestFoldersApartAreWatchedApart(t *testing.T) {
	dirs := makeTree(t, "one", "two")
	system := newFakeSystem(t)
	h := newHub(system)
	follow(t, h, dirs["one"], true)
	_, stopTwo := follow(t, h, dirs["two"], false)

	want := []string{filepath.Join(dirs["one"], "..."), dirs["two"]}
	if got := system.getWatched(); !slices.Equal(got, want) {
		t.Fatalf("watched %v", got)
	}
	stopTwo()
	stopTwo()
	if got := system.getWatched(); !slices.Equal(got, want[:1]) {
		t.Errorf("watched %v", got)
	}
}

package search

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// concurrentVault is a vault held in memory that is safe to read from several
// goroutines. Every Read counts the reads in flight at that moment, and stays
// in flight for `hold`, so the most reads that ever overlapped is what the test
// sees as the most decoded sources held at once.
type concurrentVault struct {
	*fakeVault
	mu     sync.Mutex
	hold   time.Duration
	active int
	peak   int
	reads  map[string]int
	failAt string
	// blockOthers makes every read but failAt's wait for its context to end.
	blockOthers bool
}

func newConcurrentVault(content map[string]string) *concurrentVault {
	return &concurrentVault{
		fakeVault: &fakeVault{content: content, calls: map[string]int{}},
		reads:     map[string]int{},
	}
}

func (c *concurrentVault) Read(ctx context.Context, path string) ([]byte, error) {
	c.mu.Lock()
	c.active++
	c.peak = max(c.peak, c.active)
	c.reads[path]++
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.active--
		c.mu.Unlock()
	}()

	if c.failAt != "" {
		if path == c.failAt {
			return nil, errors.New("the disk failed")
		}
		if c.blockOthers {
			<-ctx.Done()
			return nil, ctx.Err()
		}
	}
	select {
	case <-time.After(c.hold):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	raw, target := c.content[path]
	if !target {
		return nil, fs.ErrNotExist
	}
	return []byte(raw), nil
}

// manySources is `count` sources of ten lines each, and a ranking of them that
// scatters two passages of every source: the first pass names each source's
// first line, the second pass names the sixth line of each in the reverse order.
func manySources(count int) (map[string]string, []domain.Passage) {
	content := make(map[string]string, count)
	var found []domain.Passage
	for i := range count {
		name := fmt.Sprintf("book-%02d.md", i)
		content[name] = proseOf(name, 10)
		found = append(found, domain.Passage{
			ChunkID: domain.ChunkID(fmt.Sprintf("a-%02d", i)), Source: name, Start: 0, Length: len(name),
		})
	}
	for i := count - 1; i >= 0; i-- {
		name := fmt.Sprintf("book-%02d.md", i)
		found = append(found, domain.Passage{
			ChunkID: domain.ChunkID(fmt.Sprintf("b-%02d", i)), Source: name,
			Start: 5 * (len(name) + 1), Length: len(name),
		})
	}
	return content, found
}

func TestReadAnswersAsManySourcesAsItIsGivenInTheOrderOfTheRanking(t *testing.T) {
	content, found := manySources(40)

	run := func(extractors int) []domain.Passage {
		vault := newConcurrentVault(content)
		u := Search{readers: concurrentReaders{vault}, extractors: extractors}
		out, err := u.read(t.Context(), domain.Vault{ID: "v"}, found)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	want := run(1)
	if len(want) != len(found) {
		t.Fatalf("%d passages came back, want %d", len(want), len(found))
	}
	for i, p := range want {
		if p.ChunkID != found[i].ChunkID || p.Text != p.Source {
			t.Fatalf("passage %d is %+v, want %s read back with its own name", i, p, found[i].ChunkID)
		}
	}
	for _, extractors := range []int{2, defaultExtractors, 16, 100} {
		got := run(extractors)
		if len(got) != len(want) {
			t.Fatalf("%d extractors: %d passages, want %d", extractors, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%d extractors: passage %d is %+v, want %+v", extractors, i, got[i], want[i])
			}
		}
	}
}

func TestReadNeverHoldsMoreDecodedSourcesThanItHasExtractors(t *testing.T) {
	content, found := manySources(30)
	for _, extractors := range []int{1, 3, defaultExtractors} {
		vault := newConcurrentVault(content)
		vault.hold = 2 * time.Millisecond
		u := Search{readers: concurrentReaders{vault}, extractors: extractors}

		if _, err := u.read(t.Context(), domain.Vault{ID: "v"}, found); err != nil {
			t.Fatal(err)
		}
		if vault.peak > extractors {
			t.Errorf("%d extractors held %d sources at once", extractors, vault.peak)
		}
		if vault.peak < extractors {
			t.Errorf("%d extractors never ran together: the most at once was %d", extractors, vault.peak)
		}
	}
}

func TestReadUsesTheDefaultExtractorsWhereNoneAreNamed(t *testing.T) {
	content, found := manySources(30)
	vault := newConcurrentVault(content)
	vault.hold = 2 * time.Millisecond
	u := Search{readers: concurrentReaders{vault}}

	if _, err := u.read(t.Context(), domain.Vault{ID: "v"}, found); err != nil {
		t.Fatal(err)
	}
	if vault.peak != defaultExtractors {
		t.Errorf("the most sources read at once was %d, want %d", vault.peak, defaultExtractors)
	}
}

func TestReadReadsASourceNamedFarApartOnce(t *testing.T) {
	content, found := manySources(12)
	vault := newConcurrentVault(content)
	vault.hold = time.Millisecond
	u := Search{readers: concurrentReaders{vault}}

	if _, err := u.read(t.Context(), domain.Vault{ID: "v"}, found); err != nil {
		t.Fatal(err)
	}
	for name := range content {
		if vault.reads[name] != 1 {
			t.Errorf("%s was read %d times, want 1", name, vault.reads[name])
		}
	}
}

func TestReadStopsAtTheFirstErrorAndLeavesNoGoroutineRunning(t *testing.T) {
	content, found := manySources(30)
	vault := newConcurrentVault(content)
	vault.failAt = "book-00.md"
	vault.blockOthers = true
	u := Search{readers: concurrentReaders{vault}}

	before := runtime.NumGoroutine()
	out, err := u.read(t.Context(), domain.Vault{ID: "v"}, found)
	if err == nil || out != nil {
		t.Fatalf("got %v and %v, want the error of the source that failed", out, err)
	}
	if err.Error() != "read book-00.md: the disk failed" {
		t.Errorf("got %q, want the failure of book-00.md", err)
	}
	if started := len(vault.reads); started > defaultExtractors {
		t.Errorf("%d sources were started after a failure, want at most %d", started, defaultExtractors)
	}
	waitForGoroutines(t, before)
}

func TestReadStopsWhenTheContextEndsAndLeavesNoGoroutineRunning(t *testing.T) {
	content, found := manySources(30)
	vault := newConcurrentVault(content)
	vault.hold = time.Minute
	u := Search{readers: concurrentReaders{vault}}

	ctx, cancel := context.WithCancel(t.Context())
	before := runtime.NumGoroutine()
	isDone := make(chan error, 1)
	go func() {
		_, err := u.read(ctx, domain.Vault{ID: "v"}, found)
		isDone <- err
	}()
	for {
		vault.mu.Lock()
		running := vault.active
		vault.mu.Unlock()
		if running == defaultExtractors {
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel()

	select {
	case err := <-isDone:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("got %v, want the context's own error", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the read did not stop when its context ended")
	}
	if started := len(vault.reads); started > defaultExtractors {
		t.Errorf("%d sources were started, want only the %d running when the context ended", started, defaultExtractors)
	}
	waitForGoroutines(t, before)
}

// waitForGoroutines waits for the goroutines to be no more than `before`.
func waitForGoroutines(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			t.Fatalf("%d goroutines are running, there were %d before the read", runtime.NumGoroutine(), before)
		}
		time.Sleep(time.Millisecond)
	}
}

// concurrentReaders opens the one concurrentVault given.
type concurrentReaders struct{ reader *concurrentVault }

func (c concurrentReaders) Open(domain.Vault) (port.VaultReader, error) { return c.reader, nil }

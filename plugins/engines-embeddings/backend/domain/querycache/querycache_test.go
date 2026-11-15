package querycache_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/querycache"
)

func TestCacheEmptyGet(t *testing.T) {
	c := querycache.New(2)
	if _, ok := c.Get("recipe1", "query1"); ok {
		t.Fatal("expected miss on empty cache")
	}
	if c.Count() != 0 {
		t.Fatalf("expected count 0, got %d", c.Count())
	}
}

func TestCacheInsertAndGet(t *testing.T) {
	c := querycache.New(2)
	vec := []float32{1.0, 2.0, 3.0}
	c.Put("recipe1", "query1", vec)

	got, ok := c.Get("recipe1", "query1")
	if !ok {
		t.Fatal("expected hit in cache")
	}
	if len(got) != len(vec) || got[0] != vec[0] {
		t.Fatalf("got vector %v, want %v", got, vec)
	}

	if _, ok := c.Get("recipe2", "query1"); ok {
		t.Fatal("expected miss for different recipe")
	}
	if _, ok := c.Get("recipe1", "query2"); ok {
		t.Fatal("expected miss for different query")
	}
}

func TestCacheEviction(t *testing.T) {
	c := querycache.New(3)
	c.Put("r", "q1", []float32{1})
	c.Put("r", "q2", []float32{2})
	c.Put("r", "q3", []float32{3})

	// Access middle element q2, then head element q2, then tail element q1.
	// Order becomes: q1 (head), q2 (middle), q3 (tail).
	if _, ok := c.Get("r", "q2"); !ok {
		t.Fatal("expected hit for q2")
	}
	if _, ok := c.Get("r", "q2"); !ok {
		t.Fatal("expected hit for q2 at head")
	}
	if _, ok := c.Get("r", "q1"); !ok {
		t.Fatal("expected hit for q1")
	}

	// Insert q4 (evicts q3), q5 (evicts q2), q6 (evicts q1).
	c.Put("r", "q4", []float32{4})
	if _, ok := c.Get("r", "q3"); ok {
		t.Fatal("expected q3 to be evicted")
	}
	c.Put("r", "q5", []float32{5})
	if _, ok := c.Get("r", "q2"); ok {
		t.Fatal("expected q2 to be evicted")
	}
	c.Put("r", "q6", []float32{6})
	if _, ok := c.Get("r", "q1"); ok {
		t.Fatal("expected q1 to be evicted")
	}
	for _, q := range []string{"q4", "q5", "q6"} {
		if _, ok := c.Get("r", q); !ok {
			t.Fatalf("expected %s to be present", q)
		}
	}
	if c.Count() != 3 {
		t.Fatalf("expected count 3, got %d", c.Count())
	}
}

func TestCacheUpdateExisting(t *testing.T) {
	c := querycache.New(2)
	c.Put("r", "q1", []float32{1})
	c.Put("r", "q2", []float32{2})
	c.Put("r", "q1", []float32{10})
	c.Put("r", "q3", []float32{3})

	got, ok := c.Get("r", "q1")
	if !ok || got[0] != 10 {
		t.Fatalf("expected updated value 10, got %v", got)
	}
	if _, ok := c.Get("r", "q2"); ok {
		t.Fatal("expected q2 to be evicted after q1 update moved q1 to front")
	}
	if c.Count() != 2 {
		t.Fatalf("expected count 2, got %d", c.Count())
	}
}

func TestCacheCapacityBoundsAndEmptyVector(t *testing.T) {
	single := querycache.New(1)
	single.Put("r", "empty", nil)
	single.Put("r", "empty2", []float32{})
	if single.Count() != 0 {
		t.Fatalf("expected empty vectors to be ignored, got count %d", single.Count())
	}
	single.Put("r", "q1", []float32{1})
	if _, ok := single.Get("r", "q1"); !ok {
		t.Fatal("expected q1 hit")
	}
	single.Put("r", "q2", []float32{2})
	if single.Count() != 1 {
		t.Fatalf("expected count 1, got %d", single.Count())
	}
	if _, ok := single.Get("r", "q1"); ok {
		t.Fatal("expected q1 to be evicted in capacity-1 cache")
	}
	single.Put("r", "q3", []float32{3})
	if _, ok := single.Get("r", "q2"); ok {
		t.Fatal("expected q2 to be evicted in capacity-1 cache")
	}

	for _, capArg := range []int{0, -1} {
		def := querycache.New(capArg)
		for i := range querycache.DefaultCapacity {
			def.Put("r", fmt.Sprintf("q%d", i), []float32{float32(i)})
		}
		if def.Count() != 256 {
			t.Fatalf("New(%d) count = %d, want 256", capArg, def.Count())
		}
		def.Put("r", "overflow", []float32{999})
		if def.Count() != 256 {
			t.Fatalf("New(%d) count after overflow = %d, want 256", capArg, def.Count())
		}
		if _, ok := def.Get("r", "q0"); ok {
			t.Fatalf("New(%d) expected q0 to be evicted at 257th insert", capArg)
		}
	}
}

func TestCacheConcurrent(t *testing.T) {
	c := querycache.New(50)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			recipe := fmt.Sprintf("r%d", id%3)
			for j := range 100 {
				query := fmt.Sprintf("q%d", j%10)
				c.Put(recipe, query, []float32{float32(id), float32(j)})
				_, _ = c.Get(recipe, query)
			}
		}(i)
	}
	wg.Wait()
}

func TestCacheNilSafety(t *testing.T) {
	var c *querycache.Cache
	if _, ok := c.Get("r", "q"); ok {
		t.Fatal("expected false on nil cache")
	}
	c.Put("r", "q", []float32{1})
	if c.Count() != 0 {
		t.Fatalf("expected count 0 on nil cache, got %d", c.Count())
	}
}

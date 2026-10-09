package search_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

func TestQueryCacheEmptyGet(t *testing.T) {
	c := search.NewQueryCache(2)
	if _, ok := c.Get("recipe1", "query1"); ok {
		t.Fatal("expected miss on empty cache")
	}
	if c.Count() != 0 {
		t.Fatalf("expected count 0, got %d", c.Count())
	}
}

func TestQueryCachePutAndGet(t *testing.T) {
	c := search.NewQueryCache(2)
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

func TestQueryCacheEviction(t *testing.T) {
	c := search.NewQueryCache(2)
	c.Put("r", "q1", []float32{1})
	c.Put("r", "q2", []float32{2})

	// Access q1 to make q2 the least recently used
	if _, ok := c.Get("r", "q1"); !ok {
		t.Fatal("expected hit for q1")
	}

	// Insert q3, which should evict q2
	c.Put("r", "q3", []float32{3})

	if _, ok := c.Get("r", "q2"); ok {
		t.Fatal("expected q2 to be evicted")
	}
	if _, ok := c.Get("r", "q1"); !ok {
		t.Fatal("expected q1 to still be present")
	}
	if _, ok := c.Get("r", "q3"); !ok {
		t.Fatal("expected q3 to be present")
	}
	if c.Count() != 2 {
		t.Fatalf("expected count 2, got %d", c.Count())
	}
}

func TestQueryCacheUpdateExisting(t *testing.T) {
	c := search.NewQueryCache(2)
	c.Put("r", "q1", []float32{1})
	c.Put("r", "q2", []float32{2})
	c.Put("r", "q1", []float32{10})

	got, ok := c.Get("r", "q1")
	if !ok || got[0] != 10 {
		t.Fatalf("expected updated value 10, got %v", got)
	}
	if c.Count() != 2 {
		t.Fatalf("expected count 2, got %d", c.Count())
	}
}

func TestQueryCacheConcurrent(t *testing.T) {
	c := search.NewQueryCache(50)
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

func TestQueryCacheNilSafety(t *testing.T) {
	var c *search.QueryCache
	if _, ok := c.Get("r", "q"); ok {
		t.Fatal("expected false on nil cache")
	}
	c.Put("r", "q", []float32{1})
	if c.Count() != 0 {
		t.Fatalf("expected count 0 on nil cache, got %d", c.Count())
	}
}

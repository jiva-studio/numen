package preview

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
)

func TestStoreKeepsPreviews(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := NewStore(newFakeClock(now))
	if gotNow := store.Now(); !gotNow.Equal(now) {
		t.Errorf("Now() = %v, want %v", gotNow, now)
	}
	id := store.NextID()
	if id != "prev-1700000000000000000-1" {
		t.Fatalf("NextID() = %q, want prev-1700000000000000000-1", id)
	}
	if id2 := store.NextID(); id2 != "prev-1700000000000000000-2" {
		t.Fatalf("second NextID() = %q, want prev-1700000000000000000-2", id2)
	}

	store.Save(Preview{
		ID:        id,
		Path:      "notes/test.md",
		Diff:      "sample diff",
		Additions: 1,
		Deletions: 1,
	})

	retrieved, ok := store.Get(id)
	if !ok || retrieved.Path != "notes/test.md" {
		t.Errorf("Get(%q) = (%+v, %v), want found", id, retrieved, ok)
	}

	if list := store.List(); len(list) != 1 {
		t.Errorf("len(List()) = %d, want 1", len(list))
	}

	store.Clear()
	if len(store.List()) != 0 {
		t.Error("expected empty store after Clear")
	}
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}

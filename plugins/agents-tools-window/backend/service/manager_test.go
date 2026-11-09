package service

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
)

func TestWindowManagerPreviews(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	mgr := NewManager(clock.NewFake(now))
	if gotNow := mgr.GetNow(); !gotNow.Equal(now) {
		t.Errorf("GetNow() = %v, want %v", gotNow, now)
	}
	id := mgr.NextPreviewID()
	if id != "prev-1700000000000000000-1" {
		t.Fatalf("NextPreviewID() = %q, want prev-1700000000000000000-1", id)
	}
	if id2 := mgr.NextPreviewID(); id2 != "prev-1700000000000000000-2" {
		t.Fatalf("second NextPreviewID() = %q, want prev-1700000000000000000-2", id2)
	}

	prev := NoteEditPreview{
		ID:        id,
		Path:      "notes/test.md",
		Diff:      "sample diff",
		Additions: 1,
		Deletions: 1,
	}
	mgr.SavePreview(prev)

	retrieved, ok := mgr.GetPreview(id)
	if !ok || retrieved.Path != "notes/test.md" {
		t.Errorf("GetPreview(%q) = (%+v, %v), want found", id, retrieved, ok)
	}

	list := mgr.ListPreviews()
	if len(list) != 1 {
		t.Errorf("len(ListPreviews()) = %d, want 1", len(list))
	}

	mgr.ClearPreviews()
	if len(mgr.ListPreviews()) != 0 {
		t.Error("expected empty previews after ClearPreviews")
	}
}

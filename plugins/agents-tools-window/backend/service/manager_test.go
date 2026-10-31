package service

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
)

func TestWindowManagerOpenTabs(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	tabs, front := mgr.GetOpenTabs()
	if len(tabs) != 0 || front != "" {
		t.Fatalf("expected empty tabs initially, got len=%d front=%q", len(tabs), front)
	}

	initial := []TabInfo{
		{ID: "t1", Kind: "note", Path: "notes/a.md", Title: "A"},
		{ID: "t2", Kind: "note", Path: "notes/b.md", Title: "B"},
	}
	mgr.SetOpenTabs(initial, "t2")

	tabs, front = mgr.GetOpenTabs()
	if len(tabs) != 2 {
		t.Fatalf("got len(tabs) = %d, want 2", len(tabs))
	}
	if front != "t2" {
		t.Errorf("front = %q, want t2", front)
	}
	if tabs[0].IsActive || !tabs[1].IsActive {
		t.Errorf("expected tab 2 to be front, got t1=%v, t2=%v", tabs[0].IsActive, tabs[1].IsActive)
	}
}

func TestWindowManagerFocusTabExisting(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	initial := []TabInfo{
		{ID: "t1", Kind: "note", Path: "notes/a.md", Title: "A"},
		{ID: "t2", Kind: "note", Path: "notes/b.md", Title: "B"},
	}
	mgr.SetOpenTabs(initial, "t1")

	// Focus by ID
	ctx := t.Context()
	focused, err := mgr.FocusTab(ctx, WindowFocusTabInput{TabID: "t2"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.ID != "t2" || !focused.IsActive {
		t.Errorf("focused = %+v, want ID t2 and IsActive true", focused)
	}
	_, front := mgr.GetOpenTabs()
	if front != "t2" {
		t.Errorf("front = %q, want t2", front)
	}

	// Focus by Path
	focused, err = mgr.FocusTab(ctx, WindowFocusTabInput{Path: "notes/a.md"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.ID != "t1" || !focused.IsActive {
		t.Errorf("focused = %+v, want ID t1 and IsActive true", focused)
	}

	// Focus by Title
	focused, err = mgr.FocusTab(ctx, WindowFocusTabInput{Title: "B"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.ID != "t2" || !focused.IsActive {
		t.Errorf("focused = %+v, want ID t2 and IsActive true", focused)
	}
}

func TestWindowManagerFocusTabNewPath(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	ctx := t.Context()

	focused, err := mgr.FocusTab(ctx, WindowFocusTabInput{Path: "notes/new.md"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.Path != "notes/new.md" || !focused.IsActive {
		t.Errorf("focused = %+v, want Path notes/new.md and IsActive true", focused)
	}
	if focused.Title != "new" {
		t.Errorf("focused.Title = %q, want 'new'", focused.Title)
	}

	tabs, front := mgr.GetOpenTabs()
	if len(tabs) != 1 || front != focused.ID {
		t.Errorf("expected 1 tab with front %q, got len=%d front=%q", focused.ID, len(tabs), front)
	}
}

func TestWindowManagerFocusTabValidation(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	ctx := t.Context()

	_, err := mgr.FocusTab(ctx, WindowFocusTabInput{})
	if !errors.Is(err, errMissingFocusTarget) {
		t.Errorf("got error %v, want errMissingFocusTarget", err)
	}

	_, err = mgr.FocusTab(ctx, WindowFocusTabInput{TabID: "nonexistent"})
	if !errors.Is(err, errTabNotFound) {
		t.Errorf("got error %v, want errTabNotFound", err)
	}
}

func TestWindowManagerPreviews(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	id := mgr.NextPreviewID()
	if id == "" {
		t.Fatal("expected non-empty preview ID")
	}

	prev := EditPreview{
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

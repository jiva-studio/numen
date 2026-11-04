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
		{ID: "", Kind: "note", Path: "notes/empty.md", Title: "Empty"},
		{ID: "t1", Kind: "note", Path: "notes/a.md", Title: "A"},
		{ID: "t2", Kind: "note", Path: "notes/b.md", Title: "B"},
	}
	mgr.SetOpenTabs(initial, "")
	tabs, _ = mgr.GetOpenTabs()
	if tabs[0].IsActive {
		t.Errorf("expected tab with empty ID to be inactive when activeID is empty")
	}

	mgr.SetOpenTabs(initial[1:], "t2")

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
		{ID: "", Kind: "note", Path: "notes/zero.md", Title: ""},
		{ID: "t1", Kind: "note", Path: "notes/a.md", Title: "A"},
		{ID: "t2", Kind: "note", Path: "notes/b.md", Title: "B"},
	}
	mgr.SetOpenTabs(initial, "t2")

	// Focus tab at index 0 by Path
	ctx := t.Context()
	focused, err := mgr.FocusTab(ctx, WindowFocusTabInput{Path: "notes/zero.md"})
	if err != nil {
		t.Fatalf("FocusTab index 0: %v", err)
	}
	if focused.Path != "notes/zero.md" || !focused.IsActive {
		t.Errorf("focused index 0 = %+v, want Path notes/zero.md and IsActive true", focused)
	}

	// Focus by ID
	focused, err = mgr.FocusTab(ctx, WindowFocusTabInput{TabID: "t1"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.ID != "t1" || !focused.IsActive {
		t.Errorf("focused = %+v, want ID t1 and IsActive true", focused)
	}
	tabs, front := mgr.GetOpenTabs()
	if front != "t1" {
		t.Errorf("front = %q, want t1", front)
	}
	if tabs[0].IsActive || !tabs[1].IsActive || tabs[2].IsActive {
		t.Errorf("active flags = [%v, %v, %v], want [false, true, false]", tabs[0].IsActive, tabs[1].IsActive, tabs[2].IsActive)
	}

	// Focus by Path on non-zero index
	focused, err = mgr.FocusTab(ctx, WindowFocusTabInput{Path: "notes/b.md"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.ID != "t2" || !focused.IsActive {
		t.Errorf("focused = %+v, want ID t2 and IsActive true", focused)
	}

	// Focus by Title
	focused, err = mgr.FocusTab(ctx, WindowFocusTabInput{Title: "A"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.ID != "t1" || !focused.IsActive {
		t.Errorf("focused = %+v, want ID t1 and IsActive true", focused)
	}

	if got := findMatchingTabIndex(initial, WindowFocusTabInput{TabID: "missing"}); got != -1 {
		t.Errorf("findMatchingTabIndex = %d, want -1", got)
	}
}

func TestWindowManagerFocusTabNewPath(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	ctx := t.Context()

	focused, err := mgr.FocusTab(ctx, WindowFocusTabInput{Path: "notes/new.md"})
	if err != nil {
		t.Fatalf("FocusTab: %v", err)
	}
	if focused.ID != "tab-1" || focused.Path != "notes/new.md" || !focused.IsActive {
		t.Errorf("focused = %+v, want ID tab-1, Path notes/new.md, IsActive true", focused)
	}
	if focused.Title != "new" {
		t.Errorf("focused.Title = %q, want 'new'", focused.Title)
	}

	focused2, err := mgr.FocusTab(ctx, WindowFocusTabInput{
		Path:  "notes/second.md",
		TabID: "custom-tab",
		Title: "Custom Title",
	})
	if err != nil {
		t.Fatalf("FocusTab second: %v", err)
	}
	if focused2.ID != "custom-tab" || focused2.Title != "Custom Title" || !focused2.IsActive {
		t.Errorf("focused2 = %+v, want custom-tab / Custom Title / active", focused2)
	}

	focused3, err := mgr.FocusTab(ctx, WindowFocusTabInput{Path: "notes/third.md"})
	if err != nil {
		t.Fatalf("FocusTab third: %v", err)
	}
	if focused3.ID != "tab-3" {
		t.Errorf("focused3.ID = %q, want tab-3", focused3.ID)
	}

	tabs, front := mgr.GetOpenTabs()
	if len(tabs) != 3 || front != "tab-3" {
		t.Fatalf("expected 3 tabs with front tab-3, got len=%d front=%q", len(tabs), front)
	}
	if tabs[0].IsActive || tabs[1].IsActive || !tabs[2].IsActive {
		t.Errorf("active flags = [%v, %v, %v], want [false, false, true]", tabs[0].IsActive, tabs[1].IsActive, tabs[2].IsActive)
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

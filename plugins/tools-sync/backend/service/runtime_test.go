package service

import (
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
)

func TestRuntimeStart_ReturnsTheSignInRestoreSyncWatchChainLinkAndRotateLoopsByName(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})

	_, loops := r.start(t.Context(), "http://sync.example.test")

	got := slices.Sorted(maps.Keys(loops))
	want := []string{"chain", "link", "restore", "rotate", "sign-in", "sync", "watch"}
	if !slices.Equal(got, want) {
		t.Errorf("loops = %v, want %v", got, want)
	}
}

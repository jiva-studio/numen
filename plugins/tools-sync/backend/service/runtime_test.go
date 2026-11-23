package service

import (
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
)

func TestRuntimeStart_ReturnsTheLoopsByName(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})

	_, loops := r.start(t.Context(), "http://sync.example.test")

	got := slices.Sorted(maps.Keys(loops))
	want := []string{"chain", "link", "restore", "rotate", "sign-in", "sync", "watch"}
	if !slices.Equal(got, want) {
		t.Errorf("loops = %v, want %v", got, want)
	}
}

func TestRuntimeConfigure_RunsNoSessionOverPlainHTTPToAnotherHost(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	t.Cleanup(r.stop)

	r.configure(map[string]string{settingServer: "http://sync.example.test"})

	if _, err := r.getBackend(); !errors.Is(err, errNotConfigured) {
		t.Errorf("getBackend = %v, want errNotConfigured", err)
	}
}

func TestReadServerAddress_KeepsHTTPSAndLoopbackHTTPAndDropsOtherHTTP(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	for address, want := range map[string]string{
		"https://sync.example.test": "https://sync.example.test",
		"http://127.0.0.1:8080":     "http://127.0.0.1:8080",
		"http://sync.example.test":  "",
		"":                          "",
	} {
		if got := r.readServerAddress(address); got != want {
			t.Errorf("readServerAddress(%q) = %q, want %q", address, got, want)
		}
	}
}

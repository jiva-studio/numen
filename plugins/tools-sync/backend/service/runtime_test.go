package service

import (
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/settingswire"
)

func TestRuntimeStart_ReturnsTheLoopsByName(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})

	_, loops := r.start(t.Context(), "http://sync.example.test")

	got := slices.Sorted(maps.Keys(loops))
	want := []string{"chain", "link", "restore", "rotate", "share-invite", "sign-in", "sync", "watch"}
	if !slices.Equal(got, want) {
		t.Errorf("loops = %v, want %v", got, want)
	}
}

func TestRuntimeConfigure_RunsNoSessionOverPlainHTTPToAnotherHost(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	t.Cleanup(r.stop)

	r.configure(map[string]string{settingswire.SettingServer: "http://sync.example.test"})

	if _, err := r.getBackend(); !errors.Is(err, errNotConfigured) {
		t.Errorf("getBackend = %v, want errNotConfigured", err)
	}
}

func TestReadServerAddress_AcceptsHTTPSAndLoopbackHTTPAndRefusesOtherHTTP(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	for address, want := range map[string]string{
		"https://sync.example.test": "https://sync.example.test",
		"http://127.0.0.1:8080":     "http://127.0.0.1:8080",
		"http://sync.example.test":  "",
		"":                          "",
	} {
		if got := r.settings.ReadServerAddress(address); got != want {
			t.Errorf("ReadServerAddress(%q) = %q, want %q", address, got, want)
		}
	}
}

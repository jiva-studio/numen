package service

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/settingswire"
)

func TestRuntimeStart_ReturnsTheLoopsByName(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})

	_, loops := r.start(t.Context(), "http://sync.example.test")

	got := slices.Sorted(maps.Keys(loops))
	want := []string{"chain", "downloads", "lifecycle", "link", "restore", "rotate", "share-invite", "sign-in", "sync", "watch"}
	if !slices.Equal(got, want) {
		t.Errorf("loops = %v, want %v", got, want)
	}
}

func TestRuntimeStart_DownloadsLoop_EndsTheDownloadsOfTheSessionBeforeItReturns(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	backend, loops := r.start(t.Context(), "http://sync.example.test")
	ctx, stop := context.WithCancel(t.Context())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		if err := loops["downloads"](ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("the downloads loop returned %v, want context.Canceled", err)
		}
	}()

	stop()
	<-finished

	_, err := backend.Downloads.FetchFile(t.Context(), api.FetchFileRequest{VaultID: "v-1", Path: "Books/Guide.pdf"})
	if !errors.Is(err, fetchfile.ErrClosed) {
		t.Errorf("a download after the session ended returned %v, want ErrClosed", err)
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

func TestRuntimeConfigure_TurnsOnDemandOnWhenTheSettingIsMissing(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	t.Cleanup(r.stop)

	r.configure(map[string]string{})

	if !r.isOnDemand.IsOn() {
		t.Error("on demand is off, want on by default")
	}
}

func TestRuntimeConfigure_TurnsOnDemandOffWithTheSetting(t *testing.T) {
	t.Parallel()
	r := newRuntime(parts{clock: clock.System{}})
	t.Cleanup(r.stop)

	r.configure(map[string]string{settingOnDemand: "false"})

	if r.isOnDemand.IsOn() {
		t.Error("on demand is on, want off")
	}
}

func TestGetSessionOptions_DifferWhenOnDemandChanges(t *testing.T) {
	t.Parallel()
	on := getSessionOptions(map[string]string{settingOnDemand: "true"})
	off := getSessionOptions(map[string]string{settingOnDemand: "false"})

	if on == off {
		t.Error("options of both settings are equal, want the sync loops to restart on the change")
	}
	if getSessionOptions(map[string]string{}) != on {
		t.Error("a missing setting differs from true, want on by default")
	}
}

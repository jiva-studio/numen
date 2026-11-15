package process_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process/processtest"
)

func TestReleaseTellsTheHost(t *testing.T) {
	host := processtest.NewHost(t)

	if err := host.NewClient(t).Release(t.Context()); err != nil {
		t.Fatal(err)
	}

	if got := host.CountReleases(); got != 1 {
		t.Errorf("releases = %d, want 1", got)
	}
}

func TestAPluginWithoutAHostHasNoClient(t *testing.T) {
	t.Setenv(hostcall.HostPortEnv, "")

	if _, err := process.NewClientFromEnv(); !errors.Is(err, hostcall.ErrNoHost) {
		t.Errorf("error = %v, want ErrNoHost", err)
	}
}

type failingReleaser struct{ calls int }

func (r *failingReleaser) Release(context.Context) error {
	r.calls++
	return errors.New("the host is gone")
}

func TestReleaseAndLogTellsTheHost(t *testing.T) {
	var logBuf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(origLogger) })

	host := processtest.NewHost(t)

	process.ReleaseAndLog(t.Context(), host.NewClient(t))

	if got := host.CountReleases(); got != 1 {
		t.Errorf("releases = %d, want 1", got)
	}
	if logBuf.Len() != 0 {
		t.Errorf("unexpected log on clean release: %s", logBuf.String())
	}
}

func TestReleaseAndLogSurvivesAFailedReleaseAndANilReleaser(t *testing.T) {
	var logBuf bytes.Buffer
	origLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(origLogger) })

	releaser := &failingReleaser{}

	process.ReleaseAndLog(t.Context(), releaser)
	process.ReleaseAndLog(t.Context(), nil)

	if releaser.calls != 1 {
		t.Errorf("calls = %d, want 1", releaser.calls)
	}
	if logBuf.Len() == 0 {
		t.Error("expected warning log on failed release")
	}
}

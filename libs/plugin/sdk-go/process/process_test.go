package process_test

import (
	"errors"
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
	if _, err := process.NewClientFromVars(map[string]string{}); !errors.Is(err, hostcall.ErrNoHost) {
		t.Errorf("error = %v, want ErrNoHost", err)
	}
}

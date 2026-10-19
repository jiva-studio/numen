package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// refusingHost fails every activation with the error given.
type refusingHost struct {
	err error
}

func (h refusingHost) EnsureActive(context.Context, string) (port.Endpoint, error) {
	return port.Endpoint{}, h.err
}

func (refusingHost) Release(string) {}

// A plugin that cannot be started is unavailable, an unavailability the host
// sorted is kept, and the end of the caller's context stays the caller's.
func TestOpenPluginSortsTheFailureToActivate(t *testing.T) {
	sorted := &port.UnavailableError{PluginID: "md.numen.a", Reason: port.PluginDisabled}
	crash := errors.New("exit status 1")

	_, _, err := openPlugin(t.Context(), refusingHost{err: crash}, "md.numen.a", numenv1connect.NewEmbeddingServiceClient)
	var failed *port.UnavailableError
	if !errors.As(err, &failed) || failed.Reason != port.PluginFailed || failed.PluginID != "md.numen.a" || !errors.Is(err, crash) {
		t.Errorf("a crash is %v, want the plugin failed with the crash as its cause", err)
	}

	_, _, err = openPlugin(t.Context(), refusingHost{err: sorted}, "md.numen.a", numenv1connect.NewEmbeddingServiceClient)
	if !errors.Is(err, sorted) {
		t.Errorf("a disabled plugin is %v, want it as the host said", err)
	}

	_, _, err = openPlugin(t.Context(), refusingHost{err: context.Canceled}, "md.numen.a", numenv1connect.NewEmbeddingServiceClient)
	if !errors.Is(err, context.Canceled) || errors.Is(err, port.ErrUnavailable) {
		t.Errorf("a cancelled activation is %v, want the cancellation alone", err)
	}
}

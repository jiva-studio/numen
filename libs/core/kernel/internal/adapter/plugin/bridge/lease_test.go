package bridge

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// refusingHost fails every activation with the error given.
type refusingHost struct {
	err error
}

func (h refusingHost) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	return transport.Endpoint{}, h.err
}

func (refusingHost) Release(string) {}

// A plugin that cannot be started is unavailable, an unavailability the host
// sorted is kept, and the end of the caller's context stays the caller's.
func TestOpenPluginSortsTheFailureToActivate(t *testing.T) {
	sorted := &plugin.UnavailableError{PluginID: "md.numen.a", Reason: plugin.Disabled}
	crash := errors.New("exit status 1")

	_, _, err := Open(t.Context(), refusingHost{err: crash}, "md.numen.a", numenv1connect.NewSearchSourceServiceClient)
	var failed *plugin.UnavailableError
	if !errors.As(err, &failed) || failed.Reason != plugin.Failed || failed.PluginID != "md.numen.a" || !errors.Is(err, crash) {
		t.Errorf("a crash is %v, want the plugin failed with the crash as its cause", err)
	}

	_, _, err = Open(t.Context(), refusingHost{err: sorted}, "md.numen.a", numenv1connect.NewSearchSourceServiceClient)
	if !errors.Is(err, sorted) {
		t.Errorf("a disabled plugin is %v, want it as the host said", err)
	}

	_, _, err = Open(t.Context(), refusingHost{err: context.Canceled}, "md.numen.a", numenv1connect.NewSearchSourceServiceClient)
	if !errors.Is(err, context.Canceled) || errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("a canceled activation is %v, want the cancellation alone", err)
	}
}

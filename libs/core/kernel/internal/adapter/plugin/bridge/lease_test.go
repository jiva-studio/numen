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

func (h refusingHost) EnsureActiveForQuestion(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	return h.EnsureActive(ctx, pluginID)
}

func (refusingHost) ReleaseQuestion(string) {}

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

// leaseHost records the leases of each kind it hands out and takes back.
type leaseHost struct {
	uses, questions, usesReleased, questionsReleased int
}

func (h *leaseHost) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	h.uses++
	return transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: "t-atlas"}, nil
}

func (h *leaseHost) Release(string) { h.usesReleased++ }

func (h *leaseHost) EnsureActiveForQuestion(context.Context, string) (transport.Endpoint, error) {
	h.questions++
	return transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: "t-atlas"}, nil
}

func (h *leaseHost) ReleaseQuestion(string) { h.questionsReleased++ }

func TestALeaseOpenedForAQuestionIsReturnedAsAQuestion(t *testing.T) {
	host := &leaseHost{}
	service := Service[numenv1connect.SearchSourceServiceClient]{
		Host: host, PluginID: "org.example.atlas", NewClient: numenv1connect.NewSearchSourceServiceClient,
	}

	_, lease, err := service.OpenForQuestion(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	lease.Release()

	if want := (leaseHost{questions: 1, questionsReleased: 1}); *host != want {
		t.Errorf("got %+v, want %+v", *host, want)
	}
}

func TestALeaseOpenedForAUseIsReturnedAsAUse(t *testing.T) {
	host := &leaseHost{}

	_, lease, err := Open(t.Context(), host, "org.example.atlas", numenv1connect.NewSearchSourceServiceClient)
	if err != nil {
		t.Fatal(err)
	}
	lease.Release()

	if want := (leaseHost{uses: 1, usesReleased: 1}); *host != want {
		t.Errorf("got %+v, want %+v", *host, want)
	}
}

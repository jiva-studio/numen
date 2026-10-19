package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// eventRecorder serves numen.v1.PluginEventService and keeps every event it is handed.
type eventRecorder struct {
	numenv1connect.UnimplementedPluginEventServiceHandler
	mu      sync.Mutex
	events  []string
	failure error
}

func (r *eventRecorder) DeliverEvent(
	_ context.Context,
	req *connect.Request[v1.DeliverEventRequest],
) (*connect.Response[v1.DeliverEventResponse], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, req.Msg.GetEvent())
	if r.failure != nil {
		return nil, r.failure
	}
	return connect.NewResponse(&v1.DeliverEventResponse{}), nil
}

func (r *eventRecorder) getEvents() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

// servePlugin starts a plugin server answering Ping and, when recorder is set, taking events; it returns the port.
func servePlugin(t *testing.T, recorder *eventRecorder) int {
	t.Helper()
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	if recorder != nil {
		mux.Handle(numenv1connect.NewPluginEventServiceHandler(recorder))
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	addr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	return addr.Port
}

// newEventHost returns a host whose plugins subscribe to the given activation events and run on the given ports.
func newEventHost(t *testing.T, subscriptions map[string][]string, ports map[string]int) *Host {
	t.Helper()
	host := NewHost("desktop.editor")
	for id, events := range subscriptions {
		host.plugins[id] = DiscoveredPlugin{Manifest: port.PluginManifest{ID: id, ActivationEvents: events}}
	}
	host.SetLauncher(func(_ context.Context, p DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := newIdleCommand()
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { killProcess(t, cmd.Process) })
		handshake := fmt.Sprintf("NUMEN_READY port=%d\n", ports[p.Manifest.ID])
		return cmd.Process, io.NopCloser(strings.NewReader(handshake)), nil
	})
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond); err != nil {
			t.Errorf("TeardownAll failed: %v", err)
		}
	})
	return host
}

func TestAnEventReachesOnlyThePluginsSubscribedToIt(t *testing.T) {
	pdf, notes := &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string][]string{"md.test.pdf": {"onFile:*.pdf"}, "md.test.notes": {"onFile:*.md"}},
		map[string]int{"md.test.pdf": servePlugin(t, pdf), "md.test.notes": servePlugin(t, notes)},
	)

	activated, err := host.TriggerEvent(t.Context(), "onFile:report.pdf")
	if err != nil {
		t.Fatalf("TriggerEvent failed: %v", err)
	}

	if !slices.Equal(activated, []string{"md.test.pdf"}) {
		t.Fatalf("activated %v, want [md.test.pdf]", activated)
	}
	if got := pdf.getEvents(); !slices.Equal(got, []string{"onFile:report.pdf"}) {
		t.Fatalf("the subscribed plugin received %v, want [onFile:report.pdf]", got)
	}
	if got := notes.getEvents(); len(got) != 0 {
		t.Fatalf("the plugin not subscribed received %v, want nothing", got)
	}
}

func TestAnEventPublishedOnTheBusReachesTheSubscribedPlugin(t *testing.T) {
	pdf := &eventRecorder{}
	host := newEventHost(t,
		map[string][]string{"md.test.pdf": {"onFile:*.pdf"}},
		map[string]int{"md.test.pdf": servePlugin(t, pdf)},
	)

	host.EmitEvent("onFile:report.pdf")
	if err := host.GetEventBus().Flush(t.Context()); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if got := pdf.getEvents(); !slices.Equal(got, []string{"onFile:report.pdf"}) {
		t.Fatalf("the plugin received %v, want [onFile:report.pdf]", got)
	}
}

func TestAPluginThatTakesNoEventsIsActivatedWithoutAnError(t *testing.T) {
	host := newEventHost(t,
		map[string][]string{"md.test.silent": {"onOpen"}},
		map[string]int{"md.test.silent": servePlugin(t, nil)},
	)

	activated, err := host.TriggerEvent(t.Context(), "onOpen")
	if err != nil {
		t.Fatalf("TriggerEvent() = %v, want no error for a plugin that takes no events", err)
	}
	if !slices.Equal(activated, []string{"md.test.silent"}) {
		t.Fatalf("activated %v, want [md.test.silent]", activated)
	}
}

func TestAPluginThatRefusesAnEventIsReportedAndTheOthersStillReceiveIt(t *testing.T) {
	refusing := &eventRecorder{failure: connect.NewError(connect.CodeInternal, errors.New("the index is closed"))}
	working := &eventRecorder{}
	host := newEventHost(t,
		map[string][]string{"a.refusing": {"onOpen"}, "b.working": {"onOpen"}},
		map[string]int{"a.refusing": servePlugin(t, refusing), "b.working": servePlugin(t, working)},
	)

	_, err := host.TriggerEvent(t.Context(), "onOpen")

	if err == nil || !strings.Contains(err.Error(), "deliver onOpen to plugin a.refusing") || !strings.Contains(err.Error(), "the index is closed") {
		t.Fatalf("TriggerEvent() = %v, want the refusal naming the event and the plugin", err)
	}
	if got := working.getEvents(); !slices.Equal(got, []string{"onOpen"}) {
		t.Fatalf("the working plugin received %v, want [onOpen]", got)
	}
}

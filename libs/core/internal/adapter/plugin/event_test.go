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

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// eventRecorder serves numen.v1.PluginEventService and keeps every event it is handed. It refuses each
type eventRecorder struct {
	numenv1connect.UnimplementedPluginEventServiceHandler
	mu      sync.Mutex
	events  []string
	reason  string
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
	if r.reason != "" {
		return connect.NewResponse(&v1.DeliverEventResponse{
			Error:  v1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_REFUSED,
			Reason: r.reason,
		}), nil
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

// newEventHost returns a host of the given plugins, each with a backend, running on the given ports.
// The host is torn down before its processes are killed.
func newEventHost(t *testing.T, manifests map[string]pluginport.PluginManifest, ports map[string]int) *Host {
	t.Helper()
	host := NewHost("desktop.editor")
	host.SetContentTypeTable(sharedmimeinfo.Open())
	for id, manifest := range manifests {
		manifest.ID = id
		manifest.Backend = pluginport.PluginBackend{Entrypoint: "bin/plugin"}
		host.plugins[id] = manifestfile.DiscoveredPlugin{Manifest: manifest}
	}
	var mu sync.Mutex
	var procs []*os.Process
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, proc := range procs {
			plugintest.KillProcess(t, proc)
		}
	})
	host.SetLauncher(func(_ context.Context, p manifestfile.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := plugintest.NewIdleCommand()
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		mu.Lock()
		procs = append(procs, cmd.Process)
		mu.Unlock()
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

func TestAFileEventReachesOnlyThePluginsReadingItsType(t *testing.T) {
	pdf, notes := &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string]pluginport.PluginManifest{"md.test.pdf": readingType("application/pdf"), "md.test.notes": readingType("text/markdown")},
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
		map[string]pluginport.PluginManifest{"md.test.pdf": readingType("application/pdf")},
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

func TestAPluginThatConsumesNoEventsIsActivatedWithoutAnError(t *testing.T) {
	host := newEventHost(t,
		map[string]pluginport.PluginManifest{"md.test.silent": {ActivationEvents: []string{"onOpen"}}},
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
	refusing := &eventRecorder{reason: "the index is closed"}
	working := &eventRecorder{}
	host := newEventHost(t,
		map[string]pluginport.PluginManifest{"a.refusing": {ActivationEvents: []string{"onOpen"}}, "b.working": {ActivationEvents: []string{"onOpen"}}},
		map[string]int{"a.refusing": servePlugin(t, refusing), "b.working": servePlugin(t, working)},
	)

	_, err := host.TriggerEvent(t.Context(), "onOpen")

	if !errors.Is(err, transport.ErrEventRefused) || !strings.Contains(err.Error(), "deliver onOpen to plugin a.refusing") || !strings.Contains(err.Error(), "the index is closed") {
		t.Fatalf("TriggerEvent() = %v, want the rejection naming the event and the plugin", err)
	}
	if got := working.getEvents(); !slices.Equal(got, []string{"onOpen"}) {
		t.Fatalf("the working plugin received %v, want [onOpen]", got)
	}
}

func TestAPluginWhoseEventCallFailsIsReportedApartFromAError(t *testing.T) {
	failing := &eventRecorder{failure: connect.NewError(connect.CodeInternal, errors.New("the handler panicked"))}
	host := newEventHost(t,
		map[string]pluginport.PluginManifest{"md.test.failing": {ActivationEvents: []string{"onOpen"}}},
		map[string]int{"md.test.failing": servePlugin(t, failing)},
	)

	_, err := host.TriggerEvent(t.Context(), "onOpen")

	if err == nil || errors.Is(err, transport.ErrEventRefused) || connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("TriggerEvent() = %v, want the failed call reported as %v and not as an error", err, connect.CodeInternal)
	}
}

func TestReadEventErrorHandlesEveryCode(t *testing.T) {
	testsupport.CheckHandled(t, func(code v1.PluginEventErrorCode) bool {
		err := transport.ConvertEventRefusal(code, "the index is closed")
		if code == v1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_UNSPECIFIED {
			return err == nil
		}
		return errors.Is(err, transport.ErrEventRefused) && strings.Contains(err.Error(), "the index is closed") && !strings.Contains(err.Error(), "unknown code")
	})
	if err := transport.ConvertEventRefusal(v1.PluginEventErrorCode(99), "why"); !errors.Is(err, transport.ErrEventRefused) || !strings.Contains(err.Error(), "unknown code 99") {
		t.Fatalf("convertEventRefusal(99) = %v, want an error naming the unknown code", err)
	}
}

// readingType is a plugin that makes text of files of the type.
func readingType(contentType domain.ContentType) pluginport.PluginManifest {
	return pluginport.PluginManifest{Texts: &pluginport.PluginTextMaker{ContentTypes: []domain.ContentType{contentType}}}
}

package plugin

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	machineport "github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// eventRecorder serves numen.v1.PluginEventService and records every event it receives.
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

func servePlugin(t *testing.T, recorder *eventRecorder) int {
	t.Helper()
	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
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

func newEventHost(t *testing.T, manifests map[string]catalogmanifest.Manifest, ports map[string]int) *Host {
	t.Helper()
	return newEventHostWithClock(t, testsupport.GetMachine(), manifests, ports)
}

func newEventHostWithClock(t *testing.T, clock machineport.Clock, manifests map[string]catalogmanifest.Manifest, ports map[string]int) *Host {
	t.Helper()
	host := NewHost(clock, "desktop.editor")
	host.ContentTypes.SetContentTypeTable(contenttype.Open())
	for id, manifest := range manifests {
		manifest.ID = id
		manifest.Backend = catalogmanifest.Backend{Entrypoint: "bin/plugin"}
		setPlugin(host, id, catalogmanifest.Discovered{Manifest: manifest})
	}
	var mu sync.Mutex
	var procs []*os.Process
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, proc := range procs {
			wiretest.KillProcess(t, proc)
		}
	})
	host.Processes.SetLauncher(func(ctx context.Context, p catalogmanifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := wiretest.NewIdleCommand(context.WithoutCancel(ctx))
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
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond); err != nil {
			t.Errorf("TeardownAll failed: %v", err)
		}
	})
	return host
}

func TestAFileEventReachesOnlyThePluginsDeclaringItsExtension(t *testing.T) {
	pdf, notes := &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"md.test.pdf":   {ActivationEvents: []string{"onFile:pdf"}},
			"md.test.notes": {ActivationEvents: []string{"onFile:md"}},
		},
		map[string]int{"md.test.pdf": servePlugin(t, pdf), "md.test.notes": servePlugin(t, notes)},
	)

	activated, err := host.Events.TriggerEvent(t.Context(), "onFile:report.pdf")
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

func TestTriggerEventStartsOnlyPluginsDeclaringTheEvent(t *testing.T) {
	startup, vectors, pdf, onDemand := &eventRecorder{}, &eventRecorder{}, &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"md.test.startup":  {ActivationEvents: []string{catalogmanifest.ActivationEventStartup}},
			"md.test.vectors":  {ActivationEvents: []string{catalogmanifest.ActivationEventVectorsReady}},
			"md.test.pdf":      {ActivationEvents: []string{"onFile:pdf"}},
			"md.test.ondemand": {},
		},
		map[string]int{
			"md.test.startup":  servePlugin(t, startup),
			"md.test.vectors":  servePlugin(t, vectors),
			"md.test.pdf":      servePlugin(t, pdf),
			"md.test.ondemand": servePlugin(t, onDemand),
		},
	)

	activated, err := host.Events.TriggerEvent(t.Context(), "onFile:notes.txt")
	if err != nil {
		t.Fatalf("TriggerEvent failed: %v", err)
	}
	if len(activated) != 0 {
		t.Fatalf("activated %v on undeclared event, want none", activated)
	}

	activated, err = host.Events.TriggerEvent(t.Context(), catalogmanifest.ActivationEventStartup)
	if err != nil {
		t.Fatalf("TriggerEvent failed: %v", err)
	}
	if !slices.Equal(activated, []string{"md.test.startup"}) {
		t.Fatalf("activated %v on startup, want [md.test.startup]", activated)
	}

	activated, err = host.Events.TriggerEvent(t.Context(), protocol.GetVectorsReadyEvent("vault-1"))
	if err != nil {
		t.Fatalf("TriggerEvent failed: %v", err)
	}
	if !slices.Equal(activated, []string{"md.test.vectors"}) {
		t.Fatalf("activated %v on vectors ready, want [md.test.vectors]", activated)
	}

	if got := onDemand.getEvents(); len(got) != 0 {
		t.Fatalf("on-demand plugin received %v, want nothing", got)
	}
}

func TestToolsSearchRelevanceRunsOnVectorsReadyAndEvictsWhenIdle(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "..", "plugins", "tools-search-relevance", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := manifestjson.Read(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	clock := testsupport.NewClock()
	recorder := &eventRecorder{}
	host := newEventHostWithClock(t, clock,
		map[string]catalogmanifest.Manifest{parsed.ID: parsed},
		map[string]int{parsed.ID: servePlugin(t, recorder)},
	)

	for _, event := range []string{catalogmanifest.ActivationEventStartup, "onFile:md"} {
		activated, triggerErr := host.Events.TriggerEvent(t.Context(), event)
		if triggerErr != nil {
			t.Fatalf("TriggerEvent(%q) failed: %v", event, triggerErr)
		}
		if len(activated) != 0 {
			t.Fatalf("TriggerEvent(%q) activated %v, want none", event, activated)
		}
	}
	if status, ok := host.Registry.GetPluginStatus(parsed.ID); !ok || status.IsActive {
		t.Fatalf("status after startup and file listing = %+v (ok=%v), want inactive", status, ok)
	}

	readyEvent := protocol.GetVectorsReadyEvent("vault-1")
	activated, err := host.Events.TriggerEvent(t.Context(), readyEvent)
	if err != nil {
		t.Fatalf("TriggerEvent(%q) failed: %v", readyEvent, err)
	}
	if !slices.Equal(activated, []string{parsed.ID}) {
		t.Fatalf("activated %v, want [%s]", activated, parsed.ID)
	}
	if status, ok := host.Registry.GetPluginStatus(parsed.ID); !ok || !status.IsActive {
		t.Fatalf("status after vectors ready = %+v (ok=%v), want active", status, ok)
	}
	if got := recorder.getEvents(); !slices.Equal(got, []string{readyEvent}) {
		t.Fatalf("received events %v, want [%s]", got, readyEvent)
	}

	clock.Advance(supervision.IdleThreshold + time.Second)
	evicted := host.Teardown.EvictIdle(t.Context(), supervision.IdleThreshold, 50*time.Millisecond)
	if !slices.Equal(evicted, []string{parsed.ID}) {
		t.Fatalf("evicted %v, want [%s]", evicted, parsed.ID)
	}
	if status, ok := host.Registry.GetPluginStatus(parsed.ID); !ok || status.IsActive {
		t.Fatalf("status after idle eviction = %+v (ok=%v), want inactive", status, ok)
	}
}

func TestAnEventPublishedOnTheBusReachesTheSubscribedPlugin(t *testing.T) {
	pdf := &eventRecorder{}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{"md.test.pdf": {ActivationEvents: []string{"onFile:pdf"}}},
		map[string]int{"md.test.pdf": servePlugin(t, pdf)},
	)

	host.Events.EmitEvent("onFile:report.pdf")
	if err := host.Events.GetEventBus().Flush(t.Context()); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if got := pdf.getEvents(); !slices.Equal(got, []string{"onFile:report.pdf"}) {
		t.Fatalf("the plugin received %v, want [onFile:report.pdf]", got)
	}
}

func TestAPluginThatConsumesNoEventsIsActivatedWithoutAnError(t *testing.T) {
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{"md.test.silent": {ActivationEvents: []string{catalogmanifest.ActivationEventStartup}}},
		map[string]int{"md.test.silent": servePlugin(t, nil)},
	)

	activated, err := host.Events.TriggerEvent(t.Context(), catalogmanifest.ActivationEventStartup)
	if err != nil {
		t.Fatalf("TriggerEvent() = %v, want no error for a plugin that takes no events", err)
	}
	if !slices.Equal(activated, []string{"md.test.silent"}) {
		t.Fatalf("activated %v, want [md.test.silent]", activated)
	}
}

func TestTriggerEvent_PluginErrorDoesNotBlockOtherPlugins(t *testing.T) {
	refusing := &eventRecorder{reason: "the index is closed"}
	working := &eventRecorder{}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"a.refusing": {ActivationEvents: []string{catalogmanifest.ActivationEventStartup}},
			"b.working":  {ActivationEvents: []string{catalogmanifest.ActivationEventStartup}},
		},
		map[string]int{"a.refusing": servePlugin(t, refusing), "b.working": servePlugin(t, working)},
	)

	_, err := host.Events.TriggerEvent(t.Context(), catalogmanifest.ActivationEventStartup)

	if !errors.Is(err, transport.ErrEventRefused) || !strings.Contains(err.Error(), "deliver onStartup to plugin a.refusing") || !strings.Contains(err.Error(), "the index is closed") {
		t.Fatalf("TriggerEvent() = %v, want the rejection naming the event and the plugin", err)
	}
	if got := working.getEvents(); !slices.Equal(got, []string{catalogmanifest.ActivationEventStartup}) {
		t.Fatalf("the working plugin received %v, want [onStartup]", got)
	}
}

func TestAPluginWhoseEventCallFailsIsReportedApartFromAError(t *testing.T) {
	failing := &eventRecorder{failure: connect.NewError(connect.CodeInternal, errors.New("the handler panicked"))}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{"md.test.failing": {ActivationEvents: []string{catalogmanifest.ActivationEventStartup}}},
		map[string]int{"md.test.failing": servePlugin(t, failing)},
	)

	_, err := host.Events.TriggerEvent(t.Context(), catalogmanifest.ActivationEventStartup)

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

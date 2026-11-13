package delivery

import (
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

const orchardPlugin = "org.example.orchard"

// stoppedHost is a plugin host whose plugin never shows as running.
type stoppedHost struct{ revision string }

func (h *stoppedHost) GetPluginConfigRevision(string) string { return h.revision }

func (*stoppedHost) GetSnapshot(string) (Snapshot, bool, error) {
	return Snapshot{PluginID: "", Endpoint: transport.Endpoint{Host: "", Port: 0, Token: ""}, ConfigJSON: "", Values: nil}, false, nil
}

func (*stoppedHost) ListPluginIDs() []string { return []string{orchardPlugin} }

func (*stoppedHost) GetClient(transport.Endpoint) *http.Client { return http.DefaultClient }

func (*stoppedHost) RecordConfigureAnswer(string, *RefusedError) {}

// A start that a watcher's read caused configures the plugin before it shows as running. That first
// configuration has the revision the watchers already ran for, so it does not run them again.
func TestTheFirstConfigurationOfAStartThatShowsNoProcessRunsNoWatcherForTheSameRevision(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		host := &stoppedHost{revision: "pears-128"}
		d := New(host)
		var runs atomic.Int32
		stop := d.WatchConfig(orchardPlugin, func() { runs.Add(1) })
		defer stop()

		if err := d.Configure(t.Context(), orchardPlugin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if got := runs.Load(); got != 1 {
			t.Fatalf("a change of the stopped plugin ran the watchers %d times, want once", got)
		}

		d.AwaitFirstConfigure(orchardPlugin)
		if err := d.Configure(t.Context(), orchardPlugin); err != nil {
			t.Fatal(err)
		}
		d.ClearFirstConfigure(orchardPlugin)
		synctest.Wait()
		if got := runs.Load(); got != 1 {
			t.Errorf("the first configuration of the start ran the watchers again: %d runs, want 1", got)
		}

		host.revision = "pears-256"
		d.AwaitFirstConfigure(orchardPlugin)
		if err := d.Configure(t.Context(), orchardPlugin); err != nil {
			t.Fatal(err)
		}
		d.ClearFirstConfigure(orchardPlugin)
		synctest.Wait()
		if got := runs.Load(); got != 2 {
			t.Errorf("a first configuration with another revision ran the watchers %d times in all, want 2", got)
		}
	})
}

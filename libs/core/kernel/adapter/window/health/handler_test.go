package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/health"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
)

type noPlugins struct{}

func (noPlugins) GetPlugin(string) (manifest.Discovered, bool) { return manifest.Discovered{}, false }

func newClient(t *testing.T, items *pluginhealth.Items) numenv1connect.PluginHealthServiceClient {
	t.Helper()
	handler := health.NewHandler(items, pluginhealth.NewReport(items, noPlugins{}), pluginhealth.NewClear(items))
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginHealthServiceHandler(handler))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewPluginHealthServiceClient(server.Client(), server.URL)
}

// A frontend reports under the plugin the window names, and a window watching the list sees the item at
// once and its removal after.
func TestAWindowSeesTheItemAFrontendReportsAndItsClear(t *testing.T) {
	items := pluginhealth.NewItems()
	items.Set(plugin.HealthItem{PluginID: "org.example.atlas", Key: plugin.HealthKeyStart, Severity: plugin.HealthError, Text: "Failed to start"})
	client := newClient(t, items)
	stream, err := client.WatchHealth(t.Context(), connect.NewRequest(&v1.WatchHealthRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stream.Close() })
	if !stream.Receive() || len(stream.Msg().GetItems()) != 1 {
		t.Fatalf("first list = %v, %v, want the start item", stream.Msg(), stream.Err())
	}

	_, err = client.ReportHealth(t.Context(), connect.NewRequest(&v1.ReportHealthRequest{Item: &v1.HealthItem{
		PluginId: "org.example.gazette", Key: "names", Severity: v1.HealthSeverity_HEALTH_SEVERITY_WARNING, Text: "The names are old",
	}}))
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Receive() {
		t.Fatal(stream.Err())
	}
	got := stream.Msg().GetItems()
	if len(got) != 2 || got[1].GetPluginId() != "org.example.gazette" || got[1].GetSeverity() != v1.HealthSeverity_HEALTH_SEVERITY_WARNING || got[1].GetText() != "The names are old" {
		t.Fatalf("list = %v, want the start item and the reported one", got)
	}

	if _, err := client.ClearHealth(t.Context(), connect.NewRequest(&v1.ClearHealthRequest{PluginId: "org.example.gazette", Key: "names"})); err != nil {
		t.Fatal(err)
	}
	listed, err := client.ListHealth(t.Context(), connect.NewRequest(&v1.ListHealthRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := listed.Msg.GetItems(); len(got) != 1 || got[0].GetKey() != plugin.HealthKeyStart {
		t.Errorf("list after the clear = %v, want the start item", got)
	}
}

func TestAReportThatNamesNoPluginIsRefused(t *testing.T) {
	client := newClient(t, pluginhealth.NewItems())

	_, err := client.ReportHealth(t.Context(), connect.NewRequest(&v1.ReportHealthRequest{Item: &v1.HealthItem{
		Key: "names", Severity: v1.HealthSeverity_HEALTH_SEVERITY_WARNING, Text: "The names are old",
	}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("report = %v, want %v", err, connect.CodeInvalidArgument)
	}
}

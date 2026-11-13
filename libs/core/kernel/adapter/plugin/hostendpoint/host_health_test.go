package hostendpoint

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
)

// noPlugins is a host with no plugin enabled, whose commands an action could name.
type noPlugins struct{}

func (noPlugins) GetPlugin(string) (manifest.Discovered, bool) { return manifest.Discovered{}, false }

func newHealthClient(t *testing.T, items *pluginhealth.Items, token string) numenv1connect.PluginHealthServiceClient {
	t.Helper()
	callers := newCallers().asCallers()
	callers.Health = HealthCalls{Report: pluginhealth.NewReport(items, noPlugins{}), Clear: pluginhealth.NewClear(items)}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginHealthServiceHandler(&healthEndpoint{endpoint: &Endpoint{callers: callers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewPluginHealthServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

// A plugin reports and clears items of its own: the token names the plugin, whatever the request says.
func TestAPluginReportsAndClearsItsOwnItems(t *testing.T) {
	items := pluginhealth.NewItems()
	client := newHealthClient(t, items, "token-a")

	_, err := client.ReportHealth(t.Context(), connect.NewRequest(&v1.ReportHealthRequest{Item: &v1.HealthItem{
		PluginId: "md.numen.b", Key: "connection", Severity: v1.HealthSeverity_HEALTH_SEVERITY_ERROR, Text: "Choose a connection",
		Action: &v1.HealthAction{Command: "app:settings.open", Values: map[string]string{"plugin": "md.numen.b", "setting": "source"}},
	}}))
	if err != nil {
		t.Fatal(err)
	}
	list := items.List()
	if len(list) != 1 || list[0].PluginID != "md.numen.a" || list[0].Action.Values["plugin"] != "md.numen.a" {
		t.Fatalf("items = %+v, want the item of the calling plugin opening its own settings", list)
	}

	if _, err := client.ClearHealth(t.Context(), connect.NewRequest(&v1.ClearHealthRequest{PluginId: "md.numen.b", Key: "connection"})); err != nil {
		t.Fatal(err)
	}
	if list := items.List(); len(list) != 0 {
		t.Errorf("items after the clear = %+v, want none", list)
	}
}

func TestAnItemTheCoreRefusesIsAnInvalidArgument(t *testing.T) {
	client := newHealthClient(t, pluginhealth.NewItems(), "token-a")

	_, err := client.ReportHealth(t.Context(), connect.NewRequest(&v1.ReportHealthRequest{Item: &v1.HealthItem{
		Key: "connection", Severity: v1.HealthSeverity_HEALTH_SEVERITY_ERROR, Text: "Choose a connection",
		Action: &v1.HealthAction{Command: "md.numen.b:rows.edit"},
	}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("report = %v, want %v", err, connect.CodeInvalidArgument)
	}
}

func TestAnUnknownTokenReportsNothing(t *testing.T) {
	items := pluginhealth.NewItems()
	client := newHealthClient(t, items, "token-of-nobody")

	_, err := client.ReportHealth(t.Context(), connect.NewRequest(&v1.ReportHealthRequest{Item: &v1.HealthItem{
		Key: "connection", Severity: v1.HealthSeverity_HEALTH_SEVERITY_ERROR, Text: "Choose a connection",
	}}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated || len(items.List()) != 0 {
		t.Errorf("report = %v with items %+v, want it unauthenticated", err, items.List())
	}
}

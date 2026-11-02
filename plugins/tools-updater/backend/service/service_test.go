package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/release"
)

var errOffline = errors.New("offline")

// fakeFetcher answers every request with one manifest or one error, and records the endpoints requested.
type fakeFetcher struct {
	manifest  release.Manifest
	err       error
	requested []string
}

func (f *fakeFetcher) Fetch(_ context.Context, endpoint string) (release.Manifest, error) {
	f.requested = append(f.requested, endpoint)
	return f.manifest, f.err
}

func newManifest() release.Manifest {
	return release.Manifest{
		Version: "2026.10.1", Build: 950, Commit: "abc1234", Stamp: "", Channel: "stable",
		Notes: "", Released: "2026-10-15T12:00:00Z",
		Files: []release.File{
			{Name: "numen-windows-setup.exe", Platform: "windows", Arch: "amd64", Kind: "exe", URL: "https://dl.numen.md/setup.exe", Size: 10, SHA256: "hash"},
		},
	}
}

func windowsApp(build int64) api.App {
	return api.App{Version: "2026.9.0", Build: build, Platform: "windows"}
}

func TestACheckBeforeConfigureIsAFailedPrecondition(t *testing.T) {
	fetcher := &fakeFetcher{manifest: newManifest()}
	u := newUpdater(fetcher)

	_, err := u.check(t.Context(), api.CheckRequest{App: windowsApp(900)})

	var failure *server.APIError
	if !errors.As(err, &failure) || failure.Code != server.CodeFailedPrecondition {
		t.Errorf("error = %v, want failed_precondition", err)
	}
	if len(fetcher.requested) != 0 {
		t.Errorf("the endpoint was requested before Configure: %v", fetcher.requested)
	}
}

func TestACheckOfANewerBuildReportsTheUpdate(t *testing.T) {
	fetcher := &fakeFetcher{manifest: newManifest()}
	u := newUpdater(fetcher)
	u.configure(map[string]string{})

	status, err := u.check(t.Context(), api.CheckRequest{App: windowsApp(900)})

	if err != nil {
		t.Fatal(err)
	}
	if status.State != api.StateAvailable || status.Update == nil {
		t.Fatalf("status = %+v", status)
	}
	if status.Update.Version != "2026.10.1" || status.Update.DownloadURL != "https://dl.numen.md/setup.exe" {
		t.Errorf("update = %+v", status.Update)
	}
	if status.Update.File == nil || status.Update.File.SHA256 != "hash" {
		t.Errorf("file = %+v", status.Update.File)
	}
	if len(fetcher.requested) != 1 || fetcher.requested[0] != StableEndpoint {
		t.Errorf("endpoints requested = %v, want the stable feed", fetcher.requested)
	}
}

func TestACheckOfTheCurrentBuildReportsNoUpdate(t *testing.T) {
	u := newUpdater(&fakeFetcher{manifest: newManifest()})
	u.configure(map[string]string{})

	status, err := u.check(t.Context(), api.CheckRequest{App: windowsApp(950)})

	if err != nil || status.State != api.StateCurrent || status.Update != nil {
		t.Errorf("status = %+v, error = %v", status, err)
	}
}

func TestAnOfflineCheckIsNoFailure(t *testing.T) {
	u := newUpdater(&fakeFetcher{err: errOffline})
	u.configure(map[string]string{})

	status, err := u.check(t.Context(), api.CheckRequest{App: windowsApp(900)})

	if err != nil || status.State != api.StateUnreachable {
		t.Errorf("status = %+v, error = %v", status, err)
	}
}

func TestACheckOfAnUnknownBuildIsAnInvalidArgument(t *testing.T) {
	u := newUpdater(&fakeFetcher{manifest: newManifest()})
	u.configure(map[string]string{})
	cases := map[string]api.App{
		"no platform":      {Version: "2026.9.0", Build: 1, Platform: ""},
		"another platform": {Version: "2026.9.0", Build: 1, Platform: "plan9"},
		"no version":       {Version: " ", Build: 1, Platform: "linux"},
		"a negative build": {Version: "2026.9.0", Build: -1, Platform: "linux"},
	}
	for name, app := range cases {
		_, err := u.check(t.Context(), api.CheckRequest{App: app})

		var failure *server.APIError
		if !errors.As(err, &failure) || failure.Code != server.CodeInvalidArgument {
			t.Errorf("%s: error = %v, want invalid_argument", name, err)
		}
	}
}

func TestTheStatusIsTheResultOfTheLastCheck(t *testing.T) {
	u := newUpdater(&fakeFetcher{manifest: newManifest()})
	u.configure(map[string]string{})
	before, err := u.getStatus(t.Context(), api.GetStatusRequest{})
	if err != nil || before.State != api.StateUnchecked {
		t.Fatalf("status before a check = %+v, error = %v", before, err)
	}

	if _, err := u.check(t.Context(), api.CheckRequest{App: windowsApp(900)}); err != nil {
		t.Fatal(err)
	}

	after, err := u.getStatus(t.Context(), api.GetStatusRequest{})
	if err != nil || after.State != api.StateAvailable {
		t.Errorf("status after a check = %+v, error = %v", after, err)
	}
}

func TestAChangeOfTheEndpointDiscardsTheLastResult(t *testing.T) {
	u := newUpdater(&fakeFetcher{manifest: newManifest()})
	u.configure(map[string]string{"channel": "stable"})
	if _, err := u.check(t.Context(), api.CheckRequest{App: windowsApp(900)}); err != nil {
		t.Fatal(err)
	}

	u.configure(map[string]string{"channel": "stable", "checkIntervalHours": "1"})
	same, err := u.getStatus(t.Context(), api.GetStatusRequest{})
	if err != nil || same.State != api.StateAvailable {
		t.Fatalf("status after a setting that is not the endpoint = %+v, error = %v", same, err)
	}

	u.configure(map[string]string{"channel": "beta"})
	changed, err := u.getStatus(t.Context(), api.GetStatusRequest{})
	if err != nil || changed.State != api.StateUnchecked {
		t.Errorf("status after a new endpoint = %+v, error = %v", changed, err)
	}
}

func TestTheEndpointIsTheFeedOfTheChannelOrTheSettingOfTheCustomChannel(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"no settings", map[string]string{}, StableEndpoint},
		{"the stable channel", map[string]string{"channel": "stable"}, StableEndpoint},
		{"the beta channel", map[string]string{"channel": "beta"}, BetaEndpoint},
		{"the custom channel", map[string]string{"endpoint": " http://127.0.0.1:1/latest.json ", "channel": "custom"}, "http://127.0.0.1:1/latest.json"},
		{"the custom channel without an address", map[string]string{"endpoint": "  ", "channel": "custom"}, ""},
		{"an endpoint on the beta channel", map[string]string{"endpoint": "http://127.0.0.1:1/latest.json", "channel": "beta"}, BetaEndpoint},
		{"an endpoint on the stable channel", map[string]string{"endpoint": "http://127.0.0.1:1/latest.json"}, StableEndpoint},
	}
	for _, c := range cases {
		if got := getEndpoint(c.values); got != c.want {
			t.Errorf("%s: endpoint = %q, want %q", c.name, got, c.want)
		}
	}
}

// configurePath is the path of the call with which the host supplies the settings of a plugin.
const configurePath = "/numen.v1.PluginLifecycleService/Configure"

// post calls the path of the plugin with a JSON body and returns the status and the body of the answer.
func post(t *testing.T, web *httptest.Server, path, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer plugin-token")
	res, err := web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, data
}

func TestTheHostConfiguresThePluginThatThenAnswersTheAPI(t *testing.T) {
	fetcher := &fakeFetcher{manifest: newManifest()}
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		return mount(s, fetcher)
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	check := `{"app":{"version":"2026.9.0","build":900,"platform":"linux"}}`
	checkPath := "/" + api.Service + "/" + api.MethodCheck

	if status, _ := post(t, web, checkPath, check); status != http.StatusPreconditionFailed {
		t.Fatalf("Check before Configure = %d, want 412", status)
	}

	configured := `{"values":{"channel":"custom","endpoint":"http://127.0.0.1:9/latest.json"}}`
	if status, body := post(t, web, configurePath, configured); status != http.StatusOK {
		t.Fatalf("Configure = %d %s", status, body)
	}

	status, body := post(t, web, checkPath, check)
	var got api.Status
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || got.State != api.StateAvailable || got.Update == nil || got.Update.Version != "2026.10.1" {
		t.Errorf("Check after Configure = %d %s", status, body)
	}
	if len(fetcher.requested) != 1 || fetcher.requested[0] != "http://127.0.0.1:9/latest.json" {
		t.Errorf("endpoints requested = %v", fetcher.requested)
	}
	status, body = post(t, web, "/"+api.Service+"/"+api.MethodGetStatus, `{}`)
	if status != http.StatusOK || !strings.Contains(string(body), `"state":"available"`) {
		t.Errorf("GetStatus = %d %s", status, body)
	}
}

func TestTheCustomChannelWithoutAnAddressAsksNoFeedAndIsUnreachable(t *testing.T) {
	fetcher := &fakeFetcher{manifest: newManifest()}
	u := newUpdater(fetcher)
	u.configure(map[string]string{"channel": "custom", "endpoint": " "})

	status, err := u.check(t.Context(), api.CheckRequest{App: windowsApp(900)})

	if err != nil || status.State != api.StateUnreachable {
		t.Errorf("status = %+v, error = %v", status, err)
	}
	if len(fetcher.requested) != 0 {
		t.Errorf("endpoints requested = %v, want none", fetcher.requested)
	}
}

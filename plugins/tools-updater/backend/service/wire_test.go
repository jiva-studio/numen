package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/release"
)

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

// configurePath is the path of the call with which the host supplies the settings of a plugin.
const configurePath = "/numen.v1.PluginLifecycleService/Configure"

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

func TestACheckOfAnUnknownBuildIsAnInvalidArgument(t *testing.T) {
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		return mount(s, &fakeFetcher{manifest: newManifest()})
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	if status, body := post(t, web, configurePath, `{"values":{}}`); status != http.StatusOK {
		t.Fatalf("Configure = %d %s", status, body)
	}
	checkPath := "/" + api.Service + "/" + api.MethodCheck

	for name, app := range map[string]string{
		"another platform": `{"version":"2026.9.0","build":1,"platform":"plan9"}`,
		"no version":       `{"version":" ","build":1,"platform":"linux"}`,
		"a negative build": `{"version":"2026.9.0","build":-1,"platform":"linux"}`,
	} {
		if status, body := post(t, web, checkPath, `{"app":`+app+`}`); status != http.StatusBadRequest {
			t.Errorf("%s: Check = %d %s, want 400", name, status, body)
		}
	}
}

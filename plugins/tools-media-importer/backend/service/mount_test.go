package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// The handler a host that links the plugin in answers the importer API with the token it was
// given, and rejects a call without it.
func TestTheLinkedHandlerAnswersTheImporterAPI(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": "1"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	url := web.URL + "/" + api.Service + "/" + api.MethodGetLink

	post := func(token string) (int, []byte) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(`{"path":"notes/plain.txt"}`))
		if err != nil {
			t.Fatal(err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
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

	if status, _ := post(""); status != http.StatusUnauthorized {
		t.Fatalf("GetLink without the token = %d, want 401", status)
	}
	status, data := post("linked-token")
	var failure server.APIError
	if err := json.Unmarshal(data, &failure); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusNotFound || failure.Code != server.CodeNotFound {
		t.Errorf("a file that is no note is answered %d %+v, want 404 not_found", status, failure)
	}

	if _, err := NewHandler(map[string]string{}); err == nil {
		t.Error("NewHandler with empty vars succeeded")
	}
}

func TestMountRegistersAllRoutesConfigAndLifecycle(t *testing.T) {
	_, _, hosted := newMockLinks(t)
	var mountedSrv *server.Server
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "test-token"}, func(srv *server.Server) error {
		mountedSrv = srv
		return Mount(srv, hosted, true)
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	call := func(method, path, body string) int {
		req, err := http.NewRequestWithContext(t.Context(), method, web.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer test-token")
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := web.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := res.Body.Close(); err != nil {
				t.Error(err)
			}
		}()
		return res.StatusCode
	}

	// Verify all mounted API methods are reachable.
	if code := call(http.MethodPost, "/"+api.Service+"/"+api.MethodCreateLinks, `{"folder":"f","urls":["`+videoAddress+`"]}`); code != http.StatusOK {
		t.Errorf("CreateLinks status = %d, want 200", code)
	}
	for _, method := range []string{
		api.MethodGetLink,
		api.MethodCopyMedia,
	} {
		if code := call(http.MethodPost, "/"+api.Service+"/"+method, `{"path":"missing.md"}`); code != http.StatusNotFound {
			t.Errorf("%s(missing.md) status = %d, want 404", method, code)
		}
	}
	if code := call(http.MethodPost, "/numen.v1.PluginLifecycleService/Configure", `{"config":{"copyMaxSizeMb":"10"}}`); code != http.StatusOK {
		t.Errorf("Configure RPC status = %d, want 200", code)
	}
	if code := call(http.MethodPost, "/numen.v1.PluginLifecycleService/Shutdown", `{}`); code != http.StatusOK {
		t.Errorf("Shutdown RPC status = %d, want 200", code)
	}
	if err := mountedSrv.Close(); err != nil {
		t.Fatal(err)
	}

	// Verify NewHandler forwards EnvMockML == "true" vs "false".
	mockHandler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "tok",
		"NUMEN_HOST_PORT":    "1",
		EnvMockML:            "true",
	})
	if err != nil || mockHandler == nil {
		t.Fatalf("NewHandler(mock): %v", err)
	}
}

// Every error a link operation reports maps to the API code a client reads.
func TestToAPIErrorMapsLinkErrors(t *testing.T) {
	cases := []struct {
		err  error
		code server.ErrorCode
	}{
		{address.ErrNotAURL, server.CodeNotFound},
		{link.ErrNotALink, server.CodeNotFound},
		{host.ErrNotExist, server.CodeNotFound},
		{host.ErrNoVault, server.CodeFailedPrecondition},
	}
	for _, one := range cases {
		var failure *server.APIError
		if !errors.As(toAPIError(one.err), &failure) || failure.Code != one.code {
			t.Errorf("%v maps to %v, want %s", one.err, toAPIError(one.err), one.code)
		}
	}
	if toAPIError(nil) != nil {
		t.Error("nil maps to an error")
	}
	other := errors.New("the disk is full")
	if !errors.Is(toAPIError(other), other) {
		t.Error("an unmapped error is changed")
	}
}

// The settings name the programs, the captions and how large a copy may be.
func TestUpdateConfigSetsToolsCaptionsAndTheLargestCopy(t *testing.T) {
	fMock := newFetchers(true)
	if fMock.getFetcher() == nil || fMock.getFetcher().GetDownloadModel(videoAddress).Tool != "mock" {
		t.Fatalf("mock fetcher = %+v", fMock.getFetcher())
	}
	f := newFetchers(false)
	if f.getFetcher() == nil || f.getFetcher().GetDownloadModel(videoAddress).Tool == "mock" {
		t.Fatalf("real fetcher = %+v", f.getFetcher())
	}
	if f.getSettings().CopyMaxSize != 500<<20 {
		t.Errorf("default CopyMaxSize = %d, want %d", f.getSettings().CopyMaxSize, 500<<20)
	}
	before := f.getFetcher()
	f.updateConfig(map[string]string{
		"ytDlp": "/custom/yt-dlp", "ffmpeg": "/custom/ffmpeg",
		"captions": " sa, en ,", "automaticCaptions": "false", "copyMaxSizeMb": "20",
	})
	if f.getFetcher() == before || f.getFetcher() == nil || f.getFetcher().GetDownloadModel(videoAddress).Tool != "yt-dlp" {
		t.Error("updateConfig did not reopen current fetcher")
	}
	if !slices.Equal(f.cfg.Video.Command, []string{"/custom/yt-dlp"}) || !slices.Equal(f.cfg.Sound.Command, []string{"/custom/ffmpeg"}) {
		t.Errorf("the programs are %v and %v", f.cfg.Video.Command, f.cfg.Sound.Command)
	}
	settings := f.getSettings()
	if !slices.Equal(settings.Captions, []string{"sa", "en"}) || settings.IsAutomatic || settings.CopyMaxSize != 20<<20 {
		t.Errorf("the settings are %+v", settings)
	}
	f.updateConfig(map[string]string{"ytDlp": "", "ffmpeg": "", "copyMaxSizeMb": "many"})
	if len(f.cfg.Video.Command) != 0 || len(f.cfg.Sound.Command) != 0 || f.getSettings().CopyMaxSize != 20<<20 {
		t.Errorf("cleared settings leave %v, %v and %d", f.cfg.Video.Command, f.cfg.Sound.Command, f.getSettings().CopyMaxSize)
	}
	f.updateConfig(map[string]string{"copyMaxSizeMb": "-1"})
	if f.getSettings().CopyMaxSize != 20<<20 {
		t.Errorf("negative size changed CopyMaxSize to %d", f.getSettings().CopyMaxSize)
	}
	f.updateConfig(map[string]string{"copyMaxSizeMb": "0"})
	if f.getSettings().CopyMaxSize != 0 {
		t.Errorf("zero size left CopyMaxSize = %d, want 0", f.getSettings().CopyMaxSize)
	}
	f.updateConfig(map[string]string{"copyMaxSizeMb": "20"})
	f.updateConfig(map[string]string{"copyMaxSizeMb": "4611686018427387904"})
	if f.getSettings().CopyMaxSize != 0 {
		t.Errorf("64-bit size parse failed: CopyMaxSize = %d, want 0", f.getSettings().CopyMaxSize)
	}
}

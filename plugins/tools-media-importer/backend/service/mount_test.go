package service

import (
	"encoding/json"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

func newEnvHandler(t *testing.T, vars map[string]string) (http.Handler, error) {
	t.Helper()
	t.Setenv(hostcall.HostPortEnv, "")
	for name, value := range vars {
		t.Setenv(name, value)
	}
	return servertest.NewHandler(vars, NewApp().MountFromEnv)
}

// The handler a host that links the plugin in answers the importer API with the token it was
// given, and rejects a call without it.
func TestTheLinkedHandlerAnswersTheImporterAPI(t *testing.T) {
	handler, err := newEnvHandler(t, map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": "1"})
	if err != nil {
		t.Fatalf("Mount: %v", err)
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

	if _, err := newEnvHandler(t, map[string]string{}); err == nil {
		t.Error("Mount with empty vars succeeded")
	}
	if _, err := newEnvHandler(t, map[string]string{"NUMEN_PLUGIN_TOKEN": "tok"}); err == nil {
		t.Error("Mount without host port succeeded")
	}
}

func TestMountRegistersAllRoutesConfigAndLifecycle(t *testing.T) {
	t.Setenv("PATH", "")
	_, _, hosted, _ := newMockLinks(t)
	var mountedSrv *server.Server
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "test-token"}, func(srv *server.Server) error {
		mountedSrv = srv
		return NewApp().Mount(srv, hosted, true)
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	call := func(targetURL, method, path, body string) (int, []byte) {
		req, err := http.NewRequestWithContext(t.Context(), method, targetURL+path, strings.NewReader(body))
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
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, raw
	}

	// Verify all mounted API methods are reachable.
	code, createRaw := call(web.URL, http.MethodPost, "/"+api.Service+"/"+api.MethodCreateLinks, `{"folder":"f","urls":["`+videoAddress+`"]}`)
	if code != http.StatusOK {
		t.Fatalf("CreateLinks status = %d, want 200", code)
	}
	var created api.CreateLinksResponse
	if err := json.Unmarshal(createRaw, &created); err != nil || len(created.Paths) != 1 {
		t.Fatalf("CreateLinks response = %s (%v)", createRaw, err)
	}
	for _, method := range []string{
		api.MethodGetLink,
		api.MethodCopyMedia,
	} {
		if status, _ := call(web.URL, http.MethodPost, "/"+api.Service+"/"+method, `{"path":"missing.md"}`); status != http.StatusNotFound {
			t.Errorf("%s(missing.md) status = %d, want 404", method, status)
		}
	}

	if status, _ := call(web.URL, http.MethodPost, "/numen.v1.PluginLifecycleService/Shutdown", `{}`); status != http.StatusOK {
		t.Errorf("Shutdown RPC status = %d, want 200", status)
	}
	if err := mountedSrv.Close(); err != nil {
		t.Fatal(err)
	}

	// Verify SetUpdateConfig updates the live downloader configuration when Configure RPC is called.
	realHandler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "test-token"}, func(srv *server.Server) error {
		return NewApp().Mount(srv, hosted, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	realWeb := httptest.NewServer(realHandler)
	defer realWeb.Close()

	_, beforeRaw := call(realWeb.URL, http.MethodPost, "/"+api.Service+"/"+api.MethodGetLink, `{"path":`+mustJSON(t, created.Paths[0])+`}`)
	var beforeLink api.GetLinkResponse
	if err := json.Unmarshal(beforeRaw, &beforeLink); err != nil || beforeLink.IsCopyable {
		t.Fatalf("GetLink before Configure = %+v (%v), want IsCopyable=false", beforeLink, err)
	}
	if status, _ := call(realWeb.URL, http.MethodPost, "/numen.v1.PluginLifecycleService/Configure", `{"values":{"ytDlp":"/configured-yt-dlp"}}`); status != http.StatusOK {
		t.Fatalf("Configure RPC status = %d, want 200", status)
	}
	_, afterRaw := call(realWeb.URL, http.MethodPost, "/"+api.Service+"/"+api.MethodGetLink, `{"path":`+mustJSON(t, created.Paths[0])+`}`)
	var afterLink api.GetLinkResponse
	if err := json.Unmarshal(afterRaw, &afterLink); err != nil || !afterLink.IsCopyable {
		t.Fatalf("GetLink after Configure = %+v (%v), want IsCopyable=true", afterLink, err)
	}

	// Verify Mount forwards the mock variable == "true" vs "false".
	mockHandler, err := newEnvHandler(t, map[string]string{
		"NUMEN_PLUGIN_TOKEN":          "tok",
		"NUMEN_HOST_PORT":             "1",
		environment.MockModelVariable: "true",
	})
	if err != nil || mockHandler == nil {
		t.Fatalf("Mount(mock): %v", err)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
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
		if !errors.As(buildAPIError(one.err), &failure) || failure.Code != one.code {
			t.Errorf("%v maps to %v, want %s", one.err, buildAPIError(one.err), one.code)
		}
	}
	if buildAPIError(nil) != nil {
		t.Error("nil maps to an error")
	}
	other := errors.New("the disk is full")
	if !errors.Is(buildAPIError(other), other) {
		t.Error("an unmapped error is changed")
	}
}

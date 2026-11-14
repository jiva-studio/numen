package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

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

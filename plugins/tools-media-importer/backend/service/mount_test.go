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
	f := newFetchers(false)
	f.updateConfig(map[string]string{
		"yt_dlp": "/custom/yt-dlp", "ffmpeg": "/custom/ffmpeg",
		"captions": " sa, en ,", "automatic_captions": "false", "copy_max_size_mb": "20",
	})
	if !slices.Equal(f.cfg.Video.Command, []string{"/custom/yt-dlp"}) || !slices.Equal(f.cfg.Sound.Command, []string{"/custom/ffmpeg"}) {
		t.Errorf("the programs are %v and %v", f.cfg.Video.Command, f.cfg.Sound.Command)
	}
	settings := f.getSettings()
	if !slices.Equal(settings.Captions, []string{"sa", "en"}) || settings.IsAutomatic || settings.CopyMaxSize != 20<<20 {
		t.Errorf("the settings are %+v", settings)
	}
	f.updateConfig(map[string]string{"yt_dlp": "", "ffmpeg": "", "copy_max_size_mb": "many"})
	if len(f.cfg.Video.Command) != 0 || len(f.cfg.Sound.Command) != 0 || f.getSettings().CopyMaxSize != 20<<20 {
		t.Errorf("cleared settings leave %v, %v and %d", f.cfg.Video.Command, f.cfg.Sound.Command, f.getSettings().CopyMaxSize)
	}
}

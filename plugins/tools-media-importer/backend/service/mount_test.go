package service

import (
	"errors"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
	linksv1 "github.com/jiva-studio/numen/plugins/tools-media-importer/gen/numen/links/v1"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/gen/numen/links/v1/linksv1connect"
)

// The handler a host that links the plugin in serves answers LinkService with the token it was
// given, and refuses a call without it.
func TestTheLinkedHandlerAnswersTheLinkService(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": "1"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := linksv1connect.NewLinkServiceClient(web.Client(), web.URL)

	unauthReq := connect.NewRequest(&linksv1.GetLinkRequest{Path: "links/a.md"})
	if _, err := client.GetLink(t.Context(), unauthReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("GetLink without the token = %v, want unauthenticated", err)
	}
	req := connect.NewRequest(&linksv1.GetLinkRequest{Path: "notes/plain.txt"})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.GetLink(t.Context(), req)
	if err != nil {
		t.Fatalf("GetLink: %v", err)
	}
	if resp.Msg.GetError() != linksv1.LinkErrorCode_LINK_ERROR_CODE_NOT_A_LINK {
		t.Errorf("a file that is no note is answered %v", resp.Msg.GetError())
	}
}

// Every value of the schema's error code is codes for some refusal.
func TestGetErrorCodeCoversEveryLinkErrorCode(t *testing.T) {
	var codes []linksv1.LinkErrorCode
	for _, err := range []error{nil, address.ErrNotAURL, link.ErrNotALink, host.ErrNoVault} {
		code, ok := getErrorCode(err)
		if !ok {
			t.Errorf("%v is written as no code", err)
		}
		codes = append(codes, code)
	}
	for value := range linksv1.LinkErrorCode_name {
		if !slices.Contains(codes, linksv1.LinkErrorCode(value)) {
			t.Errorf("%v is written for no refusal", linksv1.LinkErrorCode(value))
		}
	}
	if _, ok := getErrorCode(errors.New("the disk is full")); ok {
		t.Error("an error the schema names no code for is written as one")
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

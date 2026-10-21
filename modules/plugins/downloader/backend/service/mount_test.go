package service

import (
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves answers the download service with the token it
// was given, and refuses a call without it.
func TestTheLinkedHandlerAnswersTheDownloadService(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := numenv1connect.NewDownloadServiceClient(web.Client(), web.URL)

	refused := connect.NewRequest(&numenv1.CanHandleRequest{Url: "https://example.org/page"})
	if _, err := client.CanHandle(t.Context(), refused); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("CanHandle without the token = %v, want unauthenticated", err)
	}
	req := connect.NewRequest(&numenv1.CanHandleRequest{Url: "https://example.org/page"})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.CanHandle(t.Context(), req)
	if err != nil {
		t.Fatalf("CanHandle: %v", err)
	}
	if !resp.Msg.GetCanHandle() {
		t.Error("a web page is not handled")
	}
}

// Config updates set custom tool paths and restore defaults when cleared.
func TestUpdateConfigSetsTools(t *testing.T) {
	srv := &downloadServer{}
	srv.updateConfig(map[string]string{
		"yt_dlp": "/custom/yt-dlp",
		"ffmpeg": "/custom/ffmpeg",
	})
	if len(srv.cfg.Video.Command) != 1 || srv.cfg.Video.Command[0] != "/custom/yt-dlp" {
		t.Errorf("yt_dlp command = %v, want [/custom/yt-dlp]", srv.cfg.Video.Command)
	}
	if len(srv.cfg.Sound.Command) != 1 || srv.cfg.Sound.Command[0] != "/custom/ffmpeg" {
		t.Errorf("ffmpeg command = %v, want [/custom/ffmpeg]", srv.cfg.Sound.Command)
	}

	srv.updateConfig(map[string]string{
		"yt_dlp": "",
		"ffmpeg": "",
	})
	if len(srv.cfg.Video.Command) != 0 {
		t.Errorf("yt_dlp command = %v, want empty", srv.cfg.Video.Command)
	}
	if len(srv.cfg.Sound.Command) != 0 {
		t.Errorf("ffmpeg command = %v, want empty", srv.cfg.Sound.Command)
	}
}

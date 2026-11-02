package editor

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func TestTheWindowFramesTheOriginsOfTheEnabledPlugins(t *testing.T) {
	services := newBareServices()
	services.PluginServices = &routedPlugins{frames: map[string][]string{"org.example.player": {"https://player.example"}}}

	response := getResponse(t, services.NewHandler(http.NotFoundHandler()), "/").Header().Get("Content-Security-Policy")
	if !strings.Contains(response, "frame-src 'self' https://player.example;") {
		t.Errorf("the policy reads %q", response)
	}
}

func TestPluginPage_ServesOriginHeadersAndFramePolicy(t *testing.T) {
	plugins := &routedPlugins{frames: map[string][]string{"org.example.player": {"https://player.example"}}}
	back, err := Listen(t.Context(), newBareServices().server, nil)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, back)
	back.setPluginServices(plugins)

	address := back.PageAddress("org.example.player", "embed?address=https%3A%2F%2Fvideo.example%2Fclip")
	resp := fetch(t, http.DefaultClient, address)
	body := resp.Body

	if resp.StatusCode != http.StatusOK || string(body) != "page bytes" {
		t.Fatalf("answered %d %q", resp.StatusCode, body)
	}
	if !slices.Equal(plugins.routed, []string{"/plugin/org.example.player/embed?address=https%3A%2F%2Fvideo.example%2Fclip"}) {
		t.Errorf("forwarded %q", plugins.routed)
	}
	if policy := resp.Header.Get("Content-Security-Policy"); !strings.Contains(policy, "frame-src 'self' https://player.example;") {
		t.Errorf("the page is served under %q", policy)
	}
	if referrer := resp.Header.Get("Referrer-Policy"); referrer != "strict-origin-when-cross-origin" {
		t.Errorf("the page sends Referrer-Policy %q", referrer)
	}
}

func TestAPluginPageIsRefusedWithoutTheToken(t *testing.T) {
	plugins := &routedPlugins{}
	back, err := Listen(t.Context(), newBareServices().server, nil)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, back)
	back.setPluginServices(plugins)

	resp := fetch(t, http.DefaultClient, back.address+"/"+pluginPageSegment+"/org.example.player/embed")
	if resp.StatusCode != http.StatusForbidden || len(plugins.routed) != 0 {
		t.Errorf("answered %d, forwarded %q", resp.StatusCode, plugins.routed)
	}
}

func TestGetPageSourceIsThePageOnTheLoopbackSocket(t *testing.T) {
	api := newBareServices()
	back, err := Listen(t.Context(), api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, back)
	api.openPlaying(back)

	answer, err := api.Media.GetPageSource(t.Context(), connect.NewRequest(&v1.GetPageSourceRequest{
		PluginId: "org.example.player", Path: "embed?address=x",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if want := back.address + "/" + back.token + "/plugin/org.example.player/embed?address=x"; answer.Msg.GetSrc() != want {
		t.Errorf("the page is at %q, want %q", answer.Msg.GetSrc(), want)
	}
	if _, err := api.Media.GetPageSource(t.Context(), connect.NewRequest(&v1.GetPageSourceRequest{Path: "embed"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("a page of no plugin was answered %v", err)
	}
}

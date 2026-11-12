package editor

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestTheWindowFramesTheOriginsOfTheEnabledPlugins(t *testing.T) {
	services := newBareServices()
	services.PluginServices = &routedPlugins{frames: map[string][]string{"org.example.player": {"https://player.example"}}}

	response := getResponse(t, services.NewHandler(http.NotFoundHandler()), "/").Header().Get("Content-Security-Policy")
	if !strings.Contains(response, "frame-src 'self' https://player.example;") {
		t.Errorf("the policy reads %q", response)
	}
}

func TestAPluginFileIsRefusedWithoutTheToken(t *testing.T) {
	plugins := &routedPlugins{}
	back, err := Listen(t.Context(), newBareServices().server, nil)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, loopbackCloser{back})
	back.setPluginServices(plugins)

	resp := fetch(t, http.DefaultClient, back.address+"/"+pluginFileSegment+"/v1/org.example.player/embed.html?size=1")
	if resp.StatusCode != http.StatusForbidden || len(plugins.routed) != 0 {
		t.Errorf("answered %d, forwarded %q", resp.StatusCode, plugins.routed)
	}
}

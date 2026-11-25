package editor

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
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

type noStart struct{}

func (noStart) Execute(context.Context, domain.Vault) {}

func TestAPluginPageFramesTheOriginsItsPluginDeclaresOnTheLoopback(t *testing.T) {
	policy := getStorePagePolicy(t, &routedPlugins{frames: map[string][]string{importer: {"https://player.example"}}})
	if !strings.Contains(policy, "frame-src 'self' https://player.example;") {
		t.Errorf("the page is served under %q", policy)
	}
}

func TestAPluginPageThatDeclaresNoOriginsFramesItselfOnTheLoopback(t *testing.T) {
	policy := getStorePagePolicy(t, &routedPlugins{})
	if !strings.Contains(policy, "frame-src 'self';") {
		t.Errorf("the page is served under %q", policy)
	}
}

func getStorePagePolicy(t *testing.T, plugins PluginServices) string {
	t.Helper()
	vault := vaulttest.NewVault(t, map[string]string{talk: sound})
	api := newBareServices()
	api.readFiles(vaultreader.Readers{}, file.NewOpenPluginFile(storedFiles{importer: {pageName: []byte(pageBytes)}}), file.NewGetMediaType(contenttype.Open()))
	api.state.Runtime.SetVault(vault)
	installation := NewInstallation(api.state.PageAccess, Lifecycle{}, nil, nil, WithPlugins(plugins, nil))
	installation.Start(t.Context(), vault, api.server, noStart{}, io.Discard)
	back := api.state.Media.playing.Load()
	testsupport.CloseAfter(t, loopbackCloser{back})

	resp := fetch(t, http.DefaultClient, back.BuildPluginFileAddress(vault, file.Address{PluginID: importer, Path: pageName}, int64(len(pageBytes))))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("answered %d %q", resp.StatusCode, resp.Body)
	}
	return resp.Header.Get("Content-Security-Policy")
}

func TestTheWindowPermitsTheCameraAndTheMicrophoneOfItsOwnPages(t *testing.T) {
	services := newBareServices()

	response := getResponse(t, services.NewHandler(http.NotFoundHandler()), "/").Header().Get("Permissions-Policy")

	if response != "camera=(self), microphone=(self)" {
		t.Errorf("the permissions policy reads %q", response)
	}
}

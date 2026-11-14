package editor

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
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
	for name, setBeforeStart := range map[string]bool{"set before the start": true, "set after the start": false} {
		t.Run(name, func(t *testing.T) {
			policy := getStorePagePolicy(t, setBeforeStart, &routedPlugins{frames: map[string][]string{importer: {"https://player.example"}}})
			if !strings.Contains(policy, "frame-src 'self' https://player.example;") {
				t.Errorf("the page is served under %q", policy)
			}
		})
	}
}

func TestAPluginPageThatDeclaresNoOriginsFramesItselfOnTheLoopback(t *testing.T) {
	policy := getStorePagePolicy(t, true, &routedPlugins{})
	if !strings.Contains(policy, "frame-src 'self';") {
		t.Errorf("the page is served under %q", policy)
	}
}

func getStorePagePolicy(t *testing.T, setBeforeStart bool, plugins PluginServices) string {
	t.Helper()
	vault := vaultfixture.NewVault(t, map[string]string{talk: sound})
	api := newBareServices()
	api.readFiles(vaultreader.Readers{}, file.NewOpenPluginFile(storedFiles{importer: {pageName: []byte(pageBytes)}}), file.NewGetMediaType(contenttype.Open()))
	api.state.Runtime.SetVault(vault)
	installation := NewInstallation(api.state, Lifecycle{Start: noStart{}}, nil, nil, nil)
	if setBeforeStart {
		installation.SetPlugins(plugins, nil)
	}
	if err := installation.Start(t.Context(), vault, api.server); err != nil {
		t.Fatal(err)
	}
	back := api.state.Playing.Load()
	testsupport.CloseAfter(t, loopbackCloser{back})
	if !setBeforeStart {
		installation.SetPlugins(plugins, nil)
	}

	resp := fetch(t, http.DefaultClient, back.PluginFileAddress(vault, file.Address{PluginID: importer, Path: pageName}, int64(len(pageBytes))))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("answered %d %q", resp.StatusCode, resp.Body)
	}
	return resp.Header.Get("Content-Security-Policy")
}

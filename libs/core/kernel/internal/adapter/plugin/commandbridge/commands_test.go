package commandbridge

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// commandPlugin answers every command with the reply it is given and keeps the requests it was sent.
type commandPlugin struct {
	numenv1connect.UnimplementedPluginCommandServiceHandler
	reply    *v1.RunCommandResponse
	received []*v1.RunCommandRequest
}

func (p *commandPlugin) RunCommand(
	_ context.Context, req *connect.Request[v1.RunCommandRequest],
) (*connect.Response[v1.RunCommandResponse], error) {
	p.received = append(p.received, req.Msg)
	return connect.NewResponse(p.reply), nil
}

// commandHost holds the plugins it is given, enabled or not, and runs those it has an endpoint for.
type commandHost struct {
	plugins   []catalogmanifest.Discovered
	disabled  map[string]bool
	endpoints map[string]transport.Endpoint
	acquired  int
	released  int
	infoReads int
}

func (h *commandHost) Discover() error { return nil }

func (h *commandHost) IsPluginEnabled(id string) bool { return !h.disabled[id] }

func (h *commandHost) IsPluginSuspended(string) bool { return false }

func (h *commandHost) GetPluginInfos() []plugindomain.Info {
	h.infoReads++
	infos := make([]plugindomain.Info, 0, len(h.plugins))
	for _, one := range h.plugins {
		infos = append(infos, plugindomain.Info{ID: one.Manifest.ID})
	}
	return infos
}

func (h *commandHost) GetPlugin(id string) (catalogmanifest.Discovered, bool) {
	for _, one := range h.plugins {
		if one.Manifest.ID == id && !h.disabled[id] {
			return one, true
		}
	}
	return catalogmanifest.Discovered{}, false
}

func (h *commandHost) ResolveCapability(string) (catalogmanifest.Discovered, error) {
	return catalogmanifest.Discovered{}, &plugindomain.UnavailableError{Reason: plugindomain.NotInstalled}
}

func (h *commandHost) ResolveService(string) (catalogmanifest.Discovered, bool) {
	return catalogmanifest.Discovered{}, false
}

func (h *commandHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

func (h *commandHost) EnsureActive(_ context.Context, id string, _ plugindomain.Holder) (transport.Endpoint, error) {
	endpoint, isRunning := h.endpoints[id]
	if !isRunning {
		return transport.Endpoint{}, errors.New("the plugin is down")
	}
	h.acquired++
	return endpoint, nil
}

func (h *commandHost) Release(string, string, plugindomain.Holder) { h.released++ }

func (h *commandHost) EnsureActiveForQuestion(ctx context.Context, id string, holder plugindomain.Holder) (transport.Endpoint, error) {
	return h.EnsureActive(ctx, id, holder)
}

func (h *commandHost) ReleaseQuestion(id, token string, holder plugindomain.Holder) { h.Release(id, token, holder) }

func serveCommandPlugin(t *testing.T, plugin *commandPlugin) transport.Endpoint {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginCommandServiceHandler(plugin))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	address, rawPort, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	return transport.Endpoint{Host: address, Port: port}
}

func declaring(id string, commands ...string) catalogmanifest.Discovered {
	manifest := catalogmanifest.Manifest{ID: id}
	for _, one := range commands {
		manifest.Contributes.Commands = append(manifest.Contributes.Commands, catalogmanifest.Command{ID: one, Title: one})
	}
	return catalogmanifest.Discovered{Manifest: manifest}
}

func TestRunCommandSendsFileAndArgumentsAndReturnsPluginResult(t *testing.T) {
	plugin := &commandPlugin{reply: &v1.RunCommandResponse{Message: "made it", OpenedPath: "Made.md"}}
	host := &commandHost{
		plugins:   []catalogmanifest.Discovered{declaring("md.numen.a", "make")},
		endpoints: map[string]transport.Endpoint{"md.numen.a": serveCommandPlugin(t, plugin)},
	}

	result, err := Open(host).RunCommand(t.Context(), plugindomain.CommandCall{
		PluginID: "md.numen.a", Command: "make", Path: "a.pdf", Arguments: map[string]string{"title": "T"}, ItemID: "audio",
	})
	if err != nil || result.Message != "made it" || result.OpenedPath != "Made.md" {
		t.Fatalf("the run answered %+v, %v", result, err)
	}
	request := plugin.received[0]
	if request.GetCommand() != "make" || request.GetPath() != "a.pdf" || request.GetArguments()["title"] != "T" ||
		request.GetItemId() != "audio" {
		t.Errorf("the plugin received %v", request)
	}
	if host.acquired != 1 || host.released != 1 {
		t.Errorf("acquired %d references, gave back %d", host.acquired, host.released)
	}
}

func TestRunCommand_FailureCarriesPluginMessage(t *testing.T) {
	plugin := &commandPlugin{reply: &v1.RunCommandResponse{
		Error: v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED, Message: "the disk is full",
	}}
	host := &commandHost{
		plugins:   []catalogmanifest.Discovered{declaring("md.numen.a", "make")},
		endpoints: map[string]transport.Endpoint{"md.numen.a": serveCommandPlugin(t, plugin)},
	}

	_, err := Open(host).RunCommand(t.Context(), plugindomain.CommandCall{PluginID: "md.numen.a", Command: "make"})

	var failed *plugindomain.CommandFailedError
	if !errors.As(err, &failed) || failed.Message != "the disk is full" {
		t.Errorf("a failed command is %v, want the plugin's words as the failure", err)
	}
}

func TestRunCommand_PluginCannotStartIsUnavailable(t *testing.T) {
	host := &commandHost{plugins: []catalogmanifest.Discovered{declaring("md.numen.a", "make")}}

	_, err := Open(host).RunCommand(t.Context(), plugindomain.CommandCall{PluginID: "md.numen.a", Command: "make"})

	var unavailable *plugindomain.UnavailableError
	if !errors.As(err, &unavailable) || !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("a plugin that cannot start is %v, want it unavailable", err)
	}
}

func TestRunCommand_PluginWithoutCommandServiceIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.NewServeMux())
	t.Cleanup(server.Close)
	address, rawPort, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	host := &commandHost{
		plugins:   []catalogmanifest.Discovered{declaring("md.numen.a", "make")},
		endpoints: map[string]transport.Endpoint{"md.numen.a": {Host: address, Port: port}},
	}

	_, err = Open(host).RunCommand(t.Context(), plugindomain.CommandCall{PluginID: "md.numen.a", Command: "make"})

	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("a plugin that serves no commands is %v, want it unavailable", err)
	}
}

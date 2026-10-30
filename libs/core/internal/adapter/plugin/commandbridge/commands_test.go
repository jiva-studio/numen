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

	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
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
	plugins   []pluginport.DiscoveredPlugin
	disabled  map[string]bool
	endpoints map[string]pluginport.Endpoint
	acquired  int
	released  int
}

func (h *commandHost) Discover() error { return nil }

func (h *commandHost) GetPluginInfos() []pluginport.PluginInfo {
	infos := make([]pluginport.PluginInfo, 0, len(h.plugins))
	for _, one := range h.plugins {
		infos = append(infos, pluginport.PluginInfo{ID: one.Manifest.ID})
	}
	return infos
}

func (h *commandHost) GetPlugin(id string) (pluginport.DiscoveredPlugin, bool) {
	for _, one := range h.plugins {
		if one.Manifest.ID == id && !h.disabled[id] {
			return one, true
		}
	}
	return pluginport.DiscoveredPlugin{}, false
}

func (h *commandHost) ResolveCapability(string) (pluginport.DiscoveredPlugin, error) {
	return pluginport.DiscoveredPlugin{}, &pluginport.UnavailableError{Reason: pluginport.PluginNotInstalled}
}

func (h *commandHost) ResolveService(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{}, false
}

func (h *commandHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

func (h *commandHost) EnsureActive(_ context.Context, id string) (pluginport.Endpoint, error) {
	endpoint, isRunning := h.endpoints[id]
	if !isRunning {
		return pluginport.Endpoint{}, errors.New("the plugin is down")
	}
	h.acquired++
	return endpoint, nil
}

func (h *commandHost) Release(string) { h.released++ }

func serveCommandPlugin(t *testing.T, plugin *commandPlugin) pluginport.Endpoint {
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
	return pluginport.Endpoint{Host: address, Port: port}
}

func declaring(id string, commands ...string) pluginport.DiscoveredPlugin {
	manifest := pluginport.PluginManifest{ID: id}
	for _, one := range commands {
		manifest.Commands = append(manifest.Commands, pluginport.PluginCommand{ID: one, Title: one})
	}
	return pluginport.DiscoveredPlugin{Manifest: manifest}
}

func TestListCommands_ReadsManifestsWithoutStartingPlugins(t *testing.T) {
	host := &commandHost{
		plugins: []pluginport.DiscoveredPlugin{
			declaring("md.numen.a", "first", "second"),
			declaring("md.numen.b", "third"),
			declaring("md.numen.c", "fourth"),
		},
		disabled: map[string]bool{"md.numen.b": true},
	}

	entries := Open(host).ListCommands()

	var ids []string
	for _, one := range entries {
		ids = append(ids, one.PluginID+"/"+one.Command.ID)
	}
	want := []string{"md.numen.a/first", "md.numen.a/second", "md.numen.c/fourth"}
	if len(ids) != len(want) || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Errorf("got %v, want %v", ids, want)
	}
	if host.acquired != 0 {
		t.Errorf("listing started %d plugins", host.acquired)
	}
}

func TestRunCommandSendsFileAndArgumentsAndReturnsPluginResult(t *testing.T) {
	plugin := &commandPlugin{reply: &v1.RunCommandResponse{Message: "made it", OpenedPath: "Made.md"}}
	host := &commandHost{
		plugins:   []pluginport.DiscoveredPlugin{declaring("md.numen.a", "make")},
		endpoints: map[string]pluginport.Endpoint{"md.numen.a": serveCommandPlugin(t, plugin)},
	}

	result, err := Open(host).RunCommand(t.Context(), pluginport.PluginCommandCall{
		PluginID: "md.numen.a", Command: "make", Path: "a.pdf", Arguments: map[string]string{"title": "T"},
	})
	if err != nil || result.Message != "made it" || result.OpenedPath != "Made.md" {
		t.Fatalf("the run answered %+v, %v", result, err)
	}
	request := plugin.received[0]
	if request.GetCommand() != "make" || request.GetPath() != "a.pdf" || request.GetArguments()["title"] != "T" {
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
		plugins:   []pluginport.DiscoveredPlugin{declaring("md.numen.a", "make")},
		endpoints: map[string]pluginport.Endpoint{"md.numen.a": serveCommandPlugin(t, plugin)},
	}

	_, err := Open(host).RunCommand(t.Context(), pluginport.PluginCommandCall{PluginID: "md.numen.a", Command: "make"})

	var failed *pluginport.CommandFailedError
	if !errors.As(err, &failed) || failed.Message != "the disk is full" {
		t.Errorf("a failed command is %v, want the plugin's words as the failure", err)
	}
}

func TestACommandOfAPluginThatCannotStartIsUnavailable(t *testing.T) {
	host := &commandHost{plugins: []pluginport.DiscoveredPlugin{declaring("md.numen.a", "make")}}

	_, err := Open(host).RunCommand(t.Context(), pluginport.PluginCommandCall{PluginID: "md.numen.a", Command: "make"})

	var unavailable *pluginport.UnavailableError
	if !errors.As(err, &unavailable) || !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("a plugin that cannot start is %v, want it unavailable", err)
	}
}

func TestACommandOfAPluginThatServesNoCommandsIsUnavailable(t *testing.T) {
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
		plugins:   []pluginport.DiscoveredPlugin{declaring("md.numen.a", "make")},
		endpoints: map[string]pluginport.Endpoint{"md.numen.a": {Host: address, Port: port}},
	}

	_, err = Open(host).RunCommand(t.Context(), pluginport.PluginCommandCall{PluginID: "md.numen.a", Command: "make"})

	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("a plugin that serves no commands is %v, want it unavailable", err)
	}
}

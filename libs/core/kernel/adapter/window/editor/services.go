package editor

import (
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"

	agentport "github.com/jiva-studio/numen/libs/core/kernel/usecase/completion/agentcontract"
)

// NewAgentRoute is the route of the service that starts the tasks of the agent in the vault the
// window shows.
func NewAgentRoute(state *State, taking agentport.Agent) Route {
	handler := newAgentHandler(state)
	handler.GetAgent = func() agentport.Agent { return taking }
	path, served := numenv1connect.NewAgentServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

func newAgentHandler(state *State) *wire.AgentHandler {
	return &wire.AgentHandler{
		GetRoot:        func() string { return state.GetCurrentVault().Path },
		GetVault:       func() domain.VaultID { return state.GetCurrentVault().ID },
		GetUnreachable: state.Unreachable.Why,
	}
}

// NewFileServer is the server of the files over the loopback socket.
func NewFileServer(
	state *State, find *registry.Find, open *file.OpenFile, plugin *file.OpenPluginFile, types *file.GetMediaType,
) *FileServer {
	return &FileServer{state: state, find: find, openFile: open, openPluginFile: plugin, mediaTypes: types}
}

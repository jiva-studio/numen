package editor

import (
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"

	agentport "github.com/jiva-studio/numen/libs/core/kernel/usecase/completion/agentcontract"
	machineport "github.com/jiva-studio/numen/libs/core/kernel/usecase/file/machinecontract"
)

// NewVaultRoute is the route of the service that answers what the vault in the window is doing.
func NewVaultRoute(state *State) Route {
	handler := &vaultHandler{state: state}
	path, served := numenv1connect.NewVaultServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewWorkspaceRoute is the route of the service that answers the tabs and the places of the window.
func NewWorkspaceRoute(state *State) Route {
	handler := &workspaceHandler{state: state}
	path, served := numenv1connect.NewWorkspaceServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewWindowRoute is the route of the service that answers the tasks of the window and its flush.
func NewWindowRoute(state *State) Route {
	path, served := numenv1connect.NewWindowServiceHandler(state.Window, state.CountQuestions())
	return NewRoute(path, served, state.Window)
}

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

// NewVaultsHandler is the handler of the vaults service; the use cases it calls are set after.
func NewVaultsHandler(state *State) *VaultsHandler { return &VaultsHandler{state: state} }

// SetScenarios sets the use cases that act on the list of vaults. The handler keeps pointers into
// target, so a use case replaced there is the one it calls.
func (s *VaultsHandler) SetScenarios(target *vaults.Scenarios) {
	s.list = &target.List
	s.find = &target.Find
	s.folderCheck = &target.FolderCheck
	s.add = &target.Add
	s.rename = &target.Rename
	s.remove = &target.Remove
}

// SetFolderDialog sets the native folder dialog a client may ask for.
func (s *VaultsHandler) SetFolderDialog(dialog machineport.PathDialog) {
	choose := picker.NewChooseFolder(dialog)
	s.chooseFolder = &choose
}

// NewFileServer is the server of the files over the loopback socket.
func NewFileServer(
	state *State, find *registry.Find, open *file.OpenFile, plugin *file.OpenPluginFile, types *file.GetMediaType,
) *FileServer {
	return &FileServer{state: state, find: find, openFile: open, openPluginFile: plugin, mediaTypes: types}
}

package editor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
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

// NoteHandler answers what a vault's notes hold and what changes them.
type NoteHandler struct {
	state *State

	getOpeningNote *query.GetOpeningNote
	getTypes       *query.GetTypes
	findTabKinds   func(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error)
	getHeadings    *query.GetHeadings
	resolveLinks   *query.ResolveLinks

	read              *note.Read
	write             *note.Write
	create            *note.Create
	editLinks         *note.EditLinks
	rename            *note.Rename
	showNeighbourhood *query.ShowNeighbourhood
	getLinkAddress    *query.GetLinkAddress

	getConnection    *query.GetConnection
	saveConnection   *note.SaveConnection
	removeSide       *note.RemoveSide
	removeConnection *note.RemoveConnection
	join             *note.Join
	createInRelation *note.CreateInRelation
	getBacklinks     *query.GetBacklinks
}

// NewNoteHandler is the handler of the notes service; the use cases it calls are set after.
func NewNoteHandler(state *State) *NoteHandler { return &NoteHandler{state: state} }

// SetQueries sets the use cases that read a note from the index.
func (h *NoteHandler) SetQueries(
	opening *query.GetOpeningNote, types *query.GetTypes, headings *query.GetHeadings, links *query.ResolveLinks,
) {
	h.getOpeningNote, h.getTypes, h.getHeadings, h.resolveLinks = opening, types, headings, links
}

// SetTabKinds sets how the kind of tab each note opens in is found.
func (h *NoteHandler) SetTabKinds(find func(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error)) {
	h.findTabKinds = find
}

// SetScenarios sets the use cases that read, change and connect notes. The handler keeps pointers
// into notes, so a use case replaced there is the one it calls.
func (h *NoteHandler) SetScenarios(notes *note.Scenarios) {
	h.read = &notes.Read
	h.write = &notes.Write
	h.create = &notes.Create
	h.editLinks = &notes.Linking
	h.rename = &notes.Rename
	h.showNeighbourhood = &notes.Neighborhood
	h.getLinkAddress = notes.GetLinkAddress
	h.getConnection = &notes.Connection
	h.saveConnection = &notes.SaveConnection
	h.removeSide = &notes.RemoveSide
	h.removeConnection = &notes.RemoveConnection
	h.join = &notes.Join
	h.createInRelation = &notes.CreateInRelation
	h.getBacklinks = &notes.Backlinks
}

// NewFileHandler is the handler of the files service; the use cases it calls are set after.
func NewFileHandler(state *State) *FileHandler { return &FileHandler{state: state} }

// SetFolders sets the use cases that list what a folder of the vault holds and make one.
func (h *FileHandler) SetFolders(entries *file.GetFolderEntries, kinds *file.GetFileKinds, create *file.CreateFolder) {
	h.getFolderEntries, h.getFileKinds, h.createFolder = entries, kinds, create
}

// SetTabKindChoice sets the choice of the kind of tab a note opens in, or none.
func (h *FileHandler) SetTabKindChoice(choose *tabkind.Choose) { h.chooseTabKind = choose }

// SetScenarios sets the use cases that move and remove files. The handler keeps pointers into
// notes and target, so a use case replaced there is the one it calls.
func (h *FileHandler) SetScenarios(notes *note.Scenarios, target *vaults.Scenarios) {
	h.remove = &notes.Remove
	h.emptyTrash = &notes.EmptyTrash
	h.move = &target.Move
}

// ListTabKinds is the kind of tab each of the files opens in.
func (h *FileHandler) ListTabKinds(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error) {
	return h.listTabKinds(ctx, v, paths)
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

// NewMediaHandler is the handler of the media service over the use cases that find a file.
func NewMediaHandler(state *State, resolve *file.ResolveAddress, types *file.GetMediaType) *MediaHandler {
	return &MediaHandler{state: state, resolveAddress: resolve, mediaTypes: types}
}

// NewFileServer is the server of the files over the loopback socket.
func NewFileServer(
	state *State, find *registry.Find, open *file.OpenFile, plugin *file.OpenPluginFile, types *file.GetMediaType,
) *FileServer {
	return &FileServer{state: state, find: find, openFile: open, openPluginFile: plugin, mediaTypes: types}
}

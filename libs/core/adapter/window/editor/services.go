package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/usecase/document"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/settings"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Services is the handlers a window answers through, one for each service of the
// schema, and the pieces beside them that are not a service.
//
// A handler holds the use cases it calls and the State it reads the vault
// from. The composition fills each in once the use cases are built.
type Services struct {
	state *State

	Vault     *vaultHandler
	Workspace *workspaceHandler
	File      *fileHandler
	Note      *noteHandler
	Search    *search.Handler
	Vaults    *vaultsHandler
	Settings  *settings.Handler
	Agent     *agentHandler
	Command   *command.Handler

	// Themes are the stylesheets the window may be dressed in. They belong to
	// the installation, so they arrive here from whatever put the window
	// together. A build put together without a catalogue serves no ThemeService.
	Themes numenv1connect.ThemeServiceHandler

	// PluginServices answers the calls for the services plugins serve to the windows.
	PluginServices PluginServices

	// Plugin answers questions about the plugins installed in the application.
	Plugin numenv1connect.PluginServiceHandler

	// getPluginDir returns the root directory of an installed plugin by ID, or false if not found.
	getPluginDir func(id string) (string, bool)

	// Playing is the socket a recording is played from. A build without one
	// answers with no address, and the window says the recording cannot be
	// played here.
	Playing *Loopback

	files  *artifactFiles
	server *fileServer
}

// createServices is every handler of a window over this state, each holding
// nothing yet that needs a vault.
func createServices(state *State) *Services {
	files := &artifactFiles{state: state}
	return &Services{
		state:     state,
		Vault:     &vaultHandler{state: state},
		Workspace: &workspaceHandler{state: state},
		File:      &fileHandler{state: state},
		Note:      &noteHandler{state: state},
		Search:    search.NewHandler(state),
		Vaults:    &vaultsHandler{state: state},
		Settings:  settings.NewHandler(),
		Agent:     &agentHandler{state: state},
		Command:   command.NewHandler(plugincommand.NewCommands(nil)),
		Plugin:    numenv1connect.UnimplementedPluginServiceHandler{},
		files:     files,
		server:    &fileServer{state: state, files: files},
	}
}

// openPlaying names the socket a recording is played from.
func (s *Services) openPlaying(playing *Loopback) {
	s.Playing = playing
}

// readFiles binds the use cases that read the files of the vaults the readers open.
func (s *Services) readFiles(readers vaultport.VaultReaders) {
	s.files.getFingerprint = file.NewGetFingerprint(readers)
	s.server.openFile = document.NewOpenFile(readers)
}

// noteHandler answers what a vault's notes hold and what changes them.
type noteHandler struct {
	state *State

	getOpeningNote *note.GetOpeningNote
	getTypes       *note.GetTypes
	getHeadings    *note.GetHeadings
	resolveLinks   *note.ResolveLinks

	read              *note.Read
	write             *note.Write
	create            *note.Create
	editLinks         *note.EditLinks
	replaceLinks      *note.ReplaceLinks
	rename            *note.Rename
	showNeighbourhood *note.ShowNeighbourhood
	getLinkAddress    *note.GetLinkAddress
}

// artifactFiles finds the file a question about an artifact is about, in the
type artifactFiles struct {
	state          *State
	getFingerprint *file.GetFingerprint
	getNoteAddress source.GetNoteAddress
}

func (f *artifactFiles) getArtifactAddress(ctx context.Context, v domain.Vault, ref domain.Fingerprint) domain.URL {
	if ref.Kind != domain.KindURL {
		return domain.URL("")
	}
	return f.getNoteAddress.Execute(ctx, v, ref.Path)
}

// SetFolderDialog puts the machine's folder dialog in front of the person when a
// client asks for one.
func (s *Services) SetFolderDialog(dialog machineport.FolderDialog) {
	choose := vaults.NewChooseFolder(dialog)
	s.Vaults.chooseFolder = &choose
}

// GetPickFolder is the use case that lets an agent put the same dialog in front
// of the person, or nil where the window has none.
func (s *Services) GetPickFolder() *vaults.PickFolder {
	if s.Vaults.chooseFolder == nil {
		return nil
	}
	pick := s.Vaults.chooseFolder.GetPickFolder()
	return &pick
}

// agentHandler answers for the agent the panel's tasks go to.
type agentHandler struct {
	numenv1connect.UnimplementedAgentServiceHandler
	state    *State
	getAgent func() agentport.Agent
}

// errNoAgent is an installation with none. Whether one can be reached is asked
// of GetAgentState.
var errNoAgent = errors.New("no agent is set up for this vault")

// GetAgentState is whether an agent can be reached at all. Which agent answers
// is the installation's, so every window asks it here.
func (h *agentHandler) GetAgentState(
	_ context.Context, _ *connect.Request[v1.GetAgentStateRequest],
) (*connect.Response[v1.GetAgentStateResponse], error) {
	return connect.NewResponse(&v1.GetAgentStateResponse{Unreachable: h.state.Unreachable.Why()}), nil
}

// AskAgent runs the task on the agent provider.
func (h *agentHandler) AskAgent(
	ctx context.Context,
	r *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	var taking agentport.Agent
	if h.getAgent != nil {
		taking = h.getAgent()
	}
	if taking == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errNoAgent)
	}

	root := r.Msg.GetRoot()
	if root == "" && h.state != nil {
		root = h.state.GetCurrentVault().Path
	}

	task := agentport.Task{
		Question:     r.Msg.GetPrompt(),
		Focus:        r.Msg.GetFocus(),
		Conversation: r.Msg.GetConversation(),
		Agent:        r.Msg.GetAgent(),
		Model:        r.Msg.GetModel(),
		Reasoning:    r.Msg.GetReasoning(),
		Root:         root,
	}
	return wire.StreamAgentTask(ctx, taking, task, stream)
}

// FinishConversation ends conversation with the agent provider.
func (h *agentHandler) FinishConversation(
	ctx context.Context,
	r *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	if h.getAgent != nil {
		if taking := h.getAgent(); taking != nil && r.Msg.GetConversation() != "" {
			_ = taking.EndConversation(ctx, r.Msg.GetConversation())
		}
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

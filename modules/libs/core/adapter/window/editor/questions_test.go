package editor_test

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
)

// questions is everything a window asks about the vault it is showing: what the
// vault is, where the person stands in it, its files, its notes and its text.
// One window asks every service that answers them, so a test holds one of these
// and calls a question by its own name.
type questions struct {
	numenv1connect.VaultServiceClient
	numenv1connect.WorkspaceServiceClient
	numenv1connect.FileServiceClient
	numenv1connect.NoteServiceClient
	numenv1connect.SearchServiceClient
}

// newQuestionsClient is that client against a server those services are mounted on.
func newQuestionsClient(client connect.HTTPClient, at string) questions {
	return questions{
		VaultServiceClient:     numenv1connect.NewVaultServiceClient(client, at),
		WorkspaceServiceClient: numenv1connect.NewWorkspaceServiceClient(client, at),
		FileServiceClient:      numenv1connect.NewFileServiceClient(client, at),
		NoteServiceClient:      numenv1connect.NewNoteServiceClient(client, at),
		SearchServiceClient:    numenv1connect.NewSearchServiceClient(client, at),
	}
}

// mountQuestionsServices puts them on a mux, which is what a window serves them from.
func mountQuestionsServices(mux *http.ServeMux, services *editor.Services) {
	mux.Handle(numenv1connect.NewVaultServiceHandler(services.Vault))
	mux.Handle(numenv1connect.NewWorkspaceServiceHandler(services.Workspace))
	mux.Handle(numenv1connect.NewFileServiceHandler(services.File))
	mux.Handle(numenv1connect.NewNoteServiceHandler(services.Note))
	mux.Handle(numenv1connect.NewSearchServiceHandler(services.Search))
}

package webui_test

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
)

// questions is everything a window asks about the vault it is showing: what the
// vault is, its files, its notes and its text. Four services answer them and one
// window asks all four, so a test holds one of these and calls a question by its
// own name.
type questions struct {
	numenv1connect.VaultServiceClient
	numenv1connect.FileServiceClient
	numenv1connect.NoteServiceClient
	numenv1connect.SearchServiceClient
}

// asks is that client against a server the four services are mounted on.
func asks(client connect.HTTPClient, at string) questions {
	return questions{
		VaultServiceClient:  numenv1connect.NewVaultServiceClient(client, at),
		FileServiceClient:   numenv1connect.NewFileServiceClient(client, at),
		NoteServiceClient:   numenv1connect.NewNoteServiceClient(client, at),
		SearchServiceClient: numenv1connect.NewSearchServiceClient(client, at),
	}
}

// answers puts the four on a mux, which is what a window serves them from.
func answers(mux *http.ServeMux, api *webui.API) {
	mux.Handle(numenv1connect.NewVaultServiceHandler(api))
	mux.Handle(numenv1connect.NewFileServiceHandler(api))
	mux.Handle(numenv1connect.NewNoteServiceHandler(api))
	mux.Handle(numenv1connect.NewSearchServiceHandler(api))
}

// Package service is the backend of the templates plugin: it makes notes from the templates of a folder
// of the vault and serves that to the plugin's frontend.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/usecase/newnote"
)

const defaultFolder = "Templates"

func normalizeFolder(raw string) string {
	var kept []string
	for segment := range strings.SplitSeq(strings.ReplaceAll(strings.TrimSpace(raw), `\`, "/"), "/") {
		switch segment {
		case "", ".":
		case "..":
			return defaultFolder
		default:
			kept = append(kept, segment)
		}
	}
	if len(kept) == 0 {
		return defaultFolder
	}
	return strings.Join(kept, "/")
}

func convertError(err error, template string) error {
	switch {
	case errors.Is(err, newnote.ErrNotATemplate):
		return server.NewAPIError(server.CodeInvalidArgument, err.Error())
	case errors.Is(err, host.ErrNotExist):
		return server.NewAPIError(server.CodeNotFound, "template not found: "+template)
	case errors.Is(err, newnote.ErrInvalidNote), errors.Is(err, newnote.ErrNoFreeName), errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return fmt.Errorf("make a note from a template: %w", err)
}

func makeCreateNoteHandler(useCase newnote.CreateFromTemplate) func(context.Context, api.CreateNoteRequest) (api.CreateNoteResponse, error) {
	return func(ctx context.Context, req api.CreateNoteRequest) (api.CreateNoteResponse, error) {
		made, err := useCase.Execute(ctx, newnote.Request{Template: req.Template, Folder: req.Folder})
		if err != nil {
			return api.CreateNoteResponse{}, convertError(err, req.Template)
		}
		return api.CreateNoteResponse{Path: made.Path}, nil
	}
}

func mount(s *server.Server, vault newnote.Vault, source newnote.Clock, ids newnote.IDs) error {
	folder := newnote.NewFolder(defaultFolder)
	s.SetUpdateConfig(func(values map[string]string) {
		if raw, isSet := values["folder"]; isSet {
			folder.Set(normalizeFolder(raw))
		}
	})
	useCase := newnote.NewCreateFromTemplate(vault, source, ids, folder.Get)
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodCreateNote, makeCreateNoteHandler(useCase))
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="api.Service and api.MethodCreateNote are valid compile-time constants."
	return a.Err()
}

// Mount registers the service on s. The vault is the one the host the process was launched for shows.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	return mount(s, host.NewClient(endpoint), clock.System{}, newnote.NewIDs())
}

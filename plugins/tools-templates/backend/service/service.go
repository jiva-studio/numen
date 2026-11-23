// Package service is the backend of the templates plugin: it makes notes from the templates of a folder
// of the vault and serves that to the plugin's frontend.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/mask"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/internal/adapter/random"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/usecase/newnote"
)

const settingTemplates = "templates"

func convertError(err error, template string) error {
	switch {
	case errors.Is(err, domain.ErrInvalidName):
		return server.NewAPIError(server.CodeInvalidArgument, err.Error())
	case errors.Is(err, newnote.ErrNameTaken):
		return server.NewAPIError(server.CodeAlreadyExists, err.Error())
	case errors.Is(err, host.ErrNotExist):
		return server.NewAPIError(server.CodeNotFound, "template not found: "+template)
	case isRefusedRequest(err):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return fmt.Errorf("make a note from a template: %w", err)
}

func isRefusedRequest(err error) bool {
	for _, refusal := range []error{
		newnote.ErrNotATemplate, newnote.ErrInvalidFolder, engine.ErrNotAnOption, newnote.ErrInvalidNote,
		newnote.ErrTemplateChanged, newnote.ErrDifferentQuestions, host.ErrNoVault,
	} {
		if errors.Is(err, refusal) {
			return true
		}
	}
	return false
}

func makeCreateNoteHandler(useCase newnote.CreateFromTemplate) func(context.Context, api.CreateNoteRequest) (api.CreateNoteResponse, error) {
	return func(ctx context.Context, req api.CreateNoteRequest) (api.CreateNoteResponse, error) {
		made, err := useCase.Execute(ctx, newnote.Request{Template: req.Template, Folder: req.Folder, Name: req.Name, Values: req.Values, Etag: req.Etag})
		if errors.Is(err, newnote.ErrNotIndexed) {
			return api.CreateNoteResponse{Path: made.Path, Warning: err.Error()}, nil
		}
		if err != nil {
			return api.CreateNoteResponse{}, convertError(err, req.Template)
		}
		return api.CreateNoteResponse{Path: made.Path, Warning: ""}, nil
	}
}

func makeListFieldsHandler(useCase newnote.ListFields) func(context.Context, api.ListFieldsRequest) (api.ListFieldsResponse, error) {
	return func(ctx context.Context, req api.ListFieldsRequest) (api.ListFieldsResponse, error) {
		found, err := useCase.Execute(ctx, newnote.FieldsRequest{Template: req.Template, Folder: req.Folder})
		if err != nil {
			return api.ListFieldsResponse{}, convertError(err, req.Template)
		}
		fields := make([]api.Field, len(found.Fields))
		for i, field := range found.Fields {
			fields[i] = api.Field{Kind: api.FieldKind(field.Kind), Label: field.Label, Options: field.Options, Default: field.Default}
		}
		return api.ListFieldsResponse{Fields: fields, Etag: found.Etag, Name: found.Name}, nil
	}
}

func mount(s *server.Server, vault newnote.Vault, source newnote.Clock, ids newnote.IDs) error {
	masks := newnote.NewMasks(mask.Parse(""))
	s.SetUpdateConfig(func(values map[string]string) {
		if raw, isSet := values[settingTemplates]; isSet {
			masks.Set(mask.Parse(raw))
		}
	})
	useCase := newnote.NewCreateFromTemplate(vault, source, ids, masks.Get)
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodListFields, makeListFieldsHandler(newnote.NewListFields(vault, source, masks.Get)))
	server.Handle(a, api.MethodCreateNote, makeCreateNoteHandler(useCase))
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="api.Service and api.MethodCreateNote are valid compile-time constants."
	return a.Err()
}

// Mount registers the service on s. The vault is the one the host the process was launched for shows.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.ReadEnv()
	if err != nil {
		return err
	}
	return mount(s, host.NewClient(endpoint), clock.System{}, random.NewIDs())
}

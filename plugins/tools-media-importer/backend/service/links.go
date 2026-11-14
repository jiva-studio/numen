package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// linkServer serves the importer API over the links of the vault the host shows.
type linkServer struct {
	links *link.Links
}

func mountAPI(s *server.Server, links *link.Links) error {
	service := linkServer{links: links}
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodGetLink, service.getLink)
	server.Handle(a, api.MethodCopyMedia, service.copyMedia)
	server.Handle(a, api.MethodCreateLinks, service.createLinks)
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="compile-time API service and method constants are valid so a.Err() is always nil"
	return a.Err()
}

func toAPIError(err error) error {
	switch {
	// gomutants:disable-next-line BRANCH_CASE reason="nil error falls through to return err which is nil"
	case err == nil:
		return nil
	case errors.Is(err, link.ErrNotALink), errors.Is(err, address.ErrNotAURL), errors.Is(err, host.ErrNotExist):
		return server.NewAPIError(server.CodeNotFound, "not a link")
	case errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, "no vault")
	}
	return err
}

func (s linkServer) readLink(ctx context.Context, path string) (link.Link, error) {
	found, err := s.links.Read(ctx, path)
	if err != nil {
		return link.Link{}, toAPIError(err)
	}
	return found, nil
}

func (s linkServer) getLink(ctx context.Context, req api.GetLinkRequest) (api.GetLinkResponse, error) {
	found, err := s.readLink(ctx, req.Path)
	if err != nil {
		return api.GetLinkResponse{}, err
	}
	return api.GetLinkResponse{
		URL: string(found.Address), IsCopyable: s.links.IsCopyable(found.Address),
	}, nil
}

func (s linkServer) copyMedia(ctx context.Context, req api.CopyMediaRequest) (api.CopyMediaResponse, error) {
	found, err := s.readLink(ctx, req.Path)
	if err != nil {
		return api.CopyMediaResponse{}, err
	}
	media, _ := fetch.ParseMedia(req.Kind)
	s.links.StartCopy(ctx, found, media)
	return api.CopyMediaResponse{}, nil
}

func (s linkServer) createLinks(ctx context.Context, req api.CreateLinksRequest) (api.CreateLinksResponse, error) {
	paths, failed := s.links.CreateLinks(ctx, req.Folder, req.URLs)
	return api.CreateLinksResponse{Paths: paths, Failed: failed}, nil
}

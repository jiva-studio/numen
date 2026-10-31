package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// linkServer serves the importer API over the links of the vault the host shows.
type linkServer struct {
	links *link.Links
}

// mountAPI mounts the importer API methods on s and returns the first failure to mount one.
func mountAPI(s *server.Server, links *link.Links) error {
	service := linkServer{links: links}
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodGetLink, service.getLink)
	server.Handle(a, api.MethodReadArticle, service.readArticle)
	server.Handle(a, api.MethodFetchText, service.fetchText)
	server.Handle(a, api.MethodCopyVideo, service.copyVideo)
	server.Handle(a, api.MethodDeleteCopy, service.deleteCopy)
	server.Handle(a, api.MethodCreateLinks, service.createLinks)
	return a.Err()
}

// toAPIError returns the API failure for a link error: not_found for a path that is no link,
// failed_precondition for a missing vault. Any other error is returned as it is.
func toAPIError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, link.ErrNotALink), errors.Is(err, address.ErrNotAURL), errors.Is(err, host.ErrNotExist):
		return server.NewAPIError(server.CodeNotFound, "not a link")
	case errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, "no vault")
	}
	return err
}

// readLink returns the link at the path.
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
	size, err := s.links.FindCopy(ctx, found)
	if err != nil {
		return api.GetLinkResponse{}, err
	}
	isDeleted, err := s.links.IsTextDeleted(ctx, found)
	if err != nil {
		return api.GetLinkResponse{}, err
	}
	return api.GetLinkResponse{
		URL: string(found.Address), IsCopyable: s.links.IsCopyable(found.Address), CopySize: size,
		IsTextDeleted: isDeleted, EmbedPage: getEmbedPage(found.Address),
	}, nil
}

func (s linkServer) readArticle(ctx context.Context, req api.ReadArticleRequest) (api.ReadArticleResponse, error) {
	found, err := s.readLink(ctx, req.Path)
	if err != nil {
		return api.ReadArticleResponse{}, err
	}
	text, err := s.links.ReadArticle(ctx, found)
	if err != nil {
		return api.ReadArticleResponse{}, err
	}
	return api.ReadArticleResponse{Text: text}, nil
}

func (s linkServer) fetchText(ctx context.Context, req api.FetchTextRequest) (api.FetchTextResponse, error) {
	found, err := s.readLink(ctx, req.Path)
	if err != nil {
		return api.FetchTextResponse{}, err
	}
	s.links.StartFetchText(ctx, found, req.IsUserStarted)
	return api.FetchTextResponse{}, nil
}

func (s linkServer) copyVideo(ctx context.Context, req api.CopyVideoRequest) (api.CopyVideoResponse, error) {
	found, err := s.readLink(ctx, req.Path)
	if err != nil {
		return api.CopyVideoResponse{}, err
	}
	s.links.StartCopy(ctx, found)
	return api.CopyVideoResponse{}, nil
}

func (s linkServer) deleteCopy(ctx context.Context, req api.DeleteCopyRequest) (api.DeleteCopyResponse, error) {
	found, err := s.readLink(ctx, req.Path)
	if err != nil {
		return api.DeleteCopyResponse{}, err
	}
	if err := s.links.DeleteCopy(ctx, found); err != nil {
		return api.DeleteCopyResponse{}, err
	}
	return api.DeleteCopyResponse{}, nil
}

// createLinks creates a link note of each web address in the folder and starts fetching what is
// published at each. A text that is no web address is skipped.
func (s linkServer) createLinks(ctx context.Context, req api.CreateLinksRequest) (api.CreateLinksResponse, error) {
	out := api.CreateLinksResponse{Paths: []string{}}
	for _, raw := range req.URLs {
		made, err := s.links.Create(ctx, req.Folder, raw)
		if errors.Is(err, address.ErrNotAURL) {
			continue
		}
		if err != nil {
			return api.CreateLinksResponse{}, toAPIError(err)
		}
		s.links.StartFetch(ctx, made)
		out.Paths = append(out.Paths, made.Path)
	}
	return out, nil
}

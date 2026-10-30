package service

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
	linksv1 "github.com/jiva-studio/numen/plugins/tools-media-importer/gen/numen/links/v1"
)

// linkServer serves LinkService over the links of the vault the host shows.
type linkServer struct {
	links *link.Links
}

// getErrorCode is the code a refusal is returned with, and false for an error the schema names no
// code for.
func getErrorCode(err error) (linksv1.LinkErrorCode, bool) {
	switch {
	case err == nil:
		return linksv1.LinkErrorCode_LINK_ERROR_CODE_UNSPECIFIED, true
	case errors.Is(err, link.ErrNotALink), errors.Is(err, address.ErrNotAURL), errors.Is(err, host.ErrNotExist):
		return linksv1.LinkErrorCode_LINK_ERROR_CODE_NOT_A_LINK, true
	case errors.Is(err, host.ErrNoVault):
		return linksv1.LinkErrorCode_LINK_ERROR_CODE_NO_VAULT, true
	}
	return 0, false
}

// readLink is the link at the path, or the code of the refusal, or the error that fails the call.
func (s linkServer) readLink(ctx context.Context, path string) (link.Link, linksv1.LinkErrorCode, error) {
	found, err := s.links.Read(ctx, path)
	code, ok := getErrorCode(err)
	if !ok {
		return link.Link{}, 0, connect.NewError(connect.CodeInternal, err)
	}
	return found, code, nil
}

func isRefused(code linksv1.LinkErrorCode) bool {
	return code != linksv1.LinkErrorCode_LINK_ERROR_CODE_UNSPECIFIED
}

func (s linkServer) GetLink(
	ctx context.Context, req *connect.Request[linksv1.GetLinkRequest],
) (*connect.Response[linksv1.GetLinkResponse], error) {
	found, code, err := s.readLink(ctx, req.Msg.GetPath())
	if err != nil || isRefused(code) {
		return connect.NewResponse(&linksv1.GetLinkResponse{Error: code}), err
	}
	size, err := s.links.FindCopy(ctx, found)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	isDeleted, err := s.links.IsTextDeleted(ctx, found)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&linksv1.GetLinkResponse{
		Url: string(found.Address), IsCopyable: s.links.IsCopyable(found.Address), CopySize: size,
		IsTextDeleted: isDeleted, EmbedPage: getEmbedPage(found.Address),
	}), nil
}

func (s linkServer) ReadArticle(
	ctx context.Context, req *connect.Request[linksv1.ReadArticleRequest],
) (*connect.Response[linksv1.ReadArticleResponse], error) {
	found, code, err := s.readLink(ctx, req.Msg.GetPath())
	if err != nil || isRefused(code) {
		return connect.NewResponse(&linksv1.ReadArticleResponse{Error: code}), err
	}
	text, err := s.links.ReadArticle(ctx, found)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&linksv1.ReadArticleResponse{Text: text}), nil
}

func (s linkServer) FetchText(
	ctx context.Context, req *connect.Request[linksv1.FetchTextRequest],
) (*connect.Response[linksv1.FetchTextResponse], error) {
	found, code, err := s.readLink(ctx, req.Msg.GetPath())
	if err != nil || isRefused(code) {
		return connect.NewResponse(&linksv1.FetchTextResponse{Error: code}), err
	}
	s.links.StartFetchText(ctx, found, req.Msg.GetIsUserStarted())
	return connect.NewResponse(&linksv1.FetchTextResponse{}), nil
}

func (s linkServer) CopyVideo(
	ctx context.Context, req *connect.Request[linksv1.CopyVideoRequest],
) (*connect.Response[linksv1.CopyVideoResponse], error) {
	found, code, err := s.readLink(ctx, req.Msg.GetPath())
	if err != nil || isRefused(code) {
		return connect.NewResponse(&linksv1.CopyVideoResponse{Error: code}), err
	}
	s.links.StartCopy(ctx, found)
	return connect.NewResponse(&linksv1.CopyVideoResponse{}), nil
}

func (s linkServer) DeleteCopy(
	ctx context.Context, req *connect.Request[linksv1.DeleteCopyRequest],
) (*connect.Response[linksv1.DeleteCopyResponse], error) {
	found, code, err := s.readLink(ctx, req.Msg.GetPath())
	if err != nil || isRefused(code) {
		return connect.NewResponse(&linksv1.DeleteCopyResponse{Error: code}), err
	}
	if err := s.links.DeleteCopy(ctx, found); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&linksv1.DeleteCopyResponse{}), nil
}

// CreateLinks makes a link note of each web address in the folder and starts fetching what is
// published at each. A text that is no web address is skipped.
func (s linkServer) CreateLinks(
	ctx context.Context, req *connect.Request[linksv1.CreateLinksRequest],
) (*connect.Response[linksv1.CreateLinksResponse], error) {
	out := &linksv1.CreateLinksResponse{}
	for _, raw := range req.Msg.GetUrls() {
		made, err := s.links.Create(ctx, req.Msg.GetFolder(), raw)
		if errors.Is(err, address.ErrNotAURL) {
			continue
		}
		if code, ok := getErrorCode(err); err != nil && ok {
			out.Error = code
			return connect.NewResponse(out), nil
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		s.links.StartFetch(ctx, made)
		out.Paths = append(out.Paths, made.Path)
	}
	return connect.NewResponse(out), nil
}

package service

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// articleServer answers ArticleService over the links of the vault the host shows.
type articleServer struct {
	links *link.Links
}

// getErrorCode is the code a refusal is answered with, and false for an error the schema names no
// code for.
func getErrorCode(err error) (v1.ArticleErrorCode, bool) {
	switch {
	case err == nil:
		return v1.ArticleErrorCode_ARTICLE_ERROR_CODE_UNSPECIFIED, true
	case errors.Is(err, domain.ErrNotAURL):
		return v1.ArticleErrorCode_ARTICLE_ERROR_CODE_NOT_AN_ADDRESS, true
	case errors.Is(err, host.ErrOccupied):
		return v1.ArticleErrorCode_ARTICLE_ERROR_CODE_OCCUPIED, true
	case errors.Is(err, link.ErrNotALink), errors.Is(err, host.ErrNotExist):
		return v1.ArticleErrorCode_ARTICLE_ERROR_CODE_NOT_A_LINK, true
	case errors.Is(err, host.ErrNoVault):
		return v1.ArticleErrorCode_ARTICLE_ERROR_CODE_NO_VAULT, true
	}
	return 0, false
}

// answer is the code a refusal is answered with, or the error that fails the call.
func answer(err error) (v1.ArticleErrorCode, error) {
	if code, ok := getErrorCode(err); ok {
		return code, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

// CreateLink makes the link and fetches for it in the background.
func (s articleServer) CreateLink(
	ctx context.Context, req *connect.Request[v1.CreateLinkRequest],
) (*connect.Response[v1.CreateLinkResponse], error) {
	made, err := s.links.Create(ctx, req.Msg.GetFolder(), req.Msg.GetUrl())
	code, err := answer(err)
	if err != nil || code != v1.ArticleErrorCode_ARTICLE_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.CreateLinkResponse{Error: code}), err
	}
	s.links.StartFetch(made)
	return connect.NewResponse(&v1.CreateLinkResponse{Path: made.Path}), nil
}

func (s articleServer) GetLink(
	ctx context.Context, req *connect.Request[v1.GetLinkRequest],
) (*connect.Response[v1.GetLinkResponse], error) {
	found, err := s.links.Read(ctx, req.Msg.GetPath())
	code, err := answer(err)
	if err != nil || code != v1.ArticleErrorCode_ARTICLE_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.GetLinkResponse{Error: code}), err
	}
	size, err := s.links.FindCopy(ctx, found)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetLinkResponse{
		Url: string(found.Address), IsCopyable: s.links.IsCopyable(found.Address), CopySize: size,
	}), nil
}

func (s articleServer) ReadArticle(
	ctx context.Context, req *connect.Request[v1.ReadArticleRequest],
) (*connect.Response[v1.ReadArticleResponse], error) {
	text, err := s.links.ReadArticle(ctx, req.Msg.GetPath())
	code, err := answer(err)
	if err != nil || code != v1.ArticleErrorCode_ARTICLE_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.ReadArticleResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.ReadArticleResponse{Text: text}), nil
}

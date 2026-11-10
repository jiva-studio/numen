package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/document"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// DocumentServer answers the document methods of the plugin API. Each call works the vault its Numen-Vault
// header names.
type DocumentServer struct {
	reader *document.Reader
}

// NewDocumentServer returns a DocumentServer that answers from reader.
func NewDocumentServer(reader *document.Reader) *DocumentServer {
	return &DocumentServer{reader: reader}
}

// Mount registers the document methods on api and returns the first failure to mount one.
func (s *DocumentServer) Mount(service *server.API) error {
	server.Handle(service, "GetDocument", s.GetDocument)
	server.Handle(service, "GetHighlights", s.GetHighlights)
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="Mount returns any registration error accumulated on api"
	return service.Err()
}

// GetDocument returns the size of each page of the PDF.
func (s *DocumentServer) GetDocument(ctx context.Context, req api.GetDocumentRequest) (api.GetDocumentResponse, error) {
	if req.Path == "" {
		return api.GetDocumentResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	pages, err := s.reader.GetPages(ctx, host.GetRequestVault(server.GetHeader(ctx)), req.Path)
	if err != nil {
		return api.GetDocumentResponse{}, getAPIError(err)
	}
	out := api.GetDocumentResponse{Pages: make([]api.Page, 0, len(pages)), Stat: nil}
	for _, page := range pages {
		out.Pages = append(out.Pages, api.Page{Width: page.Width, Height: page.Height})
	}
	return out, nil
}

// GetHighlights places each range of the file's live text version on the pages.
func (s *DocumentServer) GetHighlights(ctx context.Context, req api.GetHighlightsRequest) (api.GetHighlightsResponse, error) {
	if req.Path == "" {
		return api.GetHighlightsResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	ranges := make([]domain.ByteRange, 0, len(req.Ranges))
	for _, one := range req.Ranges {
		ranges = append(ranges, domain.ByteRange{Start: one.Start, End: one.End})
	}
	boxes, err := s.reader.GetHighlights(ctx, host.GetRequestVault(server.GetHeader(ctx)), req.Path, ranges)
	if err != nil {
		return api.GetHighlightsResponse{}, getAPIError(err)
	}
	out := api.GetHighlightsResponse{Runs: make([]api.DocumentRun, 0, len(boxes))}
	for _, run := range boxes {
		out.Runs = append(out.Runs, api.DocumentRun{Places: writePlaces(run)})
	}
	return out, nil
}

func getAPIError(err error) error {
	var badRange *document.RangeError
	switch {
	case errors.As(err, &badRange):
		return server.NewAPIError(server.CodeInvalidArgument, err.Error())
	case errors.Is(err, document.ErrOpen):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, document.ErrPageSize), errors.Is(err, document.ErrClose):
		return server.NewAPIError(server.CodeInternal, err.Error())
	case errors.Is(err, document.ErrText):
		return server.NewAPIError(server.CodeUnavailable, err.Error())
	default:
		return readVaultError(err)
	}
}

func writePlaces(boxes []pdf.HighlightBox) []api.DocumentPlace {
	places := []api.DocumentPlace{}
	for _, box := range boxes {
		rect := api.DocumentRect{MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY}
		if n := len(places); n > 0 && places[n-1].Page == box.Page {
			places[n-1].Rects = append(places[n-1].Rects, rect)
			continue
		}
		places = append(places, api.DocumentPlace{Page: box.Page, Rects: []api.DocumentRect{rect}})
	}
	return places
}

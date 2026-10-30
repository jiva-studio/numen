package service

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// DocumentServer answers the document methods of the plugin API. Each call works the vault its Numen-Vault
// header names.
type DocumentServer struct {
	files host.Vault
	texts host.Texts
}

// NewDocumentServer configures the worker pool and returns a DocumentServer that reads files from
// files and text versions from texts.
func NewDocumentServer(files host.Vault, texts host.Texts) *DocumentServer {
	pdf.SetAfterFunc(clock.AfterFunc)
	pdf.SetWorkers(runtime.NumCPU())
	pdf.SetIdleWorkers(max(2, min(runtime.NumCPU(), 4)))
	go pdf.Warmup()
	return &DocumentServer{files: files, texts: texts}
}

// Mount registers the document methods on api and returns the first failure to mount one.
func (s *DocumentServer) Mount(api *server.API) error {
	server.Handle(api, "GetDocument", s.GetDocument)
	server.Handle(api, "GetHighlights", s.GetHighlights)
	return api.Err()
}

// GetDocument returns the size of each page of the PDF.
func (s *DocumentServer) GetDocument(ctx context.Context, req GetDocumentRequest) (res GetDocumentResponse, err error) {
	if req.Path == "" {
		return GetDocumentResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	raw, _, err := getVault(s.files, server.GetHeader(ctx)).Read(ctx, req.Path)
	if err != nil {
		return GetDocumentResponse{}, readVaultError(err)
	}

	scan, err := pdf.OpenBytes(raw)
	if err != nil {
		return GetDocumentResponse{}, server.NewAPIError(server.CodeFailedPrecondition, fmt.Sprintf("read pdf: %v", err))
	}
	defer func() {
		if closeErr := scan.Close(); closeErr != nil {
			err = errors.Join(err, server.NewAPIError(server.CodeInternal, fmt.Sprintf("close pdf: %v", closeErr)))
		}
	}()

	pageCount := scan.GetPageCount()
	pages := make([]Page, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		w, h, err := scan.GetPageSize(i)
		if err != nil {
			return GetDocumentResponse{}, server.NewAPIError(server.CodeInternal, fmt.Sprintf("page size: %v", err))
		}
		pages = append(pages, Page{Width: w, Height: h})
	}
	return GetDocumentResponse{Pages: pages, Stat: nil}, nil
}

// mostSpans is the most ranges one GetHighlights request may hold.
const mostSpans = 64

// GetHighlights places each range of the file's live text version on the pages. A file with no live
// version, or a version without places, has no places.
func (s *DocumentServer) GetHighlights(ctx context.Context, req GetHighlightsRequest) (GetHighlightsResponse, error) {
	if req.Path == "" {
		return GetHighlightsResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	if _, err := getVault(s.files, server.GetHeader(ctx)).StatFile(ctx, req.Path); errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return GetHighlightsResponse{}, readVaultError(err)
	}
	if len(req.Ranges) == 0 || len(req.Ranges) > mostSpans {
		return GetHighlightsResponse{}, server.NewAPIError(server.CodeInvalidArgument, fmt.Sprintf("expected between one and %d ranges, got %d", mostSpans, len(req.Ranges)))
	}
	runs := make([]domain.ByteRange, 0, len(req.Ranges))
	end := 0
	for _, one := range req.Ranges {
		run := domain.ByteRange{Start: one.Start, End: one.End}
		if run.Start < 0 || run.End <= run.Start {
			return GetHighlightsResponse{}, server.NewAPIError(server.CodeInvalidArgument, fmt.Sprintf("%d..%d is not a range of the text", run.Start, run.End))
		}
		runs = append(runs, run)
		end = max(end, run.End)
	}

	texts := s.texts
	if id := host.GetRequestVault(server.GetHeader(ctx)); id != "" {
		texts = texts.With(id)
	}
	text, err := readLiveText(ctx, texts, req.Path, end)
	if err != nil {
		return GetHighlightsResponse{}, server.NewAPIError(server.CodeUnavailable, fmt.Sprintf("read the text of %s: %v", req.Path, err))
	}
	out := GetHighlightsResponse{Runs: make([]DocumentRun, 0, len(runs))}
	for _, run := range runs {
		out.Runs = append(out.Runs, DocumentRun{Places: writePlaces(getBoxes(text, pdf.ByteRange{Start: run.Start, End: run.End}))})
	}
	return out, nil
}

// readLiveText reads the live text version of a file until it holds the offset end. A file with
// no live version has an empty text.
func readLiveText(ctx context.Context, texts host.Texts, path string, end int) (domain.VersionText, error) {
	listing, err := texts.ListVersions(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return domain.VersionText{Text: ""}, nil
	}
	if err != nil {
		return domain.VersionText{}, err
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.GetID() == listing.Live })
	if listing.Live == "" || i < 0 {
		return domain.VersionText{Text: ""}, nil
	}
	live := listing.Versions[i]
	var parts []domain.TextSegment
	size, cursor := 0, ""
	for {
		page, err := texts.ReadText(ctx, path, live.GetID(), cursor)
		if err != nil {
			return domain.VersionText{}, err
		}
		parts = append(parts, page.Segments...)
		for _, part := range page.Segments {
			size += len(part.Text)
		}
		if page.NextCursor == "" || page.NextCursor == cursor || size >= end {
			return domain.NewVersionText(live, parts), nil
		}
		cursor = page.NextCursor
	}
}

// writePlaces groups the boxes of a run by page, in page order of appearance.
func writePlaces(boxes []pdf.HighlightBox) []DocumentPlace {
	places := []DocumentPlace{}
	for _, box := range boxes {
		rect := DocumentRect{MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY}
		if n := len(places); n > 0 && places[n-1].Page == box.Page {
			places[n-1].Rects = append(places[n-1].Rects, rect)
			continue
		}
		places = append(places, DocumentPlace{Page: box.Page, Rects: []DocumentRect{rect}})
	}
	return places
}

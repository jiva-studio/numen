package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
	pdfv1 "github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1"
	"github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1/pdfv1connect"
)

var initOnce sync.Once

// DocumentServer implements pdfv1connect.DocumentServiceHandler. Each call works the vault its
// Numen-Vault header names.
type DocumentServer struct {
	pdfv1connect.UnimplementedDocumentServiceHandler

	vaults vaultRoots
	texts  host.Texts
}

// NewDocumentServer configures the worker pool and returns a DocumentServer that finds the vault of
// a request through files, works the vault at defaultRoot for a request naming none, and reads the
// text versions of the files from texts.
func NewDocumentServer(files host.Vault, defaultRoot func() string, texts host.Texts) *DocumentServer {
	initOnce.Do(func() {
		pdf.SetWorkers(runtime.NumCPU())
		pdf.SetIdleWorkers(max(2, min(runtime.NumCPU(), 4)))
		go pdf.Warmup()
	})
	return &DocumentServer{vaults: vaultRoots{files: files, defaultRoot: defaultRoot}, texts: texts}
}

// GetDocument returns metadata and dimensions for each page in the PDF.
func (s *DocumentServer) GetDocument(
	ctx context.Context,
	req *connect.Request[pdfv1.GetDocumentRequest],
) (*connect.Response[pdfv1.GetDocumentResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	cleanPath, err := s.vaults.getFilePath(ctx, req.Header(), path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("open file: %w", err))
	}
	defer file.Close()

	scan, err := pdf.OpenReader(file)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("read pdf: %w", err))
	}
	defer scan.Close()

	pageCount := scan.GetPageCount()
	pages := make([]*pdfv1.Page, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		w, h, err := scan.GetPageSize(i)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("page size: %w", err))
		}
		pages = append(pages, &pdfv1.Page{
			Width:  w,
			Height: h,
		})
	}

	return connect.NewResponse(&pdfv1.GetDocumentResponse{
		Pages: pages,
	}), nil
}

// mostSpans is the most runs one question about highlights may name.
const mostSpans = 64

// GetHighlights answers where runs of the file's live text version stand on its pages. A file with
// no live version, or one whose version has no places, stands nowhere.
func (s *DocumentServer) GetHighlights(
	ctx context.Context,
	req *connect.Request[pdfv1.GetHighlightsRequest],
) (*connect.Response[pdfv1.GetHighlightsResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	if _, err := s.vaults.getFilePath(ctx, req.Header(), path); err != nil {
		return nil, err
	}
	spans := req.Msg.GetSpans()
	if len(spans) == 0 || len(spans) > mostSpans {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("ask about between one and %d runs, not %d", mostSpans, len(spans)))
	}
	runs := make([]domain.ByteSpan, 0, len(spans))
	end := 0
	for _, one := range spans {
		run := domain.ByteSpan{From: int(one.GetFrom()), To: int(one.GetTo())}
		if run.From < 0 || run.To <= run.From {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%d..%d is not a run of the text", run.From, run.To))
		}
		runs = append(runs, run)
		end = max(end, run.To)
	}

	texts := s.texts
	if id := host.GetRequestVault(req.Header()); id != "" {
		texts = texts.With(id)
	}
	text, err := readLiveText(ctx, texts, path, end)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read the text of %s: %w", path, err))
	}
	out := &pdfv1.GetHighlightsResponse{Runs: make([]*pdfv1.DocumentRun, 0, len(runs))}
	for _, run := range runs {
		out.Runs = append(out.Runs, &pdfv1.DocumentRun{Places: writePlaces(host.GetBoxes(text, run))})
	}
	return connect.NewResponse(out), nil
}

// readLiveText is the live text version of a file, read until it holds the offset end. A file with
// no live version has an empty text.
func readLiveText(ctx context.Context, texts host.Texts, path string, end int) (domain.VersionText, error) {
	listing, err := texts.ListVersions(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return domain.VersionText{}, nil
	}
	if err != nil {
		return domain.VersionText{}, err
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.GetID() == listing.Live })
	if listing.Live == "" || i < 0 {
		return domain.VersionText{}, nil
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

// writePlaces is the boxes of a run as places, one a page, in the order the pages come.
func writePlaces(boxes []domain.HighlightBox) []*pdfv1.DocumentPlace {
	var places []*pdfv1.DocumentPlace
	for _, box := range boxes {
		rect := &pdfv1.DocumentRect{MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY}
		if n := len(places); n > 0 && places[n-1].GetPage() == int32(box.Page) {
			places[n-1].Rects = append(places[n-1].Rects, rect)
			continue
		}
		places = append(places, &pdfv1.DocumentPlace{Page: int32(box.Page), Rects: []*pdfv1.DocumentRect{rect}})
	}
	return places
}

package service

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"os"
	"runtime"
	"sync"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/vault"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/pdf"
)

var initOnce sync.Once

// DocumentServer implements numenv1connect.DocumentServiceHandler.
type DocumentServer struct {
	numenv1connect.UnimplementedDocumentServiceHandler

	// vaultRoot is the vault the paths of requests are held to.
	vaultRoot func() string
}

// NewDocumentServer configures the worker pool and returns a DocumentServer that holds paths to the
// vault NUMEN_VAULT_PATH names.
func NewDocumentServer() *DocumentServer {
	return newDocumentServer(func() string { return os.Getenv("NUMEN_VAULT_PATH") })
}

// NewDocumentServerIn configures the worker pool and returns a DocumentServer that holds paths to
// the vault at vaultRoot.
func NewDocumentServerIn(vaultRoot string) *DocumentServer {
	return newDocumentServer(func() string { return vaultRoot })
}

func newDocumentServer(vaultRoot func() string) *DocumentServer {
	initOnce.Do(func() {
		pdf.SetWorkers(runtime.NumCPU())
		pdf.SetIdleWorkers(max(2, min(runtime.NumCPU(), 4)))
		go pdf.Warmup()
	})
	return &DocumentServer{vaultRoot: vaultRoot}
}

// GetDocument returns metadata and dimensions for each page in the PDF.
func (s *DocumentServer) GetDocument(
	_ context.Context,
	req *connect.Request[numenv1.GetDocumentRequest],
) (*connect.Response[numenv1.GetDocumentResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	cleanPath, err := vault.ValidatePathIn(s.vaultRoot(), path, vault.IsTempDocument)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope: %w", err))
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
	pages := make([]*numenv1.Page, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		w, h, err := scan.GetPageSize(i)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("page size: %w", err))
		}
		pages = append(pages, &numenv1.Page{
			Width:  w,
			Height: h,
		})
	}

	return connect.NewResponse(&numenv1.GetDocumentResponse{
		Pages: pages,
	}), nil
}

// RenderPage renders a single page of the PDF to JPEG format.
func (s *DocumentServer) RenderPage(
	_ context.Context,
	req *connect.Request[numenv1.RenderPageRequest],
) (*connect.Response[numenv1.RenderPageResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	cleanPath, err := vault.ValidatePathIn(s.vaultRoot(), path, vault.IsTempDocument)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope: %w", err))
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

	dpi := int(req.Msg.GetDpi())
	if dpi <= 0 {
		dpi = 150
	}

	img, err := scan.RenderPage(int(req.Msg.GetPage()), dpi)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("render page: %w", err))
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode jpeg: %w", err))
	}

	bounds := img.Bounds()
	return connect.NewResponse(&numenv1.RenderPageResponse{
		ImageData:   buf.Bytes(),
		ContentType: "image/jpeg",
		Width:       float64(bounds.Dx()),
		Height:      float64(bounds.Dy()),
	}), nil
}

// ExtractText extracts text and page offset markers from the PDF.
func (s *DocumentServer) ExtractText(
	_ context.Context,
	req *connect.Request[numenv1.ExtractTextRequest],
) (*connect.Response[numenv1.ExtractTextResponse], error) {
	raw := req.Msg.GetData()
	if len(raw) == 0 {
		path := req.Msg.GetPath()
		if path == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path or data required"))
		}
		cleanPath, err := vault.ValidatePathIn(s.vaultRoot(), path, vault.IsTempDocument)
		if err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope: %w", err))
		}
		var readErr error
		raw, readErr = os.ReadFile(cleanPath)
		if readErr != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read file: %w", readErr))
		}
	}

	book, err := pdf.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("parse pdf: %w", err))
	}

	starts := make([]int32, len(book.Pages))
	for i, p := range book.Pages {
		starts[i] = int32(p.Offset)
	}

	return connect.NewResponse(&numenv1.ExtractTextResponse{
		Text:       book.Text,
		PageStarts: starts,
	}), nil
}

// GetHighlights returns extracted annotations from the PDF.
func (s *DocumentServer) GetHighlights(
	_ context.Context,
	req *connect.Request[numenv1.GetHighlightsRequest],
) (*connect.Response[numenv1.GetHighlightsResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	cleanPath, err := vault.ValidatePathIn(s.vaultRoot(), path, vault.IsTempDocument)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope: %w", err))
	}

	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read file: %w", err))
	}

	book, err := pdf.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("parse pdf: %w", err))
	}

	pages := make([]int, len(book.Pages))
	for i := range pages {
		pages[i] = i
	}

	boxes, err := book.Highlights(raw, pages)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("extract highlights: %w", err))
	}

	highlights := make([]*numenv1.DocumentHighlight, 0, len(boxes))
	for _, box := range boxes {
		var text string
		if box.From >= 0 && box.To <= len(book.Text) && box.From <= box.To {
			text = book.Text[box.From:box.To]
		}
		highlights = append(highlights, &numenv1.DocumentHighlight{
			PageNumber: uint32(box.Page),
			QuadPoints: []float32{
				box.MinX, box.MinY,
				box.MaxX, box.MinY,
				box.MinX, box.MaxY,
				box.MaxX, box.MaxY,
			},
			Color: "#ffff00",
			Text:  text,
		})
	}

	return connect.NewResponse(&numenv1.GetHighlightsResponse{
		Highlights: highlights,
	}), nil
}

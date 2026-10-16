//go:build plugins_monolith

package platform

import (
	"bytes"
	"context"
	"errors"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/epub"
	"github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/pdf"
)

// BuildMode is the desktop build variant.
const BuildMode = "monolith"

// IsMonolith reports whether plugins are compiled directly into the binary.
func IsMonolith() bool {
	return true
}

func init() {
	wireMonolithHandlers()
}

func wireMonolithHandlers() {
	container.RegisterDocumentServiceHandler(&monolithDocumentServer{})
	container.RegisterBookServiceHandler(&monolithBookServer{})
	container.RegisterRecognitionServiceHandler(&monolithRecognitionServer{})
	container.RegisterTranscriptionServiceHandler(&monolithTranscriptionServer{})
	container.RegisterEmbeddingServiceHandler(&monolithEmbeddingServer{})
}

func validateMonolithVaultPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path required")
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return "", errors.New("path traversal outside allowed scope")
	}

	tempDir := filepath.Clean(os.TempDir())
	if filepath.Dir(cleanPath) == tempDir && strings.HasPrefix(filepath.Base(cleanPath), "numen-doc-") {
		return cleanPath, nil
	}

	vaultRoot := os.Getenv("NUMEN_VAULT_PATH")
	if vaultRoot != "" {
		cleanVault := filepath.Clean(vaultRoot)
		if !filepath.IsAbs(cleanPath) {
			cleanPath = filepath.Join(cleanVault, cleanPath)
		}
		prefix := cleanVault
		if !strings.HasSuffix(prefix, string(filepath.Separator)) {
			prefix += string(filepath.Separator)
		}
		if cleanPath != cleanVault && !strings.HasPrefix(cleanPath, prefix) {
			return "", errors.New("path outside allowed vault root")
		}
	}
	return cleanPath, nil
}

type monolithDocumentServer struct {
	numenv1connect.UnimplementedDocumentServiceHandler
}

func (s *monolithDocumentServer) GetDocument(
	_ context.Context,
	req *connect.Request[numenv1.GetDocumentRequest],
) (*connect.Response[numenv1.GetDocumentResponse], error) {
	cleanPath, err := validateMonolithVaultPath(req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	defer file.Close()

	scan, err := pdf.OpenReader(file)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	defer scan.Close()

	pageCount := scan.GetPageCount()
	pages := make([]*numenv1.Page, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		w, h, err := scan.GetPageSize(i)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		pages = append(pages, &numenv1.Page{Width: w, Height: h})
	}

	return connect.NewResponse(&numenv1.GetDocumentResponse{Pages: pages}), nil
}

func (s *monolithDocumentServer) RenderPage(
	_ context.Context,
	req *connect.Request[numenv1.RenderPageRequest],
) (*connect.Response[numenv1.RenderPageResponse], error) {
	cleanPath, err := validateMonolithVaultPath(req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	defer file.Close()

	scan, err := pdf.OpenReader(file)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	defer scan.Close()

	dpi := int(req.Msg.GetDpi())
	if dpi <= 0 {
		dpi = 150
	}

	img, err := scan.RenderPage(int(req.Msg.GetPage()), dpi)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	bounds := img.Bounds()
	return connect.NewResponse(&numenv1.RenderPageResponse{
		ImageData:   buf.Bytes(),
		ContentType: "image/jpeg",
		Width:       float64(bounds.Dx()),
		Height:      float64(bounds.Dy()),
	}), nil
}

func (s *monolithDocumentServer) ExtractText(
	_ context.Context,
	req *connect.Request[numenv1.ExtractTextRequest],
) (*connect.Response[numenv1.ExtractTextResponse], error) {
	raw := req.Msg.GetData()
	if len(raw) == 0 {
		cleanPath, err := validateMonolithVaultPath(req.Msg.GetPath())
		if err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		var readErr error
		raw, readErr = os.ReadFile(cleanPath)
		if readErr != nil {
			return nil, connect.NewError(connect.CodeNotFound, readErr)
		}
	}

	book, err := pdf.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
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

func (s *monolithDocumentServer) GetHighlights(
	_ context.Context,
	req *connect.Request[numenv1.GetHighlightsRequest],
) (*connect.Response[numenv1.GetHighlightsResponse], error) {
	cleanPath, err := validateMonolithVaultPath(req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	book, err := pdf.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	pages := make([]int, len(book.Pages))
	for i := range pages {
		pages[i] = i
	}
	boxes, err := book.Highlights(raw, pages)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoHighlights := make([]*numenv1.DocumentHighlight, 0, len(boxes))
	for _, box := range boxes {
		var text string
		if box.From >= 0 && box.To <= len(book.Text) && box.From <= box.To {
			text = book.Text[box.From:box.To]
		}
		protoHighlights = append(protoHighlights, &numenv1.DocumentHighlight{
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
		Highlights: protoHighlights,
	}), nil
}

type monolithBookServer struct {
	numenv1connect.UnimplementedBookServiceHandler
}

const mostListed = 1000

func (s *monolithBookServer) GetBook(
	_ context.Context,
	req *connect.Request[numenv1.GetBookRequest],
) (*connect.Response[numenv1.GetBookResponse], error) {
	cleanPath, err := validateMonolithVaultPath(req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	book, err := epub.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	resp := &numenv1.GetBookResponse{
		Title:     book.Title,
		PageCount: int32(book.PageCount()),
		TextBytes: int32(len(book.Text)),
		PageBytes: int32(book.PageBytes()),
	}

	docCount := min(len(book.Documents), mostListed)
	for _, doc := range book.Documents[:docCount] {
		resp.Documents = append(resp.Documents, &numenv1.SpineDocument{
			Path:   doc.Path,
			Offset: int32(doc.Offset),
			Length: int32(doc.Length),
		})
	}

	partCount := min(len(book.Parts), mostListed)
	for _, part := range book.Parts[:partCount] {
		resp.Parts = append(resp.Parts, &numenv1.BookPart{
			Title:  part.Title,
			Offset: int32(part.Offset),
			Level:  int32(part.Level),
		})
	}

	pageCount := min(len(book.Pages), mostListed)
	for _, page := range book.Pages[:pageCount] {
		resp.PrintedPages = append(resp.PrintedPages, &numenv1.PrintedPage{
			Label:  page.Label,
			Offset: int32(page.Offset),
		})
	}

	return connect.NewResponse(resp), nil
}

func (s *monolithBookServer) ReadBookMarkup(
	_ context.Context,
	req *connect.Request[numenv1.ReadBookMarkupRequest],
) (*connect.Response[numenv1.ReadBookMarkupResponse], error) {
	cleanPath, err := validateMonolithVaultPath(req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	book, err := epub.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	drawn, err := book.Markup(req.Msg.GetDocument())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&numenv1.ReadBookMarkupResponse{
		Markup: drawn.HTML(),
	}), nil
}

type monolithRecognitionServer struct {
	numenv1connect.UnimplementedRecognitionServiceHandler
}

func (s *monolithRecognitionServer) GetRecognitionModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetRecognitionModelRequest],
) (*connect.Response[numenv1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&numenv1.GetRecognitionModelResponse{
		Layout:     "monolith-layout",
		Recogniser: "monolith-recog",
		Dpi:        150,
		From:       "builtin",
	}), nil
}

type monolithTranscriptionServer struct {
	numenv1connect.UnimplementedTranscriptionServiceHandler
}

func (s *monolithTranscriptionServer) GetTranscriptionModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetTranscriptionModelRequest],
) (*connect.Response[numenv1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&numenv1.GetTranscriptionModelResponse{
		Model:     "monolith-transcription",
		Segmenter: "builtin",
		Cutting:   "",
		From:      "builtin",
	}), nil
}

type monolithEmbeddingServer struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
}

func (s *monolithEmbeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetEmbeddingModelRequest],
) (*connect.Response[numenv1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&numenv1.GetEmbeddingModelResponse{
		Name:        "bge-m3",
		Dimensions:  1024,
		MaxTokens:   512,
		Pooling:     "mean",
		From:        "builtin",
		Concurrency: 4,
	}), nil
}

func (s *monolithEmbeddingServer) EmbedText(
	_ context.Context,
	_ *connect.Request[numenv1.EmbedTextRequest],
) (*connect.Response[numenv1.EmbedTextResponse], error) {
	return connect.NewResponse(&numenv1.EmbedTextResponse{
		Vector: make([]float32, 1024),
	}), nil
}

func (s *monolithEmbeddingServer) EmbedBatch(
	_ context.Context,
	req *connect.Request[numenv1.EmbedBatchRequest],
) (*connect.Response[numenv1.EmbedBatchResponse], error) {
	texts := req.Msg.GetTexts()
	vectors := make([]*numenv1.FloatVector, len(texts))
	for i := range vectors {
		vectors[i] = &numenv1.FloatVector{
			Values: make([]float32, 1024),
		}
	}
	return connect.NewResponse(&numenv1.EmbedBatchResponse{
		Vectors: vectors,
	}), nil
}

package container

import (
	"context"
	"errors"
	"image"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errDocumentUnavailable = errors.New("document plugin is unavailable or not installed")
)

var _ port.Document = (*unavailableOpenDocument)(nil)

// DocumentRenderer is the port that renders pages of documents.
// Core retains abstract port.DocumentRenderer / port.Document ports.
func (c Config) DocumentRenderer() port.DocumentRenderer { return c.PageRenderer() }

// TextExtractor takes the text out of a file whose text is laid out on printed pages.
func (c Config) TextExtractor() port.TextExtractor {
	if c.Texts != nil {
		return c.Texts
	}
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("document"); ok {
			return hostDocumentPort{host: c.PluginHost}
		}
		if _, ok := c.PluginHost.FindPluginByCapability("book"); ok {
			return hostDocumentPort{host: c.PluginHost}
		}
	}
	return unavailableDocument{}
}

// PageRenderer draws such a document's pages.
func (c Config) PageRenderer() port.PageRenderer {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("document"); ok {
			return hostDocumentPort{host: c.PluginHost}
		}
	}
	return unavailableDocument{}
}

type hostDocumentPort struct {
	host port.PluginManager
}

func (h hostDocumentPort) openBridge(ctx context.Context) (*plugin.DocumentBridge, error) {
	if h.host == nil {
		return nil, errDocumentUnavailable
	}
	return plugin.OpenDocumentBridge(ctx, h.host)
}

func (h hostDocumentPort) Read(ctx context.Context, raw []byte) (domain.TextLayer, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return domain.TextLayer{}, err
	}
	defer bridge.Close()
	return bridge.Read(ctx, raw)
}

func (h hostDocumentPort) ReadBook(ctx context.Context, raw []byte) (domain.BookText, error) {
	if h.host == nil {
		return domain.BookText{}, errDocumentUnavailable
	}
	bridge, err := plugin.OpenBookBridge(ctx, h.host)
	if err != nil {
		return domain.BookText{}, err
	}
	defer bridge.Close()
	return bridge.ReadBook(ctx, raw)
}

func (h hostDocumentPort) GetHighlights(ctx context.Context, raw []byte, starts []int, pages []int) ([]domain.HighlightBox, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return nil, err
	}
	defer bridge.Close()
	return bridge.GetHighlights(ctx, raw, starts, pages)
}

func (h hostDocumentPort) OpenDocument(ctx context.Context, file io.ReadSeekCloser) (port.OpenDocument, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		return nil, err
	}
	doc, err := bridge.OpenDocument(ctx, file)
	if err != nil {
		_ = bridge.Close()
		return nil, err
	}
	return doc, nil
}

type unavailableDocument struct{}

func (unavailableDocument) Read(_ context.Context, _ []byte) (domain.TextLayer, error) {
	return domain.TextLayer{}, errDocumentUnavailable
}

func (unavailableDocument) GetHighlights(_ context.Context, _ []byte, _ []int, _ []int) ([]domain.HighlightBox, error) {
	return nil, errDocumentUnavailable
}

func (unavailableDocument) ReadBook(_ context.Context, _ []byte) (domain.BookText, error) {
	return domain.BookText{}, errDocumentUnavailable
}

func (unavailableDocument) OpenDocument(_ context.Context, file io.ReadSeekCloser) (port.OpenDocument, error) {
	if file != nil {
		_ = file.Close()
	}
	return nil, errDocumentUnavailable
}

type unavailableOpenDocument struct{}

func (unavailableOpenDocument) GetPageCount() int { return 0 }
func (unavailableOpenDocument) GetPageSize(_ int) (float64, float64, error) {
	return 0, 0, errDocumentUnavailable
}
func (unavailableOpenDocument) RenderPage(_, _ int) (image.Image, error) {
	return nil, errDocumentUnavailable
}
func (unavailableOpenDocument) Close() {}

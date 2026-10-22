package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// DocumentRenderer is the port that renders pages of documents.
// It is the same port as PageRenderer.
func (c Config) DocumentRenderer() port.DocumentRenderer { return c.PageRenderer() }

// TextExtractor takes the text out of a file whose text is laid out on printed pages.
func (c Config) TextExtractor() port.TextExtractor {
	if c.Texts != nil {
		return c.Texts
	}
	host, err := c.resolvePlugin(plugin.CapabilityDocument)
	if err != nil {
		if books, bookErr := c.resolvePlugin(plugin.CapabilityBook); bookErr == nil {
			return hostDocumentPort{host: books}
		}
		return unavailableDocument{why: err}
	}
	return hostDocumentPort{host: host}
}

// PageRenderer draws such a document's pages.
func (c Config) PageRenderer() port.PageRenderer {
	host, err := c.resolvePlugin(plugin.CapabilityDocument)
	if err != nil {
		return unavailableDocument{why: err}
	}
	return hostDocumentPort{host: host}
}

// hostDocumentPort reaches the document and the book plugins, each looked up when it is asked. A
// document it opens holds its own reference on the plugin.
type hostDocumentPort struct {
	host port.PluginManager
}

func (h hostDocumentPort) Read(ctx context.Context, raw []byte) (domain.TextLayer, error) {
	bridge, err := plugin.OpenDocumentBridge(ctx, h.host)
	if err != nil {
		return domain.TextLayer{}, err
	}
	defer bridge.Close()
	return bridge.Read(ctx, raw)
}

func (h hostDocumentPort) ReadBook(ctx context.Context, raw []byte) (domain.BookText, error) {
	bridge, err := plugin.OpenBookBridge(ctx, h.host)
	if err != nil {
		return domain.BookText{}, err
	}
	defer bridge.Close()
	return bridge.ReadBook(ctx, raw)
}

func (h hostDocumentPort) GetHighlights(ctx context.Context, raw []byte, starts []int, pages []int) ([]domain.HighlightBox, error) {
	bridge, err := plugin.OpenDocumentBridge(ctx, h.host)
	if err != nil {
		return nil, err
	}
	defer bridge.Close()
	return bridge.GetHighlights(ctx, raw, starts, pages)
}

func (h hostDocumentPort) OpenDocument(ctx context.Context, file io.ReadSeekCloser) (port.OpenDocument, error) {
	bridge, err := plugin.OpenDocumentBridge(ctx, h.host)
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		return nil, err
	}
	defer bridge.Close()
	return bridge.OpenDocument(ctx, file)
}

// unavailableDocument answers every ask with why no document plugin is there.
type unavailableDocument struct {
	why error
}

func (u unavailableDocument) Read(_ context.Context, _ []byte) (domain.TextLayer, error) {
	return domain.TextLayer{}, u.why
}

func (u unavailableDocument) GetHighlights(_ context.Context, _ []byte, _ []int, _ []int) ([]domain.HighlightBox, error) {
	return nil, u.why
}

func (u unavailableDocument) ReadBook(_ context.Context, _ []byte) (domain.BookText, error) {
	return domain.BookText{}, u.why
}

func (u unavailableDocument) OpenDocument(_ context.Context, file io.ReadSeekCloser) (port.OpenDocument, error) {
	if file != nil {
		_ = file.Close()
	}
	return nil, u.why
}

func (c Config) Books() port.Book { return hostBookPort{config: c} }

// hostBookPort reaches the book plugin, looked up when it is asked. A call made
type hostBookPort struct {
	config Config
}

// openBridge opens a bridge on the book plugin the host resolves now.
func (h hostBookPort) openBridge(ctx context.Context) (*plugin.BookBridge, error) {
	host, err := h.config.resolvePlugin(plugin.CapabilityBook)
	if err != nil {
		return nil, err
	}
	return plugin.OpenBookBridge(ctx, host)
}

func (h hostBookPort) GetBook(ctx context.Context, path string) (domain.BookShape, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return domain.BookShape{}, err
	}
	defer bridge.Close()
	return bridge.GetBook(ctx, path)
}

func (h hostBookPort) ReadBookMarkup(
	ctx context.Context, path, section string, baseline *domain.Fingerprint,
) (string, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return "", err
	}
	defer bridge.Close()
	return bridge.ReadBookMarkup(ctx, path, section, baseline)
}

func (h hostBookPort) ReadEntry(ctx context.Context, baseline domain.Fingerprint, entry string) ([]byte, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return nil, err
	}
	defer bridge.Close()
	return bridge.ReadEntry(ctx, baseline, entry)
}

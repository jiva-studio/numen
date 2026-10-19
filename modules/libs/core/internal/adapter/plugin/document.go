package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder for page images
	_ "image/png"  // register PNG decoder for page images
	"io"
	"math"
	"os"
	"strings"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// DocumentPluginHost provides plugin resolution, activation, permissions, and path validation.
type DocumentPluginHost interface {
	port.PluginResolver
	port.PluginActivator
	port.PluginAcquirer
	port.PluginAuthorizer
	port.PluginPathValidator
}

// DocumentBridge connects document and text extraction ports to an out-of-process PDF plugin via Connect-RPC.
type DocumentBridge struct {
	host     DocumentPluginHost
	pluginID string
	service  pluginService[numenv1connect.DocumentServiceClient]
	lease    *pluginLease
}

// OpenDocumentBridge activates or connects to the document plugin and returns a DocumentBridge.
func OpenDocumentBridge(ctx context.Context, host DocumentPluginHost) (*DocumentBridge, error) {
	p, err := host.ResolveCapability(CapabilityDocument)
	if err != nil {
		return nil, err
	}

	_, lease, err := openPlugin(ctx, host, p.Manifest.ID, numenv1connect.NewDocumentServiceClient)
	if err != nil {
		return nil, err
	}

	return &DocumentBridge{
		host:     host,
		pluginID: p.Manifest.ID,
		service:  pluginService[numenv1connect.DocumentServiceClient]{host: host, pluginID: p.Manifest.ID, newClient: numenv1connect.NewDocumentServiceClient},
		lease:    lease,
	}, nil
}

// Close releases the active reference count for the document plugin in the host supervisor.
func (b *DocumentBridge) Close() error {
	return b.lease.Close()
}

// Read implements port.TextExtractor.
func (b *DocumentBridge) Read(ctx context.Context, raw []byte) (domain.TextLayer, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
			return domain.TextLayer{}, err
		}
	}

	req := connect.NewRequest(&v1.ExtractTextRequest{
		Data: raw,
	})
	client, call, err := b.service.open(ctx)
	if err != nil {
		return domain.TextLayer{}, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.ExtractText(ctx, req)
	if err != nil {
		return domain.TextLayer{}, fmt.Errorf("rpc extract text: %w", err)
	}
	pages := make([]int, len(resp.Msg.GetPageStarts()))
	for i, s := range resp.Msg.GetPageStarts() {
		pages[i] = int(s)
	}
	return domain.TextLayer{
		Text:  resp.Msg.GetText(),
		Pages: pages,
	}, nil
}

// GetHighlights implements port.TextExtractor.
func (b *DocumentBridge) GetHighlights(ctx context.Context, raw []byte, starts []int, pages []int) ([]domain.HighlightBox, error) {
	if len(raw) == 0 || len(pages) == 0 {
		return nil, nil
	}

	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
			return nil, err
		}
	}

	textLayer, err := b.Read(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("read text layer: %w", err)
	}

	tmpPath, cleanup, err := bufferTempDocument(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer cleanup()

	req := connect.NewRequest(&v1.GetHighlightsRequest{
		Path: tmpPath,
	})
	client, call, err := b.service.open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.GetHighlights(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc get highlights: %w", err)
	}

	wanted := make(map[int]bool, len(pages))
	for _, p := range pages {
		if p >= 0 {
			wanted[p] = true
		}
	}

	pageStarts := starts
	if len(pageStarts) == 0 {
		pageStarts = textLayer.Pages
	}

	return mapHighlightBoxes(resp.Msg.GetHighlights(), wanted, textLayer.Text, pageStarts), nil
}

func mapHighlightBoxes(
	highlights []*v1.DocumentHighlight,
	wanted map[int]bool,
	docText string,
	pageStarts []int,
) []domain.HighlightBox {
	pageCursors := make(map[int]int)
	var boxes []domain.HighlightBox
	for _, h := range highlights {
		page := int(h.GetPageNumber())
		if !wanted[page] {
			continue
		}

		cursor, exists := pageCursors[page]
		if !exists && page < len(pageStarts) {
			cursor = pageStarts[page]
		}

		from, to, next := locateWordSpan(docText, cursor, h.GetText())
		pageCursors[page] = next

		boxes = append(boxes, domain.HighlightBox{
			Page: page,
			ByteSpan: domain.ByteSpan{
				From: from,
				To:   to,
			},
			Rect: getRectFromQuadPoints(h.GetQuadPoints()),
		})
	}
	return boxes
}

func locateWordSpan(docText string, cursor int, word string) (int, int, int) {
	from, to := cursor, cursor+len(word)
	if cursor < len(docText) {
		idx := strings.Index(docText[cursor:], word)
		if idx >= 0 {
			from = cursor + idx
			to = from + len(word)
		}
	}
	return from, to, to
}

func getRectFromQuadPoints(qp []float32) domain.Rect {
	switch {
	case len(qp) == 4:
		return domain.Rect{
			MinX: min(qp[0], qp[2]),
			MinY: min(qp[1], qp[3]),
			MaxX: max(qp[0], qp[2]),
			MaxY: max(qp[1], qp[3]),
		}
	case len(qp) >= 8:
		return domain.Rect{
			MinX: min(qp[0], qp[2], qp[4], qp[6]),
			MinY: min(qp[1], qp[3], qp[5], qp[7]),
			MaxX: max(qp[0], qp[2], qp[4], qp[6]),
			MaxY: max(qp[1], qp[3], qp[5], qp[7]),
		}
	default:
		return domain.Rect{}
	}
}

// OpenDocument implements port.PageRenderer.
func (b *DocumentBridge) OpenDocument(ctx context.Context, file io.ReadSeekCloser) (port.OpenDocument, error) {
	if file == nil {
		return nil, errors.New("nil file")
	}
	defer file.Close()

	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
			return nil, err
		}
	}

	var path string
	var cleanup func()
	if named, ok := file.(interface{ Name() string }); ok {
		validated, err := b.resolveNamedPath(named.Name())
		if err != nil {
			return nil, err
		}
		path = validated
	} else {
		tmpPath, clean, err := bufferTempDocument(file)
		if err != nil {
			return nil, err
		}
		cleanup = clean
		path = tmpPath
	}

	req := connect.NewRequest(&v1.GetDocumentRequest{
		Path: path,
	})
	client, held, err := b.service.open(ctx)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}

	resp, err := client.GetDocument(ctx, req)
	if err != nil {
		_ = held.Close()
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("rpc get document: %w", err)
	}

	return &remoteDocument{
		held:    held,
		service: b.service,
		path:    path,
		pages:   resp.Msg.GetPages(),
		cleanup: cleanup,
	}, nil
}

// remoteDocument is a document the plugin holds open. It holds its own reference on the plugin
// until it is closed, and draws each page at the endpoint the plugin is at then.
type remoteDocument struct {
	held      *pluginLease
	service   pluginService[numenv1connect.DocumentServiceClient]
	path      string
	pages     []*v1.Page
	cleanup   func()
	closeOnce sync.Once
}

func (d *remoteDocument) GetPageCount() int {
	return len(d.pages)
}

func (d *remoteDocument) GetPageSize(index int) (wide, high float64, err error) {
	if index < 0 || index >= len(d.pages) {
		return 0, 0, errors.New("page index out of range")
	}
	p := d.pages[index]
	return p.GetWidth(), p.GetHeight(), nil
}

func (d *remoteDocument) RenderPage(index, dpi int) (image.Image, error) {
	if index < 0 || index > math.MaxInt32 || dpi < 0 || dpi > math.MaxInt32 {
		return nil, errors.New("page index or dpi out of range")
	}
	ctx := context.Background()
	req := connect.NewRequest(&v1.RenderPageRequest{
		Path: d.path,
		Page: int32(index),
		Dpi:  int32(dpi),
	})
	client, call, err := d.service.open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.RenderPage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc render page: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(resp.Msg.GetImageData()))
	if err != nil {
		return nil, fmt.Errorf("decode rendered page image: %w", err)
	}
	return img, nil
}

func (d *remoteDocument) Close() {
	d.closeOnce.Do(func() {
		if d.cleanup != nil {
			d.cleanup()
			d.cleanup = nil
		}
		_ = d.held.Close()
	})
}

func (b *DocumentBridge) resolveNamedPath(rawPath string) (string, error) {
	if b.host == nil || b.host.GetVaultPath() == "" {
		return rawPath, nil
	}
	return b.host.ValidatePluginPath(b.pluginID, rawPath)
}

func bufferTempDocument(file io.Reader) (string, func(), error) {
	tmp, err := os.CreateTemp("", "numen-doc-*.pdf")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.Remove(tmp.Name()) }
	if _, err := io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		cleanup()
		return "", nil, err
	}
	_ = tmp.Close()
	return tmp.Name(), cleanup, nil
}

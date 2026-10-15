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
	"net/http"
	"os"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	// ErrNoDocumentPlugin is returned when no plugin provides document capability.
	ErrNoDocumentPlugin = errors.New("no active document plugin found")
)

// DocumentBridge connects document and text extraction ports to an out-of-process PDF plugin via Connect-RPC.
type DocumentBridge struct {
	host     *Host
	pluginID string
	client   numenv1connect.DocumentServiceClient
}

// OpenDocumentBridge activates or connects to the document plugin and returns a DocumentBridge.
func OpenDocumentBridge(ctx context.Context, host *Host) (*DocumentBridge, error) {
	p, ok := host.FindPluginByCapability("document")
	if !ok {
		return nil, ErrNoDocumentPlugin
	}

	endpoint, err := host.EnsureActive(ctx, p.Manifest.ID)
	if err != nil {
		return nil, fmt.Errorf("ensure active %s: %w", p.Manifest.ID, err)
	}

	httpClient := &http.Client{}
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := numenv1connect.NewDocumentServiceClient(httpClient, endpoint.BaseURL(), interceptor)

	return &DocumentBridge{
		host:     host,
		pluginID: p.Manifest.ID,
		client:   client,
	}, nil
}

// Close releases the active reference count for the document plugin in the host supervisor.
func (b *DocumentBridge) Close() error {
	b.host.Release(b.pluginID)
	return nil
}

// Read implements port.TextExtractor.
func (b *DocumentBridge) Read(ctx context.Context, raw []byte) (domain.TextLayer, error) {
	req := connect.NewRequest(&v1.ExtractTextRequest{
		Data: raw,
	})
	resp, err := b.client.ExtractText(ctx, req)
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
func (b *DocumentBridge) GetHighlights(_ context.Context, _ []byte, _ []int, _ []int) ([]domain.HighlightBox, error) {
	return nil, nil
}

// OpenDocument implements port.PageRenderer.
func (b *DocumentBridge) OpenDocument(ctx context.Context, file io.ReadSeekCloser) (port.OpenDocument, error) {
	if file == nil {
		return nil, errors.New("nil file")
	}
	defer file.Close()

	var path string
	if named, ok := file.(interface{ Name() string }); ok {
		path = named.Name()
	} else {
		tmp, err := os.CreateTemp("", "numen-doc-*.pdf")
		if err != nil {
			return nil, err
		}
		defer os.Remove(tmp.Name())
		if _, err := io.Copy(tmp, file); err != nil {
			_ = tmp.Close()
			return nil, err
		}
		_ = tmp.Close()
		path = tmp.Name()
	}

	req := connect.NewRequest(&v1.GetDocumentRequest{
		Path: path,
	})
	resp, err := b.client.GetDocument(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc get document: %w", err)
	}

	return &remoteDocument{
		client: b.client,
		path:   path,
		pages:  resp.Msg.GetPages(),
	}, nil
}

type remoteDocument struct {
	client numenv1connect.DocumentServiceClient
	path   string
	pages  []*v1.Page
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
	ctx := context.Background()
	req := connect.NewRequest(&v1.RenderPageRequest{
		Path: d.path,
		Page: int32(index),
		Dpi:  int32(dpi),
	})
	resp, err := d.client.RenderPage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc render page: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(resp.Msg.GetImageData()))
	if err != nil {
		return nil, fmt.Errorf("decode rendered page image: %w", err)
	}
	return img, nil
}

func (d *remoteDocument) Close() {}

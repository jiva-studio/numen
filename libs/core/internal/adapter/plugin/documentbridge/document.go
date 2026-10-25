package documentbridge

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
	"path/filepath"
	"strings"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/security"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// Host provides plugin resolution, activation, permissions, and path validation.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
	pluginport.PluginAcquirer
	pluginport.PluginAuthorizer
	pluginport.PluginPathValidator
}

// Document connects document and text extraction ports to an out-of-process PDF plugin via Connect-RPC.
type Document struct {
	host     Host
	pluginID string
	service  bridge.Service[numenv1connect.DocumentServiceClient]
	lease    *bridge.Lease
}

// Open activates or connects to the document plugin and returns a DocumentBridge.
func Open(ctx context.Context, host Host) (*Document, error) {
	p, err := host.ResolveCapability(bridge.CapabilityDocument)
	if err != nil {
		return nil, err
	}

	_, lease, err := bridge.Open(ctx, host, p.Manifest.ID, numenv1connect.NewDocumentServiceClient)
	if err != nil {
		return nil, err
	}

	return &Document{
		host:     host,
		pluginID: p.Manifest.ID,
		service:  bridge.Service[numenv1connect.DocumentServiceClient]{Host: host, PluginID: p.Manifest.ID, NewClient: numenv1connect.NewDocumentServiceClient},
		lease:    lease,
	}, nil
}

// Close releases the active reference count for the document plugin in the host supervisor.
func (b *Document) Close() error {
	return b.lease.Close()
}

// OpenDocument implements port.PageRenderer.
func (b *Document) OpenDocument(ctx context.Context, file io.ReadSeekCloser) (sourceport.OpenDocument, error) {
	if file == nil {
		return nil, errors.New("nil file")
	}
	defer file.Close()

	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionVaultRead); err != nil {
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
	client, lease, err := b.service.Open(ctx)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}

	resp, err := client.GetDocument(ctx, req)
	if err != nil {
		_ = lease.Close()
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("rpc get document: %w", err)
	}

	return &remoteDocument{
		lease:   lease,
		service: b.service,
		path:    path,
		pages:   resp.Msg.GetPages(),
		cleanup: cleanup,
	}, nil
}

// remoteDocument is a document the plugin holds open. It holds its own reference on the plugin
// until it is closed, and draws each page at the endpoint the plugin is at then.
type remoteDocument struct {
	lease     *bridge.Lease
	service   bridge.Service[numenv1connect.DocumentServiceClient]
	path      string
	pages     []*v1.Page
	cleanup   func()
	closeOnce sync.Once
}

func (d *remoteDocument) GetPageCount() int {
	return len(d.pages)
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
	client, call, err := d.service.Open(ctx)
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
		_ = d.lease.Close()
	})
}

func (b *Document) resolveNamedPath(rawPath string) (string, error) {
	clean := filepath.Clean(rawPath)
	if IsTempDocument(clean) {
		return clean, nil
	}
	if b.host == nil {
		return "", security.ErrEmptyVaultRoot
	}
	return b.host.ValidatePluginPath(b.pluginID, rawPath)
}

// tempDocumentPrefix starts the name of a document the host puts in the temp folder.
const tempDocumentPrefix = "numen-doc-"

// IsTempDocument reports whether a clean path is a regular document file directly
// in the system temp directory.
func IsTempDocument(cleanPath string) bool {
	tempDir := filepath.Clean(os.TempDir())
	realTemp, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		realTemp = tempDir
	}

	dir := filepath.Dir(cleanPath)
	if dir != tempDir && dir != realTemp {
		return false
	}
	if !strings.HasPrefix(filepath.Base(cleanPath), tempDocumentPrefix) {
		return false
	}

	if fi, err := os.Lstat(cleanPath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return false
	}

	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err == nil {
		rel, err := filepath.Rel(realTemp, realPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false
		}
	}
	return true
}

func bufferTempDocument(file io.Reader) (string, func(), error) {
	tmp, err := os.CreateTemp("", tempDocumentPrefix+"*.pdf")
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

package plugin

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	ErrNoBookPlugin = errors.New("no active book reader plugin found")
)

// BookPluginHost provides plugin resolution and activation for the book bridge.
type BookPluginHost interface {
	port.PluginResolver
	port.PluginActivator
}

// BookBridge connects BookService calls to an out-of-process book reader plugin via Connect-RPC.
type BookBridge struct {
	host     port.PluginActivator
	pluginID string
	client   numenv1connect.BookServiceClient
	lease    *pluginLease
}

// OpenBookBridge activates or connects to the book plugin and returns a BookBridge.
func OpenBookBridge(ctx context.Context, host BookPluginHost) (*BookBridge, error) {
	p, ok := host.FindPluginByCapability("book")
	if !ok {
		p, ok = host.FindPluginByFileExtension(".epub")
		if !ok {
			return nil, ErrNoBookPlugin
		}
	}

	client, lease, err := openPlugin(ctx, host, p.Manifest.ID, numenv1connect.NewBookServiceClient)
	if err != nil {
		return nil, err
	}

	return &BookBridge{
		host:     host,
		pluginID: p.Manifest.ID,
		client:   client,
		lease:    lease,
	}, nil
}

// GetBook forwards the GetBook RPC to the plugin process.
func (b *BookBridge) GetBook(
	ctx context.Context,
	req *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	return b.client.GetBook(ctx, req)
}

// ReadBookMarkup forwards the ReadBookMarkup RPC to the plugin process.
func (b *BookBridge) ReadBookMarkup(
	ctx context.Context,
	req *connect.Request[v1.ReadBookMarkupRequest],
) (*connect.Response[v1.ReadBookMarkupResponse], error) {
	return b.client.ReadBookMarkup(ctx, req)
}

// Close releases the active reference count for the book plugin in the host supervisor.
func (b *BookBridge) Close() error {
	return b.lease.Close()
}

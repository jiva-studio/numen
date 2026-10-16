package plugin

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

var (
	ErrNoBookPlugin = errors.New("no active book reader plugin found")
)

// BookBridge connects BookService calls to an out-of-process book reader plugin via Connect-RPC.
type BookBridge struct {
	host     *Host
	pluginID string
	client   numenv1connect.BookServiceClient
}

// OpenBookBridge activates or connects to the book plugin and returns a BookBridge.
func OpenBookBridge(ctx context.Context, host *Host) (*BookBridge, error) {
	p, ok := host.FindPluginByCapability("book")
	if !ok {
		p, ok = host.FindPluginByFileExtension(".epub")
		if !ok {
			return nil, ErrNoBookPlugin
		}
	}

	endpoint, err := host.EnsureActive(ctx, p.Manifest.ID)
	if err != nil {
		return nil, fmt.Errorf("ensure active %s: %w", p.Manifest.ID, err)
	}

	httpClient := EndpointClient(endpoint)
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := numenv1connect.NewBookServiceClient(httpClient, endpoint.BaseURL(), interceptor)

	return &BookBridge{
		host:     host,
		pluginID: p.Manifest.ID,
		client:   client,
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
	b.host.Release(b.pluginID)
	return nil
}

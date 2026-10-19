package plugin

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// BookPluginHost provides plugin resolution and activation for the book bridge.
type BookPluginHost interface {
	port.PluginResolver
	port.PluginActivator
	port.PluginAuthorizer
}

// BookBridge connects BookService calls to an out-of-process book reader plugin via Connect-RPC.
type BookBridge struct {
	host     BookPluginHost
	pluginID string
	client   numenv1connect.BookServiceClient
	lease    *pluginLease
}

// OpenBookBridge activates or connects to the book plugin and returns a BookBridge.
func OpenBookBridge(ctx context.Context, host BookPluginHost) (*BookBridge, error) {
	p, err := host.ResolveCapability(CapabilityBook)
	if err != nil {
		return nil, err
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

// ReadBook asks the plugin for the text of the book in the bytes given. Bytes
// that are no book are refused with port.ErrNotADocument, and a locked book with
// port.ErrEncrypted.
func (b *BookBridge) ReadBook(ctx context.Context, raw []byte) (domain.BookText, error) {
	if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
		return domain.BookText{}, err
	}
	resp, err := b.client.ExtractBookText(ctx, connect.NewRequest(&v1.ExtractBookTextRequest{Data: raw}))
	if err != nil {
		return domain.BookText{}, getBookError(err)
	}
	out := domain.BookText{Text: resp.Msg.GetText()}
	for _, part := range resp.Msg.GetParts() {
		out.Parts = append(out.Parts, domain.PartStart{Title: part.GetTitle(), Offset: int(part.GetOffset())})
	}
	for _, page := range resp.Msg.GetPrintedPages() {
		out.Pages = append(out.Pages, domain.BookPage{Label: page.GetLabel(), Offset: int(page.GetOffset())})
	}
	return out, nil
}

// getBookError is the error a refusal of the plugin is answered with: the ports'
// own for a book that cannot be read, an entry it lacks and a file that changed.
func getBookError(err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeInvalidArgument:
		return fmt.Errorf("%w: %v", port.ErrNotADocument, err)
	case connect.CodeFailedPrecondition:
		return fmt.Errorf("%w: %v", port.ErrEncrypted, err)
	case connect.CodeNotFound:
		return fmt.Errorf("%w: %v", port.ErrNoBookEntry, err)
	case connect.CodeAborted:
		return fmt.Errorf("%w: %v", port.ErrBookChanged, err)
	default:
		return fmt.Errorf("rpc book: %w", err)
	}
}

// ReadEntry asks the plugin for one entry of the archive of the book at a path,
// while the file is the bytes the baseline names.
func (b *BookBridge) ReadEntry(ctx context.Context, baseline *v1.Fingerprint, entry string) ([]byte, error) {
	if err := b.host.CheckPluginPermission(b.pluginID, PermissionVaultRead); err != nil {
		return nil, err
	}
	resp, err := b.client.ReadBookEntry(ctx, connect.NewRequest(&v1.ReadBookEntryRequest{
		Path:     baseline.GetPath(),
		Entry:    entry,
		Baseline: baseline,
	}))
	if err != nil {
		return nil, getBookError(err)
	}
	return resp.Msg.GetBody(), nil
}

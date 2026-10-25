package bookbridge

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// Host provides plugin resolution and activation for the book bridge.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
	pluginport.PluginAuthorizer
}

// Book implements port.Book and reads the text of a book over Connect-RPC, against an out-of-process book reader plugin.
type Book struct {
	host     Host
	pluginID string
	service  bridge.Service[numenv1connect.BookServiceClient]
	lease    *bridge.Lease
}

// Open activates or connects to the book plugin and returns a BookBridge.
func Open(ctx context.Context, host Host) (*Book, error) {
	p, err := host.ResolveCapability(bridge.CapabilityBook)
	if err != nil {
		return nil, err
	}

	_, lease, err := bridge.Open(ctx, host, p.Manifest.ID, numenv1connect.NewBookServiceClient)
	if err != nil {
		return nil, err
	}

	return &Book{
		host:     host,
		pluginID: p.Manifest.ID,
		service:  bridge.Service[numenv1connect.BookServiceClient]{Host: host, PluginID: p.Manifest.ID, NewClient: numenv1connect.NewBookServiceClient},
		lease:    lease,
	}, nil
}

// GetBook asks the plugin for the shape of the book at a path.
func (b *Book) GetBook(ctx context.Context, path string) (domain.BookShape, error) {
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return domain.BookShape{}, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.GetBook(ctx, connect.NewRequest(&v1.GetBookRequest{Path: path}))
	if err != nil {
		return domain.BookShape{}, getBookError(err, sourceport.ErrNoBook)
	}
	return newBookShape(resp.Msg), nil
}

// newBookShape is the shape of a book as the plugin answered it.
func newBookShape(msg *v1.GetBookResponse) domain.BookShape {
	out := domain.BookShape{
		Title:       msg.GetTitle(),
		PageCount:   int(msg.GetPageCount()),
		TextBytes:   int(msg.GetTextBytes()),
		PageBytes:   int(msg.GetPageBytes()),
		Fingerprint: newFingerprint(msg.GetFingerprint()),
	}
	for _, section := range msg.GetDocuments() {
		out.Sections = append(out.Sections, domain.BookSection{
			Path: section.GetPath(), Offset: int(section.GetOffset()), Length: int(section.GetLength()),
		})
	}
	for _, part := range msg.GetParts() {
		out.Parts = append(out.Parts, domain.BookPart{
			Title: part.GetTitle(), Offset: int(part.GetOffset()), Level: int(part.GetLevel()),
		})
	}
	for _, page := range msg.GetPrintedPages() {
		out.Pages = append(out.Pages, domain.BookPage{Label: page.GetLabel(), Offset: int(page.GetOffset())})
	}
	return out
}

// newFingerprint is a fingerprint as the schema carries it: the time in
// nanoseconds since the epoch, and zero for a file nothing was said about.
func newFingerprint(wire *v1.Fingerprint) domain.Fingerprint {
	if wire == nil {
		return domain.Fingerprint{}
	}
	out := domain.Fingerprint{Path: wire.GetPath(), Size: wire.GetSize()}
	if wire.GetMtime() != 0 {
		out.ModTime = time.Unix(0, wire.GetMtime())
	}
	return out
}

// newWireFingerprint is a fingerprint as the schema carries it.
func newWireFingerprint(mark domain.Fingerprint) *v1.Fingerprint {
	out := &v1.Fingerprint{Path: mark.Path, Size: mark.Size}
	if !mark.ModTime.IsZero() {
		out.Mtime = mark.ModTime.UnixNano()
	}
	return out
}

// ReadBookMarkup asks the plugin for the markup of one section of the book at a path.
func (b *Book) ReadBookMarkup(
	ctx context.Context, path, section string, baseline *domain.Fingerprint,
) (string, error) {
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = call.Close() }()

	req := &v1.ReadBookMarkupRequest{Path: path, Document: section}
	if baseline != nil {
		req.Baseline = newWireFingerprint(*baseline)
	}
	resp, err := client.ReadBookMarkup(ctx, connect.NewRequest(req))
	if err != nil {
		return "", getBookError(err, sourceport.ErrNoBookEntry)
	}
	return resp.Msg.GetMarkup(), nil
}

// Close releases the active reference count for the book plugin in the host supervisor.
func (b *Book) Close() error {
	return b.lease.Close()
}

// ReadBook asks the plugin for the text of the book in the bytes given. Bytes
// port.ErrEncrypted.
func (b *Book) ReadBook(ctx context.Context, raw []byte) (domain.BookText, error) {
	if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionVaultRead); err != nil {
		return domain.BookText{}, err
	}
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return domain.BookText{}, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.ExtractBookText(ctx, connect.NewRequest(&v1.ExtractBookTextRequest{Data: raw}))
	if err != nil {
		return domain.BookText{}, getBookError(err, sourceport.ErrNoBookEntry)
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

// own for a book that cannot be read, a file that changed, and, as missing, what
// the plugin did not find.
func getBookError(err, missing error) error {
	switch connect.CodeOf(err) {
	case connect.CodeInvalidArgument:
		return fmt.Errorf("%w: %v", sourceport.ErrNotADocument, err)
	case connect.CodeFailedPrecondition:
		return fmt.Errorf("%w: %v", sourceport.ErrEncrypted, err)
	case connect.CodeNotFound:
		return fmt.Errorf("%w: %v", missing, err)
	case connect.CodeAborted:
		return fmt.Errorf("%w: %v", sourceport.ErrBookChanged, err)
	default:
		return fmt.Errorf("rpc book: %w", err)
	}
}

// ReadEntry asks the plugin for one entry of the archive of the book at a path,
// while the file is the bytes the baseline names.
func (b *Book) ReadEntry(ctx context.Context, baseline domain.Fingerprint, entry string) ([]byte, error) {
	if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionVaultRead); err != nil {
		return nil, err
	}
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.ReadBookEntry(ctx, connect.NewRequest(&v1.ReadBookEntryRequest{
		Path:     baseline.Path,
		Entry:    entry,
		Baseline: newWireFingerprint(baseline),
	}))
	if err != nil {
		return nil, getBookError(err, sourceport.ErrNoBookEntry)
	}
	return resp.Msg.GetBody(), nil
}

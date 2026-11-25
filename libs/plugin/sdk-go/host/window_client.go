package host

import (
	"context"
	"errors"
	"fmt"
	"math"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

var _ Window = (*Client)(nil)

func readWindowCallError(err error) error {
	if connect.CodeOf(err) == connect.CodePermissionDenied {
		return fmt.Errorf("%w: %w", ErrPermissionDenied, err)
	}
	return err
}

func readWindowError(code v1.ErrorCode) error {
	switch code {
	case v1.ErrorCode_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.ErrorCode_ERROR_CODE_NO_WINDOW:
		return ErrNoWindow
	case v1.ErrorCode_ERROR_CODE_NOT_FOUND:
		return ErrNotExist
	default:
		return fmt.Errorf("host: rejected with code %s", code)
	}
}

func (c *Client) FocusNote(ctx context.Context, path string) (FocusedNote, error) {
	res, err := c.window.FocusNote(ctx, request(c, &v1.FocusNoteRequest{Path: path}))
	if err != nil {
		return FocusedNote{}, readWindowCallError(err)
	}
	if err := readWindowError(res.Msg.GetError()); err != nil {
		return FocusedNote{}, err
	}
	return FocusedNote{Path: res.Msg.GetPath(), Title: res.Msg.GetTitle(), ID: res.Msg.GetId()}, nil
}

// errSpanOutOfRange is returned for a span whose start or end exceeds 32 bits.
var errSpanOutOfRange = errors.New("host: passage start or length exceeds 32 bits")

func writeByteSpans(spans []ByteSpan) ([]*v1.Range, error) {
	wire := make([]*v1.Range, 0, len(spans))
	for _, span := range spans {
		if !isInt32(span.Start) || !isInt32(span.Length) || !isInt32(span.Start+span.Length) {
			return nil, fmt.Errorf("passage at %d of length %d: %w", span.Start, span.Length, errSpanOutOfRange)
		}
		wire = append(wire, &v1.Range{Start: int32(span.Start), End: int32(span.Start + span.Length)})
	}
	return wire, nil
}

func isInt32(n int) bool { return n >= math.MinInt32 && n <= math.MaxInt32 }

func (c *Client) FocusFile(ctx context.Context, path string, spans []ByteSpan) error {
	wire, err := writeByteSpans(spans)
	if err != nil {
		return err
	}
	res, err := c.window.FocusFile(ctx, request(c, &v1.FocusFileRequest{Path: path, Spans: wire}))
	if err != nil {
		return readWindowCallError(err)
	}
	return readWindowError(res.Msg.GetError())
}

func (c *Client) ReadOpenTabs(ctx context.Context) ([]OpenTab, string, error) {
	res, err := c.window.ReadOpenTabs(ctx, request(c, &v1.HostWindowServiceReadOpenTabsRequest{}))
	if err != nil {
		return nil, "", readWindowCallError(err)
	}
	if err := readWindowError(res.Msg.GetError()); err != nil {
		return nil, "", err
	}
	tabs := make([]OpenTab, 0, len(res.Msg.GetTabs()))
	for _, tab := range res.Msg.GetTabs() {
		tabs = append(tabs, readOpenTab(tab))
	}
	return tabs, res.Msg.GetActiveId(), nil
}

func (c *Client) ActivateTab(ctx context.Context, id string) (OpenTab, string, error) {
	res, err := c.window.ActivateTab(ctx, request(c, &v1.ActivateTabRequest{TabId: id}))
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded && ctx.Err() == nil {
		return OpenTab{}, "", fmt.Errorf("%w: %w", ErrTabNotShown, err)
	}
	if err != nil {
		return OpenTab{}, "", readWindowCallError(err)
	}
	if err := readWindowError(res.Msg.GetError()); err != nil {
		return OpenTab{}, "", err
	}
	return readOpenTab(res.Msg.GetTab()), res.Msg.GetActiveId(), nil
}

func readOpenTab(tab *v1.OpenTab) OpenTab {
	return OpenTab{ID: tab.GetId(), Kind: tab.GetKind(), Path: tab.GetPath(), Title: tab.GetTitle(), Place: tab.GetPlace()}
}

package host

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

var _ Window = (*Client)(nil)

// readWindowCallError converts the failure of a window call into its error: a permission the manifest
// lacks is ErrPermissionDenied, and an argument the host rejects keeps its connect code.
func readWindowCallError(err error) error {
	if connect.CodeOf(err) == connect.CodePermissionDenied {
		return fmt.Errorf("%w: %w", ErrPermissionDenied, err)
	}
	return err
}

func readWindowError(code v1.HostWindowErrorCode) error {
	switch code {
	case v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW:
		return ErrNoWindow
	case v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND:
		return ErrNotExist
	}
	return fmt.Errorf("host: rejected with code %s", code)
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

func (c *Client) FocusPassages(ctx context.Context, path string, passages []PassageSpan) (string, error) {
	spans := make([]*v1.PassageSpan, 0, len(passages))
	for _, passage := range passages {
		spans = append(spans, &v1.PassageSpan{Start: int32(passage.Start), Length: int32(passage.Length)})
	}
	res, err := c.window.FocusPassages(ctx, request(c, &v1.FocusPassagesRequest{Path: path, Passages: spans}))
	if err != nil {
		return "", readWindowCallError(err)
	}
	if err := readWindowError(res.Msg.GetError()); err != nil {
		return "", err
	}
	return res.Msg.GetSummary(), nil
}

func (c *Client) ReadOpenTabs(ctx context.Context) ([]OpenTab, string, error) {
	res, err := c.window.ReadOpenTabs(ctx, request(c, &v1.HostReadOpenTabsRequest{}))
	if err != nil {
		return nil, "", readWindowCallError(err)
	}
	if err := readWindowError(res.Msg.GetError()); err != nil {
		return nil, "", err
	}
	tabs := make([]OpenTab, 0, len(res.Msg.GetTabs()))
	for _, tab := range res.Msg.GetTabs() {
		tabs = append(tabs, OpenTab{ID: tab.GetId(), Kind: tab.GetKind(), Path: tab.GetPath(), Title: tab.GetTitle(), State: tab.GetState()})
	}
	return tabs, res.Msg.GetActiveId(), nil
}

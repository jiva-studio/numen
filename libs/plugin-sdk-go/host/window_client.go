package host

import (
	"context"
	"errors"
)

var _ Window = (*Client)(nil)

var errWindowNotImplemented = errors.New("host: the window calls are not implemented")

func (c *Client) FocusNote(_ context.Context, _ string) (FocusedNote, error) {
	return FocusedNote{}, errWindowNotImplemented
}

func (c *Client) FocusPassages(_ context.Context, _ string, _ []PassageSpan) (string, error) {
	return "", errWindowNotImplemented
}

func (c *Client) ReadOpenTabs(_ context.Context) ([]OpenTab, string, error) {
	return nil, "", errWindowNotImplemented
}

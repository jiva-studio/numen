// Package health serves PluginHealthService to the windows: the health items of every plugin, and the
// items the frontend of a plugin reports about itself.
package health

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

// Items is the health items of every plugin.
type Items interface {
	List() []plugin.HealthItem
	Watch(ctx context.Context) <-chan []plugin.HealthItem
}

// Report puts a health item in the list as the plugin's. It returns plugin.ErrInvalidHealth for an item
// the core refuses.
type Report interface {
	Execute(pluginID string, item plugin.HealthItem) error
}

// Clear removes the health item of the plugin under the key. It returns plugin.ErrInvalidHealth for a
// key the plugin cannot clear.
type Clear interface {
	Execute(pluginID, key string) error
}

// Handler answers PluginHealthService to a window.
type Handler struct {
	items    Items
	report   Report
	clearing Clear
}

var _ numenv1connect.PluginHealthServiceHandler = (*Handler)(nil)

var errNoPlugin = errors.New("the request names no plugin")

// NewHandler answers over the list and the use cases of the frontends.
func NewHandler(items Items, report Report, clearing Clear) *Handler {
	return &Handler{items: items, report: report, clearing: clearing}
}

// NewRoute is where the service answers: its path and the handler of its calls. Each call goes
// through counted.
func NewRoute(handler *Handler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewPluginHealthServiceHandler(handler, counted)
}

// ReportHealth puts the item of the plugin the request names in the list.
func (h *Handler) ReportHealth(
	_ context.Context, req *connect.Request[v1.ReportHealthRequest],
) (*connect.Response[v1.ReportHealthResponse], error) {
	pluginID := req.Msg.GetItem().GetPluginId()
	if pluginID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoPlugin)
	}
	if err := h.report.Execute(pluginID, wire.ReadHealthItem(req.Msg.GetItem())); err != nil {
		return nil, getHealthError(err)
	}
	return connect.NewResponse(&v1.ReportHealthResponse{}), nil
}

// ClearHealth removes the item of the plugin the request names under the key.
func (h *Handler) ClearHealth(
	_ context.Context, req *connect.Request[v1.ClearHealthRequest],
) (*connect.Response[v1.ClearHealthResponse], error) {
	if req.Msg.GetPluginId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoPlugin)
	}
	if err := h.clearing.Execute(req.Msg.GetPluginId(), req.Msg.GetKey()); err != nil {
		return nil, getHealthError(err)
	}
	return connect.NewResponse(&v1.ClearHealthResponse{}), nil
}

// ListHealth returns the items of every plugin.
func (h *Handler) ListHealth(
	context.Context, *connect.Request[v1.ListHealthRequest],
) (*connect.Response[v1.ListHealthResponse], error) {
	return connect.NewResponse(&v1.ListHealthResponse{Items: wire.DescribeHealthItems(h.items.List())}), nil
}

// WatchHealth sends the items at once, on every change and again every wire.ResendInterval, until the
// client is gone. A gone client is noticed through a failed write.
func (h *Handler) WatchHealth(
	ctx context.Context, _ *connect.Request[v1.WatchHealthRequest], out *connect.ServerStream[v1.WatchHealthResponse],
) error {
	watching, stop := context.WithCancel(ctx)
	defer stop()
	watch := h.items.Watch(watching)
	repeat := time.NewTicker(wire.ResendInterval)
	defer repeat.Stop()

	var last []plugin.HealthItem
	for {
		select {
		case list, isOpen := <-watch:
			if !isOpen {
				return nil
			}
			if err := out.Send(&v1.WatchHealthResponse{Items: wire.DescribeHealthItems(list)}); err != nil {
				return err
			}
			last = list
			repeat.Reset(wire.ResendInterval)
		case <-repeat.C:
			if err := out.Send(&v1.WatchHealthResponse{Items: wire.DescribeHealthItems(last)}); err != nil {
				return err
			}
		}
	}
}

func getHealthError(err error) error {
	if errors.Is(err, plugin.ErrInvalidHealth) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// Package theme maps the theme service of the protocol onto the theme use cases.
package theme

import (
	"context"
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

// Service answers what a client may ask about themes.
type Service struct {
	getThemes       *themeusecase.GetThemes
	readTheme       *themeusecase.ReadTheme
	writeAppearance *themeusecase.WriteAppearance
	watchThemes     *themeusecase.WatchThemes
}

// NewService answers the requests of the theme service with the use cases.
func NewService(
	getThemes *themeusecase.GetThemes,
	readTheme *themeusecase.ReadTheme,
	writeAppearance *themeusecase.WriteAppearance,
	watchThemes *themeusecase.WatchThemes,
) *Service {
	return &Service{getThemes: getThemes, readTheme: readTheme, writeAppearance: writeAppearance, watchThemes: watchThemes}
}

// ListThemes returns every theme and the one applied.
func (s *Service) ListThemes(
	_ context.Context,
	_ *connect.Request[v1.ListThemesRequest],
) (*connect.Response[v1.ListThemesResponse], error) {
	answer := s.getThemes.Execute()
	items := make([]*v1.Theme, 0, len(answer.List))
	for _, one := range answer.List {
		items = append(items, &v1.Theme{
			Name:       one.Name,
			Title:      one.Title,
			Origin:     encodeOrigin(one.Origin),
			IsPinned:   one.IsPinned,
			PluginId:   one.PluginID,
			Appearance: one.Appearance,
		})
	}
	return connect.NewResponse(&v1.ListThemesResponse{
		Themes:               items,
		Applied:              answer.Applied,
		Mode:                 wire.ConvertModeToWire(answer.Appearance.Mode),
		InterfaceScale:       answer.Appearance.InterfaceScale,
		TextScale:            answer.Appearance.TextScale,
		InterfaceScaleBounds: encodeBounds(answer.InterfaceScaleBounds),
		TextScaleBounds:      encodeBounds(answer.TextScaleBounds),
	}), nil
}

func encodeBounds(bounds themeusecase.Bounds) *v1.Bounds {
	return &v1.Bounds{Min: bounds.Min, Max: bounds.Max}
}

// ReadTheme returns CSS definitions for a theme. If the theme is not found,
// an empty response is returned.
func (s *Service) ReadTheme(
	_ context.Context,
	req *connect.Request[v1.ReadThemeRequest],
) (*connect.Response[v1.ReadThemeResponse], error) {
	text, err := s.readTheme.Execute(req.Msg.GetName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ReadThemeResponse{Css: text}), nil
}

// WriteAppearance saves theme name, mode, and interface/text scale values.
// Invalid inputs return an error message in the response without updating settings.
func (s *Service) WriteAppearance(
	_ context.Context,
	req *connect.Request[v1.WriteAppearanceRequest],
) (*connect.Response[v1.WriteAppearanceResponse], error) {
	respond := func(reason string) (*connect.Response[v1.WriteAppearanceResponse], error) {
		return connect.NewResponse(&v1.WriteAppearanceResponse{Error: reason}), nil
	}

	name := req.Msg.GetName()
	if req.Msg.GetMode() == v1.Mode_MODE_UNSPECIFIED {
		return respond(name + ": which half of a pair to read was not said")
	}
	selected := themedomain.Appearance{
		ThemeName:      name,
		Mode:           wire.ConvertModeFromWire(req.Msg.GetMode()),
		InterfaceScale: req.Msg.GetInterfaceScale(),
		TextScale:      req.Msg.GetTextScale(),
	}
	if err := s.writeAppearance.Execute(selected); err != nil {
		return respond(err.Error())
	}
	return connect.NewResponse(&v1.WriteAppearanceResponse{}), nil
}

// keepaliveInterval is the interval between empty reports to the client of a watch.
var keepaliveInterval = time.Second

// WatchThemes notifies the client when the themes folder or the set of plugin
// themes changes, until the client disconnects.
func (s *Service) WatchThemes(
	ctx context.Context,
	_ *connect.Request[v1.WatchThemesRequest],
	stream *connect.ServerStream[v1.WatchThemesResponse],
) error {
	// The request context belongs to the process and does not reflect whether the page that opened the
	// stream is still open. A failed write of an empty report is how a gone client is detected.
	watching, stop := context.WithCancel(ctx)
	defer stop()
	var (
		sending sync.Mutex
		gone    = make(chan error, 1)
	)
	send := func(names []string) error {
		sending.Lock()
		defer sending.Unlock()
		return stream.Send(&v1.WatchThemesResponse{Names: names})
	}
	go func() {
		repeat := time.NewTicker(keepaliveInterval)
		defer repeat.Stop()
		for {
			select {
			case <-watching.Done():
				return
			case <-repeat.C:
				if err := send(nil); err != nil {
					gone <- err
					stop()
					return
				}
			}
		}
	}()
	err := s.watchThemes.Execute(watching, send)
	select {
	case failed := <-gone:
		return failed
	default:
	}
	if errors.Is(err, themeusecase.ErrWatchUnavailable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return err
}

func encodeOrigin(origin themedomain.Origin) v1.Origin {
	switch origin {
	case themedomain.Preset:
		return v1.Origin_ORIGIN_PRESET
	case themedomain.Mine:
		return v1.Origin_ORIGIN_MINE
	case themedomain.Plugin:
		return v1.Origin_ORIGIN_PLUGIN
	}
	return v1.Origin_ORIGIN_UNSPECIFIED
}

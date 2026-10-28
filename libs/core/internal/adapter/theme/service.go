package theme

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// Appearance is what the window wears: which theme, which half of a colour pair its
// tokens are read as, and how large it is drawn and its reading text set.
type Appearance struct {
	ThemeName string
	Mode      appearance.ColorScheme

	// InterfaceScale scales UI components and TextScale scales typography.
	// Zero values preserve default scale multipliers.
	InterfaceScale, TextScale float64
}

// Bounds is how far a size goes, at each end. A client asking a person for a
// number says these.
type Bounds struct{ Min, Max float64 }

// AsDesigned is the multiplier that draws everything the size it was drawn at.
const AsDesigned = 1

// Appearances is where what the window wears is kept. Where a choice is
// written down is not this adapter's to know, so the settings arrive as what it
// does with them.
type Appearances interface {
	// Read is what the settings say the window wears.
	Read() (Appearance, error)

	// Write persists the appearance settings. Out-of-bounds scale values are
	// rejected without writing.
	Write(Appearance) error

	// Warn reports non-fatal configuration issues or missing themes.
	Warn(string)
}

// Service answers what a client may ask about themes.
//
// The catalogue is the files, and Settings is where a choice out of them is
// kept. A build with no settings behind it offers what it ships and writes
// nothing.
type Service struct {
	Catalogue Catalogue
	Settings  Appearances

	// InterfaceScaleBounds and TextScaleBounds are how far each of the two sizes
	// goes.
	InterfaceScaleBounds, TextScaleBounds Bounds

	// DebounceDelay is how long a change to the themes folder is kept before it is
	// reported. Zero is DefaultHold.
	DebounceDelay time.Duration
}

// ListThemes is every theme there is, and what the window is wearing out of it.
func (s *Service) ListThemes(
	_ context.Context,
	_ *connect.Request[v1.ListThemesRequest],
) (*connect.Response[v1.ListThemesResponse], error) {
	aged := s.getAppearance()
	applied, missing := s.Catalogue.GetApplied(aged.ThemeName)
	if missing != "" {
		s.warn(fmt.Sprintf("there is no theme called %s, so the window wears %s", missing, applied))
	}

	themes := s.Catalogue.Themes()
	items := make([]*v1.Theme, 0, len(themes))
	for _, one := range themes {
		items = append(items, &v1.Theme{
			Name:     one.Name,
			Title:    one.Title,
			Origin:   encodeOrigin(one.Origin),
			IsPinned: one.IsPinned,
		})
	}
	return connect.NewResponse(&v1.ListThemesResponse{
		Themes:               items,
		Applied:              applied,
		Mode:                 wire.ConvertModeToWire(aged.Mode),
		InterfaceScale:       aged.InterfaceScale,
		TextScale:            aged.TextScale,
		InterfaceScaleBounds: encodeBounds(s.InterfaceScaleBounds),
		TextScaleBounds:      encodeBounds(s.TextScaleBounds),
	}), nil
}

// encodeBounds converts domain Bounds to wire representation.
func encodeBounds(bounds Bounds) *v1.Bounds {
	return &v1.Bounds{Min: bounds.Min, Max: bounds.Max}
}

// ReadTheme returns CSS definitions for a theme. If the theme is not found,
// an empty response is returned.
func (s *Service) ReadTheme(
	_ context.Context,
	req *connect.Request[v1.ReadThemeRequest],
) (*connect.Response[v1.ReadThemeResponse], error) {
	text, err := s.Catalogue.Text(req.Msg.GetName())
	if errors.Is(err, ErrNoTheme) || errors.Is(err, ErrNoFolder) {
		return connect.NewResponse(&v1.ReadThemeResponse{}), nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ReadThemeResponse{Css: text}), nil
}

// WriteAppearance validates and saves theme name, mode, and interface/text scale values.
// Invalid inputs return an error message in the response without updating settings.
func (s *Service) WriteAppearance(
	_ context.Context,
	req *connect.Request[v1.WriteAppearanceRequest],
) (*connect.Response[v1.WriteAppearanceResponse], error) {
	respond := func(reason string) (*connect.Response[v1.WriteAppearanceResponse], error) {
		return connect.NewResponse(&v1.WriteAppearanceResponse{Error: reason}), nil
	}

	name := req.Msg.GetName()
	if _, err := s.Catalogue.Text(name); err != nil {
		return respond(err.Error())
	}
	if req.Msg.GetMode() == v1.Mode_MODE_UNSPECIFIED {
		return respond(fmt.Sprintf("%s: which half of a pair to read was not said", name))
	}
	if s.Settings == nil {
		return respond("this build writes no settings")
	}
	selected := Appearance{
		ThemeName:      name,
		Mode:           wire.ConvertModeFromWire(req.Msg.GetMode()),
		InterfaceScale: req.Msg.GetInterfaceScale(),
		TextScale:      req.Msg.GetTextScale(),
	}
	if err := s.Settings.Write(selected); err != nil {
		return respond(err.Error())
	}
	return connect.NewResponse(&v1.WriteAppearanceResponse{}), nil
}

// WatchThemes reports the themes folder having changed for as long as the
// caller listens.
func (s *Service) WatchThemes(
	ctx context.Context,
	_ *connect.Request[v1.WatchThemesRequest],
	stream *connect.ServerStream[v1.WatchThemesResponse],
) error {
	// The folder watch ends with this call, whichever way it ends.
	watching, stop := context.WithCancel(ctx)
	defer stop()

	changed, err := s.Catalogue.Watch(watching, s.DebounceDelay)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}

	// Naming nothing changed, so that a client that has gone fails the write.
	// The request context belongs to the process and says nothing about the page
	// a stream was opened from.
	repeat := time.NewTicker(keepaliveInterval)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			if err := stream.Send(&v1.WatchThemesResponse{}); err != nil {
				return err
			}
		case names, open := <-changed:
			if !open {
				return nil
			}
			if err := stream.Send(&v1.WatchThemesResponse{Names: names}); err != nil {
				return err
			}
		}
	}
}

// keepaliveInterval is how often the stream says nothing changed. It is the one thing that
// tells a handler its client has gone.
const keepaliveInterval = time.Second

// getAppearance is what the settings say, and this product's own palette at the size it
// was designed at, under the system's choice, where they say nothing or could
// not be read.
func (s *Service) getAppearance() Appearance {
	aged := Appearance{
		ThemeName:      Default,
		Mode:           appearance.System,
		InterfaceScale: AsDesigned,
		TextScale:      AsDesigned,
	}
	if s.Settings == nil {
		return aged
	}
	response, err := s.Settings.Read()
	if err != nil {
		s.warn(fmt.Sprintf("the settings could not be read, so the window wears %s: %v", Default, err))
		return aged
	}
	if response.ThemeName != "" {
		aged.ThemeName = response.ThemeName
	}
	aged.Mode = response.Mode
	if response.InterfaceScale > 0 {
		aged.InterfaceScale = response.InterfaceScale
	}
	if response.TextScale > 0 {
		aged.TextScale = response.TextScale
	}
	return aged
}

func (s *Service) warn(why string) {
	if s.Settings != nil {
		s.Settings.Warn(why)
	}
}

func encodeOrigin(origin Origin) v1.Origin {
	switch origin {
	case Preset:
		return v1.Origin_ORIGIN_PRESET
	case Mine:
		return v1.Origin_ORIGIN_MINE
	}
	return v1.Origin_ORIGIN_UNSPECIFIED
}


// NewAppearances adapts read, write, and warn callbacks into Appearances.
func NewAppearances(read func() (Appearance, error), write func(Appearance) error, warn func(string)) Appearances {
	return callbackAppearances{read: read, write: write, warn: warn}
}

type callbackAppearances struct {
	read  func() (Appearance, error)
	write func(Appearance) error
	warn  func(string)
}

func (a callbackAppearances) Read() (Appearance, error) { return a.read() }
func (a callbackAppearances) Write(choice Appearance) error { return a.write(choice) }
func (a callbackAppearances) Warn(why string) {
	if a.warn != nil {
		a.warn(why)
	}
}

package theme

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/appearance"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// Appearance is what the window wears: which theme, which half of a colour pair its
// tokens are read as, and how large it is drawn and its reading text set.
type Appearance struct {
	ThemeName string
	Mode      appearance.ColorScheme

	// InterfaceScale is how large the window is drawn and TextScale how large the
	// text a person reads is set, AsDesigned being the size each was designed at.
	// Zero is a size nobody named: in a choice it stands as it is, and in what is
	// worn the tokens hold their own.
	InterfaceScale, TextScale float64
}

// Bounds is how far a size goes, at each end. A client asking a person for a
// number says these.
type Bounds struct{ Least, Most float64 }

// AsDesigned is the multiplier that draws everything the size it was drawn at.
const AsDesigned = 1

// Appearances is where what the window wears is kept. Where a choice is
// written down is not this adapter's to know, so the settings arrive as what it
// does with them.
type Appearances interface {
	// Read is what the settings say the window wears.
	Read() (Appearance, error)

	// Write puts a dress into the settings, and leaves the rest of them as they
	// are. A number outside the bounds of the size it is written into is
	// refused and nothing is written.
	Write(Appearance) error

	// Warn is where a person is told what this could not do: a theme named in
	// the settings that is not in the catalogue, and settings that could not be
	// read.
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

	// Hold is how long a change to the themes folder is kept before it is
	// reported. Zero is DefaultHold.
	Hold time.Duration
}

// ListThemes is every theme there is, and what the window is wearing out of it.
func (s *Service) ListThemes(
	_ context.Context,
	_ *connect.Request[v1.ListThemesRequest],
) (*connect.Response[v1.ListThemesResponse], error) {
	worn := s.getAppearance()
	applied, missing := s.Catalogue.GetApplied(worn.ThemeName)
	if missing != "" {
		s.say(fmt.Sprintf("there is no theme called %s, so the window wears %s", missing, applied))
	}

	themes := s.Catalogue.Themes()
	listed := make([]*v1.Theme, 0, len(themes))
	for _, one := range themes {
		listed = append(listed, &v1.Theme{
			Name:   one.Name,
			Title:  one.Title,
			Shelf:  encodeShelf(one.Shelf),
			Pinned: one.Pinned,
		})
	}
	return connect.NewResponse(&v1.ListThemesResponse{
		Themes:               listed,
		Applied:              applied,
		Mode:                 wire.ModeOf(worn.Mode),
		InterfaceScale:       worn.InterfaceScale,
		TextScale:            worn.TextScale,
		InterfaceScaleBounds: encodeBounds(s.InterfaceScaleBounds),
		TextScaleBounds:      encodeBounds(s.TextScaleBounds),
	}), nil
}

// encodeBounds is how far a size goes, as the schema says it.
func encodeBounds(held Bounds) *v1.Bounds {
	return &v1.Bounds{Least: held.Least, Most: held.Most}
}

// ReadTheme is one theme's file, as the file stands. A name this build ships
// no file for and holds none under is no theme, and is answered with no CSS:
// the page wears what it already has, and nobody is shown a failure they did
// not ask for.
func (s *Service) ReadTheme(
	_ context.Context,
	req *connect.Request[v1.ReadThemeRequest],
) (*connect.Response[v1.ReadThemeResponse], error) {
	text, err := s.Catalogue.Text(req.Msg.GetName())
	if err != nil {
		//nolint:nilerr // a name with no file is no theme, and no CSS is the answer to it
		return connect.NewResponse(&v1.ReadThemeResponse{}), nil
	}
	return connect.NewResponse(&v1.ReadThemeResponse{Css: text}), nil
}

// WriteAppearance writes the theme, the mode and the two sizes into the
// settings.
//
// A theme that is not there, a mode that was not said and a size outside what
// it goes to are all refused, and the settings are left as they are. What
// stopped the write is answered with: the person is standing in front of the
// list they chose from. A size the choice does not name stands as it is.
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
	chosen := Appearance{
		ThemeName:      name,
		Mode:           wire.ModeIn(req.Msg.GetMode()),
		InterfaceScale: req.Msg.GetInterfaceScale(),
		TextScale:      req.Msg.GetTextScale(),
	}
	if err := s.Settings.Write(chosen); err != nil {
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
	changed, err := s.Catalogue.Watch(ctx, s.Hold)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}

	// Naming nothing changed, so that a client that has gone fails the write.
	// The request context belongs to the process and says nothing about the page
	// a stream was opened from.
	repeat := time.NewTicker(again)
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

// again is how often the stream says nothing changed. It is the one thing that
// tells a handler its client has gone.
const again = time.Second

// getAppearance is what the settings say, and this product's own palette at the size it
// was designed at, under the system's choice, where they say nothing or could
// not be read.
func (s *Service) getAppearance() Appearance {
	worn := Appearance{
		ThemeName:      Default,
		Mode:           appearance.System,
		InterfaceScale: AsDesigned,
		TextScale:      AsDesigned,
	}
	if s.Settings == nil {
		return worn
	}
	said, err := s.Settings.Read()
	if err != nil {
		s.say(fmt.Sprintf("the settings could not be read, so the window wears %s: %v", Default, err))
		return worn
	}
	if said.ThemeName != "" {
		worn.ThemeName = said.ThemeName
	}
	worn.Mode = said.Mode
	if said.InterfaceScale > 0 {
		worn.InterfaceScale = said.InterfaceScale
	}
	if said.TextScale > 0 {
		worn.TextScale = said.TextScale
	}
	return worn
}

func (s *Service) say(why string) {
	if s.Settings != nil {
		s.Settings.Warn(why)
	}
}

func encodeShelf(shelf Shelf) v1.Shelf {
	switch shelf {
	case Preset:
		return v1.Shelf_SHELF_PRESET
	case Mine:
		return v1.Shelf_SHELF_MINE
	}
	return v1.Shelf_SHELF_UNSPECIFIED
}

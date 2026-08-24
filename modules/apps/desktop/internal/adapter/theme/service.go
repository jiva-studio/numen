package theme

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Dress is what the window wears: which theme, and which half of a colour pair
// its tokens are read as.
type Dress struct {
	Theme string
	Mode  v1.Mode
}

// Service answers what a client may ask about themes.
//
// The catalogue is the files. Where a choice is written down is not this
// adapter's to know, so the settings arrive as the two things it does with
// them.
type Service struct {
	Catalogue Catalogue

	// Dressed is what the settings say the window wears.
	Dressed func() (Dress, error)

	// Wear writes a dress into the settings, and leaves the rest of them as
	// they are.
	Wear func(Dress) error

	// Say is where a person is told what this could not do: a theme named in
	// the settings that is not in the catalogue, and settings that could not be
	// read.
	Say func(string)

	// Hold is how long a change to the themes folder is kept before it is
	// reported. Zero is DefaultHold.
	Hold time.Duration
}

// Themes is every theme there is, and what the window is wearing out of it.
func (s *Service) Themes(
	_ context.Context,
	_ *connect.Request[v1.ThemesRequest],
) (*connect.Response[v1.ThemesResponse], error) {
	worn := s.worn()
	applied, missing := s.Catalogue.Applied(worn.Theme)
	if missing != "" {
		s.say(fmt.Sprintf("there is no theme called %s, so the window wears %s", missing, applied))
	}

	themes := s.Catalogue.Themes()
	listed := make([]*v1.Theme, 0, len(themes))
	for _, one := range themes {
		listed = append(listed, &v1.Theme{
			Name:   one.Name,
			Title:  one.Title,
			Shelf:  shelved(one.Shelf),
			Pinned: one.Pinned,
		})
	}
	return connect.NewResponse(&v1.ThemesResponse{
		Themes:  listed,
		Applied: applied,
		Mode:    worn.Mode,
	}), nil
}

// Theme is one theme's file, as the file stands.
func (s *Service) Theme(
	_ context.Context,
	req *connect.Request[v1.ThemeRequest],
) (*connect.Response[v1.ThemeResponse], error) {
	text, err := s.Catalogue.Text(req.Msg.GetName())
	if err != nil {
		return connect.NewResponse(&v1.ThemeResponse{}), nil
	}
	return connect.NewResponse(&v1.ThemeResponse{Css: text}), nil
}

// Choose writes the theme and the mode into the settings.
//
// A theme that is not there and a mode that was not said are both refused, and
// the settings are left as they are. What stopped the write is answered with:
// the person is standing in front of the list they chose from.
func (s *Service) Choose(
	_ context.Context,
	req *connect.Request[v1.ChooseRequest],
) (*connect.Response[v1.ChooseResponse], error) {
	failed := func(why string) (*connect.Response[v1.ChooseResponse], error) {
		return connect.NewResponse(&v1.ChooseResponse{Failed: why}), nil
	}

	name := req.Msg.GetName()
	if _, err := s.Catalogue.Text(name); err != nil {
		return failed(err.Error())
	}
	if req.Msg.GetMode() == v1.Mode_MODE_UNSPECIFIED {
		return failed(fmt.Sprintf("%s: which half of a pair to read was not said", name))
	}
	if s.Wear == nil {
		return failed("this build writes no settings")
	}
	if err := s.Wear(Dress{Theme: name, Mode: req.Msg.GetMode()}); err != nil {
		return failed(err.Error())
	}
	return connect.NewResponse(&v1.ChooseResponse{}), nil
}

// Changed reports the themes folder having changed for as long as the caller
// listens.
func (s *Service) Changed(
	ctx context.Context,
	_ *connect.Request[v1.ChangedRequest],
	stream *connect.ServerStream[v1.ChangedResponse],
) error {
	changed, err := s.Catalogue.Watching(ctx, s.Hold)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case names, open := <-changed:
			if !open {
				return nil
			}
			if err := stream.Send(&v1.ChangedResponse{Names: names}); err != nil {
				return err
			}
		}
	}
}

// worn is what the settings say, and this product's own palette under the
// system's choice where they say nothing or could not be read.
func (s *Service) worn() Dress {
	worn := Dress{Theme: Default, Mode: v1.Mode_MODE_SYSTEM}
	if s.Dressed == nil {
		return worn
	}
	said, err := s.Dressed()
	if err != nil {
		s.say(fmt.Sprintf("the settings could not be read, so the window wears %s: %v", Default, err))
		return worn
	}
	if said.Theme != "" {
		worn.Theme = said.Theme
	}
	if said.Mode != v1.Mode_MODE_UNSPECIFIED {
		worn.Mode = said.Mode
	}
	return worn
}

func (s *Service) say(why string) {
	if s.Say != nil {
		s.Say(why)
	}
}

func shelved(shelf Shelf) v1.Shelf {
	switch shelf {
	case Preset:
		return v1.Shelf_SHELF_PRESET
	case Mine:
		return v1.Shelf_SHELF_MINE
	}
	return v1.Shelf_SHELF_UNSPECIFIED
}

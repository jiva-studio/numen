package editor

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetWindow is this window, for whatever asks for a place to be put in front of
// the person.
func (s *State) GetWindow() port.Window { return view{s} }

// view tells the clients and nothing more. What travelling there looks
// like is theirs, and a place asked for while nobody is drawing is a place
// nobody sees.
type view struct{ *State }

func (v view) FocusPlace(_ context.Context, pos domain.Place) error {
	v.Places.publish(pos)
	return nil
}

func (v view) ShowMove(_ context.Context, went domain.Move) error {
	v.ChangeEvents.publish(change{renamed: []domain.Move{went}})
	return nil
}

func (v view) ShowEdit(_ context.Context, edit domain.Edit) error {
	v.Edits.publish(edit)
	return nil
}

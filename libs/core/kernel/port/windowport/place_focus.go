package windowport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// PlaceFocus asks the window to show a place.
type PlaceFocus interface {
	FocusPlace(ctx context.Context, pos domain.Place) error
}

package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// TellMove is told where a note went, for whoever is showing it.
//
// A person reading a note that is renamed under them is reading a name with no
// file behind it, and every change made to that note afterwards is made
// somewhere they are not looking. Nothing is told where nobody is drawing.
type TellMove func(ctx context.Context, went domain.Went)

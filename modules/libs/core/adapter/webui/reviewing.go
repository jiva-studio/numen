package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// dayStarts is the hour a day of review begins at in a build that reads no
// settings.
var dayStarts = history.Clock(history.DayStarts)

// Reviewing is the hour a day of review begins at, on the clock on the wall.
func (a *API) Reviewing(
	_ context.Context, _ *connect.Request[v1.ReviewingRequest],
) (*connect.Response[v1.ReviewingResponse], error) {
	starts := dayStarts
	if a.Reviews != nil {
		starts = a.Reviews()
	}
	return connect.NewResponse(&v1.ReviewingResponse{DayStarts: starts}), nil
}

// ChooseReviewing writes that hour into the file a person configures this
// installation in. An hour the setting does not take leaves the file as it is.
func (a *API) ChooseReviewing(
	_ context.Context, r *connect.Request[v1.ChooseReviewingRequest],
) (*connect.Response[v1.ChooseReviewingResponse], error) {
	if a.ChoosesReviewing == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := a.ChoosesReviewing(r.Msg.GetDayStarts()); err != nil {
		// An hour that is not an hour of the day is the client's to correct.
		if errors.Is(err, history.ErrNotAnHour) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		refusal, refused := wire.RefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ChooseReviewingResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.ChooseReviewingResponse{}), nil
}

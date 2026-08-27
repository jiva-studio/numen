package webui

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Hanging says whether a node in the plex hangs the headings of its note under
// the box.
func (a *API) Hanging(
	_ context.Context, _ *connect.Request[v1.HangingRequest],
) (*connect.Response[v1.HangingResponse], error) {
	return connect.NewResponse(&v1.HangingResponse{
		HangPartsUnderANode: a.Hangs == nil || a.Hangs(),
	}), nil
}

// ChooseHanging writes that setting into the file a person configures this
// installation in.
func (a *API) ChooseHanging(
	_ context.Context, r *connect.Request[v1.ChooseHangingRequest],
) (*connect.Response[v1.ChooseHangingResponse], error) {
	if a.ChoosesHanging == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := a.ChoosesHanging(r.Msg.GetHangPartsUnderANode()); err != nil {
		refusal, refused := refusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ChooseHangingResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.ChooseHangingResponse{}), nil
}

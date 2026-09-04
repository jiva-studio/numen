package webui

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/refusal"
)

// partsUnderANode is how many headings stand under a node in a build that reads
// no settings.
const partsUnderANode = 6

// Hanging says whether a node in the plex hangs the headings of its note under
// the box, and how many of them stand there at once.
func (a *API) Hanging(
	_ context.Context, _ *connect.Request[v1.HangingRequest],
) (*connect.Response[v1.HangingResponse], error) {
	parts := partsUnderANode
	if a.Configuring.Parts != nil {
		parts = a.Configuring.Parts()
	}
	return connect.NewResponse(&v1.HangingResponse{
		HangPartsUnderANode: a.Configuring.Hangs == nil || a.Configuring.Hangs(),
		PartsUnderANode:     int32(parts),
	}), nil
}

// ChooseHanging writes those settings into the file a person configures this
// installation in. The count is written where the request names one.
func (a *API) ChooseHanging(
	_ context.Context, r *connect.Request[v1.ChooseHangingRequest],
) (*connect.Response[v1.ChooseHangingResponse], error) {
	if a.Configuring.ChoosesHanging == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := a.Configuring.ChoosesHanging(r.Msg.GetHangPartsUnderANode()); err != nil {
		return refusedHanging(err)
	}
	if r.Msg.PartsUnderANode == nil {
		return connect.NewResponse(&v1.ChooseHangingResponse{}), nil
	}
	if a.Configuring.ChoosesParts == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := a.Configuring.ChoosesParts(int(r.Msg.GetPartsUnderANode())); err != nil {
		return refusedHanging(err)
	}
	return connect.NewResponse(&v1.ChooseHangingResponse{}), nil
}

// refusedHanging is what a write of one of the two settings comes back as when
// it did not happen.
func refusedHanging(err error) (*connect.Response[v1.ChooseHangingResponse], error) {
	reason, refused := refusal.By(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ChooseHangingResponse{Refusal: &reason}), nil
}

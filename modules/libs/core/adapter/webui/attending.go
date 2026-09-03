package webui

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Attending takes what the person has open, as the window last said it.
//
// It is the other direction to Focus: a place is put in front of the person
// there, and here the window says what is in front of them now.
func (a *API) Attending(
	_ context.Context, r *connect.Request[v1.AttendingRequest],
) (*connect.Response[v1.AttendingResponse], error) {
	told := r.Msg.GetTabs()
	open := domain.Attention{Tabs: make([]domain.Tab, 0, len(told)), FrontID: r.Msg.GetFront()}
	for _, one := range told {
		open.Tabs = append(open.Tabs, domain.Tab{
			ID:    one.GetId(),
			Kind:  one.GetKind(),
			Path:  one.GetPath(),
			Title: one.GetTitle(),
			At:    int(one.GetAt()),
			Of:    int(one.GetOf()),
		})
	}
	a.attention.Store(&open)
	if a.Attends != nil {
		a.Attends(open)
	}
	return connect.NewResponse(&v1.AttendingResponse{}), nil
}

// Attended is what the person has open, as the window last said. A window that
// has said nothing has nothing open as far as anyone here knows.
func (a *API) Attended() domain.Attention {
	if open := a.attention.Load(); open != nil {
		return *open
	}
	return domain.Attention{}
}

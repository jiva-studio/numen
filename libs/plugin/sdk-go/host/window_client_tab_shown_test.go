package host_test

import (
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

func TestClientActivateTabReturnsErrTabNotShownOnHostDeadline(t *testing.T) {
	c := dialActivate(t, &activateHost{status: connect.NewError(connect.CodeDeadlineExceeded, errors.New(`activate tab "tab-4": the window did not show the tab in time`))})

	_, _, err := c.ActivateTab(t.Context(), "tab-4")
	if !errors.Is(err, host.ErrTabNotShown) {
		t.Errorf("ActivateTab got %v, want ErrTabNotShown", err)
	}
}

func TestClientActivateTabReturnsOtherHostErrorsAsIs(t *testing.T) {
	c := dialActivate(t, &activateHost{status: connect.NewError(connect.CodeInternal, errors.New("the window broke"))})

	_, _, err := c.ActivateTab(t.Context(), "tab-4")
	if err == nil || errors.Is(err, host.ErrTabNotShown) {
		t.Errorf("ActivateTab got %v, want the internal error alone", err)
	}
}

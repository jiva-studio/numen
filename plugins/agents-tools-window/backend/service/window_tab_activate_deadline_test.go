package service_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"

	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
)

// deadlineWindow is a window whose ActivateTab returns what the client of the host returns when the
// host returns a deadline: ErrTabNotShown over the deadline error with its code.
type deadlineWindow struct {
	*hosttest.Window
}

func (w deadlineWindow) ActivateTab(_ context.Context, id string) (host.OpenTab, string, error) {
	deadline := fmt.Errorf("deadline_exceeded: editor: activate tab %q: the window did not show the tab in time", id)
	return host.OpenTab{}, "", fmt.Errorf("%w: %w", host.ErrTabNotShown, deadline)
}

func TestWindowTabActivateReturnsAHostDeadlineAsText(t *testing.T) {
	vault := hosttest.New(testVault)
	window := hosttest.NewWindow(vault)
	window.SetOpenTabs(getGuideTabs(), "tab-1")
	w := windowTools{window: window, registry: newRegistry(t, service.Host{Vault: vault, Window: deadlineWindow{Window: window}})}

	failure := w.call(t, "window_tab_activate", `{"tabId":"tab-2"}`, nil, nil)

	if want := `window_tab_activate: the window did not show tab "tab-2" within 2 s`; failure != want {
		t.Errorf("the tool error is %q, want %q", failure, want)
	}
	if strings.Contains(failure, "deadline_exceeded") {
		t.Errorf("the tool error %q has the code of the host", failure)
	}
}

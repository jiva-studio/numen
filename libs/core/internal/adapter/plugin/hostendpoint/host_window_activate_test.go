package hostendpoint

import (
	"fmt"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestHostWindowActivateTabAnswersTheTabAndTheActiveID(t *testing.T) {
	window := &recordingWindow{}
	client, fakeCallers := newWindowClient(t, window, "token-a")

	got, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-1"}))
	if err != nil {
		t.Fatal(err)
	}
	tab := got.Msg.GetTab()
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED || got.Msg.GetActiveId() != "tab-2" {
		t.Errorf("answered %v, want no error and the active id tab-2 the window returned", got.Msg)
	}
	if tab.GetId() != "tab-1" || tab.GetKind() != "note" || tab.GetPath() != "notes/Heron.md" || tab.GetTitle() != "Grey Heron" {
		t.Errorf("answered the tab %v, want tab-1 of notes/Heron.md", tab)
	}
	if activated := window.getActivated(); !slices.Equal(activated, []string{"tab-1"}) {
		t.Errorf("the window was asked to activate %v, want [tab-1]", activated)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus asked once", fakeCallers.checked)
	}

	reader, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-2"}))
	if err != nil {
		t.Fatal(err)
	}
	if reader.Msg.GetTab().GetId() != "tab-2" || reader.Msg.GetTab().GetState() != `{"page":4}` || reader.Msg.GetActiveId() != "tab-2" {
		t.Errorf("answered %v, want tab-2 with its state, active", reader.Msg)
	}
}

func TestHostWindowActivateTabAnswersNotFoundForAnIDNoTabHas(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-a")

	got, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-9"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND || got.Msg.GetTab() != nil || got.Msg.GetActiveId() != "" {
		t.Errorf("answered %v, want not found and no tab", got.Msg)
	}
	if activated := window.getActivated(); len(activated) != 0 {
		t.Errorf("the window activated %v", activated)
	}
}

func TestHostWindowActivateTabNeedsWindowFocus(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-b")

	_, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-1"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("ActivateTab from a plugin without window:focus got %v, want permission denied", err)
	}
	if activated := window.getActivated(); len(activated) != 0 {
		t.Errorf("the window activated %v for a call that was denied", activated)
	}
}

func TestHostWindowActivateTabWithoutAWindowAnswersNoWindow(t *testing.T) {
	for name, window := range map[string]pluginport.PluginWindow{
		"no window bound":                 nil,
		"a window that reports no window": &recordingWindow{err: fmt.Errorf("activate: %w", pluginport.ErrNoWindow)},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := newWindowClient(t, window, "token-a")
			got := readWindowCode(client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-1"})))
			if got != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW.String() {
				t.Errorf("ActivateTab answered %s, want no window", got)
			}
		})
	}
}

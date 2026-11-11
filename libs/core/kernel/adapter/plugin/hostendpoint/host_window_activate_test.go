package hostendpoint

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestHostWindowActivateTabReturnsTabPlaceAndActiveID(t *testing.T) {
	window := &recordingWindow{}
	client, fakeCallers := newWindowClient(t, window, "token-a")

	got, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-1"}))
	if err != nil {
		t.Fatal(err)
	}
	tab := got.Msg.GetTab()
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED || got.Msg.GetActiveId() != "tab-2" {
		t.Errorf("returned %v, want no error and the active id tab-2 the window returned", got.Msg)
	}
	if tab.GetId() != "tab-1" || tab.GetKind() != "note" || tab.GetPath() != "notes/Heron.md" || tab.GetTitle() != "Gray Heron" {
		t.Errorf("returned the tab %v, want tab-1 of notes/Heron.md", tab)
	}
	if activated := window.getActivatedIDs(); !slices.Equal(activated, []string{"tab-1"}) {
		t.Errorf("the window received a call to activate %v, want [tab-1]", activated)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus checked once", fakeCallers.checked)
	}

	reader, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-2"}))
	if err != nil {
		t.Fatal(err)
	}
	readerTab := reader.Msg.GetTab()
	if readerTab.GetId() != "tab-2" || readerTab.GetPlace().GetPage() != 4 || readerTab.GetPlace().GetPageCount() != 80 || reader.Msg.GetActiveId() != "tab-2" {
		t.Errorf("returned %v, want tab-2 at page 4 of 80, active", reader.Msg)
	}
	requireOpenTabMessage(t, (&v1.ActivateTabResponse{}).ProtoReflect().Descriptor(), "tab")
}

func TestHostWindowActivateTabReturnsNotFoundForUnknownID(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-a")

	got, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-9"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND || got.Msg.GetTab() != nil || got.Msg.GetActiveId() != "" {
		t.Errorf("returned %v, want not found and no tab", got.Msg)
	}
	if activated := window.getActivatedIDs(); len(activated) != 0 {
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
	if activated := window.getActivatedIDs(); len(activated) != 0 {
		t.Errorf("the window activated %v for a call that was denied", activated)
	}
}

func TestHostWindowActivateTabWithoutAWindowReturnsNoWindow(t *testing.T) {
	for name, window := range map[string]WindowControl{
		"no window bound":                 nil,
		"a window that reports no window": &recordingWindow{err: fmt.Errorf("activate: %w", plugin.ErrNoWindow)},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := newWindowClient(t, window, "token-a")
			got := readWindowCode(client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-1"})))
			if got != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW.String() {
				t.Errorf("ActivateTab returned %s, want no window", got)
			}
		})
	}
}

func TestHostWindowActivateTabReturnsDeadlineWhenTabNotShown(t *testing.T) {
	window := &recordingWindow{err: fmt.Errorf("activate tab %q: %w", "tab-1", plugin.ErrTabNotShown)}
	client, _ := newWindowClient(t, window, "token-a")

	got, err := client.ActivateTab(t.Context(), connect.NewRequest(&v1.ActivateTabRequest{TabId: "tab-1"}))
	if connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("ActivateTab returned %v (%v), want deadline exceeded", got, err)
	}
	if !strings.Contains(err.Error(), `"tab-1"`) || !strings.Contains(err.Error(), plugin.ErrTabNotShown.Error()) {
		t.Errorf("ActivateTab got %v, want the message of the window naming tab-1", err)
	}
}
